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
			for i := 0; i < size; i++ {
				m.Set(i, i)
			}
			b.Run("get", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(i % size)
				}
			})
			b.Run("miss", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(size + i%size)
				}
			})
			b.Run("insert", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(size+i, i)
				}
			})
			b.Run("replace", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(i%size, i)
				}
			})
			b.Run("delete", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k := i % size
					m.Delete(k)
					m.Set(k, i)
				}
			})
			b.Run("clear", func(b *testing.B) {
				b.StopTimer()
				b.StartTimer()
				for i := 0; i < b.N; i++ {
					m.Clear()
					for k := 0; k < size; k++ {
						m.Set(k, k)
					}
				}
			})
		})
	}
}
