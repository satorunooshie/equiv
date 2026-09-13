# 0008: Concurrent consistency

Point operations and global snapshots are linearizable. `All` and `Snapshot`
acquire an exclusive operation gate while copying every shard, then release
all internal locks before invoking user code. This gives the returned entries a
single global view while avoiding callbacks under lock.
