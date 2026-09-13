// Package sketch provides non-concurrent Count-Min and HyperLogLog sketches.
// Count-Min estimates may overcount, while HyperLogLog estimates cardinality.
package sketch

import (
	"errors"
	"hash/maphash"
	"math"
)

// CountMin is a non-concurrent Count-Min sketch whose estimates may overcount
// but never undercount inserted increments.
type CountMin[T any] struct {
	h     maphash.Hasher[T]
	seed  maphash.Seed
	rows  [][]uint64
	width uint64
	total uint64
}

// NewCountMin constructs a Count-Min sketch targeting additive error epsilon
// with confidence 1-delta.
func NewCountMin[T any](h maphash.Hasher[T], epsilon, delta float64) (*CountMin[T], error) {
	if h == nil {
		panic("sketch: nil Hasher")
	}
	if math.IsNaN(epsilon) || math.IsNaN(delta) || epsilon <= 0 || epsilon >= 1 || delta <= 0 || delta >= 1 {
		return nil, errors.New("sketch: invalid parameters")
	}
	wFloat := math.Ceil(math.E / epsilon)
	dFloat := math.Ceil(math.Log(1 / delta))
	maxInt := uint64(^uint(0) >> 1)
	if math.IsInf(wFloat, 0) || wFloat > float64(maxInt) || math.IsInf(dFloat, 0) || dFloat > float64(maxInt) || dFloat < 1 {
		return nil, errors.New("sketch: parameters require too much memory")
	}
	d := int(dFloat)
	w := uint64(wFloat)
	r := make([][]uint64, d)
	for i := range r {
		r[i] = make([]uint64, w)
	}
	return &CountMin[T]{h: h, seed: maphash.MakeSeed(), rows: r, width: w}, nil
}

func (s *CountMin[T]) hashes(v T) (uint64, uint64) {
	var x maphash.Hash
	x.SetSeed(s.seed)
	s.h.Hash(&x, v)
	a := x.Sum64()
	var y maphash.Hash
	y.SetSeed(s.seed)
	y.WriteString("equiv:count-min")
	s.h.Hash(&y, v)
	return a, y.Sum64()
}

func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// Add increases v's estimated frequency by n.
func (s *CountMin[T]) Add(v T, n uint64) {
	added := n
	a, b := s.hashes(v)
	minv := uint64(math.MaxUint64)
	for i := range s.rows {
		p := mix64(a+uint64(i)*0x9e3779b97f4a7c15) ^ mix64(b+uint64(i)*0x9e3779b97f4a7c15+0x6a09e667f3bcc909)
		if x := s.rows[i][p%s.width]; x < minv {
			minv = x
		}
	}
	if n > math.MaxUint64-minv {
		n = math.MaxUint64 - minv
	}
	target := minv + n
	for i := range s.rows {
		p := mix64(a+uint64(i)*0x9e3779b97f4a7c15) ^ mix64(b+uint64(i)*0x9e3779b97f4a7c15+0x6a09e667f3bcc909)
		j := p % s.width
		if s.rows[i][j] < target {
			s.rows[i][j] = target
		}
	}
	if added > math.MaxUint64-s.total {
		s.total = math.MaxUint64
	} else {
		s.total += added
	}
}

// Estimate returns an estimate that does not underestimate unsaturated counts.
func (s *CountMin[T]) Estimate(v T) uint64 {
	a, b := s.hashes(v)
	x := uint64(math.MaxUint64)
	for i := range s.rows {
		p := mix64(a+uint64(i)*0x9e3779b97f4a7c15) ^ mix64(b+uint64(i)*0x9e3779b97f4a7c15+0x6a09e667f3bcc909)
		x = min(x, s.rows[i][p%s.width])
	}
	return x
}

// Total returns the saturated sum of all increments.
func (s *CountMin[T]) Total() uint64 { return s.total }

// Clear resets counters while preserving the sketch's hash domain.
func (s *CountMin[T]) Clear() {
	for i := range s.rows {
		clear(s.rows[i])
	}
	s.total = 0
}
