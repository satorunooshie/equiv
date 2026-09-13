# Contributing

Run `gofmt -w` on changed Go files and run `make verify`. This executes the
unit suite, race detector, vet, fuzz smoke test, and benchmark suite. Changes
to collection behavior require tests for
semantic equality, collisions, mutation, and the relevant concurrency or
expiration contract.
