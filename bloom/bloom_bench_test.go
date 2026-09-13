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
	i := 0
	for b.Loop() {
		f.Add(i % 10000)
		i++
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
	i := 0
	for b.Loop() {
		f.Contains(i % 10000)
		i++
	}
}

func BenchmarkCountingBloomAddRemove(b *testing.B) {
	f, err := NewCounting[int](maphash.ComparableHasher[int]{}, 10000, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		f.Add(i % 10000)
		f.Remove(i % 10000)
		i++
	}
}
