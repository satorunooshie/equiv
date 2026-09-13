package map_test

import (
	"hash/maphash"
	"strings"
	"testing"

	"github.com/aristanetworks/gomap"
	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func BenchmarkSemanticKeyShapes(b *testing.B) {
	b.Run("BytesContentIdentity/Gomap", func(b *testing.B) {
		keys := byteKeys(1024)
		h := hashers.Bytes()
		m := gomap.NewHint[[]byte, int](len(keys), bytesEqual, gomapHash(h))
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})
	b.Run("BytesContentIdentity/Equiv", func(b *testing.B) {
		keys := byteKeys(1024)
		m := equiv.NewMap[[]byte, int](hashers.Bytes())
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})

	b.Run("SliceContentIdentity/Gomap", func(b *testing.B) {
		keys := stringSliceKeys(1024)
		h := hashers.Slice(maphash.ComparableHasher[string]{})
		m := gomap.NewHint[[]string, int](len(keys), stringSliceEqual, gomapHash(h))
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})
	b.Run("SliceContentIdentity/Equiv", func(b *testing.B) {
		keys := stringSliceKeys(1024)
		m := equiv.NewMap[[]string, int](hashers.Slice(maphash.ComparableHasher[string]{}))
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})

	b.Run("ProjectedStruct/Gomap", func(b *testing.B) {
		keys := projectedKeys(1024)
		h := hashers.By(func(key projectedKey) string { return strings.ToLower(key.Name) }, maphash.ComparableHasher[string]{})
		m := gomap.NewHint[projectedKey, int](len(keys), h.Equal, gomapHash(h))
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})
	b.Run("ProjectedStruct/Equiv", func(b *testing.B) {
		keys := projectedKeys(1024)
		h := hashers.By(func(key projectedKey) string { return strings.ToLower(key.Name) }, maphash.ComparableHasher[string]{})
		m := equiv.NewMap[projectedKey, int](h)
		for i, key := range keys {
			m.Set(key, i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			semanticSink, _ = m.Get(keys[0])
		}
	})
}

var semanticSink int

type projectedKey struct {
	ID   int
	Name string
}

func bytesEqual(a, b []byte) bool { return string(a) == string(b) }

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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
