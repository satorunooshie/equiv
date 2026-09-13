package sketch

import (
	"hash/maphash"
	"math"
	"testing"
)

func TestSketches(t *testing.T) {
	s, e := NewCountMin[int](maphash.ComparableHasher[int]{}, .01, .01)
	if e != nil {
		t.Fatal(e)
	}
	s.Add(4, 3)
	if s.Estimate(4) < 3 {
		t.Fatal("count-min false undercount")
	}
	f := NewFamily[int](maphash.ComparableHasher[int]{})
	a, _ := f.NewHyperLogLog(8)
	b, _ := f.NewHyperLogLog(8)
	for i := 0; i < 100; i++ {
		a.Add(i)
		b.Add(i)
	}
	if e := a.Merge(b); e != nil {
		t.Fatal(e)
	}
	if a.Estimate() == 0 {
		t.Fatal("HLL estimate")
	}
}

func TestCountMinParameterValidation(t *testing.T) {
	tests := []struct {
		name           string
		epsilon, delta float64
	}{
		{"nan epsilon", math.NaN(), .1},
		{"nan delta", .1, math.NaN()},
		{"unallocatable width", math.SmallestNonzeroFloat64, .1},
		{"unrepresentable depth", .1, math.SmallestNonzeroFloat64},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewCountMin[int](maphash.ComparableHasher[int]{}, tc.epsilon, tc.delta); err == nil {
				t.Fatal("expected constructor error")
			}
		})
	}
}

func TestHLLFamiliesAreNotInterchangeable(t *testing.T) {
	a := NewFamily[int](maphash.ComparableHasher[int]{})
	b := NewFamily[int](maphash.ComparableHasher[int]{})
	x, _ := a.NewHyperLogLog(8)
	y, _ := b.NewHyperLogLog(8)
	if x.Merge(y) == nil {
		t.Fatal("different families merged")
	}
	z, _ := a.NewHyperLogLog(9)
	if x.Merge(z) == nil {
		t.Fatal("different precisions merged")
	}
}

func TestHLLEstimateIsWithinExpectedRange(t *testing.T) {
	f := NewFamily[int](maphash.ComparableHasher[int]{})
	h, e := f.NewHyperLogLog(12)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 10000; i++ {
		h.Add(i)
	}
	got := h.Estimate()
	if got < 8000 || got > 12500 {
		t.Fatalf("estimate=%d, want approximately 10000", got)
	}
}

func TestCountMinRejectsNaNParameters(t *testing.T) {
	h := maphash.ComparableHasher[int]{}
	if _, e := NewCountMin[int](h, math.NaN(), .1); e == nil {
		t.Fatal("expected NaN epsilon error")
	}
	if _, e := NewCountMin[int](h, .1, math.NaN()); e == nil {
		t.Fatal("expected NaN delta error")
	}
}

func TestCountMinTotalSaturatesIndependently(t *testing.T) {
	s, e := NewCountMin[int](maphash.ComparableHasher[int]{}, .1, .1)
	if e != nil {
		t.Fatal(e)
	}
	s.Add(1, ^uint64(0))
	s.Add(1, 1)
	if s.Total() != ^uint64(0) {
		t.Fatalf("total=%d, want MaxUint64", s.Total())
	}
}

func TestCountMinRejectsUnallocatableParameters(t *testing.T) {
	if _, e := NewCountMin[int](maphash.ComparableHasher[int]{}, math.SmallestNonzeroFloat64, .1); e == nil {
		t.Fatal("expected oversized Count-Min configuration error")
	}
}

func TestCountMinRejectsUnrepresentableDepth(t *testing.T) {
	if _, e := NewCountMin[int](maphash.ComparableHasher[int]{}, .1, math.SmallestNonzeroFloat64); e == nil {
		t.Fatal("expected unrepresentable depth error")
	}
}

func TestCountMinErrorIsBoundedOnDistinctKeys(t *testing.T) {
	s, err := NewCountMin[int](maphash.ComparableHasher[int]{}, .01, .01)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		s.Add(i, 1)
	}
	for i := 0; i < 1000; i++ {
		if got := s.Estimate(i); got < 1 || got > 100 {
			t.Fatalf("estimate(%d)=%d, want [1,100]", i, got)
		}
	}
}
