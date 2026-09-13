package equiv_test

import (
	"hash/maphash"
	"math"
	"testing"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func TestMapHandlesFullHashCollisions(t *testing.T) {
	m := equiv.NewMap[int, int](constantHasher{})
	for i := 0; i < 100; i++ {
		m.Set(i, i*i)
	}
	for i := 0; i < 100; i++ {
		if v, ok := m.Get(i); !ok || v != i*i {
			t.Fatalf("Get(%d)=(%d,%v)", i, v, ok)
		}
	}
	for i := 0; i < 50; i++ {
		if !m.Delete(i) {
			t.Fatalf("Delete(%d) failed", i)
		}
	}
	for i := 0; i < 50; i++ {
		m.Set(i, -i)
	}
	clone := m.Clone()
	if clone.Len() != 100 {
		t.Fatalf("Clone Len=%d", clone.Len())
	}
	for i := 0; i < 50; i++ {
		if v, ok := clone.Get(i); !ok || v != -i {
			t.Fatalf("Clone Get(%d)=(%d,%v)", i, v, ok)
		}
	}
}

func TestMapSemanticRepresentativeAndSlice(t *testing.T) {
	h := hashers.Slice[string](maphash.ComparableHasher[string]{})
	m := equiv.NewMap[[]string, int](h)
	k := []string{"a", "b"}
	if _, ok := m.Set(k, 1); ok {
		t.Fatal("first Set replaced")
	}
	if v, ok := m.Get([]string{"a", "b"}); !ok || v != 1 {
		t.Fatalf("Get=(%v,%v)", v, ok)
	}
	old, ok := m.Set([]string{"a", "b"}, 2)
	if !ok || old != 1 {
		t.Fatalf("replacement=(%v,%v)", old, ok)
	}
	stored, _, ok := m.GetEntry([]string{"a", "b"})
	if !ok || &stored[0] != &k[0] {
		t.Fatal("first representative was not retained")
	}
}

func TestMapIteratorFailFast(t *testing.T) {
	m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
	m.Set(1, 1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected mutation panic")
		}
	}()
	for k := range m.Keys() {
		m.Set(k+1, 1)
	}
}

func TestDeleteFuncDoesNotSkipEntries(t *testing.T) {
	m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
	for i := 0; i < 100; i++ {
		m.Set(i, i)
	}
	if n := m.DeleteFunc(func(k, v int) bool { return k%2 == 0 }); n != 50 || m.Len() != 50 {
		t.Fatalf("removed=%d len=%d", n, m.Len())
	}
}

func TestMapRejectsOverflowingCapacity(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected oversized capacity panic")
		}
	}()
	equiv.NewMapCapacity[int, int](maphash.ComparableHasher[int]{}, math.MaxInt)
}

func TestMapIteratorsDoNotAllocate(t *testing.T) {
	m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
	for i := 0; i < 32; i++ {
		m.Set(i, i)
	}
	for name, run := range map[string]func(){
		"get": func() {
			_, _ = m.Get(17)
		},
		"failed-get": func() {
			_, _ = m.Get(-1)
		},
		"replace": func() {
			m.Set(17, 17)
		},
		"all": func() {
			for range m.All() {
			}
		},
		"keys": func() {
			for range m.Keys() {
			}
		},
		"values": func() {
			for range m.Values() {
			}
		},
	} {
		if got := testing.AllocsPerRun(100, run); got != 0 {
			t.Errorf("%s allocations = %v, want 0", name, got)
		}
	}
}
