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

func TestCountsTotalsAndElements(t *testing.T) {
	s := New[string](maphash.ComparableHasher[string]{})
	s.AddN("go", 3)
	s.Add("cache")
	if s.Count("go") != 3 || s.Count("cache") != 1 || s.DistinctLen() != 2 || s.Total() != 4 {
		t.Fatalf("counts=(%d,%d), distinct=%d total=%d", s.Count("go"), s.Count("cache"), s.DistinctLen(), s.Total())
	}
	if removed := s.RemoveN("go", 2); removed != 2 || s.Count("go") != 1 || s.Total() != 2 {
		t.Fatalf("remove=%d count=%d total=%d", removed, s.Count("go"), s.Total())
	}
	var elements []string
	for e := range s.Elements() {
		elements = append(elements, e)
	}
	if len(elements) != 2 {
		t.Fatalf("elements=%v, want two values", elements)
	}
	seen := 0
	for _, count := range s.All() {
		if count > 0 {
			seen++
		}
	}
	if seen != 2 {
		t.Fatalf("All visited %d distinct values", seen)
	}
	s.Clear()
	if s.Total() != 0 || s.DistinctLen() != 0 {
		t.Fatal("clear did not empty multiset")
	}
}

func TestRemoveAbsentIsNoOp(t *testing.T) {
	s := New[int](maphash.ComparableHasher[int]{})
	if got := s.Remove(99); got != 0 || s.Total() != 0 || s.DistinctLen() != 0 {
		t.Fatalf("absent remove=(%d,%d,%d)", got, s.Total(), s.DistinctLen())
	}
}
