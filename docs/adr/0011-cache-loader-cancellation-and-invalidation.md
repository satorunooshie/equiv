# 0011: Loader cancellation and invalidation

Shared loads use a detached cache-level context. Explicit mutation invalidates
the current generation without canceling existing waiters.
