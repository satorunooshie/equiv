package ordered

import (
	"hash/maphash"
	"iter"
)

// Set stores distinct semantic elements in insertion order. Its zero value is
// invalid and it is not safe for concurrent use or copying after first use.
type Set[E any] struct{ m *Map[E, struct{}] }

// NewSet constructs an insertion-ordered set using h for element identity.
func NewSet[E any](h maphash.Hasher[E]) *Set[E] { return NewSetCapacity[E](h, 0) }

// NewSetCapacity constructs an insertion-ordered set with a capacity hint.
func NewSetCapacity[E any](h maphash.Hasher[E], capacity int) *Set[E] {
	return &Set[E]{NewMapCapacity[E, struct{}](h, capacity)}
}

// Insert adds e and reports whether it was new.
func (s *Set[E]) Insert(e E) bool { _, ok := s.m.Set(e, struct{}{}); return !ok }

// Lookup returns the stored canonical representation of e.
func (s *Set[E]) Lookup(e E) (E, bool) { k, _, ok := s.m.GetEntry(e); return k, ok }

// Contains reports whether e is present.
func (s *Set[E]) Contains(e E) bool { _, ok := s.m.Get(e); return ok }

// Delete removes e if present.
func (s *Set[E]) Delete(e E) bool { return s.m.Delete(e) }

// Clear removes all elements.
func (s *Set[E]) Clear() { s.m.Clear() }

// Len returns the number of distinct elements.
func (s *Set[E]) Len() int { return s.m.Len() }

// Clone returns an independent set with the same insertion order.
func (s *Set[E]) Clone() *Set[E] {
	n := NewSet[E](s.m.h)
	for e := range s.All() {
		n.Insert(e)
	}
	return n
}

// DeleteFunc removes elements selected by f in insertion order.
func (s *Set[E]) DeleteFunc(f func(E) bool) int {
	n := 0
	keys := make([]E, 0, s.Len())
	for e := range s.All() {
		keys = append(keys, e)
	}
	for _, e := range keys {
		if f(e) && s.Delete(e) {
			n++
		}
	}
	return n
}

// All returns elements in insertion order.
func (s *Set[E]) All() iter.Seq[E] { return s.m.Keys() }

// Backward returns elements in reverse insertion order.
func (s *Set[E]) Backward() iter.Seq[E] {
	return func(y func(E) bool) {
		for k := range s.m.Backward() {
			if !y(k) {
				return
			}
		}
	}
}

// MoveToFront moves an existing element to the front.
func (s *Set[E]) MoveToFront(e E) bool { return s.m.MoveToFront(e) }

// MoveToBack moves an existing element to the back.
func (s *Set[E]) MoveToBack(e E) bool { return s.m.MoveToBack(e) }
