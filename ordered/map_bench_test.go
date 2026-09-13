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
				i := 0
				for b.Loop() {
					_, _ = m.Get(i % size)
					i++
				}
			})
			b.Run("miss", func(b *testing.B) {
				m := filledOrderedMap(size)
				i := 0
				for b.Loop() {
					_, _ = m.Get(size + i%size)
					i++
				}
			})
			b.Run("insert", func(b *testing.B) {
				m := NewMap[int, int](maphash.ComparableHasher[int]{})
				i := 0
				for b.Loop() {
					m.Set(i, i)
					i++
				}
			})
			b.Run("replace", func(b *testing.B) {
				m := filledOrderedMap(size)
				i := 0
				for b.Loop() {
					m.Set(i%size, i)
					i++
				}
			})
			b.Run("iterate", func(b *testing.B) {
				m := filledOrderedMap(size)
				for b.Loop() {
					for range m.All() {
					}
				}
			})
			b.Run("delete", func(b *testing.B) {
				m := filledOrderedMap(size)
				i := 0
				for b.Loop() {
					k := i % size
					m.Delete(k)
					m.Set(k, i)
					i++
				}
			})
			b.Run("clear", func(b *testing.B) {
				b.StopTimer()
				m := filledOrderedMap(size)
				b.StartTimer()
				for b.Loop() {
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
