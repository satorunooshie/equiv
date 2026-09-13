# Benchmark report

This report is generated from the repository benchmark suite. It records the
release-gate command and the environment used for the latest verification run;
the benchmark tests themselves remain the source of truth for raw measurements.

## Reproduction

```text
make verify
```

The isolated `bench/` module additionally compares built-in `map`,
`aristanetworks/gomap`, and `equiv.Map` for comparable and semantic-key
workloads, and contains concurrent, cache, and probabilistic baselines. Its
competitor versions are pinned in `bench/go.mod` and are not production
dependencies.

For a statistically meaningful comparison, collect multiple samples and use
`benchstat` (or an equivalent method). For example:

```text
cd bench
go test -run '^$' -bench . -benchmem -count 10 ./... > run.txt
```

Do not publish a performance claim from a single run, a cherry-picked minimum,
or a difference within measurement noise. Record the Go version, OS, GOARCH,
CPU, command, dependency versions, sample count, and comparison method.

The command runs the unit suite, race detector, vet, the deterministic fuzz
smoke test, and all package benchmarks with `-benchtime=1x`. Exact-table
benchmarks cover 8, 64, 1K, 64K, and 1M entries. Cache benchmarks cover all nine
replacement policies and uniform, Zipfian, looping, scan-pollution, hot/cold,
high-write, read-mostly, expiration-heavy, and weighted-object workloads.

## Latest verification environment

```text
OS:      darwin
Arch:    arm64
CPU:     Apple M4 Max
Go:      1.27.1
Commit:  210ddc1
```

The release suite completed successfully. `internal/table` also reports probe,
`Equal`-call, tombstone, resize, and rebuild metrics during its benchmark. The
numbers are intentionally not treated as a public API or as a cross-machine
performance promise; compare runs on the same machine and toolchain when
evaluating regressions.
