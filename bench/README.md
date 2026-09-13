# Isolated benchmark module

This module keeps competitor dependencies out of the production `go.mod`.
Versions are pinned in `go.mod`; update them deliberately and record the
resulting `go.sum`.

Run reproducible samples from this directory:

```sh
GOTOOLCHAIN=local go test -run '^$' -bench . -benchmem -count 10 ./...
```

Use `benchstat` (or an equivalent statistical comparison) for before/after
claims. A single run or a cherry-picked minimum is not evidence of a public
performance difference. Benchmarks compare `equiv` with built-in `map`,
`gomap`, and a representative specialized LRU cache where applicable. They
separate raw comparable-key operation cost from semantic workloads that include
per-operation conversion or precomputed canonical keys.

The concurrent package uses `RunParallel` specifically for its parallel
workloads; the other new benchmarks use `B.Loop`. Probabilistic benchmarks
cover Bloom, Cuckoo, and XOR filters.

Category results include hit rate and resident-entry metrics for caches, and
bits/key and measured false-positive rate for probabilistic filters. Throughput
does not establish an overall ranking: concurrent snapshot/iteration semantics,
cache policy and loader behavior, and probabilistic accuracy remain separate
contract dimensions.
