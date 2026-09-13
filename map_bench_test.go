package equiv_test

import (
	"hash/maphash"
	"strconv"
	"testing"

	"github.com/satorunooshie/equiv"
)

var benchmarkSizes = []int{8, 64, 1 << 10, 1 << 16, 1 << 20}

func BenchmarkMapOperations(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(sizeName(size), func(b *testing.B) {
			b.Run("equiv/hit", func(b *testing.B) {
				m := filledMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(i % size)
				}
			})
			b.Run("equiv/miss", func(b *testing.B) {
				m := filledMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = m.Get(size + i%size)
				}
			})
			b.Run("equiv/insert", func(b *testing.B) {
				m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(i, i)
				}
			})
			b.Run("equiv/replace", func(b *testing.B) {
				m := filledMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Set(i%size, i)
				}
			})
			b.Run("equiv/delete", func(b *testing.B) {
				m := filledMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k := i % size
					m.Delete(k)
					m.Set(k, i)
				}
			})
			b.Run("equiv/iterate", func(b *testing.B) {
				m := filledMap(size)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for range m.All() {
					}
				}
			})
			b.Run("equiv/resize", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
					for k := 0; k < size; k++ {
						m.Set(k, k)
					}
				}
			})
			b.Run("equiv/clear", func(b *testing.B) {
				b.StopTimer()
				m := filledMap(size)
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

func BenchmarkComparableMapGet(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(sizeName(size), func(b *testing.B) {
			m := make(map[int]int, size)
			for i := 0; i < size; i++ {
				m[i] = i
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = m[i%size]
			}
		})
	}
}

func BenchmarkMapCollisionLookup(b *testing.B) {
	// Full-hash collisions are intentionally quadratic in this scalar table;
	// keep the pathological workload bounded so release verification terminates.
	for _, size := range []int{8, 64, 1 << 10, 1 << 16} {
		b.Run(sizeName(size), func(b *testing.B) {
			m := equiv.NewMap[int, int](constantHasher{})
			for i := 0; i < size; i++ {
				m.Set(i, i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = m.Get(i % size)
			}
		})
	}
}

func BenchmarkStringEncodedMapGet(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(sizeName(size), func(b *testing.B) {
			m := make(map[string]int, size)
			for i := 0; i < size; i++ {
				m[strconv.Itoa(i)] = i
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = m[strconv.Itoa(i%size)]
			}
		})
	}
}

func filledMap(size int) *equiv.Map[int, int] {
	m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
	for i := 0; i < size; i++ {
		m.Set(i, i)
	}
	return m
}

func sizeName(size int) string {
	switch size {
	case 1 << 10:
		return "1K"
	case 1 << 16:
		return "64K"
	case 1 << 20:
		return "1M"
	default:
		return strconv.Itoa(size)
	}
}
