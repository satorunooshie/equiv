// Package equiv provides semantic hash-based maps and sets for Go 1.27.
// Containers use the supplied maphash.Hasher as their identity protocol.
package equiv

import (
	"hash/maphash"
	"iter"

	"github.com/satorunooshie/equiv/internal/table"
)

// Map stores values under semantic keys. Its zero value is invalid and the
// map must not be copied after first use; it is not safe for concurrent use.
type Map[K, V any] struct {
	h   maphash.Hasher[K]
	t   *table.Table[K, V]
	ver uint64
}

// NewMap constructs a Map using h as its semantic identity protocol.
func NewMap[K, V any](h maphash.Hasher[K]) *Map[K, V] { return NewMapCapacity[K, V](h, 0) }

// NewMapCapacity constructs a Map with an initial capacity hint.
func NewMapCapacity[K, V any](h maphash.Hasher[K], capacity int) *Map[K, V] {
	if h == nil {
		panic("equiv: nil Hasher")
	}
	if capacity < 0 {
		panic("equiv: negative capacity")
	}
	if capacity > int(^uint(0)>>1)/2 {
		panic("equiv: capacity too large")
	}
	return &Map[K, V]{h: h, t: table.New[K, V](h, capacity)}
}

// Get returns the value for a semantically equal key, if present.
func (m *Map[K, V]) Get(k K) (V, bool) {
	return m.t.Get(k)
}

// GetEntry returns the stored canonical key, value, and presence flag.
func (m *Map[K, V]) GetEntry(k K) (K, V, bool) {
	return m.t.GetEntry(k)
}

// Set inserts or replaces a value and reports the previous value and presence.
func (m *Map[K, V]) Set(k K, v V) (V, bool) {
	old, ok := m.t.Set(k, v)
	if !ok {
		m.ver++
	}
	return old, ok
}

// GetOrSet returns the existing value or inserts and returns v.
func (m *Map[K, V]) GetOrSet(k K, v V) (V, bool) {
	d := m.t.Hash(k)
	if x, ok := m.t.GetHashed(k, d); ok {
		return x, true
	}
	m.t.SetHashed(k, v, d)
	m.ver++
	return v, false
}

// GetOrCompute computes and inserts a missing value; f is not called on a hit.
func (m *Map[K, V]) GetOrCompute(k K, f func() V) (V, bool) {
	if v, ok := m.Get(k); ok {
		return v, true
	}
	v := f()
	return m.GetOrSet(k, v)
}

// Delete removes the semantically equal key, if present.
func (m *Map[K, V]) Delete(k K) bool {
	ok := m.t.Delete(k)
	if ok {
		m.ver++
	}
	return ok
}

// DeleteFunc removes entries whose predicate returns true.
func (m *Map[K, V]) DeleteFunc(f func(K, V) bool) int {
	type candidate struct {
		k K
		v V
	}
	candidates := make([]candidate, 0, m.Len())
	for k, v := range m.t.All() {
		candidates = append(candidates, candidate{k, v})
	}
	n := 0
	for _, x := range candidates {
		if f(x.k, x.v) && m.Delete(x.k) {
			n++
		}
	}
	return n
}

// Clear removes all entries without changing the map's hash domain.
func (m *Map[K, V]) Clear() {
	if m.t.Len() > 0 {
		m.t.Clear()
		m.ver++
	}
}

// Len returns the number of resident entries.
func (m *Map[K, V]) Len() int { return m.t.Len() }

// Clone returns an independent map with a fresh hash domain.
func (m *Map[K, V]) Clone() *Map[K, V] {
	n := NewMapCapacity[K, V](m.h, m.Len())
	for k, v := range m.All() {
		n.Set(k, v)
	}
	return n
}

// All returns an allocation-free sequence in table order. Structural mutation
// during iteration panics; replacing an existing value is permitted.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		v := m.ver
		for k, x := range m.t.All() {
			if v != m.ver {
				panic("equiv: collection structurally mutated during iteration")
			}
			if !y(k, x) {
				return
			}
			if v != m.ver {
				panic("equiv: collection structurally mutated during iteration")
			}
		}
	}
}

// Keys returns the map's keys using the same iteration contract as All.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(y func(K) bool) {
		for k := range m.All() {
			if !y(k) {
				return
			}
		}
	}
}

// Values returns the map's values using the same iteration contract as All.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(y func(V) bool) {
		for _, v := range m.All() {
			if !y(v) {
				return
			}
		}
	}
}
