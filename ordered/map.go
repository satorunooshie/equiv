// Package ordered provides insertion-ordered semantic maps and sets.
// Its containers are non-concurrent and use the supplied Hasher for identity.
package ordered

import (
	"hash/maphash"
	"iter"
	"slices"

	"github.com/satorunooshie/equiv"
)

// Map stores values by semantic key in insertion order. Its zero value is
// invalid, it is not concurrent, and it must not be copied after first use.
type Map[K, V any] struct {
	h     maphash.Hasher[K]
	m     *equiv.Map[K, V]
	order []K
	ver   uint64
}

// NewMap constructs an insertion-ordered map using h for key identity.
func NewMap[K, V any](h maphash.Hasher[K]) *Map[K, V] {
	return NewMapCapacity[K, V](h, 0)
}

// NewMapCapacity constructs an insertion-ordered map with a capacity hint.
func NewMapCapacity[K, V any](h maphash.Hasher[K], capacity int) *Map[K, V] {
	if h == nil {
		panic("ordered: nil Hasher")
	}
	return &Map[K, V]{h: h, m: equiv.NewMapCapacity[K, V](h, capacity)}
}

// Get returns the value for a semantically equal key.
func (m *Map[K, V]) Get(k K) (V, bool) { return m.m.Get(k) }

// GetEntry returns the stored canonical key and value.
func (m *Map[K, V]) GetEntry(k K) (K, V, bool) { return m.m.GetEntry(k) }

// GetOrSet returns the existing value or inserts v.
func (m *Map[K, V]) GetOrSet(k K, v V) (V, bool) {
	old, ok := m.m.Get(k)
	if ok {
		return old, true
	}
	m.Set(k, v)
	return v, false
}

// GetOrCompute computes a missing value and leaves existing values unchanged.
func (m *Map[K, V]) GetOrCompute(k K, f func() V) (V, bool) {
	if v, ok := m.Get(k); ok {
		return v, true
	}
	v := f()
	return m.GetOrSet(k, v)
}

// Set inserts or replaces a value without changing the order of an existing key.
func (m *Map[K, V]) Set(k K, v V) (V, bool) {
	old, ok := m.m.Set(k, v)
	if !ok {
		m.order = append(m.order, k)
		m.ver++
	}
	return old, ok
}

// Delete removes a key and its position.
func (m *Map[K, V]) Delete(k K) bool {
	stored, _, ok := m.m.GetEntry(k)
	if !ok {
		return false
	}
	m.m.Delete(k)
	for i, k := range m.order {
		if m.h.Equal(k, stored) {
			m.order = slices.Delete(m.order, i, i+1)
			m.ver++
			break
		}
	}
	return true
}

// Clear removes all entries without changing the hash domain.
func (m *Map[K, V]) Clear() {
	if len(m.order) > 0 {
		m.m.Clear()
		m.order = nil
		m.ver++
	}
}

// Len returns the number of entries.
func (m *Map[K, V]) Len() int { return m.m.Len() }

// Clone returns an independent map with the same order and a fresh hash domain.
func (m *Map[K, V]) Clone() *Map[K, V] {
	n := NewMapCapacity[K, V](m.h, len(m.order))
	for _, k := range m.order {
		if v, ok := m.m.Get(k); ok {
			n.Set(k, v)
		}
	}
	return n
}

// DeleteFunc removes entries selected by f in insertion order.
func (m *Map[K, V]) DeleteFunc(f func(K, V) bool) int {
	removed := 0
	keys := append([]K(nil), m.order...)
	for _, k := range keys {
		if v, ok := m.Get(k); ok && f(k, v) && m.Delete(k) {
			removed++
		}
	}
	return removed
}

// All returns entries in insertion order. Structural mutation during iteration
// panics; replacing an existing value is permitted.
func (m *Map[K, V]) All() iter.Seq2[K, V] { return m.forward(false) }

// Backward returns entries in reverse insertion order.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] { return m.forward(true) }

func (m *Map[K, V]) forward(back bool) iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		ver := m.ver
		if back {
			for _, k := range slices.Backward(m.order) {
				if ver != m.ver {
					panic("ordered: collection structurally mutated during iteration")
				}

				v, ok := m.m.Get(k)
				if ok && !y(k, v) {
					return
				}
				if ver != m.ver {
					panic("ordered: collection structurally mutated during iteration")
				}
			}
		} else {
			for _, k := range m.order {
				if ver != m.ver {
					panic("ordered: collection structurally mutated during iteration")
				}
				v, ok := m.m.Get(k)
				if ok && !y(k, v) {
					return
				}
				if ver != m.ver {
					panic("ordered: collection structurally mutated during iteration")
				}
			}
		}
	}
}

// MoveToFront moves an existing key to the front and invalidates active iterators.
func (m *Map[K, V]) MoveToFront(k K) bool {
	stored, _, ok := m.m.GetEntry(k)
	if !ok {
		return false
	}
	for i, x := range m.order {
		if m.h.Equal(x, stored) {
			if i == 0 {
				return true
			}
			m.order = slices.Insert(slices.Delete(m.order, i, i+1), 0, stored)
			m.ver++
			return true
		}
	}
	return false
}

// MoveToBack moves an existing key to the back and invalidates active iterators.
func (m *Map[K, V]) MoveToBack(k K) bool {
	stored, _, ok := m.m.GetEntry(k)
	if !ok {
		return false
	}
	for i, x := range m.order {
		if m.h.Equal(x, stored) {
			if i == len(m.order)-1 {
				return true
			}
			m.order = append(slices.Delete(m.order, i, i+1), stored)
			m.ver++
			return true
		}
	}
	return false
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
