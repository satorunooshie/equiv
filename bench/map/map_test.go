package map_test

import (
	"hash/maphash"
	"strings"
	"testing"

	"github.com/aristanetworks/gomap"
	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

var sinkInt int

func BenchmarkIntGet(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 65536} {
		b.Run("Builtin/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			m := make(map[int]int, size)
			for _, key := range keys {
				m[key] = key
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m[keys[0]]
			}
		})
		b.Run("Gomap/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			m := gomap.NewHint[int, int](size, intEqual, intHash)
			for _, key := range keys {
				m.Set(key, key)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m.Get(keys[0])
			}
		})
		b.Run("Equiv/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
			for _, key := range keys {
				m.Set(key, key)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m.Get(keys[0])
			}
		})
	}
}

func BenchmarkSemanticGet(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 65536} {
		b.Run("RawCanonical/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			m := make(map[string]int, size)
			for i, key := range keys {
				m[key] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m[keys[0]]
			}
		})
		b.Run("PerOperationConversion/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			m := make(map[string]int, size)
			for i, key := range keys {
				m[strings.ToLower(key)] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m[strings.ToLower(keys[0])]
			}
		})
		b.Run("PrecomputedCanonical/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			canonical := strings.ToLower(keys[0])
			m := make(map[string]int, size)
			for i, key := range keys {
				m[strings.ToLower(key)] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m[canonical]
			}
		})
		b.Run("SharedHasher/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			h := hashers.By(strings.ToLower, maphash.ComparableHasher[string]{})
			m := equiv.NewMap[string, int](h)
			for i, key := range keys {
				m.Set(key, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				sinkInt, _ = m.Get(keys[0])
			}
		})
	}
}

func BenchmarkCollisionGet(b *testing.B) {
	keys := ints(1024)
	m := equiv.NewMap[int, int](constantHasher{})
	for _, key := range keys {
		m.Set(key, key)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		sinkInt, _ = m.Get(keys[len(keys)-1])
	}
}

type constantHasher struct{}

func (constantHasher) Hash(h *maphash.Hash, _ int) { h.WriteByte(1) }
func (constantHasher) Equal(a, b int) bool         { return a == b }

func intEqual(a, b int) bool { return a == b }

func intHash(_ maphash.Seed, value int) uint64 { return uint64(value) }

func ints(size int) []int {
	keys := make([]int, size)
	for i := range keys {
		keys[i] = i
	}
	return keys
}

func stringsFor(size int) []string {
	keys := make([]string, size)
	for i := range keys {
		keys[i] = "Key-" + itoa(i)
	}
	return keys
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
