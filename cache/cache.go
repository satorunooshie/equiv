// Package cache provides semantic-key caches with expiration and loading.
package cache

import (
	"context"
	"errors"
	"hash/maphash"
	"iter"
	"slices"
	"sync"
	"time"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/internal/clock"
)

// Policy selects the cache replacement policy.
type Policy uint8

const (
	FIFO Policy = iota
	LRU
	MRU
	LFU
	SLRU
	TwoQ
	ARC
	Clock
	WTinyLFU
)

// Admission selects admission control for new cache entries.
type Admission uint8

const (
	AdmitAlways Admission = iota
	AdmitTinyLFU
)

// Weigher returns a strictly positive resident weight for an entry.
type Weigher[K, V any] func(K, V) uint64

// Loader computes a value for a cache miss.
type Loader[K, V any] func(context.Context, K) (V, error)

// Stats contains cache operation counters.
type Stats struct{ Hits, Misses, Sets, Updates, Evictions, Expirations, Rejections, LoadSuccesses, LoadErrors, LoadCoalesced uint64 }

// EventType identifies a cache mutation event.
type EventType uint8

const (
	EventSet EventType = iota
	EventUpdate
	EventDelete
	EventEvict
	EventExpire
	EventReject
)

// Event describes a cache mutation delivered after internal locks are released.
type Event[K, V any] struct {
	Type        EventType
	Key         K
	Value       V
	Previous    V
	HasPrevious bool
	Weight      uint64
}

// Config controls cache capacity, policy, expiration, loading, and events.
// A cache must be constructed with New, NewSync, or NewSharded.
type Config[K, V any] struct {
	Policy                              Policy
	Admission                           Admission
	MaxEntries                          int
	MaxWeight                           uint64
	Weigher                             Weigher[K, V]
	DefaultTTL, DefaultTTI, LoadTimeout time.Duration
	Shards                              int
	Observer                            func(Event[K, V])
}
type item[K, V any] struct {
	k                   K
	v                   V
	weight              uint64
	freq                uint64
	segment             uint8
	ref                 bool
	expires, timeToIdle time.Time
	last                time.Time
}

// Cache is a non-concurrent semantic cache with synchronous loading.
// Its zero value is invalid and it must not be copied after first use.
type Cache[K, V any] struct {
	h         maphash.Hasher[K]
	seed      maphash.Seed
	m         *equiv.Map[K, item[K, V]]
	cfg       Config[K, V]
	items     []K
	weightSum uint64
	stats     Stats
	mu        sync.Mutex
	loads     map[uint64][]*load[K, V]
	freq      map[uint64]uint64
	pending   []Event[K, V]
	clockHand int
	clock     clock.Clock
	ghost1    map[uint64]struct{}
	ghost2    map[uint64]struct{}
	arcP      int
	twoQGhost map[uint64]struct{}
	accesses  uint64
}
type load[K, V any] struct {
	done chan struct{}
	v    V
	err  error
	key  K
}

func validate[K, V any](h maphash.Hasher[K], c Config[K, V]) error {
	if h == nil {
		panic("cache: nil Hasher")
	}
	if (c.MaxEntries > 0) == (c.MaxWeight > 0) {
		return errors.New("cache: exactly one capacity is required")
	}
	if c.MaxWeight > 0 && c.Weigher == nil {
		return errors.New("cache: Weigher is required")
	}
	if c.Shards < 0 {
		return errors.New("cache: invalid shard count")
	}
	if c.Policy > WTinyLFU || c.Admission > AdmitTinyLFU {
		return errors.New("cache: invalid policy or admission")
	}
	if c.Policy == ARC && c.Admission == AdmitTinyLFU {
		return errors.New("cache: ARC cannot use independent TinyLFU admission")
	}
	if c.Policy == ARC && c.MaxWeight > 0 {
		return errors.New("cache: ARC is entry-count based")
	}
	if c.Policy == WTinyLFU && c.Admission != AdmitAlways {
		return errors.New("cache: WTinyLFU owns admission")
	}
	if c.Shards != 0 && c.MaxEntries > 0 && c.Shards > c.MaxEntries {
		return errors.New("cache: too many shards")
	}
	return nil
}

