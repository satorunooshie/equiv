// Package interner provides strong and weak semantic canonicalization maps.
// Interner values are non-concurrent and must be constructed with a Hasher.
package interner

import (
	"hash/maphash"
	"iter"

	"github.com/satorunooshie/equiv"
)

// Strong retains canonical values strongly. Its zero value is invalid and it
// is not safe for concurrent use or copying after first use.
type Strong[T any] struct{ m *equiv.Set[T] }

// NewStrong constructs a strong interner using h as its semantic identity.
func NewStrong[T any](h maphash.Hasher[T]) *Strong[T] { return &Strong[T]{equiv.NewSet[T](h)} }

// Intern returns the resident canonical value, inserting v when absent.
func (i *Strong[T]) Intern(v T) T {
	if x, ok := i.m.Lookup(v); ok {
		return x
	}
	i.m.Insert(v)
	return v
}

// Lookup returns the resident canonical value for v.
func (i *Strong[T]) Lookup(v T) (T, bool) { return i.m.Lookup(v) }

// Delete removes v if present.
func (i *Strong[T]) Delete(v T) bool { return i.m.Delete(v) }

// All returns all canonical values.
func (i *Strong[T]) All() iter.Seq[T] { return i.m.All() }

// Len returns the number of canonical values.
func (i *Strong[T]) Len() int { return i.m.Len() }

// Clear removes all canonical values without changing the hash domain.
func (i *Strong[T]) Clear() { i.m.Clear() }
