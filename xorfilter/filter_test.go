package xorfilter

import (
	"hash/maphash"
	"iter"
	"testing"

	"github.com/satorunooshie/equiv/hashers"
)

func sliceSeq[T any](values []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, value := range values {
			if !yield(value) {
				return
			}
		}
	}
}

func TestNewDeduplicatesSemanticValues(t *testing.T) {
	h := hashers.By[string, string](func(s string) string {
		if len(s) == 0 {
			return s
		}
		return string([]byte{s[0]})
	}, maphash.ComparableHasher[string]{})
	f, e := New8(h, sliceSeq([]string{"a", "apple", "b", "boat"}))
	if e != nil {
		t.Fatal(e)
	}
	if f.Len() != 2 {
		t.Fatalf("Len=%d", f.Len())
	}
}

func TestPeelingHasNoFalseNegatives(t *testing.T) {
	values := make([]int, 1000)
	for i := range values {
		values[i] = i
	}
	f, e := New16(maphash.ComparableHasher[int]{}, sliceSeq(values))
	if e != nil {
		t.Fatal(e)
	}
	if f.Len() != len(values) {
		t.Fatalf("Len=%d", f.Len())
	}
	for _, v := range values {
		if !f.Contains(v) {
			t.Fatalf("false negative %d", v)
		}
	}
}

func TestXor16FalsePositiveRateIsLow(t *testing.T) {
	values := make([]int, 1000)
	for i := range values {
		values[i] = i
	}
	f, err := New16(maphash.ComparableHasher[int]{}, sliceSeq(values))
	if err != nil {
		t.Fatal(err)
	}
	falsePositives := 0
	for i := 1000; i < 11000; i++ {
		if f.Contains(i) {
			falsePositives++
		}
	}
	if rate := float64(falsePositives) / 10000; rate > 0.01 {
		t.Fatalf("false-positive rate=%0.4f, want <= 0.01", rate)
	}
}

func TestXor8FalsePositiveRateIsBounded(t *testing.T) {
	values := make([]int, 1000)
	for i := range values {
		values[i] = i
	}
	f, err := New8(maphash.ComparableHasher[int]{}, sliceSeq(values))
	if err != nil {
		t.Fatal(err)
	}
	falsePositives := 0
	for i := 1000; i < 11000; i++ {
		if f.Contains(i) {
			falsePositives++
		}
	}
	if rate := float64(falsePositives) / 10000; rate > .03 {
		t.Fatalf("false-positive rate=%0.4f, want <= 0.03", rate)
	}
}
