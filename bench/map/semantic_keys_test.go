package map_test

import (
	"bytes"
	"hash/maphash"
	"slices"
	"strings"
	"testing"

	"github.com/aristanetworks/gomap"
	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func BenchmarkSemanticKeyShapes(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 65536} {
		b.Run("BytesContentIdentity/"+itoa(size), func(b *testing.B) {
			benchmarkBytesGomap(b, size)
		})
		b.Run("BytesContentIdentity/Equiv/"+itoa(size), func(b *testing.B) {
			benchmarkBytesEquiv(b, size)
		})
		b.Run("SliceContentIdentity/"+itoa(size), func(b *testing.B) {
			benchmarkSlicesGomap(b, size)
		})
		b.Run("SliceContentIdentity/Equiv/"+itoa(size), func(b *testing.B) {
			benchmarkSlicesEquiv(b, size)
		})
		b.Run("ProjectedStruct/"+itoa(size), func(b *testing.B) {
			benchmarkProjectedGomap(b, size)
		})
		b.Run("ProjectedStruct/Equiv/"+itoa(size), func(b *testing.B) {
			benchmarkProjectedEquiv(b, size)
		})
	}
}

func benchmarkBytesGomap(b *testing.B, size int) {
	keys := byteKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	h := hashers.Bytes()
	m := gomap.NewHint[[]byte, int](size, bytesEqual, gomapHash(h))
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func benchmarkBytesEquiv(b *testing.B, size int) {
	keys := byteKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	m := equiv.NewMap[[]byte, int](hashers.Bytes())
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func benchmarkSlicesGomap(b *testing.B, size int) {
	keys := stringSliceKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	h := hashers.Slice(maphash.ComparableHasher[string]{})
	m := gomap.NewHint[[]string, int](size, stringSliceEqual, gomapHash(h))
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func benchmarkSlicesEquiv(b *testing.B, size int) {
	keys := stringSliceKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	m := equiv.NewMap[[]string, int](hashers.Slice(maphash.ComparableHasher[string]{}))
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func benchmarkProjectedGomap(b *testing.B, size int) {
	keys := projectedKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	h := hashers.By(func(key projectedKey) string { return strings.ToLower(key.Name) }, maphash.ComparableHasher[string]{})
	m := gomap.NewHint[projectedKey, int](size, h.Equal, gomapHash(h))
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

func benchmarkProjectedEquiv(b *testing.B, size int) {
	keys := projectedKeys(size)
	accesses := accessSequence(4096, size, 0x53484150)
	h := hashers.By(func(key projectedKey) string { return strings.ToLower(key.Name) }, maphash.ComparableHasher[string]{})
	m := equiv.NewMap[projectedKey, int](h)
	for i, key := range keys {
		m.Set(key, i)
	}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		semanticSink, _ = m.Get(keys[accesses[i%len(accesses)]])
	}
}

var semanticSink int

type projectedKey struct {
	ID   int
	Name string
}

func bytesEqual(a, b []byte) bool         { return bytes.Equal(a, b) }
func stringSliceEqual(a, b []string) bool { return slices.Equal(a, b) }

func gomapHash[T any](h maphash.Hasher[T]) func(maphash.Seed, T) uint64 {
	return func(seed maphash.Seed, value T) uint64 {
		var hash maphash.Hash
		hash.SetSeed(seed)
		h.Hash(&hash, value)
		return hash.Sum64()
	}
}

func byteKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte("key-" + itoa(i))
	}
	return keys
}

func stringSliceKeys(n int) [][]string {
	keys := make([][]string, n)
	for i := range keys {
		keys[i] = []string{"group", itoa(i), "value"}
	}
	return keys
}

func projectedKeys(n int) []projectedKey {
	keys := make([]projectedKey, n)
	for i := range keys {
		keys[i] = projectedKey{ID: i, Name: "Key-" + itoa(i)}
	}
	return keys
}
