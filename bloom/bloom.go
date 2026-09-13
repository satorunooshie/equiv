// Package bloom provides non-concurrent Bloom and counting Bloom filters.
// Bloom membership is probabilistic: inserted values have no false negatives,
// while absent values may produce false positives.
package bloom

import (
	"errors"
	"hash/maphash"
	"math"
)

// Filter is a non-concurrent probabilistic set with no false negatives for
// inserted values and a configurable false-positive probability.
type Filter[T any] struct {
	h    maphash.Hasher[T]
	seed maphash.Seed
	bits []uint64
	m, k uint64
	n    uint64
}

// New constructs a Bloom filter sized for expected insertions and false-positive
// probability p. Inserted values are never reported absent.
func New[T any](h maphash.Hasher[T], expected uint64, p float64) (*Filter[T], error) {
	if h == nil {
		panic("bloom: nil Hasher")
	}
	if expected == 0 || math.IsNaN(p) || p <= 0 || p >= 1 {
		return nil, errors.New("bloom: invalid parameters")
	}
	mFloat := math.Ceil(-float64(expected) * math.Log(p) / (math.Ln2 * math.Ln2))
	maxInt := uint64(^uint(0) >> 1)
	if math.IsInf(mFloat, 0) || mFloat > float64(maxInt-63) {
		return nil, errors.New("bloom: parameters require too much memory")
	}
	m := uint64(mFloat)
	if m < 1 {
		m = 1
	}
	k := uint64(math.Round(float64(m) / float64(expected) * math.Ln2))
	if k < 1 {
		k = 1
	}
	return &Filter[T]{h: h, seed: maphash.MakeSeed(), bits: make([]uint64, (m+63)/64), m: m, k: k}, nil
}

func (f *Filter[T]) hash(v T) (uint64, uint64) {
	var x maphash.Hash
	x.SetSeed(f.seed)
	f.h.Hash(&x, v)
	a := x.Sum64()
	var y maphash.Hash
	y.SetSeed(f.seed)
	y.WriteString("equiv:bloom")
	f.h.Hash(&y, v)
	return a, y.Sum64() | 1
}

// Add records v in the filter.
func (f *Filter[T]) Add(v T) {
	a, b := f.hash(v)
	for j := uint64(0); j < f.k; j++ {
		p := (a + j*b) % f.m
		f.bits[p/64] |= 1 << (p % 64)
	}
	f.n++
}

// Contains reports whether v may be present; false positives are possible.
func (f *Filter[T]) Contains(v T) bool {
	a, b := f.hash(v)
	for j := uint64(0); j < f.k; j++ {
		p := (a + j*b) % f.m
		if f.bits[p/64]&(1<<(p%64)) == 0 {
			return false
		}
	}
	return true
}

// BitLen returns the number of bits allocated by the filter.
func (f *Filter[T]) BitLen() uint64 { return f.m }

// EstimatedFalsePositiveRate estimates the rate after inserted insertions.
func (f *Filter[T]) EstimatedFalsePositiveRate(inserted uint64) float64 {
	return math.Pow(1-math.Exp(-float64(f.k)*float64(inserted)/float64(f.m)), float64(f.k))
}