// New constructs a non-concurrent cache. Exactly one of MaxEntries and
// MaxWeight must be positive.
func New[K, V any](h maphash.Hasher[K], cfg Config[K, V]) (*Cache[K, V], error) {
	if e := validate(h, cfg); e != nil {
		return nil, e
	}
	if cfg.Shards != 0 {
		return nil, errors.New("cache: Shards is only valid for NewSharded")
	}
	return &Cache[K, V]{h: h, seed: maphash.MakeSeed(), m: equiv.NewMap[K, item[K, V]](h), cfg: cfg, loads: map[uint64][]*load[K, V]{}, freq: map[uint64]uint64{}, ghost1: map[uint64]struct{}{}, ghost2: map[uint64]struct{}{}, twoQGhost: map[uint64]struct{}{}, clock: clock.Real{}}, nil
}
func (c *Cache[K, V]) now() time.Time { return c.clock.Now() }
func (c *Cache[K, V]) expired(x item[K, V], now time.Time) bool {
	return (!x.expires.IsZero() && !now.Before(x.expires)) || (!x.timeToIdle.IsZero() && !now.Before(x.timeToIdle))
}

func (c *Cache[K, V]) weight(k K, v V) uint64 {
	if c.cfg.MaxWeight == 0 {
		return 1
	}
	w := c.cfg.Weigher(k, v)
	if w == 0 {
		panic("cache: Weigher returned zero")
	}
	return w
}

func (c *Cache[K, V]) remove(k K, ev EventType) bool {
	x, ok := c.m.Get(k)
	if !ok {
		return false
	}
	if ev == EventEvict && c.cfg.Policy == ARC {
		d := digest(c.h, c.seed, k)
		if x.segment == 0 {
			c.ghost1[d] = struct{}{}
			delete(c.ghost2, d)
		} else {
			c.ghost2[d] = struct{}{}
			delete(c.ghost1, d)
		}
	}
	if ev == EventEvict && c.cfg.Policy == TwoQ {
		d := digest(c.h, c.seed, k)
		c.twoQGhost[d] = struct{}{}
		if c.cfg.MaxEntries > 0 && len(c.twoQGhost) > c.cfg.MaxEntries {
			for q := range c.twoQGhost {
				delete(c.twoQGhost, q)
				break
			}
		}
	}
	c.m.Delete(k)
	c.weightSum -= x.weight
	for i, q := range c.items {
		if c.h.Equal(q, k) {
			last := len(c.items) - 1
			copy(c.items[i:], c.items[i+1:])
			var zero K
			c.items[last] = zero
			c.items = c.items[:last]
			break
		}
	}
	if ev == EventExpire {
		c.stats.Expirations++
	}
	if ev == EventEvict {
		c.stats.Evictions++
	}
	if c.cfg.Observer != nil {
		c.pending = append(c.pending, Event[K, V]{Type: ev, Key: x.k, Value: x.v, Weight: x.weight})
	}
	return true
}

func (c *Cache[K, V]) flush() {
	if c.cfg.Observer == nil {
		return
	}
	for {
		c.mu.Lock()
		if len(c.pending) == 0 {
			c.mu.Unlock()
			return
		}
		events := slices.Clone(c.pending)
		c.pending = nil
		c.mu.Unlock()
		for _, e := range events {
			c.cfg.Observer(e)
		}
	}
}

func (c *Cache[K, V]) invalidateLoad(k K) {
	d := digest(c.h, c.seed, k)
	keep := c.loads[d][:0]
	for _, x := range c.loads[d] {
		if !c.h.Equal(x.key, k) {
			keep = append(keep, x)
		}
	}
	if len(keep) == 0 {
		delete(c.loads, d)
	} else {
		c.loads[d] = keep
	}
}

