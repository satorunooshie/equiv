package ordered

import (
	"hash/maphash"
	"strconv"
	"testing"
)

var orderedBenchmarkSizes = []int{8, 64, 1 << 10, 1 << 16, 1 << 20}

func BenchmarkOrderedMapOperations(b *testing.B) {
	for _, size := range orderedBenchmarkSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.Run("get", func(b *testing.B) {
				m := filledOrderedMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(i % size)
				}
			})
			b.Run("miss", func(b *testing.B) {
				m := filledOrderedMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(size + i%size)
				}
			})
			b.Run("insert", func(b *testing.B) {
				m := NewMap[int, int](maphash.ComparableHasher[int]{})
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(i, i)
				}
			})
			b.Run("replace", func(b *testing.B) {
				m := filledOrderedMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(i%size, i)
				}
			})
			b.Run("iterate", func(b *testing.B) {
				m := filledOrderedMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for range m.All() {
					}
				}
			})
			b.Run("delete", func(b *testing.B) {
				m := filledOrderedMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k := i % size
					m.Delete(k)
					m.Set(k, i)
				}
			})
			b.Run("clear", func(b *testing.B) {
				b.StopTimer()
				m := filledOrderedMap(size)
				b.StartTimer()
				for i := 0; i < b.N; i++ {
					m.Clear()
					for k := range size {
						m.Set(k, k)
					}
				}
			})
		})
	}
}

func filledOrderedMap(size int) *Map[int, int] {
	m := NewMap[int, int](maphash.ComparableHasher[int]{})
	for i := range size {
		m.Set(i, i)
	}
	return m
}
