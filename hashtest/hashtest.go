// Package hashtest checks the laws required by maphash.Hasher implementations.
package hashtest

import (
	"fmt"
	"hash/maphash"
	"iter"
	"math/rand"
	"slices"
	"testing"
)

// CheckHasher checks the base maphash.Hasher contract for values.
//
// The checks are diagnostic checks over the supplied finite corpus and finite
// call sequences. They cannot prove logical statelessness for arbitrary inputs.
// In particular, CheckHasher does not require reflexivity, symmetry, or
// transitivity because maphash.ComparableHasher follows the == semantics of T.
func CheckHasher[T any](t testing.TB, h maphash.Hasher[T], values []T) {
	t.Helper()
	for _, issue := range checkHasher(h, values) {
		t.Error(issue)
	}
}

func checkHasher[T any](h maphash.Hasher[T], values []T) []string {
	var issues []string
	// Re-evaluate the complete equality matrix. This catches equality
	// implementations whose result depends on their observed call history.
	equality := make([]bool, len(values)*len(values))
	for i, x := range values {
		for j, y := range values {
			equality[i*len(values)+j] = h.Equal(x, y)
		}
	}
	for i, x := range values {
		for j, y := range values {
			if got := h.Equal(x, y); got != equality[i*len(values)+j] {
				issues = append(issues, fmt.Sprintf("Equal result is not stable at %d,%d", i, j))
			}
		}
	}

	seeds := []maphash.Seed{maphash.MakeSeed(), maphash.MakeSeed(), maphash.MakeSeed()}
	for _, s := range seeds {
		for i, x := range values {
			// A value that is not equal to itself has no repeatability
			// obligation under == semantics. ComparableHasher intentionally
			// gives NaN values unstable hashes for this reason.
			if !equality[i*len(values)+i] {
				continue
			}
			if hashValue(h, s, x) != hashValue(h, s, x) {
				issues = append(issues, "hasher is not repeatable")
			}
		}
		for i, x := range values {
			for j, y := range values {
				if equality[i*len(values)+j] && hashValue(h, s, x) != hashValue(h, s, y) {
					issues = append(issues, fmt.Sprintf("Equal values have different hashes at %d,%d", i, j))
				}
			}
		}
		// Exercise several finite call orders. A pure hasher must produce
		// the same result for a value after each history. This is a
		// diagnostic check over the supplied corpus, not a proof for all
		// possible call histories.
		sequences := [][]T{values, reversed(values), rotated(values)}
		for _, sequence := range sequences {
			for i, x := range values {
				if !equality[i*len(values)+i] {
					continue
				}
				for _, y := range sequence {
					_ = hashValue(h, s, y)
				}
				if got := hashValue(h, s, x); got != hashValue(h, s, x) {
					issues = append(issues, "hasher depends on call history")
				}
			}
		}
	}
	return issues
}

func reversed[T any](values []T) []T {
	result := slices.Clone(values)
	slices.Reverse(result)
	return result
}

func rotated[T any](values []T) []T {
	if len(values) < 2 {
		return slices.Clone(values)
	}
	return slices.Concat(values[1:], values[:1])
}

func hashValue[T any](h maphash.Hasher[T], seed maphash.Seed, value T) uint64 {
	var hash maphash.Hash
	hash.SetSeed(seed)
	h.Hash(&hash, value)
	return hash.Sum64()
}

// CheckEquivalence checks the base Hasher contract and exhaustively checks
// reflexivity, symmetry, and transitivity for values.
//
// The supplied corpus is checked exhaustively. The transitivity check may
// require O(n^3) equality calls, so callers should provide a small
// representative corpus. Larger domains should additionally use fuzzing or
// domain-specific property tests.
func CheckEquivalence[T any](t testing.TB, h maphash.Hasher[T], values []T) {
	t.Helper()
	for _, issue := range checkEquivalence(h, values) {
		t.Error(issue)
	}
}

func checkEquivalence[T any](h maphash.Hasher[T], values []T) []string {
	issues := checkHasher(h, values)
	for i, x := range values {
		if !h.Equal(x, x) {
			issues = append(issues, fmt.Sprintf("hasher is not reflexive at %d", i))
		}
		for j, y := range values {
			xy := h.Equal(x, y)
			yx := h.Equal(y, x)
			if xy != yx {
				issues = append(issues, fmt.Sprintf("hasher is not symmetric at %d,%d", i, j))
			}
			if !xy {
				continue
			}
			for k, z := range values {
				if h.Equal(y, z) && !h.Equal(x, z) {
					issues = append(issues, fmt.Sprintf("hasher is not transitive at %d,%d,%d", i, j, k))
				}
			}
		}
	}
	return issues
}

// Check verifies the strict equivalence laws in addition to the base Hasher
// contract.
func Check[T any](t testing.TB, h maphash.Hasher[T], values []T) {
	t.Helper()
	CheckEquivalence(t, h, values)
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
