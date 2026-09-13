# equiv

`equiv` lets keyed data structures share a definition of key identity across
maps, caches, filters, and other data structures.

```go
key := hashers.Slice(maphash.ComparableHasher[string]{})
m := equiv.NewMap[[]string, Value](key)
c, _ := cache.New[[]string, Value](key, cache.Config[[]string, Value]{
	MaxEntries: 10_000,
})
f, _ := bloom.New(key, 100_000, 0.01)
```

The values remain `[]string`; only their identity is defined separately.

`equiv` uses Go 1.27’s `maphash.Hasher[T]` as that boundary.

```text
                         maphash.Hasher[T]
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                 │
           Exact            Operational      Probabilistic
              │                 │                 │
       Map / Set / Multi*      Cache          Bloom / Cuckoo
       Ordered / Interner    Concurrent       XOR / Sketch
```

For comparable keys whose `==` semantics are what you want, prefer Go’s
built-in `map`.

Use `equiv` when key identity needs to be defined separately—for example, for
non-comparable values, identity based on selected fields, or semantics shared
across multiple keyed data structures.

## Defining identity

Key identity does not always match the representation of a value.

```go
type Request struct {
	TenantID string
	Method   string
	Path     []string
	TraceID  string
}
```

Suppose `TenantID`, `Method`, and `Path` identify a request while `TraceID`
does not.

```go
hasher := maphash.ComparableHasher[string]{}

requestKey := hashers.Struct[Request]().
	Field(func(r Request) string { return r.TenantID }, hasher).
	Field(func(r Request) string { return r.Method }, hasher).
	Field(func(r Request) []string { return r.Path }, hashers.Slice(hasher)).
	Build()
```

The same definition can then be used wherever request identity is needed:

```go
responses, _ := cache.New[Request, Response](requestKey, cache.Config[Request, Response]{
	MaxEntries: 10_000,
})
inflight, _ := concurrent.NewMap[Request, *Call](requestKey)
seen := equiv.NewSet[Request](requestKey)
filter, _ := bloom.New(requestKey, 100_000, 0.01)
```

Each structure has different storage and algorithmic behavior, but they agree
on what makes two `Request` values the same key.

## Components

| Component | What it provides |
| --- | --- |
| `equiv` | Exact Map and Set |
| `hashers` | Reflection-free `maphash.Hasher[T]` composition |
| `ordered` | Insertion-ordered collections |
| `multimap` / `multiset` | Multiple values and multiplicities |
| `interner` | Strong and weak canonicalization |
| `concurrent` | Concurrent collections |
| `cache` | Loading, expiration, and eviction |
| `bloom` / `cuckoo` / `xorfilter` | Probabilistic membership |
| `sketch` | Frequency and cardinality estimation |

Exact collections use both `Hash` and `Equal`, so hash collisions do not
change equality semantics. Probabilistic structures retain the guarantees of
their respective algorithms.

## Contracts

A `maphash.Hasher[T]` used with `equiv` must satisfy:

```text
Equal(a, b) => Hash(a) == Hash(b)
```

Data participating in a stored key’s identity must not be mutated in a way
that changes its hash or equality semantics while the key is retained.

Stateful collections own independent random hash domains. Hash values are
implementation details and must not be used as persistent identifiers.

See [docs/key-semantics-contracts.md](docs/key-semantics-contracts.md) for the
complete contract.

## Design

`maphash.Hasher[T]` is the shared boundary between key identity and the
algorithms that consume it.

`equiv` does not introduce a separate equality interface, serialized key
representation, or reflection-based fallback. Values remain `T`; each
consumer adds only its own storage or algorithmic behavior.

For implementation details and design rationale, see [DESIGN.md](DESIGN.md),
the ADRs under [docs/](docs/), and runnable examples under [examples/](examples/).
