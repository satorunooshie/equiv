package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/hashers"
	"github.com/satorunooshie/equiv/interner"
)

func main() {
	i := interner.NewStrong[string](maphash.ComparableHasher[string]{})
	a := i.Intern(string([]byte("shared")))
	b := i.Intern(string([]byte("shared")))
	fmt.Println(a == b, i.Len())

	// A weak interner keeps canonical objects only while another part of the
	// program owns them, which is useful for deduplicating temporary objects.
	weak := interner.NewWeak[string](hashers.Deref(maphash.ComparableHasher[string]{}))
	first := "temporary"
	canonical := weak.Intern(&first)
	probe := "temporary"
	found, ok := weak.Lookup(&probe)
	fmt.Println(found == canonical, ok, weak.LenApprox())
}
