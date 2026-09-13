# equiv

> **Semantic hash infrastructure for Go.**

Define identity once with `hash/maphash.Hasher[T]`, then reuse it across maps,
sets, interners, caches, concurrent collections, filters, and sketches.

`equiv` provides semantic hash-based collections for Go 1.27. The library is
designed around semantic expressiveness, standard-library interoperability,
type safety, policy completeness, and predictable performance—not merely as a
faster built-in map.

```go
m := equiv.NewMap[[]string, int](hashers.Slice(maphash.ComparableHasher[string]{}))
m.Set([]string{"foo", "bar"}, 42)
v, ok := m.Get([]string{"foo", "bar"})
```

Data participating in a resident key's semantic identity must remain unchanged
while that key is stored. The zero value of stateful collections is invalid;
construct them with their `New` function.

## Components at a glance

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
