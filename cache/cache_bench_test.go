package cache

import (
	"hash/maphash"
	"testing"
	"time"
)

var benchmarkPolicies = []struct {
	name   string
	policy Policy
}{
	{"FIFO", FIFO},
	{"LRU", LRU},
	{"MRU", MRU},
	{"LFU", LFU},
	{"SLRU", SLRU},
	{"2Q", TwoQ},
	{"ARC", ARC},
	{"CLOCK", Clock},
	{"WTinyLFU", WTinyLFU},
}

func BenchmarkCacheGet(b *testing.B) {
	c, _ := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 1024})
	for i := range 1024 {
		c.Set(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i & 1023)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c, _ := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 1024})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(i&1023, i)
	}
}

func BenchmarkCachePoliciesUniform(b *testing.B) {
	for _, tc := range benchmarkPolicies {
		b.Run(tc.name, func(b *testing.B) {
			c, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
				Policy: tc.policy, MaxEntries: 1024,
			})
			if err != nil {
				b.Fatal(err)
			}
			for i := range 1024 {
				c.Set(i, i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := (i*17 + i/31) & 2047
				if i&3 == 0 {
					c.Set(key, i)
				} else {
					c.Get(key)
				}
			}
		})
	}
}

func BenchmarkCachePolicyWorkloads(b *testing.B) {
	workloads := []string{"zipfian", "looping", "scan-pollution", "hot-cold", "high-write", "read-mostly", "expiration-heavy"}
	for _, tc := range benchmarkPolicies {
		for _, workload := range workloads {
			b.Run(tc.name+"/"+workload, func(b *testing.B) {
				c, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
					Policy: tc.policy, MaxEntries: 256,
				})
				if err != nil {
					b.Fatal(err)
				}
				for i := range 256 {
					c.Set(i, i)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					workloadCacheOp(c, workload, i)
				}
			})
		}
	}
}

func BenchmarkCacheWeightedObjects(b *testing.B) {
	for _, policy := range []Policy{FIFO, LRU, MRU, LFU, SLRU, TwoQ, Clock, WTinyLFU} {
		b.Run(policyName(policy), func(b *testing.B) {
			c, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
				Policy: policy, MaxWeight: 256,
				Weigher: func(_ int, v int) uint64 { return uint64(v%8 + 1) },
			})
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				k := i & 511
				c.Set(k, (k&7)+1)
				c.Get((i * 17) & 511)
			}
		})
	}
}

func workloadCacheOp(c *Cache[int, int], workload string, i int) {
	switch workload {
	case "zipfian":
		k := (i*i + 13*i) & 255
		c.Get(k)
	case "looping":
		c.Get(i & 63)
	case "scan-pollution":
		if i&7 == 0 {
			c.Get(1024 + i&1023)
		} else {
			c.Get(i & 255)
		}
	case "hot-cold":
		if i&3 != 0 {
			c.Get(i & 15)
		} else {
			c.Get(128 + i&127)
		}
	case "high-write":
		c.Set(i&255, i)
	case "read-mostly":
		if i&15 == 0 {
			c.Set(i&255, i)
		} else {
			c.Get(i & 255)
		}
	case "expiration-heavy":
		c.SetTTL(i&255, i, time.Nanosecond)
		c.Get(i & 255)
	}
}

func policyName(policy Policy) string {
	for _, tc := range benchmarkPolicies {
		if tc.policy == policy {
			return tc.name
		}
	}
	return "unknown"
}
