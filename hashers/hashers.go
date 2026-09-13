// Package hashers provides reflection-free semantic Hasher combinators.
package hashers

import (
	"bytes"
	"hash/maphash"
)

// bytesHasher compares byte slices by contents; nil and empty are equivalent.
// It is stateless and safe to reuse across container constructors.
type bytesHasher struct{}

// Bytes returns a content-based Hasher for byte slices. Nil and empty slices
// are semantically equal.
func Bytes() maphash.Hasher[[]byte]                { return bytesHasher{} }
func (bytesHasher) Hash(h *maphash.Hash, v []byte) { h.Write(v) }
func (bytesHasher) Equal(a, b []byte) bool         { return bytes.Equal(a, b) }

// byHasher defines identity by projecting values to a child semantic key.
type byHasher[T, K any, H maphash.Hasher[K]] struct {
	project func(T) K
	child   H
}

// By constructs a Hasher whose identity is the projected child identity.
func By[T, K any, H maphash.Hasher[K]](project func(T) K, hasher H) maphash.Hasher[T] {
	if project == nil {
		panic("hashers: nil projection")
	}
	if any(hasher) == nil {
		panic("hashers: nil Hasher")
	}
	return byHasher[T, K, H]{project, hasher}
}
func (h byHasher[T, K, H]) Hash(out *maphash.Hash, v T) { h.child.Hash(out, h.project(v)) }
func (h byHasher[T, K, H]) Equal(a, b T) bool           { return h.child.Equal(h.project(a), h.project(b)) }

// derefHasher hashes pointers with an explicit nil tag and delegates
// non-nil identity to its child Hasher.
type derefHasher[T any, H maphash.Hasher[T]] struct{ child H }

// Deref constructs a pointer Hasher with a distinct nil representation.
func Deref[T any, H maphash.Hasher[T]](h H) maphash.Hasher[*T] {
	if any(h) == nil {
		panic("hashers: nil Hasher")
	}
	return derefHasher[T, H]{h}
}

func (h derefHasher[T, H]) Hash(out *maphash.Hash, p *T) {
	if p == nil {
		out.WriteByte(0)
		return
	}
	out.WriteByte(1)
	h.child.Hash(out, *p)
}

func (h derefHasher[T, H]) Equal(a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return h.child.Equal(*a, *b)
}

// sliceHasher hashes slices structurally using length and child semantic
// digests; it does not retain the input slice.
type sliceHasher[T any, H maphash.Hasher[T]] struct{ child H }

// Slice constructs a structural Hasher for ordered slices.
func Slice[T any, H maphash.Hasher[T]](h H) maphash.Hasher[[]T] {
	if any(h) == nil {
		panic("hashers: nil Hasher")
	}
	return sliceHasher[T, H]{h}
}

func (h sliceHasher[T, H]) Hash(out *maphash.Hash, v []T) {
	var buf [8]byte
	put64(buf[:], uint64(len(v)))
	out.Write(buf[:])
	for _, x := range v {
		var sub maphash.Hash
		sub.SetSeed(out.Seed())
		h.child.Hash(&sub, x)
		put64(buf[:], sub.Sum64())
		out.Write(buf[:])
	}
}

