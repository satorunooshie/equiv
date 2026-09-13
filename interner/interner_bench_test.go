package interner

import (
	"hash/maphash"
	"strconv"
	"testing"
)

func BenchmarkStrongOperations(b *testing.B) {
	for _, size := range []int{8, 64, 1 << 10, 1 << 16, 1 << 20} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			i := NewStrong[int](maphash.ComparableHasher[int]{})
			for k := range size {
				i.Intern(k)
			}
			b.Run("lookup", func(b *testing.B) {
				b.ResetTimer()
				for k := 0; k < b.N; k++ {
					_, _ = i.Lookup(k % size)
				}
			})
			b.Run("intern", func(b *testing.B) {
				b.ResetTimer()
				for k := 0; k < b.N; k++ {
					i.Intern(k % size)
				}
			})
			b.Run("iterate", func(b *testing.B) {
				b.ResetTimer()
				for k := 0; k < b.N; k++ {
					for range i.All() {
					}
				}
			})
		})
	}
}
