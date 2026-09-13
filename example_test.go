package equiv_test

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/hashers"
)

func ExampleMap() {
	m := equiv.NewMap[[]string, int](hashers.Slice(maphash.ComparableHasher[string]{}))
	m.Set([]string{"foo", "bar"}, 42)

	v, ok := m.Get([]string{"foo", "bar"})
	fmt.Println(v, ok)
	// Output: 42 true
}

func ExampleMap_customEquality() {
	h := hashers.By(
		strings.ToLower,
		maphash.ComparableHasher[string]{},
	)
	m := equiv.NewMap[string, int](h)
	m.Set("equiv", 1)

	v, ok := m.Get("equiv")
	fmt.Println(v, ok)
	// Output: 1 true
}
