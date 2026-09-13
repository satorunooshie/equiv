# 0009: Weak interner lifetime

Weak interners retain only `weak.Pointer` references and remove dead entries
during lookup, insertion, or explicit sweep.
