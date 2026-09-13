package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/bloom"
	"github.com/satorunooshie/equiv/sketch"
)

func main() {
	h := maphash.ComparableHasher[string]{}
	f, err := bloom.New[string](h, 10000, .01)
	if err != nil {
		panic(err)
	}
	f.Add("user-42")

	family := sketch.NewFamily[string](h)
	hll, err := family.NewHyperLogLog(12)
	if err != nil {
		panic(err)
	}
	hll.Add("user-42")
	fmt.Println(f.Contains("user-42"), hll.Estimate() > 0)
}
