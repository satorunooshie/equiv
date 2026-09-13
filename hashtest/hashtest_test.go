package hashtest

import (
	"hash/maphash"
	"math/rand"
	"testing"
)

func TestCheckAndHelpers(t *testing.T) {
	h := maphash.ComparableHasher[int]{}
	Check(t, h, []int{1, 2, 3})
	CheckPairs(t, h, func(yield func(int, int) bool) {
		if !yield(1, 1) {
			return
		}
		yield(2, 2)
	})
	CheckFunc(t, h, func(r *rand.Rand) int {
		return int(r.Int63())
	}, 10)
}
