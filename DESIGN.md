# Design

`equiv` uses `maphash.Hasher[T]` as its sole semantic identity protocol.
`hashers` constructs reusable definitions and `hashtest` checks their base and
strict-equivalence contracts. Collections retain the first equivalent key
representation, own independent random hash domains, and do not serialize or
reflect over keys.

Consumers are intentionally grouped by contract:

- Exact: `Map`, `Set`, `ordered`, `multimap`, `multiset`, and `interner` use
  `Hash` plus `Equal` for collision-safe lookup.
- Operational: `concurrent` and `cache` add synchronization, expiration,
  loading, eviction, or admission state while preserving exact resident-key
  correctness.
- Probabilistic: `bloom`, `cuckoo`, `xorfilter`, and `sketch` reuse the hashing
  definition under structure-specific approximate guarantees.

See [`docs/key-semantics-contracts.md`](docs/key-semantics-contracts.md),
`docs.md`, and the ADRs in `docs/adr/` for the normative decisions.
