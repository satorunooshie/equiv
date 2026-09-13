// Package xorfilter implements immutable XOR filters.
package xorfilter

import (
	"errors"
	"hash/maphash"
	"iter"
	"slices"

	"github.com/satorunooshie/equiv"
)

// Filter8 is an immutable non-concurrent-read XOR filter using 8-bit
// fingerprints. It has no false negatives for its construction input.
type Filter8[T any] struct {
	h     maphash.Hasher[T]
	seed  maphash.Seed
	table []uint16
	n     int
}

// Filter16 is an immutable non-concurrent-read XOR filter using 16-bit
// fingerprints. It has no false negatives for its construction input.
type Filter16[T any] struct {
	h     maphash.Hasher[T]
	seed  maphash.Seed
	table []uint16
	n     int
}

func mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	return x ^ (x >> 31)
}

func digest[T any](h maphash.Hasher[T], s maphash.Seed, v T) uint64 {
	var x maphash.Hash
	x.SetSeed(s)
	h.Hash(&x, v)
	return x.Sum64()
}

func positions(d uint64, n int) [3]int {
	a := int(mix(d) % uint64(n))
	b := int(mix(d+0x9e3779b97f4a7c15) % uint64(n))
	for b == a {
		b = (b + 1) % n
	}
	c := int(mix(d+0x3c6ef372fe94f82a) % uint64(n))
	for c == a || c == b {
		c = (c + 1) % n
	}
	return [3]int{a, b, c}
}

func build[T any](h maphash.Hasher[T], vals []T, bits uint8) (maphash.Seed, []uint16, error) {
	if len(vals) == 0 {
		return maphash.MakeSeed(), make([]uint16, 1), nil
	}
	for range 128 {
		seed := maphash.MakeSeed()
		m := int(float64(len(vals))*1.30) + 3
		deg := make([]uint8, m)
		xr := make([]int, m)
		for i, v := range vals {
			p := positions(digest(h, seed, v), m)
			for _, q := range p {
				deg[q]++
				xr[q] ^= i
			}
		}
		queue := make([]int, 0, m)
		for q, d := range deg {
			if d == 1 {
				queue = append(queue, q)
			}
		}
		type peel struct{ edge, vertex int }
		order := make([]peel, 0, len(vals))
		removed := make([]bool, len(vals))
		for len(queue) > 0 {
			q := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			if deg[q] != 1 {
				continue
			}
			e := xr[q]
			if e < 0 || e >= len(vals) || removed[e] {
				continue
			}
			removed[e] = true
			order = append(order, peel{e, q})
			p := positions(digest(h, seed, vals[e]), m)
			for _, z := range p {
				if deg[z] > 0 {
					deg[z]--
					xr[z] ^= e
					if deg[z] == 1 {
						queue = append(queue, z)
					}
				}
			}
		}
		if len(order) != len(vals) {
			continue
		}
		mask := uint16((1 << bits) - 1)
		out := make([]uint16, m)
		for _, o := range slices.Backward(order) {
			e, q := o.edge, o.vertex
			d := digest(h, seed, vals[e])
			p := positions(d, m)
			fp := uint16((d ^ (d >> 23) ^ (d >> 41)) & uint64(mask))
			if fp == 0 {
				fp = 1
			}
			out[q] = fp ^ out[p[0]] ^ out[p[1]] ^ out[p[2]]
		}
		return seed, out, nil
	}
	return maphash.Seed{}, nil, errors.New("xorfilter: unable to build filter after retries")
}

// New8 materializes values once and constructs an immutable 8-bit XOR filter.
func New8[T any](h maphash.Hasher[T], values iter.Seq[T]) (*Filter8[T], error) {
	if h == nil {
		panic("xorfilter: nil Hasher")
	}
	if values == nil {
		return nil, errors.New("xorfilter: nil values")
	}
	seen := equiv.NewSet[T](h)
	vals := make([]T, 0)
	for v := range values {
		if seen.Insert(v) {
			vals = append(vals, v)
		}
	}
	s, t, e := build(h, vals, 8)
	return &Filter8[T]{h: h, seed: s, table: t, n: len(vals)}, e
}

// Contains reports whether v may be present; inserted values have no false negatives.
func (f *Filter8[T]) Contains(v T) bool {
	if f.n == 0 {
		return false
	}
	d := digest(f.h, f.seed, v)
	p := positions(d, len(f.table))
	fp := uint16((d ^ (d >> 23) ^ (d >> 41)) & 255)
	if fp == 0 {
		fp = 1
	}
	return f.table[p[0]]^f.table[p[1]]^f.table[p[2]] == fp
}

// Len returns the number of distinct materialized input values.
func (f *Filter8[T]) Len() int { return f.n }

// New16 materializes values once and constructs an immutable 16-bit XOR filter.
func New16[T any](h maphash.Hasher[T], values iter.Seq[T]) (*Filter16[T], error) {
	if h == nil {
		panic("xorfilter: nil Hasher")
	}
	if values == nil {
		return nil, errors.New("xorfilter: nil values")
	}
	seen := equiv.NewSet[T](h)
	vals := make([]T, 0)
	for v := range values {
		if seen.Insert(v) {
			vals = append(vals, v)
		}
	}
	s, t, e := build(h, vals, 16)
	return &Filter16[T]{h: h, seed: s, table: t, n: len(vals)}, e
}

// Contains reports whether v may be present; inserted values have no false negatives.
func (f *Filter16[T]) Contains(v T) bool {
	if f.n == 0 {
		return false
	}
	d := digest(f.h, f.seed, v)
	p := positions(d, len(f.table))
	fp := uint16((d ^ (d >> 23) ^ (d >> 41)) & 65535)
	if fp == 0 {
		fp = 1
	}
	return f.table[p[0]]^f.table[p[1]]^f.table[p[2]] == fp
}

// Len returns the number of distinct materialized input values.
func (f *Filter16[T]) Len() int { return f.n }
