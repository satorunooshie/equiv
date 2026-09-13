// cuckoo-filter demonstrates a bounded mutable membership index.
package main

import (
	"fmt"
	"hash/maphash"

	"github.com/satorunooshie/equiv/cuckoo"
)

func main() {
	f, err := cuckoo.New[string](maphash.ComparableHasher[string]{}, cuckoo.Config{Capacity: 100, FingerprintBits: 12})
	if err != nil {
		panic(err)
	}
	f.Insert("revoked-token-42")
	f.Delete("revoked-token-42")
	f.Insert("revoked-token-99")

	fmt.Println("token 42 may be revoked:", f.Contains("revoked-token-42"))
	fmt.Println("token 99 may be revoked:", f.Contains("revoked-token-99"))
	// Output:
	// token 42 may be revoked: false
	// token 99 may be revoked: true
}
