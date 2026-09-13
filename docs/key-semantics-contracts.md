# Key-semantics consumer contracts

`maphash.Hasher[T]` is the sole key-semantics protocol in this repository.
`hashers` builds definitions, `hashtest` checks them, and the structures below
consume them. A hash collision is not an equality match for exact consumers.

| Consumer | Class | Uses `Hash` | Uses `Equal` | Exact lookup | Digest-only state and collision consequence |
| --- | --- | --- | --- | --- | --- |
| `equiv.Map`, `equiv.Set` | exact | yes | yes | yes | none; collisions affect probe cost only |
| `ordered` | exact | yes | yes | yes | none; order is independent of digest |
| `multimap`, `multiset` | exact | yes | yes | yes | none; multiplicity/value ownership remains exact |
| `interner` | exact | yes | yes | yes | none; equivalent values share a canonical value only |
| `concurrent` | operational | yes | yes | yes | shard selection uses the digest; collision affects distribution only |
| resident `cache` entries | operational | yes | yes | yes | resident map is exact; collisions can affect probe cost only |
| cache load coalescing | operational | yes | yes | yes | digest buckets are only an index; non-equivalent keys never share a load |
| cache frequency / ghost metadata | operational policy | yes | no | no | digest collision may change admission or eviction policy decisions |
| `bloom` | probabilistic | yes | no | no | false positives are possible; inserted values have no false negatives |
| `cuckoo` | probabilistic | yes | no | no | fingerprint collisions are part of the probabilistic contract |
| `xorfilter` | probabilistic | yes | no | no | false positives are possible; the filter is immutable after construction |
| `sketch` | probabilistic | yes | no | no | estimates and merge behavior are approximate by design |

## Cache safety boundary

Resident entries and in-flight loads are correctness-critical. A collision may
not return the wrong value, overwrite a non-equivalent key, delete an unrelated
entry, corrupt TTL/TTI or capacity accounting, or share a loader result between
non-equivalent keys. Frequency estimators and ARC/2Q ghost state are policy
metadata; digest collisions may change future policy decisions, but never the
resident lookup result.

## Shared hasher requirements

Consumers pass the standard `maphash.Hasher[T]` directly. The hasher must be
safe for the caller's concurrency model and logically stateless. `hashtest`
provides finite diagnostic contract checks; it does not mathematically prove
statelessness for arbitrary inputs.
