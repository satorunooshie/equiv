package bloom

import (
	"hash/maphash"
	"testing"
)

func BenchmarkBloomAdd(b *testing.B) {
	f, err := New[int](maphash.ComparableHasher[int]{}, 10000, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Add(i % 10000)
	}
}

func BenchmarkBloomContains(b *testing.B) {
	f, err := New[int](maphash.ComparableHasher[int]{}, 10000, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	for i := range 10000 {
		f.Add(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Contains(i % 10000)
	}
}

func BenchmarkCountingBloomAddRemove(b *testing.B) {
	f, err := NewCounting[int](maphash.ComparableHasher[int]{}, 10000, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Add(i % 10000)
		f.Remove(i % 10000)
	}
}
