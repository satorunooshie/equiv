// multiset-frequency counts repeated events while retaining the distinct keys.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/multiset"
)

func main() {
	words := multiset.New[string](maphash.ComparableHasher[string]{})
	for _, word := range []string{"go", "cache", "go", "map", "go", "cache"} {
		words.Add(word)
	}
	words.Remove("cache")

	for word, count := range words.All() {
		fmt.Printf("%s=%d\n", word, count)
	}
	fmt.Println("total events:", words.Total())
	// Output order follows the hash-table order and may vary.
	// The meaningful result is go=3, map=1, cache=1, total events: 5.
}
