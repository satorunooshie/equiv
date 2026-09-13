package concurrent_test

import (
	"hash/maphash"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/satorunooshie/equiv/concurrent"
)

// RunParallel is used here because the unit under test is concurrent
// scalability. Unlike the single-threaded B.Loop benchmarks, this benchmark
// intentionally uses the testing package's parallel-worker protocol.
func BenchmarkParallelMapWorkloads(b *testing.B) {
	for _, workload := range []struct {
		name       string
		writeEvery int
		keyModulo  int
	}{
		{name: "ReadOnly", keyModulo: 1024},
		{name: "Read95Write5", writeEvery: 20, keyModulo: 1024},
		{name: "Read50Write50", writeEvery: 2, keyModulo: 1024},
		{name: "HotKeyContention", writeEvery: 20, keyModulo: 1},
	} {
		b.Run("Equiv/"+workload.name, func(b *testing.B) {
			accesses := accessSequence(4096, workload.keyModulo, 0x434f4e43)
			m, err := concurrent.NewMap[int, int](maphash.ComparableHasher[int]{}, concurrent.WithShards(64))
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				m.Set(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := accesses[i%len(accesses)]
					if workload.writeEvery > 0 && i%workload.writeEvery == 0 {
						m.Set(key, i)
					} else {
						_, _ = m.Get(key)
					}
					i++
				}
			})
		})
		b.Run("XSync/"+workload.name, func(b *testing.B) {
			accesses := accessSequence(4096, workload.keyModulo, 0x434f4e43)
			m := xsync.NewMap[int, int]()
			for i := range 1024 {
				m.Store(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := accesses[i%len(accesses)]
					if workload.writeEvery > 0 && i%workload.writeEvery == 0 {
						m.Store(key, i)
					} else {
						_, _ = m.Load(key)
					}
					i++
				}
			})
		})
	}
}
