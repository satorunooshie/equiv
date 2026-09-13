package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/interner"
)

func main() {
	i := interner.NewStrong[string](maphash.ComparableHasher[string]{})
	a := i.Intern(string([]byte("shared")))
	b := i.Intern(string([]byte("shared")))
	fmt.Println(a == b, i.Len())
}
