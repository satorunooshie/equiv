package table

import (
	"hash/maphash"
	"testing"
)

type collisionHasher struct{}

func (collisionHasher) Hash(h *maphash.Hash, _ int) { h.WriteByte(0) }
func (collisionHasher) Equal(a, b int) bool         { return a == b }
func TestResizeDeleteAndClear(t *testing.T) {
	x := New[int, string](collisionHasher{}, 1)
	for i := range 100 {
		x.Set(i, string(rune(i)))
	}
	for i := range 100 {
		if v, ok := x.Get(i); !ok || v != string(rune(i)) {
			t.Fatalf("lookup %d", i)
		}
	}
	for i := range 50 {
		if !x.Delete(i) {
			t.Fatalf("delete %d", i)
		}
	}
	if x.Len() != 50 {
		t.Fatal(x.Len())
	}
	x.Clear()
	if x.Len() != 0 {
		t.Fatal(x.Len())
	}
}
