package xorfilter

import (
	"hash/maphash"
	"testing"
)

func BenchmarkXorContains(b *testing.B) {
	v := make([]int, 10000)
	for i := range v {
		v[i] = i
	}
	f, _ := New16(maphash.ComparableHasher[int]{}, sliceSeq(v))
	i := 0
	for b.Loop() {
		f.Contains(i & 9999)
		i++
	}
}
