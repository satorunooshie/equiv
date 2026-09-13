// Package concurrent provides sharded, linearizable semantic collections.
package concurrent

import (
	"errors"
	"hash/maphash"
	"iter"
	"sync"
	"sync/atomic"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/internal/hashpool"
	"github.com/satorunooshie/equiv/internal/table"
)

type config struct{ shards int }

// Option configures a concurrent collection at construction time.
type (
	Option   interface{ apply(*config) error }
	shardOpt int
)

func (o shardOpt) apply(c *config) error {
	if o <= 0 || o&(o-1) != 0 {
		return errors.New("concurrent: shards must be a positive power of two")
	}
	c.shards = int(o)
	return nil
}

// WithShards requests n positive power-of-two shards.
func WithShards(n int) Option { return shardOpt(n) }

// Map is a sharded, linearizable semantic map. Its zero value is invalid and
// it must not be copied after first use.
type Map[K, V any] struct {
	h      maphash.Hasher[K]
	seed   maphash.Seed
	shards []mapShard[K, V]
	count  atomic.Int64
	gate   sync.RWMutex
	pool   *hashpool.Pool
}
type mapShard[K, V any] struct {
	sync.RWMutex
	m *table.Table[K, V]
}

// NewMap constructs a linearizable sharded map using h for key identity.
func NewMap[K, V any](h maphash.Hasher[K], opts ...Option) (*Map[K, V], error) {
	if h == nil {
		panic("concurrent: nil Hasher")
	}
	c := config{shards: 64}
	for _, o := range opts {
		if o == nil {
			return nil, errors.New("concurrent: nil option")
		}
		if err := o.apply(&c); err != nil {
			return nil, err
		}
	}
	seed := maphash.MakeSeed()
	m := &Map[K, V]{h: h, seed: seed, pool: hashpool.New(seed), shards: make([]mapShard[K, V], c.shards)}
	for i := range m.shards {
		m.shards[i].m = table.NewWithSeed[K, V](h, 0, seed)
	}
	return m, nil
}

func (m *Map[K, V]) index(k K) (int, uint64) {
	d := hashpool.Hash(m.pool, m.h, k)
	return int(d & uint64(len(m.shards)-1)), d
}

// Get returns the value for a semantically equal key.
func (m *Map[K, V]) Get(k K) (V, bool) {
	m.gate.RLock()
	defer m.gate.RUnlock()
	i, d := m.index(k)
	s := &m.shards[i]
	s.RLock()
	defer s.RUnlock()
	return s.m.GetHashed(k, d)
}

// GetEntry returns the stored canonical key and value.
func (m *Map[K, V]) GetEntry(k K) (K, V, bool) {
	m.gate.RLock()
	defer m.gate.RUnlock()
	i, d := m.index(k)
	s := &m.shards[i]
	s.RLock()
	defer s.RUnlock()
	return s.m.GetEntryHashed(k, d)
}

// Set inserts or replaces a value and reports the previous value and presence.
func (m *Map[K, V]) Set(k K, v V) (V, bool) {
	m.gate.RLock()
	defer m.gate.RUnlock()
	i, d := m.index(k)
	s := &m.shards[i]
	s.Lock()
	defer s.Unlock()
	old, ok := s.m.SetHashed(k, v, d)
	if !ok {
		m.count.Add(1)
	}
	return old, ok
}

// GetOrSet returns the existing value or inserts v atomically.
func (m *Map[K, V]) GetOrSet(k K, v V) (V, bool) {
	m.gate.RLock()
	defer m.gate.RUnlock()
	i, d := m.index(k)
	s := &m.shards[i]
	s.Lock()
	defer s.Unlock()
	old, ok := s.m.GetHashed(k, d)
	if !ok {
		s.m.SetHashed(k, v, d)
		old = v
	}
	if !ok {
		m.count.Add(1)
	}
	return old, ok
}

