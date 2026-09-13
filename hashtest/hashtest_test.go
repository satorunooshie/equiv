package hashtest

import (
	"hash/maphash"
	"math"
	"math/rand"
	"testing"
)

func TestCheckAndHelpers(t *testing.T) {
	h := maphash.ComparableHasher[int]{}
	CheckHasher(t, h, []int{1, 2, 3})
	CheckEquivalence(t, h, []int{1, 2, 3})
	Check(t, h, []int{1, 2, 3})
	CheckPairs(t, h, func(yield func(int, int) bool) {
		if !yield(1, 1) {
			return
		}
		yield(2, 2)
	})
	CheckFunc(t, h, func(r *rand.Rand) int {
		return int(r.Int63())
	}, 10)
}

func TestCheckHasherAcceptsNaN(t *testing.T) {
	h := maphash.ComparableHasher[float64]{}
	values := []float64{0, 1, math.NaN()}

	CheckHasher(t, h, values)
	if issues := checkEquivalence(h, values); len(issues) == 0 {
		t.Fatal("CheckEquivalence accepted a non-reflexive NaN value")
	}
}

func TestCheckHasherRejectsHashCompatibilityViolation(t *testing.T) {
	if issues := checkHasher(parityHasher{}, []int{1, 2, 3}); len(issues) == 0 {
		t.Fatal("CheckHasher accepted equal values with different hashes")
	}
}

func TestCheckHasherRejectsStatefulHasher(t *testing.T) {
	if issues := checkHasher(&statefulHasher{}, []int{1, 2, 3}); len(issues) == 0 {
		t.Fatal("CheckHasher accepted a stateful hash implementation")
	}
}

func TestCheckEquivalenceRejectsAsymmetricRelation(t *testing.T) {
	if issues := checkEquivalence(asymmetricHasher{}, []int{1, 2}); len(issues) == 0 {
		t.Fatal("CheckEquivalence accepted an asymmetric relation")
	}
}

func TestCheckEquivalenceRejectsNonTransitiveRelation(t *testing.T) {
	if issues := checkEquivalence(nonTransitiveHasher{}, []int{1, 2, 3}); len(issues) == 0 {
		t.Fatal("CheckEquivalence accepted a non-transitive relation")
	}
}

type parityHasher struct{}

func (parityHasher) Hash(h *maphash.Hash, value int) {
	_, _ = h.Write([]byte{byte(value)})
}

func (parityHasher) Equal(a, b int) bool { return a%2 == b%2 }

type statefulHasher struct{ calls uint8 }

func (h *statefulHasher) Hash(dst *maphash.Hash, value int) {
	h.calls++
	_, _ = dst.Write([]byte{byte(value), h.calls})
}

func (*statefulHasher) Equal(a, b int) bool { return a == b }

type asymmetricHasher struct{}

func (asymmetricHasher) Hash(*maphash.Hash, int) {}

func (asymmetricHasher) Equal(a, b int) bool { return a <= b }

type nonTransitiveHasher struct{}

func (nonTransitiveHasher) Hash(*maphash.Hash, int) {}

func (nonTransitiveHasher) Equal(a, b int) bool {
	if a == b {
		return true
	}
	return (a == 1 && b == 2) || (a == 2 && b == 1) ||
		(a == 2 && b == 3) || (a == 3 && b == 2)
}
