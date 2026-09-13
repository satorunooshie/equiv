package multimap

import (
	"hash/maphash"
	"strconv"
	"testing"
)

func BenchmarkMapOperations(b *testing.B) {
	for _, size := range []int{8, 64, 1 << 10, 1 << 16, 1 << 20} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			m := New[int, int](maphash.ComparableHasher[int]{})
			for i := range size {
				m.Add(i, i)
			}
			b.Run("values", func(b *testing.B) {
				i := 0
				for b.Loop() {
					for range m.Values(i % size) {
					}
					i++
				}
			})
			b.Run("add", func(b *testing.B) {
				i := 0
				for b.Loop() {
					m.Add(i%size, i)
					i++
				}
			})
			b.Run("iterate", func(b *testing.B) {
				for b.Loop() {
					for range m.All() {
					}
				}
			})
		})
	}
}
