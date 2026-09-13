package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/cache"
)

func main() {
	c, err := cache.NewSharded[string, int](maphash.ComparableHasher[string]{}, cache.Config[string, int]{
		Policy: cache.WTinyLFU, MaxWeight: 100,
		Weigher: func(string, int) uint64 { return 1 }, Shards: 8,
	})
	if err != nil {
		panic(err)
	}
	c.Set("hot-key", 1)
	fmt.Println(c.Get("hot-key"))
}
