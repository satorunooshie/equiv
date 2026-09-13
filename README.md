# equiv

> Define key semantics once. Reuse them everywhere.

`equiv` is a Go 1.27 ecosystem centered on the standard
`hash/maphash.Hasher[T]` protocol. Define a key once, then reuse that definition
with exact collections, operational structures, and probabilistic structures.

`equiv` provides semantic hash-based collections for Go 1.27. The library is
designed around semantic expressiveness, standard-library interoperability,
type safety, policy completeness, and predictable performance.

Go's built-in `map` is the right default when `==` already expresses the
desired key semantics. `equiv` is for non-comparable or domain-specific keys,
and for cases where one key definition must be reused across multiple
structures.

```go
identity := hashers.Slice(maphash.ComparableHasher[string]{})
exact := equiv.NewMap[[]string, int](identity)
seen := equiv.NewSet[[]string](identity)
filter, err := bloom.New(identity, 100_000, 0.01)
if err != nil {
	panic(err)
}

exact.Set([]string{"foo", "bar"}, 42)
seen.Insert([]string{"foo", "bar"})
filter.Add([]string{"foo", "bar"})
```

Exact collections use hashing and equality for collision-safe lookup.
Probabilistic structures reuse the same hashing definition under their own
accuracy guarantees.

Data participating in a resident key's semantic identity must not be mutated
while that key remains stored. The zero value of stateful collections is
invalid; construct them with their `New` function.

## Components

### Identity layer

`hashers` builds reflection-free key definitions. `hashtest` provides
diagnostic contract checks; it does not prove correctness or statelessness for
arbitrary inputs.

### Exact consumers

`Map` / `Set`, `ordered`, `multimap` / `multiset`, and `interner` preserve exact
semantic lookup using both `Hash` and `Equal`.

### Operational consumers

`concurrent` and `cache` add synchronization, expiration, loading, eviction,
and admission policies while keeping resident-key correctness exact.

### Probabilistic consumers

`bloom`, `cuckoo`, `xorfilter`, and `sketch` provide structure-specific
approximate or probabilistic contracts.

## Package details

| Component | What it provides | Typical use | Why it exists |
| --- | --- | --- | --- |
| `equiv.Map` / `equiv.Set` | Exact semantic-key map and set, including non-comparable keys and collision-safe lookup | Content-based `[]byte`, `[]string`, or structural keys | Extends Go collection semantics beyond built-in comparability without a second equality system |
| `hashers` | Reflection-free `Bytes`, `Slice`, `By`, `Deref`, tuple, struct, and function hashers | Define identity once from fields, projections, or contents | Makes one semantic identity reusable across every component |
| `ordered` | Exact maps and sets with insertion order and explicit move operations | Stable output, ordered configuration, insertion-ordered workflows | Combines hash lookup with deterministic order |
| `multimap` / `multiset` | Multiple values per key, or multiplicity counts per element | Tags, relationships, frequencies, and bags | Models duplicate values directly instead of forcing ad-hoc slices or counters |
| `interner` | Strong and weak canonicalization | Share equivalent strings or objects | Reduces duplicate representations while making lifetime explicit |
| `concurrent` | Sharded, linearizable semantic maps and sets | Concurrent typed collections with snapshots | Provides semantic identity and predictable synchronization together |
| `cache` | FIFO, LRU, MRU, LFU, SLRU, 2Q, ARC, Clock, and W-TinyLFU caches with expiry, loading, stats, and events | Bounded local, synchronized, or sharded caches | Makes eviction, expiry, load coalescing, and observability explicit contracts |
| `bloom` / `cuckoo` / `xorfilter` | Probabilistic membership structures with different mutation and accuracy trade-offs | Fast “definitely absent?” checks or immutable allow-lists | Uses bounded memory when exact membership storage is unnecessary |
| `sketch` | Count-Min frequency estimates and merge-compatible HyperLogLog cardinality estimates | Streaming frequency and distinct-count telemetry | Summarizes large streams without retaining every key |

For runnable, practical examples, see [`examples/`](examples/). Choose an
exact collection when false positives are unacceptable; choose a probabilistic
structure when bounded memory and approximate answers are the right trade-off.
