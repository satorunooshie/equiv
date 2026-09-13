// multimap-tags groups all labels assigned to each document.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/multimap"
)

func main() {
	tags := multimap.New[string, string](maphash.ComparableHasher[string]{})
	tags.Add("release-notes", "go")
	tags.Add("release-notes", "performance")
	tags.Add("release-notes", "examples")
	tags.DeleteFunc("release-notes", func(tag string) bool { return tag == "performance" })

	for tag := range tags.Values("release-notes") {
		fmt.Println(tag)
	}
	// Output:
	// go
	// examples
}
