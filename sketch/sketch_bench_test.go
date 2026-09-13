package sketch

import (
	"hash/maphash"
	"testing"
)

func BenchmarkCountMinAdd(b *testing.B) {
	s, _ := NewCountMin[int](maphash.ComparableHasher[int]{}, .01, .01)
	i := 0
	for b.Loop() {
		s.Add(i&9999, 1)
		i++
	}
}

func BenchmarkHLLAdd(b *testing.B) {
	f := NewFamily[int](maphash.ComparableHasher[int]{})
	h, _ := f.NewHyperLogLog(12)
	i := 0
	for b.Loop() {
		h.Add(i & 9999)
		i++
	}
}
