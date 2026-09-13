// sketch-stream summarizes a large event stream without retaining every key.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/sketch"
)

func main() {
	h := maphash.ComparableHasher[string]{}
	frequency, err := sketch.NewCountMin[string](h, .01, .01)
	if err != nil {
		panic(err)
	}
	for _, path := range []string{"/", "/search", "/", "/", "/checkout", "/search"} {
		frequency.Add(path, 1)
	}

	family := sketch.NewFamily[string](h)
	regions, _ := family.NewHyperLogLog(10)
	for _, region := range []string{"apac", "us", "apac", "eu"} {
		regions.Add(region)
	}
	fmt.Println("/ estimated hits:", frequency.Estimate("/"))
	fmt.Println("distinct regions:", regions.Estimate())
	// Output is approximate: / estimated hits is at least 3 and
	// distinct regions is close to 3.
}
