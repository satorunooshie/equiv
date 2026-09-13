# 0008: Concurrent consistency

Point operations are linearizable. Iterators snapshot shard contents before
yielding and never hold a shard lock across user code.
