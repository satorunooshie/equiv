package cache_test

import (
	"hash/maphash"
	"testing"

	"github.com/hashicorp/golang-lru/v2"
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

func BenchmarkSpecializedCacheBaseline(b *testing.B) {
	for _, name := range []string{"EquivLRU", "HashicorpLRU"} {
		b.Run(name, func(b *testing.B) {
			if name == "EquivLRU" {
				c, err := cache.New[int, int](maphash.ComparableHasher[int]{}, cache.Config[int, int]{Policy: cache.LRU, MaxEntries: 1024})
				if err != nil {
					b.Fatal(err)
				}
				for i := range 1024 {
					c.Set(i, i)
				}
				b.ReportAllocs()
				b.ResetTimer()
				hits, misses := 0, 0
				for i := 0; b.Loop(); i++ {
					key := (i*17 + i/31) & 2047
					if i%20 == 0 {
						c.Set(key, i)
					} else if value, ok := c.Get(key); ok {
						hits++
						cacheSink = value
					} else {
						misses++
					}
				}
				if hits+misses > 0 {
					b.ReportMetric(float64(hits)/float64(hits+misses), "hit-rate")
				}
				b.ReportMetric(float64(c.Len()), "resident-entries")
				return
			}

			c, err := lru.New[int, int](1024)
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				c.Add(i, i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			hits, misses := 0, 0
			for i := 0; b.Loop(); i++ {
				key := (i*17 + i/31) & 2047
				if i%20 == 0 {
					c.Add(key, i)
				} else if value, ok := c.Get(key); ok {
					hits++
					cacheSink = value
				} else {
					misses++
				}
			}
			if hits+misses > 0 {
				b.ReportMetric(float64(hits)/float64(hits+misses), "hit-rate")
			}
			b.ReportMetric(float64(c.Len()), "resident-entries")
		})
	}
}

func policyName(policy cache.Policy) string {
	return []string{"FIFO", "LRU", "MRU", "LFU", "SLRU", "TwoQ", "ARC", "Clock", "WTinyLFU"}[policy]
}