func (c *Cache[K, V]) get(k K, peek bool) (V, bool) {
	x, ok := c.m.Get(k)
	if !ok {
		c.stats.Misses++
		var z V
		return z, false
	}
	now := c.now()
	if c.expired(x, now) {
		c.remove(k, EventExpire)
		c.stats.Misses++
		var z V
		return z, false
	}
	if !peek && !x.timeToIdle.IsZero() {
		x.timeToIdle = now.Add(x.timeToIdle.Sub(x.last))
		x.last = now
		c.m.Set(k, x)
	}
	if !peek {
		c.accesses++
		if c.cfg.Policy == Clock {
			x.ref = true
		}
		x.freq++
		d := digest(c.h, c.seed, k)
		if c.freq[d] < ^uint64(0) {
			c.freq[d]++
		}
		c.m.Set(k, x)
		if c.accesses >= 4096 {
			c.ageFrequencies()
			x, _ = c.m.Get(k)
		}
		if c.cfg.Policy == LRU || c.cfg.Policy == MRU || c.cfg.Policy == LFU || c.cfg.Policy == SLRU || c.cfg.Policy == TwoQ || c.cfg.Policy == Clock || c.cfg.Policy == WTinyLFU {
			c.promote(k, true)
		}
		if c.cfg.Policy == SLRU && x.segment == 0 {
			x.segment = 1
			c.m.Set(k, x)
			c.rebalanceSLRU()
		}
		if c.cfg.Policy == ARC && x.segment == 0 {
			x.segment = 1
			c.m.Set(k, x)
			c.promote(k, true)
		}
		if c.cfg.Policy == TwoQ && x.segment == 0 {
			x.segment = 1
			c.m.Set(k, x)
			c.promote(k, true)
		}
		if c.cfg.Policy == WTinyLFU && x.segment == 0 {
			x.segment = 1
			c.m.Set(k, x)
			c.promote(k, true)
		}
	}
	if c.cfg.Policy == WTinyLFU {
		x.segment = 0
	}
	c.stats.Hits++
	return x.v, true
}

// Get returns a resident value and updates policy recency and TTI state.
func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	v, ok := c.get(k, false)
	c.mu.Unlock()
	c.flush()
	return v, ok
}

// Peek returns a resident value without updating policy recency or TTI state.
func (c *Cache[K, V]) Peek(k K) (V, bool) {
	c.mu.Lock()
	v, ok := c.get(k, true)
	c.mu.Unlock()
	c.flush()
	return v, ok
}

// Contains reports whether k is resident and unexpired.
func (c *Cache[K, V]) Contains(k K) bool { _, ok := c.Get(k); return ok }

// Set inserts or updates v using the configured default expiration.
func (c *Cache[K, V]) Set(k K, v V) bool { return c.set(k, v, 0, 0) }

// SetTTL inserts or updates v with a relative TTL.
func (c *Cache[K, V]) SetTTL(k K, v V, d time.Duration) bool { return c.set(k, v, d, 0) }

// SetExpiration inserts or updates v with relative TTL and TTI durations.
func (c *Cache[K, V]) SetExpiration(k K, v V, ttl, tti time.Duration) bool {
	return c.set(k, v, ttl, tti)
}

// SetUntil inserts or updates v with an absolute expiration deadline.
func (c *Cache[K, V]) SetUntil(k K, v V, t time.Time) bool {
	return c.setWithWeightPreparation(k, v, t.Sub(c.now()), 0)
}

func (c *Cache[K, V]) set(k K, v V, ttl, tti time.Duration) bool {
	return c.setWithWeightPreparation(k, v, ttl, tti)
}

func (c *Cache[K, V]) setWithWeightPreparation(k K, v V, ttl, tti time.Duration) bool {
	for {
		c.mu.Lock()
		c.invalidateLoad(k)
		old, exists := c.m.Get(k)
		canonical := k
		if exists {
			canonical = old.k
		}
		c.mu.Unlock()
		w := c.weight(canonical, v)
		c.mu.Lock()
		current, still := c.m.Get(k)
		if still != exists || (still && !c.h.Equal(current.k, canonical)) {
			c.mu.Unlock()
			continue
		}
		ok := c.setLockedWithWeight(k, v, ttl, tti, w)
		c.mu.Unlock()
		c.flush()
		return ok
	}
}

