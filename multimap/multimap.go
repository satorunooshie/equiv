// Package multimap provides insertion-ordered values grouped by semantic keys.
// Containers are non-concurrent and require a constructor-supplied Hasher.
package multimap

import (
	"hash/maphash"
	"iter"

	"github.com/satorunooshie/equiv"
)

// Map groups insertion-ordered values by semantic key. Its zero value is
// invalid and it is not safe for concurrent use or copying after first use.
type Map[K, V any] struct{ m *equiv.Map[K, []V] }

// New constructs an insertion-ordered multimap using h for key identity.
func New[K, V any](h maphash.Hasher[K]) *Map[K, V] { return &Map[K, V]{equiv.NewMap[K, []V](h)} }

// Add appends v to the values associated with k.
func (m *Map[K, V]) Add(k K, v V) {
	x, ok := m.m.Get(k)
	x = append(x, v)
	if ok {
		m.m.Set(k, x)
	} else {
		m.m.Set(k, x)
	}
}

// Values returns the values associated with k in insertion order.
func (m *Map[K, V]) Values(k K) iter.Seq[V] {
	x, ok := m.m.Get(k)
	return func(y func(V) bool) {
		if ok {
			for _, v := range x {
				if !y(v) {
					return
				}
			}
		}
	}
}

// ContainsKey reports whether k has at least one associated value.
func (m *Map[K, V]) ContainsKey(k K) bool { _, ok := m.m.Get(k); return ok }
func (m *Map[K, V]) KeyLen() int          { return m.m.Len() }
func (m *Map[K, V]) ValueLen() int {
	n := 0
	for x := range m.m.Values() {
		n += len(x)
	}
	return n
}

// DeleteKey removes k and all of its associated values.
func (m *Map[K, V]) DeleteKey(k K) bool { return m.m.Delete(k) }

// DeleteFunc removes values for k whose predicate returns true.
func (m *Map[K, V]) DeleteFunc(k K, f func(V) bool) int {
	x, ok := m.m.Get(k)
	if !ok {
		return 0
	}
	n := 0
	keep := x[:0]
	for _, v := range x {
		if f(v) {
			n++
		} else {
			keep = append(keep, v)
		}
	}
	clear(x[len(keep):])
	if len(keep) == 0 {
		m.m.Delete(k)
	} else {
		m.m.Set(k, keep)
	}
	return n
}

func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		for k, x := range m.m.All() {
			for _, v := range x {
				if !y(k, v) {
					return
				}
			}
		}
	}
}
func (m *Map[K, V]) Keys() iter.Seq[K] { return m.m.Keys() }
func (m *Map[K, V]) Clear()            { m.m.Clear() }
