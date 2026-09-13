package cuckoo

import (
	"hash/maphash"
	"testing"
)

func BenchmarkCuckooContains(b *testing.B) {
	f, _ := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 10000})
	for i := 0; i < 5000; i++ {
		f.Insert(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Contains(i & 4999)
	}
}
