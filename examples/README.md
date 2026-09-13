# Examples

Each directory is an independent `go run`-able program:

```bash
go run ./examples/semantic-map
go run ./examples/custom-identity
go run ./examples/loading-cache
go run ./examples/concurrent-cache
go run ./examples/interner
go run ./examples/probabilistic
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
