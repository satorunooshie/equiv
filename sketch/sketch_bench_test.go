package sketch

import (
	"hash/maphash"
	"testing"
)

func BenchmarkCountMinAdd(b *testing.B) {
	s, _ := NewCountMin[int](maphash.ComparableHasher[int]{}, .01, .01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(i&9999, 1)
	}
}

func BenchmarkHLLAdd(b *testing.B) {
	f := NewFamily[int](maphash.ComparableHasher[int]{})
	h, _ := f.NewHyperLogLog(12)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(i & 9999)
	}
}
