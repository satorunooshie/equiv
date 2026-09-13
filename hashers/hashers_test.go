package hashers_test

import (
	"hash/maphash"
	"testing"

	"github.com/satorunooshie/equiv/hashers"
	"github.com/satorunooshie/equiv/hashtest"
)

type request struct {
	method string
	path   string
}

func TestCompositeHashersSatisfyHasherLaws(t *testing.T) {
	intHasher := maphash.ComparableHasher[int]{}
	stringHasher := maphash.ComparableHasher[string]{}
	hashtest.CheckEquivalence(t, hashers.Bytes(), [][]byte{nil, {}, []byte("a"), []byte("b")})
	hashtest.CheckEquivalence(t, hashers.Deref(stringHasher), []*string{nil, new("a"), new("b")})
	hashtest.CheckEquivalence(t, hashers.Slice(stringHasher), [][]string{{}, {"a"}, {"a", "b"}})
	hashtest.CheckEquivalence(t, hashers.Tuple2Of(intHasher, stringHasher), []hashers.Tuple2[int, string]{
		{First: 1, Second: "a"}, {First: 2, Second: "b"},
	})
	h := hashers.Struct[request]().
		Field(func(r request) string { return r.method }, stringHasher).
		Field(func(r request) string { return r.path }, stringHasher).
		Build()
	hashtest.CheckEquivalence(t, h, []request{{"GET", "/"}, {"POST", "/items"}})
	hashtest.CheckEquivalence(t, hashers.By(func(r request) string { return r.path }, stringHasher), []request{{"GET", "/"}, {"POST", "/items"}})
	hashtest.CheckEquivalence(t, hashers.Tuple3Of(intHasher, stringHasher, stringHasher), []hashers.Tuple3[int, string, string]{
		{First: 1, Second: "a", Third: "x"}, {First: 2, Second: "b", Third: "y"},
	})
	custom := hashers.Func(
		func(out *maphash.Hash, v string) { out.WriteString(v) },
		func(a, b string) bool { return a == b },
	)
	hashtest.CheckEquivalence(t, custom, []string{"a", "b"})
}

func TestCompositeConstructorsRejectNilHashers(t *testing.T) {
	var h maphash.Hasher[int]
	assertPanics(t, func() { hashers.By(func(string) int { return 0 }, h) })
	assertPanics(t, func() { hashers.Deref(h) })
	assertPanics(t, func() { hashers.Slice(h) })
	assertPanics(t, func() { hashers.Tuple2Of(h, maphash.ComparableHasher[string]{}) })
	assertPanics(t, func() { hashers.Struct[string]().Field(func(string) int { return 0 }, h) })
	assertPanics(t, func() { hashers.Tuple3Of(h, maphash.ComparableHasher[string]{}, maphash.ComparableHasher[string]{}) })
	assertPanics(t, func() { hashers.Func[int](nil, func(int, int) bool { return true }) })
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
}
