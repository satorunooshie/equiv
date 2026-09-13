package table

import (
	"hash/maphash"
	"sort"
	"testing"
)

func BenchmarkTableProbeMetrics(b *testing.B) {
	t := New[int, int](maphash.ComparableHasher[int]{}, 8192)
	for i := 0; i < 5000; i++ {
		t.Set(i, i)
	}
	t.resetMetrics()
	sampleN := b.N
	if sampleN > 10000 {
		sampleN = 10000
	}
	samples := make([]uint64, sampleN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		before := t.probes.Load()
		t.Get(i % 5000)
		if i < sampleN {
			samples[i] = t.probes.Load() - before
		}
	}
	b.StopTimer()
	m := t.metrics()
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p95 := uint64(0)
	if len(samples) > 0 {
		p95 = samples[(len(samples)*95-1)/100]
	}
	b.ReportMetric(float64(m.probes)/float64(b.N), "probes/op")
	b.ReportMetric(float64(p95), "probe_p95/op")
	b.ReportMetric(float64(m.equalCalls)/float64(b.N), "equal/op")
	b.ReportMetric(float64(m.resizes), "resizes")
	b.ReportMetric(float64(m.rebuilds), "rebuilds")
	b.ReportMetric(float64(t.tombs)/float64(len(t.ctrl)), "tombstone_ratio")
}
