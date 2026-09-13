// ordered-workflow keeps a user-facing menu stable while still allowing
// semantic keys such as case-insensitive command names.
package main

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/satorunooshie/equiv/hashers"
	"github.com/satorunooshie/equiv/ordered"
)

func main() {
	h := hashers.By(strings.ToLower, maphash.ComparableHasher[string]{})
	menu := ordered.NewMap[string, string](h)
	menu.Set("Home", "/")
	menu.Set("Settings", "/settings")
	menu.Set("Help", "/help")
	menu.Set("settings", "/account/settings") // replaces, keeps its position
	menu.MoveToFront("help")

	for name, path := range menu.All() {
		fmt.Printf("%s=%s\n", name, path)
	}
	// Output:
	// Help=/help
	// Home=/
	// Settings=/account/settings
}