// setLoaded installs a loader result without invoking user code while the
// cache mutex is held. The weight callback is user supplied and may call back
// into the cache, just like the callbacks used by Set.
func (c *Cache[K, V]) setLoaded(k K, v V) bool {
	for {
		c.mu.Lock()
		if resident, exists := c.m.Get(k); exists {
			if c.expired(resident, c.now()) {
				c.remove(k, EventExpire)
			} else {
				c.mu.Unlock()
				return false
			}
		}
		c.mu.Unlock()

		w := c.weight(k, v)
		c.mu.Lock()
		if resident, exists := c.m.Get(k); exists {
			if c.expired(resident, c.now()) {
				c.remove(k, EventExpire)
			} else {
				c.mu.Unlock()
				return false
			}
		}
		ok := c.setLockedWithWeight(k, v, 0, 0, w)
		c.mu.Unlock()
		c.flush()
		return ok
	}
}

func (c *Cache[K, V]) setLockedWithWeight(k K, v V, ttl, tti time.Duration, w uint64) bool {
	if ttl == 0 {
		ttl = c.cfg.DefaultTTL
	}
	if tti == 0 {
		tti = c.cfg.DefaultTTI
	}
	if c.cfg.MaxWeight > 0 && w > c.cfg.MaxWeight {
		c.stats.Rejections++
		if c.cfg.Observer != nil {
			c.pending = append(c.pending, Event[K, V]{Type: EventReject, Key: k, Value: v, Weight: w})
		}
		return false
	}
	old, ok := c.m.Get(k)
	if !ok && (c.cfg.Admission == AdmitTinyLFU || c.cfg.Policy == WTinyLFU) && ((c.cfg.MaxEntries > 0 && len(c.items) >= c.cfg.MaxEntries) || (c.cfg.MaxWeight > 0 && len(c.items) > 0)) {
		candidate := c.freq[digest(c.h, c.seed, k)]
		victim := c.victim()
		victimFreq := c.freq[digest(c.h, c.seed, victim)]
		if candidate <= victimFreq {
			c.stats.Rejections++
			if c.cfg.Observer != nil {
				c.pending = append(c.pending, Event[K, V]{Type: EventReject, Key: k, Value: v, Weight: w})
			}
			return false
		}
	}
	if c.cfg.Policy == ARC && !ok {
		d := digest(c.h, c.seed, k)
		if _, hit := c.ghost1[d]; hit {
			if c.arcP < c.cfg.MaxEntries {
				c.arcP++
			}
			delete(c.ghost1, d)
		}
		if _, hit := c.ghost2[d]; hit {
			if c.arcP > 0 {
				c.arcP--
			}
			delete(c.ghost2, d)
		}
	}
	twoQHit := false
	if c.cfg.Policy == TwoQ && !ok {
		d := digest(c.h, c.seed, k)
		if _, hit := c.twoQGhost[d]; hit {
			twoQHit = true
			delete(c.twoQGhost, d)
		}
	}
	if !ok && c.cfg.Policy == LFU && ((c.cfg.MaxEntries > 0 && len(c.items) >= c.cfg.MaxEntries) || (c.cfg.MaxWeight > 0 && len(c.items) > 0)) {
		c.remove(c.victim(), EventEvict)
	}
	if !ok && c.cfg.Policy == MRU && ((c.cfg.MaxEntries > 0 && len(c.items) >= c.cfg.MaxEntries) || (c.cfg.MaxWeight > 0 && len(c.items) > 0)) {
		c.remove(c.victim(), EventEvict)
	}
	if ok && c.cfg.MaxWeight > 0 {
		remaining := c.weightSum - old.weight
		for remaining > c.cfg.MaxWeight-w && len(c.items) > 1 {
			c.remove(c.victimExcept(old.k), EventEvict)
			remaining = c.weightSum - old.weight
		}
	}
	if !ok && c.cfg.MaxWeight > 0 && c.weightSum > c.cfg.MaxWeight-w {
		for c.weightSum > c.cfg.MaxWeight-w && len(c.items) > 0 {
			c.remove(c.victim(), EventEvict)
		}
	}
	canonical := k
	if ok {
		canonical = old.k
	}
	x := item[K, V]{k: canonical, v: v, weight: w, freq: 1, last: c.now()}
	if c.cfg.Policy == SLRU {
		x.segment = 0
	}
	if c.cfg.Policy == TwoQ && twoQHit {
		x.segment = 1
	}
	if c.cfg.Policy == Clock {
		x.ref = true
	}
	if ok {
		x.freq = old.freq + 1
		x.segment = old.segment
		x.ref = old.ref
	}
	if ttl != 0 {
		x.expires = x.last.Add(ttl)
	}
	if tti != 0 {
		x.timeToIdle = x.last.Add(tti)
	}
	if ok {
		if w >= old.weight {
			c.weightSum += w - old.weight
		} else {
			c.weightSum -= old.weight - w
		}
		c.m.Set(k, x)
		c.stats.Updates++
	} else {
		d := digest(c.h, c.seed, k)
		if c.freq[d] < ^uint64(0) {
			c.freq[d]++
		}
		c.m.Set(k, x)
		c.items = append(c.items, k)
		c.weightSum += w
		c.stats.Sets++
	}
	if c.cfg.Observer != nil {
		typ := EventSet
		if ok {
			typ = EventUpdate
		}
		c.pending = append(c.pending, Event[K, V]{Type: typ, Key: canonical, Value: v, Previous: old.v, HasPrevious: ok, Weight: w})
	}
	if ok && (c.cfg.Policy == LRU || c.cfg.Policy == MRU || c.cfg.Policy == LFU || c.cfg.Policy == SLRU || c.cfg.Policy == TwoQ || c.cfg.Policy == Clock || c.cfg.Policy == WTinyLFU || c.cfg.Policy == ARC) {
		c.promote(canonical, true)
	}
	if c.cfg.Policy == SLRU {
		c.rebalanceSLRU()
	}
	if c.cfg.Policy == WTinyLFU {
		c.rebalanceWTiny()
	}
	for (c.cfg.MaxEntries > 0 && len(c.items) > c.cfg.MaxEntries) || (c.cfg.MaxWeight > 0 && c.totalWeight() > c.cfg.MaxWeight) {
		c.remove(c.victim(), EventEvict)
	}
	return true
}

