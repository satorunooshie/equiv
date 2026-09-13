package cache

import (
	"context"
	"errors"
	"hash/maphash"
	"iter"
	"time"
)

// ShardedCache partitions a semantic cache across independent shards.
type ShardedCache[K, V any] struct {
	h      maphash.Hasher[K]
	seed   maphash.Seed
	shards []*Cache[K, V]
}

// NewSharded constructs a cache partitioned across independent shards.
func NewSharded[K, V any](h maphash.Hasher[K], c Config[K, V]) (*ShardedCache[K, V], error) {
	if e := validate(h, c); e != nil {
		return nil, e
	}
	units := uint64(c.MaxEntries)
	if units == 0 {
		units = c.MaxWeight
	}
	requested := c.Shards
	if requested == 0 {
		requested = 64
		if units < uint64(requested) {
			requested = int(units)
		}
	}
	if requested <= 0 || uint64(requested) > units {
		return nil, errors.New("cache: invalid effective shard count")
	}
	s := &ShardedCache[K, V]{h: h, seed: maphash.MakeSeed(), shards: make([]*Cache[K, V], requested)}
	base := units / uint64(requested)
	rem := units % uint64(requested)
	for i := range s.shards {
		cc := c
		cc.Shards = 0
		if c.MaxEntries > 0 {
			cc.MaxEntries = int(base)
			if uint64(i) < rem {
				cc.MaxEntries++
			}
		} else {
			cc.MaxWeight = base
			if uint64(i) < rem {
				cc.MaxWeight++
			}
		}
		s.shards[i], _ = New(h, cc)
	}
	return s, nil
}

func (s *ShardedCache[K, V]) index(k K) int {
	var x maphash.Hash
	x.SetSeed(s.seed)
	s.h.Hash(&x, k)
	return int(x.Sum64() % uint64(len(s.shards)))
}
func (s *ShardedCache[K, V]) shard(k K) *Cache[K, V] { return s.shards[s.index(k)] }

// Get returns a resident value for k.
func (s *ShardedCache[K, V]) Get(k K) (V, bool) { return s.shard(k).Get(k) }

// Peek returns a resident value without updating recency or TTI state.
func (s *ShardedCache[K, V]) Peek(k K) (V, bool) { return s.shard(k).Peek(k) }

// Contains reports whether k is resident and unexpired.
func (s *ShardedCache[K, V]) Contains(k K) bool { _, ok := s.Get(k); return ok }

// Set inserts or updates v using the configured default expiration.
func (s *ShardedCache[K, V]) Set(k K, v V) bool { return s.shard(k).Set(k, v) }

// SetTTL inserts or updates v with a relative TTL.
func (s *ShardedCache[K, V]) SetTTL(k K, v V, d time.Duration) bool {
	return s.shard(k).SetTTL(k, v, d)
}

// SetExpiration inserts or updates v with relative TTL and TTI durations.
func (s *ShardedCache[K, V]) SetExpiration(k K, v V, a, b time.Duration) bool {
	return s.shard(k).SetExpiration(k, v, a, b)
}

// SetUntil inserts or updates v with an absolute expiration deadline.
func (s *ShardedCache[K, V]) SetUntil(k K, v V, t time.Time) bool {
	return s.shard(k).SetUntil(k, v, t)
}

// Delete removes k if present.
func (s *ShardedCache[K, V]) Delete(k K) bool { return s.shard(k).Delete(k) }

// Touch refreshes TTI state for k.
func (s *ShardedCache[K, V]) Touch(k K) bool { return s.shard(k).Touch(k) }

// Len returns the total number of resident entries.
func (s *ShardedCache[K, V]) Len() int {
	n := 0
	for _, x := range s.shards {
		n += x.Len()
	}
	return n
}

// Weight returns the total resident weight.
func (s *ShardedCache[K, V]) Weight() uint64 {
	n := uint64(0)
	for _, x := range s.shards {
		n += x.Weight()
	}
	return n
}

// Clear removes all entries from every shard.
func (s *ShardedCache[K, V]) Clear() {
	for _, x := range s.shards {
		x.Clear()
	}
}

// PruneExpired removes all currently expired entries from every shard.
func (s *ShardedCache[K, V]) PruneExpired() int {
	n := 0
	for _, x := range s.shards {
		n += x.PruneExpired()
	}
	return n
}

// All returns entries from each shard without holding locks while yielding.
func (s *ShardedCache[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		for _, x := range s.shards {
			for k, v := range x.All() {
				if !y(k, v) {
					return
				}
			}
		}
	}
}

// Stats returns the aggregate counters from all shards.
func (s *ShardedCache[K, V]) Stats() Stats {
	var z Stats
	for _, x := range s.shards {
		a := x.Stats()
		z.Hits += a.Hits
		z.Misses += a.Misses
		z.Sets += a.Sets
		z.Updates += a.Updates
		z.Evictions += a.Evictions
		z.Expirations += a.Expirations
		z.Rejections += a.Rejections
		z.LoadSuccesses += a.LoadSuccesses
		z.LoadErrors += a.LoadErrors
		z.LoadCoalesced += a.LoadCoalesced
	}
	return z
}

// GetOrLoad loads a missing value, sharing one in-flight load per key shard.
func (s *ShardedCache[K, V]) GetOrLoad(ctx context.Context, k K, l Loader[K, V]) (V, error) {
	return s.shard(k).getOrLoadCoalesced(ctx, k, l)
}
