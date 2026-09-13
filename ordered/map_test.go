package ordered

import (
	"hash/maphash"
	"testing"
)

func TestIteratorFailsFastAfterOrderMutation(t *testing.T) {
	m := NewMap[int, int](maphash.ComparableHasher[int]{})
	m.Set(1, 1)
	m.Set(2, 2)
	defer func() {
		if recover() == nil {
			t.Fatal("expected order mutation panic")
		}
	}()
	for k := range m.All() {
		m.MoveToBack(k)
	}
}

func TestValueUpdateDoesNotInvalidateIterator(t *testing.T) {
	m := NewMap[int, int](maphash.ComparableHasher[int]{})
	m.Set(1, 1)
	m.Set(2, 2)
	seen := 0
	for k, v := range m.All() {
		m.Set(k, v+1)
		seen++
	}
	if seen != 2 {
		t.Fatalf("visited %d entries", seen)
	}
}

func TestDeleteFuncDoesNotUseFailFastIterator(t *testing.T) {
	m := NewSet[int](maphash.ComparableHasher[int]{})
	for i := range 10 {
		m.Insert(i)
	}
	if got := m.DeleteFunc(func(v int) bool { return v%2 == 0 }); got != 5 || m.Len() != 5 {
		t.Fatalf("deleted=%d len=%d", got, m.Len())
	}
}

func TestOrderedCapacityConstructor(t *testing.T) {
	m := NewMapCapacity[int, int](maphash.ComparableHasher[int]{}, 32)
	m.Set(1, 1)
	if v, ok := m.Get(1); !ok || v != 1 {
		t.Fatalf("capacity constructor lookup=(%d,%v)", v, ok)
	}
	s := NewSetCapacity[int](maphash.ComparableHasher[int]{}, 32)
	if !s.Insert(1) || !s.Contains(1) {
		t.Fatal("ordered set capacity constructor failed")
	}
}

func TestOrderedMapClonePreservesOrderAndIndependence(t *testing.T) {
	m := NewMap[int, string](maphash.ComparableHasher[int]{})
	m.Set(2, "two")
	m.Set(1, "one")
	clone := m.Clone()
	var got []int
	for k := range clone.Keys() {
		got = append(got, k)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Fatalf("clone order=%v", got)
	}
	m.Set(2, "updated")
	if v, _ := clone.Get(2); v != "two" {
		t.Fatalf("clone changed after source update: %q", v)
	}
}

func TestMoveOperationsAndBackwardIteration(t *testing.T) {
	m := NewMap[string, int](maphash.ComparableHasher[string]{})
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)
	if !m.MoveToFront("c") || !m.MoveToBack("a") {
		t.Fatal("move operation failed")
	}

	var forward []string
	for k := range m.Keys() {
		forward = append(forward, k)
	}
	var backward []string
	for k := range m.Backward() {
		backward = append(backward, k)
	}
	if len(forward) != 3 || forward[0] != "c" || forward[1] != "b" || forward[2] != "a" {
		t.Fatalf("forward=%v, want [c b a]", forward)
	}
	if len(backward) != 3 || backward[0] != "a" || backward[1] != "b" || backward[2] != "c" {
		t.Fatalf("backward=%v, want [a b c]", backward)
	}
}

func TestOrderedMapHelpersAndSetLifecycle(t *testing.T) {
	m := NewMap[string, int](maphash.ComparableHasher[string]{})
	if v, existed := m.GetOrSet("a", 1); existed || v != 1 {
		t.Fatalf("first GetOrSet=(%d,%v)", v, existed)
	}
	if v, existed := m.GetOrSet("a", 2); !existed || v != 1 {
		t.Fatalf("second GetOrSet=(%d,%v)", v, existed)
	}
	calls := 0
	if v, hit := m.GetOrCompute("b", func() int { calls++; return 2 }); hit || v != 2 || calls != 1 {
		t.Fatal("GetOrCompute failed")
	}
	if k, v, ok := m.GetEntry("a"); !ok || k != "a" || v != 1 {
		t.Fatalf("GetEntry=(%q,%d,%v)", k, v, ok)
	}
	var values []int
	for v := range m.Values() {
		values = append(values, v)
	}
	if len(values) != 2 {
		t.Fatalf("values=%v", values)
	}
	if got := m.DeleteFunc(func(_ string, v int) bool { return v == 1 }); got != 1 || m.Len() != 1 {
		t.Fatal("DeleteFunc failed")
	}
	m.Clear()
	if m.Len() != 0 {
		t.Fatal("Clear failed")
	}

	s := NewSet[string](maphash.ComparableHasher[string]{})
	s.Insert("a")
	s.Insert("b")
	if v, ok := s.Lookup("a"); !ok || v != "a" {
		t.Fatalf("set lookup=(%q,%v)", v, ok)
	}
	if !s.MoveToFront("b") || !s.MoveToBack("b") {
		t.Fatal("set move failed")
	}
	clone := s.Clone()
	var reverse []string
	for e := range s.Backward() {
		reverse = append(reverse, e)
	}
	if len(reverse) != 2 {
		t.Fatalf("set backward=%v", reverse)
	}
	s.Clear()
	if s.Len() != 0 || clone.Len() != 2 {
		t.Fatal("set clear/clone failed")
	}
}

func TestOrderedMissingKeyOperationsAreNoOps(t *testing.T) {
	m := NewMap[int, int](maphash.ComparableHasher[int]{})
	if m.Delete(1) || m.MoveToFront(1) || m.MoveToBack(1) {
		t.Fatal("missing map key operation reported success")
	}
	s := NewSet[int](maphash.ComparableHasher[int]{})
	if s.Delete(1) || s.MoveToFront(1) || s.MoveToBack(1) {
		t.Fatal("missing set element operation reported success")
	}
}