// Delete removes a semantically equal key, if present.
func (m *Map[K, V]) Delete(k K) bool {
	m.gate.RLock()
	defer m.gate.RUnlock()
	i, d := m.index(k)
	s := &m.shards[i]
	s.Lock()
	defer s.Unlock()
	ok := s.m.DeleteHashed(k, d)
	if ok {
		m.count.Add(-1)
	}
	return ok
}

// Len returns a linearizable count of resident entries.
func (m *Map[K, V]) Len() int { m.gate.RLock(); defer m.gate.RUnlock(); return int(m.count.Load()) }

// Clear atomically removes all entries.
func (m *Map[K, V]) Clear() {
	m.gate.Lock()
	defer m.gate.Unlock()
	for i := range m.shards {
		m.shards[i].Lock()
	}
	for i := range m.shards {
		m.shards[i].m.Clear()
	}
	m.count.Store(0)
	for i := len(m.shards) - 1; i >= 0; i-- {
		m.shards[i].Unlock()
	}
}

// All snapshots entries at one linearization point and invokes yield without
// holding internal locks. Mutations may continue while the returned sequence
// is being consumed.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		// The exclusive gate prevents point mutations from changing a shard
		// after it has been copied. The shard locks protect the local table.
		m.gate.Lock()
		all := make([]mapEntry[K, V], 0, int(m.count.Load()))
		for i := range m.shards {
			s := &m.shards[i]
			s.RLock()
			for k, v := range s.m.All() {
				all = append(all, mapEntry[K, V]{k, v})
			}
			s.RUnlock()
		}
		m.gate.Unlock()
		for _, e := range all {
			if !y(e.k, e.v) {
				return
			}
		}
	}
}

type mapEntry[K, V any] struct {
	k K
	v V
}

func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(y func(K) bool) {
		for k := range m.All() {
			if !y(k) {
				return
			}
		}
	}
}

func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(y func(V) bool) {
		for _, v := range m.All() {
			if !y(v) {
				return
			}
		}
	}
}

// Snapshot returns an independent local map containing a linearizable view.
func (m *Map[K, V]) Snapshot() *equiv.Map[K, V] {
	// Hold the exclusive gate for the entire copy so the result corresponds
	// to one global point in the mutation history.
	m.gate.Lock()
	defer m.gate.Unlock()
	n := equiv.NewMap[K, V](m.h)
	for i := range m.shards {
		s := &m.shards[i]
		s.RLock()
		for k, v := range s.m.All() {
			n.Set(k, v)
		}
		s.RUnlock()
	}
	return n
}

// Set is a sharded, linearizable semantic set. Its zero value is invalid and
// it must not be copied after first use.
type Set[E any] struct{ m *Map[E, struct{}] }

// NewSet constructs a linearizable sharded set using h for element identity.
func NewSet[E any](h maphash.Hasher[E], opts ...Option) (*Set[E], error) {
	m, e := NewMap[E, struct{}](h, opts...)
	if e != nil {
		return nil, e
	}
	return &Set[E]{m}, nil
}

// Insert adds e and reports whether it was new.
func (s *Set[E]) Insert(e E) bool { _, ok := s.m.Set(e, struct{}{}); return !ok }

// Lookup returns the stored canonical representation of e.
func (s *Set[E]) Lookup(e E) (E, bool) { k, _, ok := s.m.GetEntry(e); return k, ok }

// Contains reports whether e is present.
func (s *Set[E]) Contains(e E) bool { _, ok := s.m.Get(e); return ok }

// Delete removes e if present.
func (s *Set[E]) Delete(e E) bool { return s.m.Delete(e) }

// Len returns the number of distinct elements.
func (s *Set[E]) Len() int { return s.m.Len() }

// Clear removes all elements.
func (s *Set[E]) Clear() { s.m.Clear() }

// All returns a linearizable snapshot of the elements.
func (s *Set[E]) All() iter.Seq[E] { return s.m.Keys() }

// Snapshot returns an independent local set containing a linearizable view.
func (s *Set[E]) Snapshot() *equiv.Set[E] {
	n := equiv.NewSet[E](s.m.h)
	for e := range s.All() {
		n.Insert(e)
	}
	return n
}
