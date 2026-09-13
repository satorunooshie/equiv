package probabilistic_test

import (
	"hash/maphash"
	"testing"

	"github.com/satorunooshie/equiv/cuckoo"
	"github.com/satorunooshie/equiv/xorfilter"
)

func BenchmarkCuckoo(b *testing.B) {
	f, err := cuckoo.New[int](maphash.ComparableHasher[int]{}, cuckoo.Config{Capacity: 131072})
	if err != nil {
		b.Fatal(err)
	}
	for i := range 65536 {
		if !f.Insert(i) {
			b.Fatalf("insert %d failed", i)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		filterSink = f.Contains(i & 65535)
	}
}

func BenchmarkXORFilter(b *testing.B) {
	values := make([]int, 65536)
	for i := range values {
		values[i] = i
	}
	f, err := xorfilter.New8[int](maphash.ComparableHasher[int]{}, func(yield func(int) bool) {
		for _, value := range values {
			if !yield(value) {
				return
			}
		}
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		filterSink = f.Contains(i & 65535)
	}
}
