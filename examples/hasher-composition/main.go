// hasher-composition defines identity once and reuses it in an exact map and
// a probabilistic index.
package main

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/bloom"
	"github.com/satorunooshie/equiv/hashers"
)

type Request struct {
	Tenant string
	Path   string
	Trace  string // metadata, not part of identity
}

func main() {
	h := hashers.Struct[Request]().
		Field(func(r Request) string { return r.Tenant }, maphash.ComparableHasher[string]{}).
		Field(func(r Request) string { return strings.ToLower(r.Path) }, maphash.ComparableHasher[string]{}).
		Build()

	routes := equiv.NewSet[Request](h)
	routes.Insert(Request{"acme", "/Search", "first"})
	routes.Insert(Request{"acme", "/search", "duplicate"})
	filter, _ := bloom.New[Request](h, 100, .01)
	filter.Add(Request{"acme", "/search", "probe"})

	canonical, _ := routes.Lookup(Request{"acme", "/SEARCH", "lookup"})
	fmt.Println("distinct routes:", routes.Len(), "canonical trace:", canonical.Trace, "maybe indexed:", filter.Contains(Request{"acme", "/search", "other"}))
	// Output: distinct routes: 1 canonical trace: first maybe indexed: true
}
