package main

import (
	"context"
	"fmt"
	"hash/maphash"
	"time"

	"github.com/satorunooshie/equiv/cache"
)

func main() {
	c, err := cache.NewSync[string, string](maphash.ComparableHasher[string]{}, cache.Config[string, string]{
		Policy: cache.LRU, MaxEntries: 256, DefaultTTL: time.Minute,
	})
	if err != nil {
		panic(err)
	}
	v, err := c.GetOrLoad(context.Background(), "profile:42", func(context.Context, string) (string, error) {
		return "loaded from the service", nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(v, c.Stats().LoadSuccesses)
}
