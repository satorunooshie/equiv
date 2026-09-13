package map_test

import (
	"hash/maphash"
	"math/rand"
	"strings"
	"testing"

	"github.com/aristanetworks/gomap"
	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

var sinkInt int

func BenchmarkIntGet(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 65536, 1 << 20} {
		b.Run("Builtin/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			accesses := accessSequence(4096, size, 0x4d415000+int64(size))
			m := make(map[int]int, size)
			for _, key := range keys {
				m[key] = key
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m[keys[accesses[i%len(accesses)]]]
			}
		})
		b.Run("Gomap/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			accesses := accessSequence(4096, size, 0x4d415000+int64(size))
			m := gomap.NewHint[int, int](size, intEqual, intHash)
			for _, key := range keys {
				m.Set(key, key)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m.Get(keys[accesses[i%len(accesses)]])
			}
		})
		b.Run("Equiv/"+itoa(size), func(b *testing.B) {
			keys := ints(size)
			accesses := accessSequence(4096, size, 0x4d415000+int64(size))
			m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
			for _, key := range keys {
				m.Set(key, key)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m.Get(keys[accesses[i%len(accesses)]])
			}
		})
	}
}

func BenchmarkSemanticGet(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 65536, 1 << 20} {
		b.Run("RawCanonical/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			accesses := accessSequence(4096, size, 0x53454d000+int64(size))
			m := make(map[string]int, size)
			for i, key := range keys {
				m[key] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m[keys[accesses[i%len(accesses)]]]
			}
		})
		b.Run("PerOperationConversion/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			accesses := accessSequence(4096, size, 0x53454d000+int64(size))
			m := make(map[string]int, size)
			for i, key := range keys {
				m[strings.ToLower(key)] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m[strings.ToLower(keys[accesses[i%len(accesses)]])]
			}
		})
		b.Run("PrecomputedCanonical/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			accesses := accessSequence(4096, size, 0x53454d000+int64(size))
			canonical := make([]string, size)
			for i, key := range keys {
				canonical[i] = strings.ToLower(key)
			}
			m := make(map[string]int, size)
			for i := range keys {
				m[canonical[i]] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m[canonical[accesses[i%len(accesses)]]]
			}
		})
		b.Run("SharedHasher/"+itoa(size), func(b *testing.B) {
			keys := stringsFor(size)
			accesses := accessSequence(4096, size, 0x53454d000+int64(size))
			h := hashers.By(strings.ToLower, maphash.ComparableHasher[string]{})
			m := equiv.NewMap[string, int](h)
			for i, key := range keys {
				m.Set(key, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkInt, _ = m.Get(keys[accesses[i%len(accesses)]])
			}
		})
	}
}

func BenchmarkCollisionGet(b *testing.B) {
	keys := ints(1024)
	accesses := accessSequence(4096, len(keys), 0x434f4c4c)
	m := equiv.NewMap[int, int](constantHasher{})
	for _, key := range keys {
		m.Set(key, key)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		sinkInt, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func BenchmarkIntOperations(b *testing.B) {
	for _, operation := range []string{"GetHit", "GetMiss", "SetNew", "Replace", "Delete", "Range", "Clear"} {
		b.Run("Builtin/"+operation, func(b *testing.B) { benchmarkBuiltinOperation(b, operation) })
		b.Run("Gomap/"+operation, func(b *testing.B) { benchmarkGomapOperation(b, operation) })
		b.Run("Equiv/"+operation, func(b *testing.B) { benchmarkEquivOperation(b, operation) })
	}
}

func benchmarkBuiltinOperation(b *testing.B, operation string) {
	keys := ints(1024)
	accesses := accessSequence(4096, len(keys), 0x4f505300)
	m := make(map[int]int, len(keys)*2)
	for _, key := range keys {
		m[key] = key
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := accesses[i%len(accesses)]
		switch operation {
		case "GetHit":
			sinkInt, _ = m[key]
		case "GetMiss":
			sinkInt, _ = m[key+len(keys)]
		case "SetNew":
			m[key+len(keys)] = i
		case "Replace":
			m[key] = i
		case "Delete":
			delete(m, key)
			m[key] = key
		case "Range":
			for _, value := range m {
				sinkInt += value
			}
		case "Clear":
			clear(m)
		}
	}
}

func benchmarkGomapOperation(b *testing.B, operation string) {
	keys := ints(1024)
	accesses := accessSequence(4096, len(keys), 0x4f505300)
	m := gomap.NewHint[int, int](len(keys)*2, intEqual, intHash)
	for _, key := range keys {
		m.Set(key, key)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := accesses[i%len(accesses)]
		switch operation {
		case "GetHit":
			sinkInt, _ = m.Get(key)
		case "GetMiss":
			sinkInt, _ = m.Get(key + len(keys))
		case "SetNew":
			m.Set(key+len(keys), i)
		case "Replace":
			m.Set(key, i)
		case "Delete":
			m.Delete(key)
			m.Set(key, key)
		case "Range":
			for _, value := range m.All() {
				sinkInt += value
			}
		case "Clear":
			m.Clear()
		}
	}
}

func benchmarkEquivOperation(b *testing.B, operation string) {
	keys := ints(1024)
	accesses := accessSequence(4096, len(keys), 0x4f505300)
	m := equiv.NewMap[int, int](maphash.ComparableHasher[int]{})
	for _, key := range keys {
		m.Set(key, key)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := accesses[i%len(accesses)]
		switch operation {
		case "GetHit":
			sinkInt, _ = m.Get(key)
		case "GetMiss":
			sinkInt, _ = m.Get(key + len(keys))
		case "SetNew":
			m.Set(key+len(keys), i)
		case "Replace":
			m.Set(key, i)
		case "Delete":
			m.Delete(key)
			m.Set(key, key)
		case "Range":
			for value := range m.Values() {
				sinkInt += value
			}
		case "Clear":
			m.Clear()
		}
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

func accessSequence(length, size int, seed int64) []int {
	accesses := make([]int, length)
	r := rand.New(rand.NewSource(seed))
	for i := range accesses {
		accesses[i] = r.Intn(size)
	}
	return accesses
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
