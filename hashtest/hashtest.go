// Package hashtest checks the laws required by maphash.Hasher implementations.
package hashtest

import (
	"hash/maphash"
	"iter"
	"math/rand"
	"testing"
)

// Check verifies semantic equality laws, repeatability, and equal-hash
// compatibility across independent hash seeds.
func Check[T any](t testing.TB, h maphash.Hasher[T], values []T) {
	t.Helper()
	for _, x := range values {
		if !h.Equal(x, x) {
			t.Errorf("hasher is not reflexive")
		}
		for _, y := range values {
			if h.Equal(x, y) != h.Equal(y, x) {
				t.Errorf("hasher is not symmetric")
			}
			if h.Equal(x, y) {
				for _, z := range values {
					if h.Equal(y, z) && !h.Equal(x, z) {
						t.Errorf("hasher is not transitive")
					}
				}
			}
		}
	}
	seeds := []maphash.Seed{maphash.MakeSeed(), maphash.MakeSeed(), maphash.MakeSeed()}
	for _, s := range seeds {
		for _, x := range values {
			var a, b maphash.Hash
			a.SetSeed(s)
			b.SetSeed(s)
			h.Hash(&a, x)
			h.Hash(&b, x)
			if a.Sum64() != b.Sum64() {
				t.Errorf("hasher is not repeatable")
			}
		}
		for i, x := range values {
			for j, y := range values {
				if h.Equal(x, y) {
					var a, b maphash.Hash
					a.SetSeed(s)
					b.SetSeed(s)
					h.Hash(&a, x)
					h.Hash(&b, y)
					if a.Sum64() != b.Sum64() {
						t.Errorf("Equal values have different hashes at %d,%d", i, j)
					}
				}
			}
		}
	}
}

// CheckPairs applies Check to each supplied pair of values.
func CheckPairs[T any](t testing.TB, h maphash.Hasher[T], pairs iter.Seq2[T, T]) {
	for a, b := range pairs {
		Check(t, h, []T{a, b})
	}
}

// CheckFunc generates deterministic values and applies Check to them.
func CheckFunc[T any](t testing.TB, h maphash.Hasher[T], gen func(*rand.Rand) T, n int) {
	r := rand.New(rand.NewSource(1))
	v := make([]T, n)
	for i := range v {
		v[i] = gen(r)
	}
	Check(t, h, v)
}
