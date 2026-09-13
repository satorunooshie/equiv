package cache_test

import (
	"hash/maphash"
	"testing"

	"github.com/satorunooshie/equiv/cache"
)

var cacheSink int

func BenchmarkCacheWorkloads(b *testing.B) {
	for _, policy := range []cache.Policy{cache.FIFO, cache.LRU, cache.LFU, cache.TwoQ, cache.ARC, cache.WTinyLFU} {
		b.Run(policyName(policy), func(b *testing.B) {
			c, err := cache.New[int, int](maphash.ComparableHasher[int]{}, cache.Config[int, int]{Policy: policy, MaxEntries: 1024})
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				c.Set(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := (i * 17) % 4096
				if i%20 == 0 {
					c.Set(key, i)
					continue
				}
				cacheSink, _ = c.Get(key)
			}
		})
	}
}

func policyName(policy cache.Policy) string {
	return []string{"FIFO", "LRU", "MRU", "LFU", "SLRU", "TwoQ", "ARC", "Clock", "WTinyLFU"}[policy]
}
