// counting-bloom uses a counting filter for a compact, removable membership
// hint. The database remains the source of truth after a positive result.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/bloom"
)

func main() {
	f, err := bloom.NewCounting[string](maphash.ComparableHasher[string]{}, 1000, .01)
	if err != nil {
		panic(err)
	}
	f.Add("session:42")
	f.Add("session:42") // two independent references
	f.Remove("session:42")

	fmt.Println("maybe present:", f.Contains("session:42"), "estimated copies:", f.Estimate("session:42"))
	// Output: maybe present: true estimated copies: 1
}
