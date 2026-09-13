package bloom

import (
	"hash/maphash"
	"math"
)

// CountingFilter is a non-concurrent counting Bloom filter supporting
// multiplicity-sensitive removal with possible false positives.
type CountingFilter[T any] struct {
	h    maphash.Hasher[T]
	seed maphash.Seed
	c    []uint8
	m, k uint64
}

// NewCounting constructs a counting Bloom filter with the requested capacity
// and target false-positive probability.
func NewCounting[T any](h maphash.Hasher[T], expected uint64, p float64) (*CountingFilter[T], error) {
	f, e := New(h, expected, p)
	if e != nil {
		return nil, e
	}
	return &CountingFilter[T]{h: h, seed: f.seed, c: make([]uint8, f.m), m: f.m, k: f.k}, nil
}

func (f *CountingFilter[T]) hash(v T) (uint64, uint64) {
	var x maphash.Hash
	x.SetSeed(f.seed)
	f.h.Hash(&x, v)
	a := x.Sum64()
	var y maphash.Hash
	y.SetSeed(f.seed)
	y.WriteString("equiv:counting-bloom")
	f.h.Hash(&y, v)
	return a, y.Sum64() | 1
}

// Add increments the counters associated with v.
func (f *CountingFilter[T]) Add(v T) {
	a, b := f.hash(v)
	for j := uint64(0); j < f.k; j++ {
		p := (a + j*b) % f.m
		if f.c[p] < math.MaxUint8 {
			f.c[p]++
		}
	}
}

// Estimate returns the minimum counter among v's hash positions.
func (f *CountingFilter[T]) Estimate(v T) uint8 {
	a, b := f.hash(v)
	x := uint8(math.MaxUint8)
	for j := uint64(0); j < f.k; j++ {
		x = min(x, f.c[(a+j*b)%f.m])
	}
	return x
}

// Contains reports whether v may be present.
func (f *CountingFilter[T]) Contains(v T) bool { return f.Estimate(v) > 0 }

// Remove decrements v's counters and reports whether every counter supported it.
func (f *CountingFilter[T]) Remove(v T) bool {
	if !f.Contains(v) {
		return false
	}
	a, b := f.hash(v)
	for j := uint64(0); j < f.k; j++ {
		p := (a + j*b) % f.m
		if f.c[p] > 0 {
			f.c[p]--
		}
	}
	return true
}
