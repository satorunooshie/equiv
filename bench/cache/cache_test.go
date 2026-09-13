package cache_test

import (
	"context"
	"hash/maphash"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/satorunooshie/equiv/cache"
)

var cacheSink int

func BenchmarkCacheWorkloads(b *testing.B) {
	for _, policy := range []cache.Policy{cache.FIFO, cache.LRU, cache.LFU, cache.TwoQ, cache.ARC, cache.WTinyLFU} {
		b.Run(policyName(policy), func(b *testing.B) {
			accesses := accessSequence(4096, 4096, 0x43414348)
			c, err := cache.New[int, int](maphash.ComparableHasher[int]{}, cache.Config[int, int]{Policy: policy, MaxEntries: 1024})
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				c.Set(i, i)
			}
			b.ReportMetric(float64(residentBytes()), "resident-bytes")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := accesses[i%len(accesses)]
				if i%20 == 0 {
					c.Set(key, i)
					continue
				}
				cacheSink, _ = c.Get(key)
			}
		})
	}
}

func BenchmarkCacheZipf(b *testing.B) {
	accesses := zipfSequence(4096, 4096, 0x5a495046)
	for _, policy := range []cache.Policy{cache.LRU, cache.LFU, cache.TwoQ, cache.ARC, cache.WTinyLFU} {
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
				cacheSink, _ = c.Get(accesses[i%len(accesses)])
			}
			b.ReportMetric(float64(c.Len()), "resident-entries")
		})
	}
}

func BenchmarkSpecializedCacheBaseline(b *testing.B) {
	for _, name := range []string{"EquivLRU", "HashicorpLRU"} {
		b.Run(name, func(b *testing.B) {
			accesses := accessSequence(4096, 2048, 0x43414348)
			if name == "EquivLRU" {
				c, err := cache.New[int, int](maphash.ComparableHasher[int]{}, cache.Config[int, int]{Policy: cache.LRU, MaxEntries: 1024})
				if err != nil {
					b.Fatal(err)
				}
				for i := range 1024 {
					c.Set(i, i)
				}
				b.ReportMetric(float64(residentBytes()), "resident-bytes")
				b.ReportAllocs()
				b.ResetTimer()
				hits, misses := 0, 0
				for i := 0; b.Loop(); i++ {
					key := accesses[i%len(accesses)]
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
			b.ReportMetric(float64(residentBytes()), "resident-bytes")
			b.ReportAllocs()
			b.ResetTimer()
			hits, misses := 0, 0
			for i := 0; b.Loop(); i++ {
				key := accesses[i%len(accesses)]
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

func BenchmarkLoaderDuplicateExecution(b *testing.B) {
	c, err := cache.NewSync[int, int](maphash.ComparableHasher[int]{}, cache.Config[int, int]{MaxEntries: 1024})
	if err != nil {
		b.Fatal(err)
	}
	var totalCalls atomic.Int64
	iterations := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		c.Clear()
		var calls atomic.Int64
		release := make(chan struct{})
		loader := func(context.Context, int) (int, error) {
			calls.Add(1)
			<-release
			return 42, nil
		}
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = c.GetOrLoad(context.Background(), 1, loader)
			}()
		}
		for calls.Load() == 0 {
			runtime.Gosched()
		}
		close(release)
		wg.Wait()
		totalCalls.Add(calls.Load())
		iterations++
	}
	if iterations > 0 {
		b.ReportMetric(float64(totalCalls.Load())/float64(iterations), "loader-calls/op")
	}
}

func policyName(policy cache.Policy) string {
	return []string{"FIFO", "LRU", "MRU", "LFU", "SLRU", "TwoQ", "ARC", "Clock", "WTinyLFU"}[policy]
}

func accessSequence(length, size int, seed int64) []int {
	accesses := make([]int, length)
	r := rand.New(rand.NewSource(seed))
	for i := range accesses {
		accesses[i] = r.Intn(size)
	}
	return accesses
}

func zipfSequence(length, size int, seed int64) []int {
	accesses := make([]int, length)
	r := rand.New(rand.NewSource(seed))
	z := rand.NewZipf(r, 1.2, 1, uint64(size-1))
	for i := range accesses {
		accesses[i] = int(z.Uint64())
	}
	return accesses
}

func residentBytes() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.Alloc
}
