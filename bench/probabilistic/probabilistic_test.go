package probabilistic_test

import (
	"hash/maphash"
	"testing"

	"github.com/satorunooshie/equiv/bloom"
	"github.com/satorunooshie/equiv/sketch"
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
	b.ReportMetric(0.01, "configured-false-positive-target")
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

func BenchmarkBloomAdd(b *testing.B) {
	f, err := bloom.New[int](maphash.ComparableHasher[int]{}, 1<<20, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ReportMetric(float64(f.BitLen())/(1<<20), "bits/key")
	b.ReportMetric(0.01, "configured-false-positive-target")
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		f.Add(i)
	}
}

func BenchmarkHLLMerge(b *testing.B) {
	family := sketch.NewFamily[int](maphash.ComparableHasher[int]{})
	left, err := family.NewHyperLogLog(12)
	if err != nil {
		b.Fatal(err)
	}
	right, err := family.NewHyperLogLog(12)
	if err != nil {
		b.Fatal(err)
	}
	for i := range 32768 {
		left.Add(i)
		right.Add(i + 32768)
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := left.Merge(right); err != nil {
			b.Fatal(err)
		}
	}
}
