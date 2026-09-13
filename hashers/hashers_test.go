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
	hashtest.Check(t, hashers.Bytes(), [][]byte{nil, {}, []byte("a"), []byte("b")})
	hashtest.Check(t, hashers.Deref(stringHasher), []*string{nil, ptr("a"), ptr("b")})
	hashtest.Check(t, hashers.Slice(stringHasher), [][]string{{}, {"a"}, {"a", "b"}})
	hashtest.Check(t, hashers.Tuple2Of(intHasher, stringHasher), []hashers.Tuple2[int, string]{
		{First: 1, Second: "a"}, {First: 2, Second: "b"},
	})
	h := hashers.Struct[request]().
		Field(func(r request) string { return r.method }, stringHasher).
		Field(func(r request) string { return r.path }, stringHasher).
		Build()
	hashtest.Check(t, h, []request{{"GET", "/"}, {"POST", "/items"}})
	hashtest.Check(t, hashers.By(func(r request) string { return r.path }, stringHasher), []request{{"GET", "/"}, {"POST", "/items"}})
}

func ptr(v string) *string { return &v }

func TestCompositeConstructorsRejectNilHashers(t *testing.T) {
	var h maphash.Hasher[int]
	assertPanics(t, func() { hashers.By(func(string) int { return 0 }, h) })
	assertPanics(t, func() { hashers.Deref(h) })
	assertPanics(t, func() { hashers.Slice(h) })
	assertPanics(t, func() { hashers.Tuple2Of(h, maphash.ComparableHasher[string]{}) })
	assertPanics(t, func() { hashers.Struct[string]().Field(func(string) int { return 0 }, h) })
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
