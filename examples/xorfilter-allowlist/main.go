// xorfilter-allowlist builds an immutable, compact allow-list for a hot read
// path. Changes require building a new filter.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/xorfilter"
)

func main() {
	allowed := []string{"plan:free", "plan:pro", "plan:team"}
	f, err := xorfilter.New8[string](maphash.ComparableHasher[string]{}, func(yield func(string) bool) {
		for _, plan := range allowed {
			if !yield(plan) {
				return
			}
		}
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("plan:pro may be allowed:", f.Contains("plan:pro"))
	fmt.Println("plan:enterprise may be allowed:", f.Contains("plan:enterprise"))
	// Output:
	// plan:pro may be allowed: true
	// plan:enterprise may be allowed: false
}
