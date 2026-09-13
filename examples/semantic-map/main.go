package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func main() {
	// []string is not comparable, but its contents can still define identity.
	m := equiv.NewMap[[]string, int](hashers.Slice(maphash.ComparableHasher[string]{}))
	m.Set([]string{"tenant-a", "admin"}, 42)
	v, ok := m.Get([]string{"tenant-a", "admin"})
	fmt.Println(v, ok)
}