func (c *Cache[K, V]) rebalanceSLRU() {
	if c.cfg.Policy != SLRU || len(c.items) < 2 {
		return
	}
	limit := uint64(c.cfg.MaxEntries / 2)
	weighted := c.cfg.MaxWeight > 0
	if weighted {
		limit = c.cfg.MaxWeight / 2
	}
	if limit < 1 {
		limit = 1
	}
	var protected uint64
	for _, k := range c.items {
		if x, ok := c.m.Get(k); ok && x.segment == 1 {
			if weighted {
				protected += x.weight
			} else {
				protected++
			}
		}
	}
	for protected > limit {
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 1 {
				x.segment = 0
				c.m.Set(k, x)
				if weighted {
					protected -= x.weight
				} else {
					protected--
				}
				break
			}
		}
	}
}

func (c *Cache[K, V]) rebalanceWTiny() {
	if c.cfg.Policy != WTinyLFU || len(c.items) == 0 {
		return
	}
	limit := uint64(c.cfg.MaxEntries / 100)
	weighted := c.cfg.MaxWeight > 0
	if weighted {
		limit = c.cfg.MaxWeight / 100
	}
	if limit < 1 {
		limit = 1
	}
	var window uint64
	for _, k := range c.items {
		if x, ok := c.m.Get(k); ok && x.segment == 0 {
			if weighted {
				window += x.weight
			} else {
				window++
			}
		}
	}
	for window > limit {
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 0 {
				x.segment = 1
				c.m.Set(k, x)
				c.promote(k, true)
				if weighted {
					window -= x.weight
				} else {
					window--
				}
				break
			}
		}
	}
}

func (c *Cache[K, V]) ageFrequencies() {
	c.accesses = 0
	for i := range c.freq {
		c.freq[i] /= 2
	}
	for k, x := range c.m.All() {
		x.freq /= 2
		c.m.Set(k, x)
	}
}

