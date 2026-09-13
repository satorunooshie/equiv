package cuckoo

import (
	"hash/maphash"
	"testing"
)

func BenchmarkCuckooContains(b *testing.B) {
	f, _ := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 10000})
	for i := range 5000 {
		f.Insert(i)
	}
	i := 0
	for b.Loop() {
		f.Contains(i & 4999)
		i++
	}
}
