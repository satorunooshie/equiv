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
	b.ReportMetric(float64(f.BitLen())/65536, "bits/key")
	falsePositives := 0
	for i := 65536; i < 131072; i++ {
		if f.Contains(i) {
			falsePositives++
		}
	}
	b.ReportMetric(float64(falsePositives)/65536, "false-positive-rate")
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		filterSink = f.Contains(i & 65535)
	}
}
