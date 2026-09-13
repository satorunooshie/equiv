// concurrent-map shows atomic updates from many goroutines and a snapshot for
// reporting without keeping the concurrent collection locked.
package main

import (
	"fmt"
	"hash/maphash"
	"sync"

	"github.com/satorunooshie/equiv/concurrent"
)

func main() {
	m, err := concurrent.NewMap[string, int](maphash.ComparableHasher[string]{}, concurrent.WithShards(8))
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for worker := range 4 {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := range 100 {
				m.GetOrSet(fmt.Sprintf("user:%d", i), worker)
			}
		}(worker)
	}
	wg.Wait()

	snapshot := m.Snapshot()
	value, _ := snapshot.Get("user:42")
	fmt.Println("entries:", m.Len(), "user:42:", value)
	// Output: entries: 100 user:42: <one of the workers' values>
}
