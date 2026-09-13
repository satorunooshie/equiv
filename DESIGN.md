# Design

`equiv` uses `maphash.Hasher[T]` as its sole semantic identity protocol.
Collections retain the first equivalent key representation, own independent
random hash domains, and do not serialize or reflect over keys. See `docs.md`
and the ADRs in `docs/adr/` for the normative decisions.