func (c *Cache[K, V]) promote(k K, back bool) {
	for i, q := range c.items {
		if c.h.Equal(q, k) {
			c.items = append(c.items[:i], c.items[i+1:]...)
			if back {
				c.items = append(c.items, k)
			} else {
				c.items = append([]K{k}, c.items...)
			}
			return
		}
	}
}

func (c *Cache[K, V]) victim() K {
	if len(c.items) == 0 {
		var z K
		return z
	}
	idx := 0
	switch c.cfg.Policy {
	case MRU:
		idx = len(c.items) - 1
	case LFU:
		best := uint64(^uint64(0))
		for i, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.freq < best {
				best = x.freq
				idx = i
			}
		}
	case Clock:
		for n := 0; n < len(c.items)*2; n++ {
			if len(c.items) == 0 {
				break
			}
			i := c.clockHand % len(c.items)
			c.clockHand = (i + 1) % len(c.items)
			k := c.items[i]
			if x, ok := c.m.Get(k); ok && x.ref {
				x.ref = false
				c.m.Set(k, x)
				continue
			}
			return k
		}
	}
	if c.cfg.Policy == SLRU {
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 0 {
				return k
			}
		}
	}
	if c.cfg.Policy == TwoQ {
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 0 {
				return k
			}
		}
	}
	if c.cfg.Policy == WTinyLFU {
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 1 {
				return k
			}
		}
		for _, k := range c.items {
			return k
		}
	}
	if c.cfg.Policy == ARC {
		t1 := 0
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 0 {
				t1++
			}
		}
		if t1 > c.arcP {
			for _, k := range c.items {
				if x, ok := c.m.Get(k); ok && x.segment == 0 {
					return k
				}
			}
		}
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 1 {
				return k
			}
		}
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && x.segment == 0 {
				return k
			}
		}
	}
	return c.items[idx]
}

func (c *Cache[K, V]) victimExcept(exclude K) K {
	v := c.victim()
	if !c.h.Equal(v, exclude) {
		return v
	}
	for _, k := range c.items {
		if !c.h.Equal(k, exclude) {
			return k
		}
	}
	return exclude
}

func (c *Cache[K, V]) totalWeight() uint64 {
	return c.weightSum
}

// Delete removes k and invalidates any in-flight load generation for it.
func (c *Cache[K, V]) Delete(k K) bool {
	c.mu.Lock()
	c.invalidateLoad(k)
	ok := c.remove(k, EventDelete)
	c.mu.Unlock()
	c.flush()
	return ok
}

// Touch refreshes TTI state for k without changing its TTL deadline.
func (c *Cache[K, V]) Touch(k K) bool {
	c.mu.Lock()
	x, ok := c.m.Get(k)
	if !ok {
		c.mu.Unlock()
		return false
	}
	if c.expired(x, c.now()) {
		c.remove(k, EventExpire)
		c.mu.Unlock()
		c.flush()
		return false
	}
	now := c.now()
	tti := x.timeToIdle.Sub(x.last)
	x.last = now
	if !x.timeToIdle.IsZero() {
		x.timeToIdle = now.Add(tti)
	}
	c.m.Set(k, x)
	c.mu.Unlock()
	c.flush()
	return true
}

// Len returns the number of resident entries.
func (c *Cache[K, V]) Len() int { c.mu.Lock(); defer c.mu.Unlock(); return c.m.Len() }

// Weight returns the total resident weight.
func (c *Cache[K, V]) Weight() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.totalWeight() }

// Clear removes entries and invalidates current load generations.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m.Clear()
	c.items = nil
	c.weightSum = 0
	c.loads = map[uint64][]*load[K, V]{}
	c.freq = map[uint64]uint64{}
	c.ghost1 = map[uint64]struct{}{}
	c.ghost2 = map[uint64]struct{}{}
	c.arcP = 0
	c.twoQGhost = map[uint64]struct{}{}
}

// PruneExpired removes all currently expired entries and returns its count.
func (c *Cache[K, V]) PruneExpired() int {
	c.mu.Lock()
	n := 0
	now := c.now()
	for _, k := range slices.Clone(c.items) {
		if x, ok := c.m.Get(k); ok && c.expired(x, now) {
			if c.remove(k, EventExpire) {
				n++
			}
		}
	}
	c.mu.Unlock()
	c.flush()
	return n
}

