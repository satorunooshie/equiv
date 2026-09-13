package multiset

import (
	"hash/maphash"
	"math"
	"testing"
)

func TestAddNZeroDoesNotCreateMember(t *testing.T) {
	s := New[int](maphash.ComparableHasher[int]{})
	if got := s.AddN(1, 0); got != 0 || s.DistinctLen() != 0 || s.Total() != 0 {
		t.Fatalf("zero add changed set: got=%d distinct=%d total=%d", got, s.DistinctLen(), s.Total())
	}
}

func TestAddNPanicsWhenTotalOverflows(t *testing.T) {
	s := New[int](maphash.ComparableHasher[int]{})
	s.AddN(1, math.MaxUint64)
	defer func() {
		if recover() == nil {
			t.Fatal("expected total overflow panic")
		}
	}()
	s.AddN(2, 1)
}
