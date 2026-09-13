// Package multiset provides non-concurrent semantic bags with uint64 counts.
// Counter overflow is reported by panic rather than silently saturated.
package multiset

import (
	"hash/maphash"
	"iter"
	"math"

	"github.com/satorunooshie/equiv"
)

// Set stores semantic elements with uint64 multiplicities. Its zero value is
// invalid and it is not safe for concurrent use or copying after first use.
type Set[E any] struct {
	m     *equiv.Map[E, uint64]
	total uint64
}

// New constructs a multiset using h for element identity.
func New[E any](h maphash.Hasher[E]) *Set[E] { return &Set[E]{m: equiv.NewMap[E, uint64](h)} }

// Add increments e's multiplicity and returns the new count.
func (s *Set[E]) Add(e E) uint64 { return s.AddN(e, 1) }

// AddN increments e's multiplicity by n and returns the new count.
func (s *Set[E]) AddN(e E, n uint64) uint64 {
	old, _ := s.m.Get(e)
	if n == 0 {
		return old
	}
	if n > math.MaxUint64-old {
		panic("multiset: count overflow")
	}
	if n > math.MaxUint64-s.total {
		panic("multiset: total overflow")
	}
	s.m.Set(e, old+n)
	s.total += n
	return old + n
}

// Remove decrements e's multiplicity by one and returns the remaining count.
func (s *Set[E]) Remove(e E) uint64 { return s.RemoveN(e, 1) }

// RemoveN decrements e's multiplicity by n and returns the remaining count.
func (s *Set[E]) RemoveN(e E, n uint64) uint64 {
	old, ok := s.m.Get(e)
	if !ok {
		return 0
	}
	if n >= old {
		s.m.Delete(e)
		s.total -= old
		return old
	}
	s.m.Set(e, old-n)
	s.total -= n
	return n
}

// Count returns e's multiplicity.
func (s *Set[E]) Count(e E) uint64 { n, _ := s.m.Get(e); return n }

// DistinctLen returns the number of elements with nonzero multiplicity.
func (s *Set[E]) DistinctLen() int { return s.m.Len() }

// Total returns the sum of all multiplicities.
func (s *Set[E]) Total() uint64 { return s.total }

// All returns each distinct element and its multiplicity.
func (s *Set[E]) All() iter.Seq2[E, uint64] { return s.m.All() }

// Elements returns elements repeated according to their multiplicity.
func (s *Set[E]) Elements() iter.Seq[E] {
	return func(y func(E) bool) {
		for e, n := range s.m.All() {
			for i := uint64(0); i < n; i++ {
				if !y(e) {
					return
				}
			}
		}
	}
}

// Clear removes all elements and resets the total count.
func (s *Set[E]) Clear() { s.m.Clear(); s.total = 0 }
