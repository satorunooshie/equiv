package probabilistic_test

import (
	"hash/maphash"
	"testing"

	"github.com/satorunooshie/equiv/bloom"
)

var filterSink bool

func BenchmarkBloom(b *testing.B) {
	h := maphash.ComparableHasher[int]{}
	f, err := bloom.New(h, 65536, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	for i := range 65536 {
		f.Add(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		filterSink = f.Contains(i & 65535)
	}
}
