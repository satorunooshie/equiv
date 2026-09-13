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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Contains(i & 9999)
	}
}
