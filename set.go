package equiv

import (
	"hash/maphash"
	"iter"
)

// Set stores distinct semantic elements. Its zero value is invalid and it is
// not safe for concurrent use or copying after first use.
type Set[E any] struct{ m *Map[E, struct{}] }

// NewSet constructs a semantic set using h as its identity protocol.
func NewSet[E any](h maphash.Hasher[E]) *Set[E] { return &Set[E]{NewMap[E, struct{}](h)} }

// NewSetCapacity constructs a set with an initial capacity hint.
func NewSetCapacity[E any](h maphash.Hasher[E], n int) *Set[E] {
	return &Set[E]{NewMapCapacity[E, struct{}](h, n)}
}

// Insert adds e and reports whether it was new.
func (s *Set[E]) Insert(e E) bool { _, ok := s.m.Set(e, struct{}{}); return !ok }

// Lookup returns the stored canonical representation of e.
func (s *Set[E]) Lookup(e E) (E, bool) { k, _, ok := s.m.GetEntry(e); return k, ok }

// Contains reports whether e is present.
func (s *Set[E]) Contains(e E) bool { _, ok := s.m.Get(e); return ok }

// Delete removes e if present.
func (s *Set[E]) Delete(e E) bool { return s.m.Delete(e) }

// Clear removes all elements without changing the hash domain.
func (s *Set[E]) Clear() { s.m.Clear() }

// Len returns the number of distinct elements.
func (s *Set[E]) Len() int { return s.m.Len() }

// Clone returns an independent set with a fresh hash domain.
func (s *Set[E]) Clone() *Set[E] { return &Set[E]{s.m.Clone()} }

// All returns the elements in the underlying map's iteration order.
func (s *Set[E]) All() iter.Seq[E] { return s.m.Keys() }

func (s *Set[E]) DeleteFunc(f func(E) bool) int {
	return s.m.DeleteFunc(func(e E, _ struct{}) bool { return f(e) })
}

func (s *Set[E]) Union(x iter.Seq[E]) *Set[E] {
	n := s.Clone()
	for e := range x {
		n.Insert(e)
	}
	return n
}

func (s *Set[E]) Intersection(x iter.Seq[E]) *Set[E] {
	n := NewSet[E](s.m.h)
	for e := range x {
		if s.Contains(e) {
			n.Insert(e)
		}
	}
	return n
}

func (s *Set[E]) Difference(x iter.Seq[E]) *Set[E] {
	n := s.Clone()
	for e := range x {
		n.Delete(e)
	}
	return n
}

func (s *Set[E]) SymmetricDifference(x iter.Seq[E]) *Set[E] {
	n := s.Clone()
	for e := range x {
		if !n.Delete(e) {
			n.Insert(e)
		}
	}
	return n
}

func (s *Set[E]) ContainsAll(x iter.Seq[E]) bool {
	for e := range x {
		if !s.Contains(e) {
			return false
		}
	}
	return true
}

func (s *Set[E]) Intersects(x iter.Seq[E]) bool {
	for e := range x {
		if s.Contains(e) {
			return true
		}
	}
	return false
}
