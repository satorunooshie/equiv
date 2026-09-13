package concurrent

import (
	"hash/maphash"
	"strconv"
	"testing"
)

var concurrentBenchmarkSizes = []int{8, 64, 1 << 10, 1 << 16, 1 << 20}

func BenchmarkConcurrentMapOperations(b *testing.B) {
	for _, size := range concurrentBenchmarkSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			m, err := NewMap[int, int](maphash.ComparableHasher[int]{}, WithShards(64))
			if err != nil {
				b.Fatal(err)
			}
			for i := range size {
				m.Set(i, i)
			}
			b.Run("get", func(b *testing.B) {
				i := 0
				for b.Loop() {
					_, _ = m.Get(i % size)
					i++
				}
			})
			b.Run("miss", func(b *testing.B) {
				i := 0
				for b.Loop() {
					_, _ = m.Get(size + i%size)
					i++
				}
			})
			b.Run("insert", func(b *testing.B) {
				i := 0
				for b.Loop() {
					m.Set(size+i, i)
					i++
				}
			})
			b.Run("replace", func(b *testing.B) {
				i := 0
				for b.Loop() {
					m.Set(i%size, i)
					i++
				}
			})
			b.Run("delete", func(b *testing.B) {
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
