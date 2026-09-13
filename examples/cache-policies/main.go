// cache-policies makes the replacement-policy choice visible. Both caches
// have room for two entries; touching "hot" matters only for LRU.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/cache"
)

func newCache(policy cache.Policy) *cache.Cache[string, string] {
	c, err := cache.New[string, string](maphash.ComparableHasher[string]{}, cache.Config[string, string]{
		Policy: policy, MaxEntries: 2,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func main() {
	lru := newCache(cache.LRU)
	fifo := newCache(cache.FIFO)
	for _, c := range []*cache.Cache[string, string]{lru, fifo} {
		c.Set("hot", "cached")
		c.Set("cold", "cached")
		c.Get("hot")
		c.Set("new", "cached")
	}

	_, lruHasHot := lru.Get("hot")
	_, fifoHasHot := fifo.Get("hot")
	fmt.Println("LRU keeps hot:", lruHasHot)
	fmt.Println("FIFO keeps hot:", fifoHasHot)
	// Output:
	// LRU keeps hot: true
	// FIFO keeps hot: false
}
