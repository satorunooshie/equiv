package cuckoo

import (
	"hash/maphash"
	"math"
	"testing"
)

func TestMultiplicity(t *testing.T) {
	f, e := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 4})
	if e != nil {
		t.Fatal(e)
	}
	if !f.Insert(1) || !f.Insert(1) {
		t.Fatal("insert")
	}
	if f.Len() != 2 {
		t.Fatal(f.Len())
	}
	f.Delete(1)
	if !f.Contains(1) || f.Len() != 1 {
		t.Fatal("multiplicity")
	}
	if f.Insert(2) == false || f.Insert(3) == false || f.Insert(4) == false || f.Insert(5) {
		t.Fatal("capacity")
	}
}

func TestConstructorValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"zero capacity", Config{}},
		{"overflow capacity", Config{Capacity: math.MaxUint64}},
		{"invalid fingerprint", Config{Capacity: 8, FingerprintBits: 7}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New[int](maphash.ComparableHasher[int]{}, tc.cfg); err == nil {
				t.Fatal("expected constructor error")
			}
		})
	}
}

func TestFingerprintFilterHasNoFalseNegative(t *testing.T) {
	f, e := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 100, FingerprintBits: 8})
	if e != nil {
		t.Fatal(e)
	}
	for i := range 50 {
		if !f.Insert(i) {
			t.Fatalf("insert %d", i)
		}
	}
	for i := range 50 {
		if !f.Contains(i) {
			t.Fatalf("false negative %d", i)
		}
	}
}

func TestFailedInsertPreservesMembership(t *testing.T) {
	f, e := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 4, BucketSize: 4, MaxKicks: 500})
	if e != nil {
		t.Fatal(e)
	}
	for i := range 4 {
		if !f.Insert(i) {
			t.Fatalf("insert %d", i)
		}
	}
	before := make([]bool, 4)
	for i := range before {
		before[i] = f.Contains(i)
	}
	if f.Insert(100) {
		t.Fatal("expected bounded insertion failure")
	}
	for i, v := range before {
		if f.Contains(i) != v {
			t.Fatalf("membership changed for %d", i)
		}
	}
}

func TestCuckooRejectsOverflowingCapacity(t *testing.T) {
	if _, err := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: math.MaxUint64}); err == nil {
		t.Fatal("expected oversized Cuckoo configuration error")
	}
}

func TestCuckooLengthLoadFactorAndDeleteMiss(t *testing.T) {
	f, err := New[string](maphash.ComparableHasher[string]{}, Config{Capacity: 8, BucketSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if f.Len() != 0 || f.LoadFactor() != 0 {
		t.Fatalf("empty filter len/load=(%d,%v)", f.Len(), f.LoadFactor())
	}
	if !f.Insert("a") || f.Len() != 1 || f.LoadFactor() <= 0 {
		t.Fatalf("insert len/load=(%d,%v)", f.Len(), f.LoadFactor())
	}
	if f.Delete("missing") {
		t.Fatal("delete reported absent value")
	}
}

func TestCuckooExercisesRelocationPath(t *testing.T) {
	f, err := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 100, BucketSize: 2, MaxKicks: 500})
	if err != nil {
		t.Fatal(err)
	}
	inserted := 0
	for i := 0; i < 100 && f.Insert(i); i++ {
		inserted++
	}
	if inserted < 50 {
		t.Fatalf("filter became unusable too early: inserted=%d", inserted)
	}
	for i := 0; i < inserted; i++ {
		if !f.Contains(i) {
			t.Fatalf("false negative after relocation at %d", i)
		}
	}
}

func TestCuckooFalsePositiveRateIsBounded(t *testing.T) {
	f, err := New[int](maphash.ComparableHasher[int]{}, Config{Capacity: 2000, FingerprintBits: 8})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		if !f.Insert(i) {
			t.Fatalf("insert %d", i)
		}
	}
	falsePositives := 0
	for i := 1000; i < 11000; i++ {
		if f.Contains(i) {
			falsePositives++
		}
	}
	if rate := float64(falsePositives) / 10000; rate > 0.10 {
		t.Fatalf("false-positive rate=%0.4f, want <= 0.10", rate)
	}
}
