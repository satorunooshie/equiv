package equiv_test

import (
	"hash/maphash"
	"slices"
	"testing"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func FuzzMapBytes(f *testing.F) {
	f.Add([]byte("alpha"), []byte("beta"))
	f.Fuzz(func(t *testing.T, a, b []byte) {
		m := equiv.NewMap[[]byte, int](hashers.Bytes())
		m.Set(a, 1)
		if v, ok := m.Get(slices.Clone(a)); !ok || v != 1 {
			t.Fatalf("semantic lookup failed")
		}
		m.Set(b, 2)
		if m.Len() < 1 || m.Len() > 2 {
			t.Fatalf("invalid length %d", m.Len())
		}
		m.Delete(a)
	})
}

type constantHasher struct{}

func (constantHasher) Hash(h *maphash.Hash, _ int) { h.WriteByte(1) }
func (constantHasher) Equal(a, b int) bool         { return a == b }
func TestMapCollisionCorrectness(t *testing.T) {
	m := equiv.NewMap[int, int](constantHasher{})
	for i := range 1000 {
		m.Set(i, i*i)
	}
	for i := range 1000 {
		if v, ok := m.Get(i); !ok || v != i*i {
			t.Fatalf("collision lookup %d=(%d,%v)", i, v, ok)
		}
	}
	if m.Len() != 1000 {
		t.Fatal(m.Len())
	}
}
