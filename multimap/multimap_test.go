package multimap

import (
	"hash/maphash"
	"testing"
)

type constantHasher struct{}

func (constantHasher) Hash(h *maphash.Hash, _ string) { h.WriteString("constant") }
func (constantHasher) Equal(a, b string) bool         { return a == b }

func TestAddPreservesValueOrderAndGroupsByKey(t *testing.T) {
	m := New[string, string](maphash.ComparableHasher[string]{})
	m.Add("doc-1", "go")
	m.Add("doc-1", "hashing")
	m.Add("doc-2", "cache")
	var got []string
	for v := range m.Values("doc-1") {
		got = append(got, v)
	}
	if len(got) != 2 || got[0] != "go" || got[1] != "hashing" {
		t.Fatalf("values=%v, want [go hashing]", got)
	}
	if m.KeyLen() != 2 || m.ValueLen() != 3 {
		t.Fatalf("key/value lengths=(%d,%d), want (2,3)", m.KeyLen(), m.ValueLen())
	}
}

func TestDeleteFuncRemovesValuesAndEmptyKeys(t *testing.T) {
	m := New[string, int](maphash.ComparableHasher[string]{})
	m.Add("job", 1)
	m.Add("job", 2)
	if got := m.DeleteFunc("job", func(v int) bool { return v < 3 }); got != 2 {
		t.Fatalf("deleted=%d, want 2", got)
	}
	if m.ContainsKey("job") || m.KeyLen() != 0 || m.ValueLen() != 0 {
		t.Fatalf("empty key was retained: keys=%d values=%d", m.KeyLen(), m.ValueLen())
	}
	if got := m.DeleteFunc("missing", func(int) bool { return true }); got != 0 {
		t.Fatalf("missing delete=%d, want 0", got)
	}
}

func TestDeleteFuncKeepsNonMatchingValues(t *testing.T) {
	cases := []struct {
		name      string
		remove    func(int) bool
		deleted   int
		remaining []int
	}{
		{name: "first value", remove: func(v int) bool { return v == 1 }, deleted: 1, remaining: []int{2}},
		{name: "no values", remove: func(int) bool { return false }, deleted: 0, remaining: []int{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New[string, int](maphash.ComparableHasher[string]{})
			m.Add("job", 1)
			m.Add("job", 2)
			if got := m.DeleteFunc("job", tc.remove); got != tc.deleted {
				t.Fatalf("deleted=%d, want %d", got, tc.deleted)
			}
			var remaining []int
			for v := range m.Values("job") {
				remaining = append(remaining, v)
			}
			if len(remaining) != len(tc.remaining) {
				t.Fatalf("remaining=%v, want %v", remaining, tc.remaining)
			}
			for i := range remaining {
				if remaining[i] != tc.remaining[i] {
					t.Fatalf("remaining=%v, want %v", remaining, tc.remaining)
				}
			}
		})
	}
}

func TestAllKeysAndClear(t *testing.T) {
	m := New[string, int](maphash.ComparableHasher[string]{})
	m.Add("a", 1)
	m.Add("b", 2)
	seen := 0
	for range m.All() {
		seen++
	}
	if seen != 2 {
		t.Fatalf("All visited %d values", seen)
	}
	keys := 0
	for range m.Keys() {
		keys++
	}
	if keys != 2 || !m.DeleteKey("a") || m.ContainsKey("a") {
		t.Fatalf("keys/delete contract failed: keys=%d", keys)
	}
	m.Clear()
	if m.KeyLen() != 0 || m.ValueLen() != 0 {
		t.Fatal("clear did not empty multimap")
	}
}

func TestCollisionSafeKeyOperations(t *testing.T) {
	m := New[string, int](constantHasher{})
	m.Add("a", 1)
	m.Add("b", 2)
	if !m.ContainsKey("a") || !m.ContainsKey("b") || m.KeyLen() != 2 {
		t.Fatalf("collision keys were not retained: len=%d", m.KeyLen())
	}
	if !m.DeleteKey("a") || m.ContainsKey("a") || !m.ContainsKey("b") {
		t.Fatal("collision-safe delete removed the wrong key")
	}
}
