package cache

import (
	"context"
	"hash/maphash"
	"time"
)

// SyncCache is a mutex-protected cache with coalesced concurrent loads.
type SyncCache[K, V any] struct{ *Cache[K, V] }

// NewSync constructs a synchronized cache with coalesced concurrent loads.
func NewSync[K, V any](h maphash.Hasher[K], c Config[K, V]) (*SyncCache[K, V], error) {
	x, e := New(h, c)
	return &SyncCache[K, V]{x}, e
}

// GetOrLoad loads a missing value, sharing one in-flight load per key.
func (s *SyncCache[K, V]) GetOrLoad(ctx context.Context, k K, l Loader[K, V]) (V, error) {
	return s.Cache.getOrLoadCoalesced(ctx, k, l)
}

// RunJanitor periodically removes expired entries until ctx is canceled.
func (s *SyncCache[K, V]) RunJanitor(ctx context.Context, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.PruneExpired()
		}
	}
}