// All snapshots unexpired entries and invokes its yield callback without locks.
func (c *Cache[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		c.mu.Lock()
		a := make([]item[K, V], 0, len(c.items))
		for _, k := range c.items {
			if x, ok := c.m.Get(k); ok && !c.expired(x, c.now()) {
				a = append(a, x)
			}
		}
		c.mu.Unlock()
		for _, x := range a {
			if !y(x.k, x.v) {
				return
			}
		}
	}
}

// Stats returns a snapshot of operation counters.
func (c *Cache[K, V]) Stats() Stats { c.mu.Lock(); defer c.mu.Unlock(); return c.stats }

// GetOrLoad synchronously loads a miss in the caller's goroutine.
func (c *Cache[K, V]) GetOrLoad(ctx context.Context, k K, l Loader[K, V]) (V, error) {
	if l == nil {
		panic("cache: nil Loader")
	}
	if v, ok := c.Get(k); ok {
		return v, nil
	}
	v, err := l(ctx, k)
	if err != nil {
		return v, err
	}
	if resident, ok := c.Get(k); ok {
		return resident, nil
	}
	c.setLoaded(k, v)
	return v, nil
}

func (c *Cache[K, V]) getOrLoadCoalesced(ctx context.Context, k K, l Loader[K, V]) (V, error) {
	if l == nil {
		panic("cache: nil Loader")
	}
	if v, ok := c.Get(k); ok {
		return v, nil
	}
	d := digest(c.h, c.seed, k)
	c.mu.Lock()
	var x *load[K, V]
	for _, candidate := range c.loads[d] {
		if c.h.Equal(candidate.key, k) {
			x = candidate
			break
		}
	}
	if x != nil {
		c.stats.LoadCoalesced++
		c.mu.Unlock()
		select {
		case <-x.done:
			return x.v, x.err
		case <-ctx.Done():
			var z V
			return z, ctx.Err()
		}
	}
	x = &load[K, V]{done: make(chan struct{}), key: k}
	c.loads[d] = append(c.loads[d], x)
	c.mu.Unlock()
	base := context.WithoutCancel(ctx)
	var cancel context.CancelFunc
	if c.cfg.LoadTimeout > 0 {
		base, cancel = context.WithTimeout(base, c.cfg.LoadTimeout)
	} else {
		base, cancel = context.WithCancel(base)
	}
	go c.runLoad(base, cancel, k, x, l)
	select {
	case <-x.done:
		return x.v, x.err
	case <-ctx.Done():
		var z V
		return z, ctx.Err()
	}
}

func (c *Cache[K, V]) runLoad(ctx context.Context, cancel context.CancelFunc, k K, x *load[K, V], l Loader[K, V]) {
	x.v, x.err = l(ctx, k)
	cancel()
	d := digest(c.h, c.seed, k)
	c.mu.Lock()
	current := false
	bucket := c.loads[d]
	keep := bucket[:0]
	for _, candidate := range bucket {
		if candidate == x {
			current = true
		} else {
			keep = append(keep, candidate)
		}
	}
	if len(keep) == 0 {
		delete(c.loads, d)
	} else {
		c.loads[d] = keep
	}
	// An explicit mutation may have invalidated this generation while its
	// loader was running. A resident equivalent value wins for the old
	// generation's waiters, including when the loader itself returned an error.
	if resident, ok := c.m.Get(k); ok {
		if c.expired(resident, c.now()) {
			c.remove(k, EventExpire)
		} else {
			x.v = resident.v
			x.err = nil
		}
	}
	if x.err == nil {
		c.stats.LoadSuccesses++
	} else {
		c.stats.LoadErrors++
	}
	c.mu.Unlock()
	if x.err == nil && current {
		c.setLoaded(k, x.v)
	}
	close(x.done)
	c.flush()
}

func digest[K any](h maphash.Hasher[K], seed maphash.Seed, k K) uint64 {
	var x maphash.Hash
	x.SetSeed(seed)
	h.Hash(&x, k)
	return x.Sum64()
}
