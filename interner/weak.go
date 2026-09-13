package interner

import (
	"hash/maphash"
	"iter"
	"slices"
	"weak"
)

type weakEntry[T any] struct {
	hash uint64
	ptr  weak.Pointer[T]
}

// Weak retains canonical pointers only through weak references. Nil pointers
// are invalid; the zero value is invalid and the interner is non-concurrent.
type Weak[T any] struct {
	h       maphash.Hasher[*T]
	seed    maphash.Seed
	entries []weakEntry[T]
}

// NewWeak constructs a weak interner. Nil pointers are rejected by Intern and Lookup.
func NewWeak[T any](h maphash.Hasher[*T]) *Weak[T] {
	if h == nil {
		panic("interner: nil Hasher")
	}
	return &Weak[T]{h: h, seed: maphash.MakeSeed()}
}

func (i *Weak[T]) digest(v *T) uint64 {
	var x maphash.Hash
	x.SetSeed(i.seed)
	i.h.Hash(&x, v)
	return x.Sum64()
}

// Intern returns the live canonical pointer, inserting v when absent.
func (i *Weak[T]) Intern(v *T) *T {
	if v == nil {
		panic("interner: nil pointer")
	}
	d := i.digest(v)
	keep := i.entries[:0]
	for _, e := range i.entries {
		p := e.ptr.Value()
		if p == nil {
			continue
		}
		keep = append(keep, e)
		if e.hash == d && i.h.Equal(p, v) {
			i.entries = keep
			return p
		}
	}
	i.entries = keep
	i.entries = append(i.entries, weakEntry[T]{d, weak.Make(v)})
	return v
}

// Lookup returns the live canonical pointer for v, if present.
func (i *Weak[T]) Lookup(v *T) (*T, bool) {
	if v == nil {
		panic("interner: nil pointer")
	}
	d := i.digest(v)
	keep := i.entries[:0]
	var found *T
	for _, e := range i.entries {
		p := e.ptr.Value()
		if p == nil {
			continue
		}
		keep = append(keep, e)
		if found == nil && e.hash == d && i.h.Equal(p, v) {
			found = p
		}
	}
	i.entries = keep
	return found, found != nil
}

// Sweep removes entries whose canonical objects are no longer live.
func (i *Weak[T]) Sweep() int {
	before := len(i.entries)
	i.entries = slices.DeleteFunc(i.entries, func(e weakEntry[T]) bool {
		return e.ptr.Value() == nil
	})
	return before - len(i.entries)
}

// LenApprox returns the number of index entries before dead-entry compaction.
func (i *Weak[T]) LenApprox() int { return len(i.entries) }

// All returns currently live canonical pointers.
func (i *Weak[T]) All() iter.Seq[*T] {
	return func(y func(*T) bool) {
		for _, e := range i.entries {
			if p := e.ptr.Value(); p != nil && !y(p) {
				return
			}
		}
	}
}
