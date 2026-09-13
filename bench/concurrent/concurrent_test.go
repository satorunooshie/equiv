package concurrent_test

import (
	"hash/maphash"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/satorunooshie/equiv/concurrent"
)

var concurrentSink int

func BenchmarkMapWorkloads(b *testing.B) {
	for _, workload := range []struct {
		name       string
		writeEvery int
		keyModulo  int
	}{
		{name: "ReadOnly", writeEvery: 0, keyModulo: 1024},
		{name: "Read95Write5", writeEvery: 20, keyModulo: 1024},
		{name: "Read50Write50", writeEvery: 2, keyModulo: 1024},
		{name: "HotKeyContention", writeEvery: 20, keyModulo: 1},
	} {
		b.Run("Equiv/"+workload.name, func(b *testing.B) {
			m, err := concurrent.NewMap[int, int](maphash.ComparableHasher[int]{}, concurrent.WithShards(64))
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				m.Set(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := i % workload.keyModulo
				if workload.writeEvery > 0 && i%workload.writeEvery == 0 {
					m.Set(key, i)
					continue
				}
				concurrentSink, _ = m.Get(key)
			}
		})
		b.Run("XSync/"+workload.name, func(b *testing.B) {
			m := xsync.NewMap[int, int]()
			for i := range 1024 {
				m.Store(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := i % workload.keyModulo
				if workload.writeEvery > 0 && i%workload.writeEvery == 0 {
					m.Store(key, i)
					continue
				}
				concurrentSink, _ = m.Load(key)
			}
		})
	}
}
