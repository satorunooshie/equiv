# Examples

Each directory is an independent `go run`-able program:

```bash
go run ./examples/semantic-map
go run ./examples/custom-identity
go run ./examples/loading-cache
go run ./examples/concurrent-cache
go run ./examples/interner
go run ./examples/probabilistic
go run ./examples/ordered-workflow
go run ./examples/multimap-tags
go run ./examples/multiset-frequency
go run ./examples/concurrent-map
go run ./examples/counting-bloom
go run ./examples/cuckoo-filter
go run ./examples/xorfilter-allowlist
go run ./examples/sketch-stream
go run ./examples/hasher-composition
go run ./examples/cache-policies
```

Use them as small starting points rather than production-ready application
templates. In particular, every resident key's identity data must remain
unchanged while it is stored, and stateful containers must be constructed with
their `New` function and not copied after first use.

| Example | Demonstrates |
| --- | --- |
| `semantic-map` | A non-comparable `[]string` key |
| `custom-identity` | Explicit identity fields with `hashers.Struct` |
| `loading-cache` | TTL and `SyncCache.GetOrLoad` |
| `concurrent-cache` | Weighted `ShardedCache` with W-TinyLFU |
| `interner` | Strong canonicalization |
| `probabilistic` | Bloom membership and HLL cardinality |
| `ordered-workflow` | Case-insensitive keys, stable insertion order, and explicit reordering |
| `multimap-tags` | Multiple ordered values per document and value deletion |
| `multiset-frequency` | Event frequencies, total multiplicity, and removal |
| `concurrent-map` | Atomic concurrent updates, sharding, and snapshots |
| `counting-bloom` | Removable approximate membership with multiplicity |
| `cuckoo-filter` | Bounded mutable membership for token revocation |
| `xorfilter-allowlist` | Immutable compact allow-list membership |
| `sketch-stream` | Streaming frequency and distinct-count estimates |
| `hasher-composition` | One structural identity reused across exact and probabilistic indexes |
| `cache-policies` | Why LRU and FIFO produce different eviction results |

The package-specific examples intentionally show the trade-off behind each
constructor: exact collections preserve the answer, while filters and sketches
answer quickly with bounded memory and may require a second source of truth.

追加例の出力例（並行処理や probabilistic な構造では値が変わり得ます）は、各
`main.go` のコメントと `go run` の出力を参照してください。
