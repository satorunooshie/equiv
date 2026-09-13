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
	for i := 0; i < 10; i++ {
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
