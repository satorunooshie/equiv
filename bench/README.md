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
performance difference. Benchmarks separate raw comparable-key operation cost
from semantic workloads that include per-operation conversion or precomputed
canonical keys.
