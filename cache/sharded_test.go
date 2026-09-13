package cache

import (
	"context"
	"hash/maphash"
	"testing"
	"time"
)

func TestShardedPartitionsCapacity(t *testing.T) {
	c, e := NewSharded[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4, Shards: 2})
	if e != nil {
		t.Fatal(e)
	}
	for i := range 20 {
		c.Set(i, i)
	}
	if c.Len() > 4 {
		t.Fatalf("capacity exceeded: %d", c.Len())
	}
	if _, e := NewSharded[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 2, Shards: 3}); e == nil {
		t.Fatal("expected invalid explicit shard count")
	}
}

func TestShardedEffectiveShardCounts(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config[int, int]
		want int
	}{
		{name: "minimum entry capacity", cfg: Config[int, int]{MaxEntries: 1}, want: 1},
		{name: "entry capacity", cfg: Config[int, int]{MaxEntries: 32}, want: 32},
		{name: "minimum weight capacity", cfg: Config[int, int]{MaxWeight: 1, Weigher: func(int, int) uint64 { return 1 }}, want: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewSharded[int, int](maphash.ComparableHasher[int]{}, tc.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(c.shards); got != tc.want {
				t.Fatalf("effective shard count = %d, want %d", got, tc.want)
			}
		})
	}
	if _, err := NewSharded[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4, Shards: 64}); err == nil {
		t.Fatal("expected shard count greater than capacity to be rejected")
	}
}

func TestShardedCacheDelegatesOperationsAndStats(t *testing.T) {
	c, err := NewSharded[string, string](maphash.ComparableHasher[string]{}, Config[string, string]{MaxEntries: 8, Shards: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Set("a", "one") || !c.Contains("a") {
		t.Fatal("set/contains failed")
	}
	if v, ok := c.Get("a"); !ok || v != "one" {
		t.Fatalf("get=(%q,%v)", v, ok)
	}
	if v, ok := c.Peek("a"); !ok || v != "one" {
		t.Fatalf("peek=(%q,%v)", v, ok)
	}
	if !c.Touch("a") || c.Weight() != 1 || c.Len() != 1 {
		t.Fatalf("touch/weight/len failed: weight=%d len=%d", c.Weight(), c.Len())
	}
	if !c.SetTTL("ttl", "value", time.Hour) || !c.SetExpiration("tti", "value", time.Hour, time.Hour) || !c.SetUntil("until", "value", time.Now().Add(time.Hour)) {
		t.Fatal("expiration setters failed")
	}
	var seen int
	for range c.All() {
		seen++
	}
	if seen != c.Len() {
		t.Fatalf("All visited %d entries, len=%d", seen, c.Len())
	}
	if !c.Delete("a") {
		t.Fatal("delete failed")
	}
	if c.Stats().Sets == 0 {
		t.Fatal("stats did not aggregate sets")
	}
	c.Clear()
	if c.Len() != 0 || c.PruneExpired() != 0 {
		t.Fatal("clear/prune failed")
	}
}

func TestShardedCacheLoadsAndExpires(t *testing.T) {
	c, err := NewSharded[string, string](maphash.ComparableHasher[string]{}, Config[string, string]{
		MaxEntries: 2, Shards: 2, DefaultTTL: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	loader := func(context.Context, string) (string, error) {
		loads++
		return "loaded", nil
	}
	if v, err := c.GetOrLoad(context.Background(), "key", loader); err != nil || v != "loaded" {
		t.Fatalf("first load=(%q,%v)", v, err)
	}
	if _, err := c.GetOrLoad(context.Background(), "key", loader); err != nil || loads != 1 {
		t.Fatalf("cache hit did not reuse value: loads=%d", loads)
	}
	time.Sleep(3 * time.Millisecond)
	if _, err := c.GetOrLoad(context.Background(), "key", loader); err != nil || loads != 2 {
		t.Fatalf("expired value was not reloaded: loads=%d", loads)
	}
}
