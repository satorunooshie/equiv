// Package cuckoo implements a bounded, fingerprint-based Cuckoo filter.
package cuckoo

import (
	"errors"
	"hash/maphash"
)

// Config controls Cuckoo filter capacity, fingerprint width, bucket size, and
// bounded insertion relocation.
type Config struct {
	Capacity        uint64
	FingerprintBits uint8
	BucketSize      uint8
	MaxKicks        uint32
}

// Filter is a non-concurrent Cuckoo filter. Delete is multiplicity-sensitive
// and may remove a false-positive fingerprint for an absent value.
type Filter[T any] struct {
	h        maphash.Hasher[T]
	seed     maphash.Seed
	buckets  [][]uint16
	mask     uint64
	bits     uint8
	maxKicks uint32
	capacity uint64
	n        uint64
}

// New constructs a bounded Cuckoo filter according to cfg.
func New[T any](h maphash.Hasher[T], cfg Config) (*Filter[T], error) {
	if h == nil {
		panic("cuckoo: nil Hasher")
	}
	if cfg.Capacity == 0 {
		return nil, errors.New("cuckoo: zero capacity")
	}
	if cfg.FingerprintBits == 0 {
		cfg.FingerprintBits = 8
	}
	if cfg.FingerprintBits != 8 && cfg.FingerprintBits != 12 && cfg.FingerprintBits != 16 {
		return nil, errors.New("cuckoo: unsupported fingerprint size")
	}
	if cfg.BucketSize == 0 {
		cfg.BucketSize = 4
	}
	if cfg.MaxKicks == 0 {
		cfg.MaxKicks = 500
	}
	maxInt := uint64(^uint(0) >> 1)
	if cfg.Capacity > maxInt-uint64(cfg.BucketSize)+1 {
		return nil, errors.New("cuckoo: capacity is too large")
	}
	need := (cfg.Capacity + uint64(cfg.BucketSize) - 1) / uint64(cfg.BucketSize)
	bc := uint64(1)
	for bc < need {
		if bc > maxInt/2 {
			return nil, errors.New("cuckoo: capacity is too large")
		}
		bc <<= 1
	}
	b := make([][]uint16, bc)
	for i := range b {
		b[i] = make([]uint16, cfg.BucketSize)
	}
	return &Filter[T]{h: h, seed: maphash.MakeSeed(), buckets: b, mask: bc - 1, bits: cfg.FingerprintBits, maxKicks: cfg.MaxKicks, capacity: cfg.Capacity}, nil
}

func (f *Filter[T]) digest(v T) uint64 {
	var x maphash.Hash
	x.SetSeed(f.seed)
	f.h.Hash(&x, v)
	return x.Sum64()
}

func (f *Filter[T]) fingerprint(d uint64) uint16 {
	mask := uint64(1<<f.bits) - 1
	v := uint16((d ^ (d >> 17) ^ (d >> 37)) & mask)
	if v == 0 {
		v = 1
	}
	return v
}

func mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	return x ^ (x >> 31)
}

func (f *Filter[T]) locations(d uint64, fp uint16) (uint64, uint64) {
	a := d & f.mask
	b := (a ^ (mix(uint64(fp)) & f.mask)) & f.mask
	return a, b
}

func (f *Filter[T]) find(i uint64, fp uint16) int {
	for j, x := range f.buckets[i] {
		if x == fp {
			return j
		}
	}
	return -1
}

func (f *Filter[T]) empty(i uint64) int {
	for j, x := range f.buckets[i] {
		if x == 0 {
			return j
		}
	}
	return -1
}

// Insert adds v, returning false when the bounded table cannot accept it.
func (f *Filter[T]) Insert(v T) bool {
	if f.n >= f.capacity {
		return false
	}
	d := f.digest(v)
	fp := f.fingerprint(d)
	a, b := f.locations(d, fp)
	if j := f.empty(a); j >= 0 {
		f.buckets[a][j] = fp
		f.n++
		return true
	}
	if j := f.empty(b); j >= 0 {
		f.buckets[b][j] = fp
		f.n++
		return true
	}
	i := a
	if mix(d)&1 != 0 {
		i = b
	}
	backup := make([][]uint16, len(f.buckets))
	for q := range f.buckets {
		backup[q] = append([]uint16(nil), f.buckets[q]...)
	}
	cur := fp
	for k := uint32(0); k < f.maxKicks; k++ {
		j := int(mix(d+uint64(k)) % uint64(len(f.buckets[i])))
		cur, f.buckets[i][j] = f.buckets[i][j], cur
		i = (i ^ (mix(uint64(cur)) & f.mask)) & f.mask
		if q := f.empty(i); q >= 0 {
			f.buckets[i][q] = cur
			f.n++
			return true
		}
	}
	for q := range f.buckets {
		copy(f.buckets[q], backup[q])
	}
	return false
}

// Contains reports whether v may be present; false positives are possible.
func (f *Filter[T]) Contains(v T) bool {
	d := f.digest(v)
	fp := f.fingerprint(d)
	a, b := f.locations(d, fp)
	return f.find(a, fp) >= 0 || f.find(b, fp) >= 0
}

// Delete removes one matching fingerprint and is safe only for inserted
// values with matching multiplicity.
func (f *Filter[T]) Delete(v T) bool {
	d := f.digest(v)
	fp := f.fingerprint(d)
	a, b := f.locations(d, fp)
	i := f.find(a, fp)
	if i < 0 {
		i = f.find(b, fp)
		if i < 0 {
			return false
		}
		f.buckets[b][i] = 0
	} else {
		f.buckets[a][i] = 0
	}
	f.n--
	return true
}

// Len returns the number of successful insertions not yet deleted.
func (f *Filter[T]) Len() uint64 { return f.n }

// LoadFactor returns the fraction of occupied bucket slots.
func (f *Filter[T]) LoadFactor() float64 {
	return float64(f.n) / float64(len(f.buckets)*len(f.buckets[0]))
}
