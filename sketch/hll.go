package sketch

import (
	"errors"
	"hash/maphash"
	"math"
	"math/bits"
)

// HyperLogLog estimates distinct semantic values and can merge only with an
// instance from the same Family and precision.
type HyperLogLog[T any] struct {
	family *Family[T]
	p      uint8
	reg    []uint8
}

// Add records v for cardinality estimation.
func (h *HyperLogLog[T]) Add(v T) {
	var x maphash.Hash
	x.SetSeed(h.family.seed)
	h.family.h.Hash(&x, v)
	z := x.Sum64()
	idx := z >> uint(64-h.p)
	w := z << h.p
	r := uint8(bitsLeading(w) + 1)
	if r > h.reg[idx] {
		h.reg[idx] = r
	}
}
func bitsLeading(x uint64) int { return bits.LeadingZeros64(x) }

// Estimate returns the approximate number of distinct values.
func (h *HyperLogLog[T]) Estimate() uint64 {
	m := float64(len(h.reg))
	sum := 0.0
	zeros := 0
	for _, r := range h.reg {
		sum += math.Pow(2, -float64(r))
		if r == 0 {
			zeros++
		}
	}
	a := 0.7213 / (1 + 1.079/m)
	e := a * m * m / sum
	if e <= 2.5*m && zeros > 0 {
		e = m * math.Log(m/float64(zeros))
	}
	return uint64(e + 0.5)
}

// Merge combines o into h when both sketches share family and precision.
func (h *HyperLogLog[T]) Merge(o *HyperLogLog[T]) error {
	if o == nil || o.family != h.family || o.p != h.p {
		return errors.New("sketch: incompatible HyperLogLog")
	}
	for i, r := range o.reg {
		if r > h.reg[i] {
			h.reg[i] = r
		}
	}
	return nil
}

// Clear resets registers without changing family compatibility.
func (h *HyperLogLog[T]) Clear() { clear(h.reg) }

var _ maphash.Seed
