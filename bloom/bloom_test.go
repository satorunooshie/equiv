package bloom

import (
	"hash/maphash"
	"math"
	"testing"
)

func TestCountingBloomMultiplicity(t *testing.T) {
	f, e := NewCounting[int](maphash.ComparableHasher[int]{}, 100, .01)
	if e != nil {
		t.Fatal(e)
	}
	f.Add(7)
	f.Add(7)
	if f.Estimate(7) != 2 {
		t.Fatal(f.Estimate(7))
	}
	if !f.Remove(7) || f.Estimate(7) != 1 {
		t.Fatal("first remove")
	}
	if !f.Remove(7) || f.Contains(7) {
		t.Fatal("second remove")
	}
}

func TestFilterConstructorValidation(t *testing.T) {
	tests := []struct {
		name string
		new  func() error
	}{
		{"zero expected", func() error { _, err := New[int](maphash.ComparableHasher[int]{}, 0, .1); return err }},
		{"nan probability", func() error { _, err := New[int](maphash.ComparableHasher[int]{}, 10, math.NaN()); return err }},
		{"unallocatable", func() error {
			_, err := New[int](maphash.ComparableHasher[int]{}, math.MaxUint64, math.SmallestNonzeroFloat64)
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.new() == nil {
				t.Fatal("expected constructor error")
			}
		})
	}
}

func TestBloomParameters(t *testing.T) {
	if _, e := New[int](maphash.ComparableHasher[int]{}, 0, .1); e == nil {
		t.Fatal("expected error")
	}
	f, _ := New[int](maphash.ComparableHasher[int]{}, 100, .01)
	f.Add(3)
	if !f.Contains(3) {
		t.Fatal("false negative")
	}
}

func TestBloomFalsePositiveRateIsCalibrated(t *testing.T) {
	f, e := New[int](maphash.ComparableHasher[int]{}, 1000, .01)
	if e != nil {
		t.Fatal(e)
	}
	for i := range 1000 {
		f.Add(i)
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

func TestBloomRejectsNaNProbability(t *testing.T) {
	if _, e := New[int](maphash.ComparableHasher[int]{}, 10, math.NaN()); e == nil {
		t.Fatal("expected NaN probability error")
	}
	if _, e := NewCounting[int](maphash.ComparableHasher[int]{}, 10, math.NaN()); e == nil {
		t.Fatal("expected NaN counting probability error")
	}
}

func TestBloomEstimatedRateHandlesLargeInput(t *testing.T) {
	f, e := New[int](maphash.ComparableHasher[int]{}, 10, .01)
	if e != nil {
		t.Fatal(e)
	}
	if got := f.EstimatedFalsePositiveRate(^uint64(0)); got < 0 || got > 1 {
		t.Fatalf("estimated rate=%v, want [0,1]", got)
	}
}

func TestBloomRejectsUnallocatableParameters(t *testing.T) {
	if _, e := New[int](maphash.ComparableHasher[int]{}, math.MaxUint64, math.SmallestNonzeroFloat64); e == nil {
		t.Fatal("expected oversized Bloom configuration error")
	}
}

func TestCountingBloomFalsePositiveRateIsCalibrated(t *testing.T) {
	f, err := NewCounting[int](maphash.ComparableHasher[int]{}, 1000, .01)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		f.Add(i)
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
