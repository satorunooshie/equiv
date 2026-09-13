# equiv

> Define semantic identity once. Reuse it everywhere.

`equiv` is a Go 1.27 library for defining what makes two keys equivalent once, then reusing that definition across maps, sets, caches, probabilistic filters, and other keyed data structures.

Go’s built-in map is the right default when `==` already expresses the identity you need.

But sometimes identity means something else:

* `[]byte` compared by contents
* `[]string` used as a composite key
* structs identified by selected fields
* normalized or case-insensitive values
* domain objects with application-specific identity

And when the same kind of key is used across multiple data structures, those structures should agree on what “the same key” means.

A map, cache, and Bloom filter can otherwise silently encode different hashing and equality rules.

`equiv` makes `hash/maphash.Hasher[T]` the shared definition of that identity.

## Define once

```go
identity := hashers.Slice(
	maphash.ComparableHasher[string]{},
)
```

This value defines the semantic identity of a `[]string`.

## Reuse everywhere

```go
exact := equiv.NewMap[[]string, int](identity)
seen := equiv.NewSet[[]string](identity)
filter, err := bloom.New(identity, 100_000, 0.01)
if err != nil {
	panic(err)
}
```

The same identity definition is reused by every structure.

```text
                  semantic identity
                         │
                    Hasher[T]
                         │
          ┌──────────────┼──────────────┐
          │              │              │
       Exact         Operational    Probabilistic
          │              │              │
      Map / Set         Cache       Bloom / Cuckoo
      Ordered         Concurrent        XOR
      Multimap                         Sketch
      Interner
```

One identity definition. Different data structures. Consistent semantics.

## Why equiv?

Without a shared identity definition, the same domain rule tends to be reimplemented by every keyed structure.

For example, suppose an application considers two values equivalent after normalization:

```text
Map       → normalization + hashing + equality
Cache     → normalization + hashing + equality
Bloom     → normalization + hashing
Set       → normalization + hashing + equality
```

Each implementation can drift independently.

With `equiv`:

```text
                    Hasher[T]
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
         Map           Cache         Bloom
```

Identity becomes a reusable value rather than an implementation detail of each data structure.

`equiv` deliberately uses the standard `hash/maphash.Hasher[T]` protocol instead of introducing another hashing or equality abstraction.

## When should I use equiv?

Use `equiv` when:

* your key is not Go-comparable
* `==` does not express your domain’s notion of identity
* you need content-based, normalized, or projected identity
* the same key semantics must be shared across multiple structures

For example:

```go
// []string cannot be a built-in map key.
// Here its contents define its identity.
identity := hashers.Slice(
	maphash.ComparableHasher[string]{},
)
users := equiv.NewMap[[]string, User](identity)
users.Set([]string{"acme", "alice"}, user)
u, ok := users.Get([]string{"acme", "alice"})
```

The two slices do not need to be the same slice. Their contents determine whether they represent the same key.

## When should I not use equiv?

If ordinary Go equality already expresses the semantics you need, prefer the built-in map.

```go
map[string]User
map[int]Session
map[UserID]User
```

`equiv` is not intended to replace Go’s built-in collections.

It exists for cases where key identity needs to be richer than `==`, or where that identity needs to become a reusable contract across multiple data structures.

## Identity

`hashers` provides reflection-free building blocks for constructing reusable `maphash.Hasher[T]` definitions.

They can define identity for:

* bytes and slices
* projections
* dereferenced values
* tuples
* structs
* custom functions

For example, a domain object can be identified by one of its fields rather than its entire representation:

```go
identity := hashers.By(
	func(u User) UserID { return u.ID },
	maphash.ComparableHasher[UserID]{},
)
```

Every consumer receiving this hasher now uses the same definition of `User` identity.

`hashtest` can be used to verify the base and strict-equivalence contracts of custom definitions.

## Consumers

Consumers are grouped by the guarantees they provide.

### Exact

Exact structures use both hashing and equality for collision-safe lookup.

| Package | Purpose |
| --- | --- |
| `equiv.Map` / `equiv.Set` | Semantic-key maps and sets |
| `ordered` | Insertion-ordered maps and sets |
| `multimap` / `multiset` | Multiple values or multiplicities |
| `interner` | Strong and weak canonicalization |

### Operational

Operational structures preserve exact resident-key semantics while adding runtime behavior.

| Package | Purpose |
| --- | --- |
| `concurrent` | Sharded, linearizable maps and sets |
| `cache` | FIFO, LRU, MRU, LFU, SLRU, 2Q, ARC, Clock, and W-TinyLFU caches |

Caches additionally support expiry, loading, statistics, events, and admission policies.

### Probabilistic

Probabilistic structures reuse the same identity definition while providing structure-specific approximate guarantees.

| Package | Purpose |
| --- | --- |
| `bloom` | Bloom filters |
| `cuckoo` | Cuckoo filters |
| `xorfilter` | XOR filters |
| `sketch` | Count-Min Sketch and HyperLogLog |

Choose an exact structure when false positives or approximation are unacceptable.

Choose a probabilistic structure when bounded memory and approximate answers are the appropriate trade-off.

## Key semantics

`equiv` follows a few important rules:

* `maphash.Hasher[T]` is the sole semantic identity protocol.
* Exact consumers use both `Hash` and `Equal`.
* Collections retain the first equivalent key representation.
* Collections own independent random hash domains.
* Keys are not serialized or reflected over.
* Data participating in a resident key’s semantic identity must not be mutated while the key is stored.
* Stateful collections must be constructed with their `New` function; their zero value is invalid.

For the normative contracts and design decisions, see:

* [docs/key-semantics-contracts.md](docs/key-semantics-contracts.md)
* [DESIGN.md](DESIGN.md)
* [docs/adr/](docs/adr/)
* [examples/](examples/)

## Design principle

The individual data structures are useful, but they are not the core abstraction of `equiv`.

The core abstraction is the identity definition they share.

Define identity once.
Pass it to any compatible structure.
Keep key semantics consistent across the application.

That is what `equiv` is for.