func (h sliceHasher[T, H]) Equal(a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !h.child.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Tuple2 is the semantic pair value consumed by Tuple2Of.
type Tuple2[A, B any] struct {
	First  A
	Second B
}

// Tuple3 is the semantic triple value consumed by Tuple3Of.
type Tuple3[A, B, C any] struct {
	First  A
	Second B
	Third  C
}

// tuple2Hasher composes two child Hashers into structural pair identity.
type tuple2Hasher[A, B any, HA maphash.Hasher[A], HB maphash.Hasher[B]] struct {
	a HA
	b HB
}

// Tuple2Of constructs a Hasher for semantic pairs.
func Tuple2Of[A, B any, HA maphash.Hasher[A], HB maphash.Hasher[B]](a HA, b HB) maphash.Hasher[Tuple2[A, B]] {
	if any(a) == nil || any(b) == nil {
		panic("hashers: nil Hasher")
	}
	return tuple2Hasher[A, B, HA, HB]{a, b}
}

func (h tuple2Hasher[A, B, HA, HB]) Hash(out *maphash.Hash, v Tuple2[A, B]) {
	writeChild(out, h.a, v.First)
	writeChild(out, h.b, v.Second)
}

func (h tuple2Hasher[A, B, HA, HB]) Equal(x, y Tuple2[A, B]) bool {
	return h.a.Equal(x.First, y.First) && h.b.Equal(x.Second, y.Second)
}

// tuple3Hasher composes three child Hashers into structural triple identity.
type tuple3Hasher[A, B, C any, HA maphash.Hasher[A], HB maphash.Hasher[B], HC maphash.Hasher[C]] struct {
	a HA
	b HB
	c HC
}

// Tuple3Of constructs a Hasher for semantic triples.
func Tuple3Of[A, B, C any, HA maphash.Hasher[A], HB maphash.Hasher[B], HC maphash.Hasher[C]](a HA, b HB, c HC) maphash.Hasher[Tuple3[A, B, C]] {
	if any(a) == nil || any(b) == nil || any(c) == nil {
		panic("hashers: nil Hasher")
	}
	return tuple3Hasher[A, B, C, HA, HB, HC]{a, b, c}
}

func (h tuple3Hasher[A, B, C, HA, HB, HC]) Hash(out *maphash.Hash, v Tuple3[A, B, C]) {
	writeChild(out, h.a, v.First)
	writeChild(out, h.b, v.Second)
	writeChild(out, h.c, v.Third)
}

func (h tuple3Hasher[A, B, C, HA, HB, HC]) Equal(x, y Tuple3[A, B, C]) bool {
	return h.a.Equal(x.First, y.First) && h.b.Equal(x.Second, y.Second) && h.c.Equal(x.Third, y.Third)
}

func writeChild[T any, H maphash.Hasher[T]](out *maphash.Hash, h H, v T) {
	var sub maphash.Hash
	sub.SetSeed(out.Seed())
	h.Hash(&sub, v)
	var b [8]byte
	put64(b[:], sub.Sum64())
	out.Write(b[:])
}

func put64(b []byte, v uint64) {
	for i := range b {
		b[i] = byte(v >> uint(8*i))
	}
}

// funcHasher adapts caller-provided Hash and Equal functions. Both functions
// must be logically stateless and safe for the container's concurrency model.
type funcHasher[T any] struct {
	hash  func(*maphash.Hash, T)
	equal func(T, T) bool
}

// Func constructs a Hasher from caller-provided hash and equality functions.
// They must be pure, equivalent, and safe for the container's concurrency model.
func Func[T any](hash func(*maphash.Hash, T), equal func(T, T) bool) maphash.Hasher[T] {
	if hash == nil || equal == nil {
		panic("hashers: nil function")
	}
	return funcHasher[T]{hash, equal}
}
func (h funcHasher[T]) Hash(x *maphash.Hash, v T) { h.hash(x, v) }
func (h funcHasher[T]) Equal(a, b T) bool         { return h.equal(a, b) }

type field[T any] interface {
	hash(*maphash.Hash, T)
	equal(T, T) bool
}
type fieldImpl[T, F any, H maphash.Hasher[F]] struct {
	get func(T) F
	h   H
}

func (f fieldImpl[T, F, H]) hash(out *maphash.Hash, v T) { writeChild(out, f.h, f.get(v)) }
func (f fieldImpl[T, F, H]) equal(a, b T) bool           { return f.h.Equal(f.get(a), f.get(b)) }

// StructBuilder builds a reflection-free structural Hasher from field
// projections. The builder itself is mutable until Build is called.
type StructBuilder[T any] struct{ fields []field[T] }

// Struct starts a reflection-free structural Hasher builder.
func Struct[T any]() *StructBuilder[T] { return &StructBuilder[T]{} }

// Field adds a projected semantic field to the builder.
func (b *StructBuilder[T]) Field[F any, H maphash.Hasher[F]](get func(T) F, h H) *StructBuilder[T] {
	if get == nil {
		panic("hashers: nil field projection")
	}
	if any(h) == nil {
		panic("hashers: nil Hasher")
	}
	b.fields = append(b.fields, fieldImpl[T, F, H]{get, h})
	return b
}

type structHasher[T any] struct{ fields []field[T] }

func (h structHasher[T]) Hash(out *maphash.Hash, v T) {
	for _, f := range h.fields {
		f.hash(out, v)
	}
}

func (h structHasher[T]) Equal(a, b T) bool {
	for _, f := range h.fields {
		if !f.equal(a, b) {
			return false
		}
	}
	return true
}

// Build freezes the builder's current fields into an immutable Hasher value.
func (b *StructBuilder[T]) Build() maphash.Hasher[T] {
	fs := append([]field[T](nil), b.fields...)
	return structHasher[T]{fs}
}
