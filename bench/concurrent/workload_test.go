package concurrent_test

import "math/rand"

func accessSequence(length, size int, seed int64) []int {
	accesses := make([]int, length)
	r := rand.New(rand.NewSource(seed))
	for i := range accesses {
		accesses[i] = r.Intn(size)
	}
	return accesses
}
