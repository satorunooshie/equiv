package cache

import (
	"hash/maphash"
	"testing"
)

func TestShardedPartitionsCapacity(t *testing.T) {
	c, e := NewSharded[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4, Shards: 2})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 20; i++ {
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
