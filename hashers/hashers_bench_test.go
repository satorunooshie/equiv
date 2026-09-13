package hashers

import (
	"hash/maphash"
	"testing"
)

var (
	benchmarkHash  uint64
	benchmarkEqual bool
)

func BenchmarkBytesHash(b *testing.B) {
	h := Bytes()
	value := []byte("semantic identity benchmark payload")
	benchmarkHashHasher(b, h, value)
}

func BenchmarkBytesEqual(b *testing.B) {
	h := Bytes()
	a := []byte("semantic identity benchmark payload")
	bvalue := append([]byte(nil), a...)
	benchmarkEqualHasher(b, h, a, bvalue)
}

func BenchmarkByHash(b *testing.B) {
	h := By(func(value string) string { return value }, maphash.ComparableHasher[string]{})
	benchmarkHashHasher(b, h, "semantic identity benchmark payload")
}

func BenchmarkByEqual(b *testing.B) {
	h := By(func(value string) string { return value }, maphash.ComparableHasher[string]{})
	benchmarkEqualHasher(b, h, "same", "same")
}

func BenchmarkDerefHash(b *testing.B) {
	h := Deref(maphash.ComparableHasher[string]{})
	value := new("semantic identity benchmark payload")
	benchmarkHashHasher(b, h, value)
}

func BenchmarkDerefEqual(b *testing.B) {
	h := Deref(maphash.ComparableHasher[string]{})
	a := new("same")
	bvalue := new("same")
	benchmarkEqualHasher(b, h, a, bvalue)
}

func BenchmarkSliceHash(b *testing.B) {
	for _, size := range []int{0, 1, 8, 64} {
		b.Run("Len"+itoa(size), func(b *testing.B) {
			h := Slice(maphash.ComparableHasher[string]{})
			value := make([]string, size)
			for i := range value {
				value[i] = "value"
			}
			benchmarkHashHasher(b, h, value)
		})
	}
}

func BenchmarkSliceEqual(b *testing.B) {
	for _, size := range []int{0, 1, 8, 64} {
		b.Run("Len"+itoa(size), func(b *testing.B) {
			h := Slice(maphash.ComparableHasher[string]{})
			a := make([]string, size)
			bvalue := append([]string(nil), a...)
			benchmarkEqualHasher(b, h, a, bvalue)
		})
	}
}

func BenchmarkTuple2Hash(b *testing.B) {
	h := Tuple2Of(maphash.ComparableHasher[int]{}, maphash.ComparableHasher[string]{})
	benchmarkHashHasher(b, h, Tuple2[int, string]{First: 42, Second: "value"})
}

func BenchmarkTuple2Equal(b *testing.B) {
	h := Tuple2Of(maphash.ComparableHasher[int]{}, maphash.ComparableHasher[string]{})
	value := Tuple2[int, string]{First: 42, Second: "value"}
	benchmarkEqualHasher(b, h, value, value)
}

func BenchmarkTuple3Hash(b *testing.B) {
	h := Tuple3Of(maphash.ComparableHasher[int]{}, maphash.ComparableHasher[string]{}, maphash.ComparableHasher[bool]{})
	benchmarkHashHasher(b, h, Tuple3[int, string, bool]{First: 42, Second: "value", Third: true})
}

func BenchmarkTuple3Equal(b *testing.B) {
	h := Tuple3Of(maphash.ComparableHasher[int]{}, maphash.ComparableHasher[string]{}, maphash.ComparableHasher[bool]{})
	value := Tuple3[int, string, bool]{First: 42, Second: "value", Third: true}
	benchmarkEqualHasher(b, h, value, value)
}

func BenchmarkStructHash(b *testing.B) {
	for _, fields := range []int{1, 2, 4} {
		b.Run("Fields"+itoa(fields), func(b *testing.B) {
			h := benchmarkStructHasher(fields)
			benchmarkHashHasher(b, h, benchmarkStruct{a: 1, b: "value", c: true, d: 2})
		})
	}
}

func BenchmarkStructEqual(b *testing.B) {
	for _, fields := range []int{1, 2, 4} {
		b.Run("Fields"+itoa(fields), func(b *testing.B) {
			h := benchmarkStructHasher(fields)
			value := benchmarkStruct{a: 1, b: "value", c: true, d: 2}
			benchmarkEqualHasher(b, h, value, value)
		})
	}
}

func BenchmarkNestedSliceByHash(b *testing.B) {
	h := Slice(By(func(value benchmarkStruct) string { return value.b }, maphash.ComparableHasher[string]{}))
	value := []benchmarkStruct{{b: "one"}, {b: "two"}, {b: "three"}}
	benchmarkHashHasher(b, h, value)
}

func BenchmarkNestedStructSliceHash(b *testing.B) {
	h := Struct[benchmarkWithSlice]().
		Field(func(value benchmarkWithSlice) []string { return value.values }, Slice(maphash.ComparableHasher[string]{})).
		Build()
	value := benchmarkWithSlice{values: []string{"one", "two", "three"}}
	benchmarkHashHasher(b, h, value)
}

func BenchmarkSliceHasherHandwrittenHash(b *testing.B) {
	value := []string{"one", "two", "three"}
	seed := maphash.MakeSeed()
	b.ReportAllocs()
	for b.Loop() {
		var hash maphash.Hash
		hash.SetSeed(seed)
		hash.WriteString(itoa(len(value)))
		for _, item := range value {
			hash.WriteString(item)
		}
		benchmarkHash = hash.Sum64()
	}
}

func benchmarkHashHasher[T any](b *testing.B, h maphash.Hasher[T], value T) {
	b.Helper()
	seed := maphash.MakeSeed()
	b.ReportAllocs()
	for b.Loop() {
		var hash maphash.Hash
		hash.SetSeed(seed)
		h.Hash(&hash, value)
		benchmarkHash = hash.Sum64()
	}
}

func benchmarkEqualHasher[T any](b *testing.B, h maphash.Hasher[T], a, value T) {
	b.Helper()
	b.ReportAllocs()
	for b.Loop() {
		benchmarkEqual = h.Equal(a, value)
	}
}

type benchmarkStruct struct {
	a int
	b string
	c bool
	d uint64
}

type benchmarkWithSlice struct {
	values []string
}

func benchmarkStructHasher(fields int) maphash.Hasher[benchmarkStruct] {
	b := Struct[benchmarkStruct]().Field(func(value benchmarkStruct) int { return value.a }, maphash.ComparableHasher[int]{})
	if fields >= 2 {
		b.Field(func(value benchmarkStruct) string { return value.b }, maphash.ComparableHasher[string]{})
	}
	if fields >= 3 {
		b.Field(func(value benchmarkStruct) bool { return value.c }, maphash.ComparableHasher[bool]{})
	}
	if fields >= 4 {
		b.Field(func(value benchmarkStruct) uint64 { return value.d }, maphash.ComparableHasher[uint64]{})
	}
	return b.Build()
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var result [20]byte
	i := len(result)
	for value > 0 {
		i--
		result[i] = digits[value%10]
		value /= 10
	}
	return string(result[i:])
}
