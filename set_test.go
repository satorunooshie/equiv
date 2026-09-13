package equiv

import (
	"hash/maphash"
	"testing"
)

func TestSetOperationsAndAlgebra(t *testing.T) {
	h := maphash.ComparableHasher[int]{}
	s := NewSet[int](h)
	if !s.Insert(1) || !s.Insert(2) || s.Insert(1) || !s.Contains(2) {
		t.Fatal("set insert/contains contract failed")
	}
	if k, ok := s.Lookup(1); !ok || k != 1 {
		t.Fatalf("lookup=(%d,%v)", k, ok)
	}
	other := NewSet[int](h)
	other.Insert(2)
	other.Insert(3)
	seq := other.All()
	if !s.ContainsAll(func(y func(int) bool) { y(1); y(2) }) || !s.Intersects(seq) {
		t.Fatal("set predicates failed")
	}
	if got := s.Union(other.All()); got.Len() != 3 {
		t.Fatalf("union len=%d", got.Len())
	}
	if got := s.Intersection(other.All()); got.Len() != 1 || !got.Contains(2) {
		t.Fatal("intersection failed")
	}
	if got := s.Difference(other.All()); got.Len() != 1 || !got.Contains(1) {
		t.Fatal("difference failed")
	}
	if got := s.SymmetricDifference(other.All()); got.Len() != 2 || !got.Contains(1) || !got.Contains(3) {
		t.Fatal("symmetric difference failed")
	}
	if removed := s.DeleteFunc(func(v int) bool { return v == 1 }); removed != 1 || s.Len() != 1 {
		t.Fatal("set delete func failed")
	}
	s.Clear()
	if s.Len() != 0 {
		t.Fatal("set clear failed")
	}
}

func TestMapGetOrSetComputeAndClone(t *testing.T) {
	m := NewMap[string, int](maphash.ComparableHasher[string]{})
	if v, existed := m.GetOrSet("a", 1); existed || v != 1 {
		t.Fatalf("first GetOrSet=(%d,%v)", v, existed)
	}
	if v, existed := m.GetOrSet("a", 2); !existed || v != 1 {
		t.Fatalf("second GetOrSet=(%d,%v)", v, existed)
	}
	calls := 0
	if v, hit := m.GetOrCompute("a", func() int { calls++; return 3 }); !hit || v != 1 || calls != 0 {
		t.Fatal("GetOrCompute called on hit")
	}
	if v, hit := m.GetOrCompute("b", func() int { calls++; return 4 }); hit || v != 4 || calls != 1 {
		t.Fatal("GetOrCompute missed insert")
	}
	clone := m.Clone()
	m.Delete("a")
	if _, ok := m.Get("a"); ok {
		t.Fatal("delete failed")
	}
	if v, ok := clone.Get("a"); !ok || v != 1 {
		t.Fatal("clone was not independent")
	}
	m.Clear()
	if m.Len() != 0 {
		t.Fatal("map clear failed")
	}
}

func TestEmptySetAlgebraAndPredicates(t *testing.T) {
	left := NewSet[int](maphash.ComparableHasher[int]{})
	right := NewSet[int](maphash.ComparableHasher[int]{})
	right.Insert(1)
	empty := func(y func(int) bool) {}
	if left.ContainsAll(empty) == false || left.Intersects(right.All()) {
		t.Fatal("empty set predicates failed")
	}
	if left.Union(right.All()).Len() != 1 || left.Intersection(right.All()).Len() != 0 {
		t.Fatal("empty set union/intersection failed")
	}
	if left.Difference(right.All()).Len() != 0 || left.SymmetricDifference(right.All()).Len() != 1 {
		t.Fatal("empty set difference failed")
	}
}
