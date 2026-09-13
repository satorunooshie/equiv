package equiv_test

import (
	"fmt"
	"hash/maphash"
	"strings"

	"github.com/satorunooshie/equiv"
	"github.com/satorunooshie/equiv/bloom"
	"github.com/satorunooshie/equiv/hashers"
)

func Example_sharedHasher() {
	identity := hashers.Slice(maphash.ComparableHasher[string]{})
	exact := equiv.NewMap[[]string, int](identity)
	seen := equiv.NewSet[[]string](identity)
	filter, err := bloom.New(identity, 100, 0.01)
	if err != nil {
		panic(err)
	}

	key := []string{"foo", "bar"}
	exact.Set(key, 42)
	seen.Insert(key)
	filter.Add(key)

	v, _ := exact.Get([]string{"foo", "bar"})
	fmt.Println(v, seen.Contains([]string{"foo", "bar"}), filter.Contains([]string{"foo", "bar"}))
	// Output: 42 true true
}

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
