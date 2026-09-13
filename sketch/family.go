package sketch

import (
	"errors"
	"hash/maphash"
)

// Family owns the immutable hash domain shared by compatible HLL instances.
// It is non-concurrent and must not be copied after first use.
type Family[T any] struct {
	h    maphash.Hasher[T]
	seed maphash.Seed
	id   *struct{}
}

// NewFamily constructs a hash domain for merge-compatible HLL sketches.
func NewFamily[T any](h maphash.Hasher[T]) *Family[T] {
	if h == nil {
		panic("sketch: nil Hasher")
	}
	return &Family[T]{h: h, seed: maphash.MakeSeed(), id: &struct{}{}}
}

// NewHyperLogLog constructs an HLL member with precision p in f's domain.
func (f *Family[T]) NewHyperLogLog(p uint8) (*HyperLogLog[T], error) {
	if p < 4 || p > 18 {
		return nil, errors.New("sketch: precision must be between 4 and 18")
	}
	n := 1 << p
	return &HyperLogLog[T]{family: f, p: p, reg: make([]uint8, n)}, nil
}
