// Package table is the exact, scalar open-addressed table backend.
package table

import (
	"hash/maphash"
	"iter"
	"sync/atomic"
)

const (
	empty    uint8 = 0
	deleted  uint8 = 1
	occupied uint8 = 2
)

type Entry[K, V any] struct {
	Hash  uint64
	Key   K
	Value V
}
type Table[K, V any] struct {
	h           maphash.Hasher[K]
	seed        maphash.Seed
	scratch     maphash.Hash
	slots       []Entry[K, V]
	ctrl        []uint8
	live, tombs int
	probes      atomic.Uint64
	equalCalls  atomic.Uint64
	resizes     atomic.Uint64
	rebuilds    atomic.Uint64
}

func New[K, V any](h maphash.Hasher[K], capacity int) *Table[K, V] {
	if h == nil {
		panic("table: nil Hasher")
	}
	return NewWithSeed[K, V](h, capacity, maphash.MakeSeed())
}

// NewWithSeed constructs a table in an existing hash domain. It is used by
// sharded containers that must reuse the full hash computed before routing.
func NewWithSeed[K, V any](h maphash.Hasher[K], capacity int, seed maphash.Seed) *Table[K, V] {
	if h == nil {
		panic("table: nil Hasher")
	}
	t := &Table[K, V]{h: h, seed: seed}
	t.scratch.SetSeed(seed)
	t.init(capacity)
	return t
}

func (t *Table[K, V]) init(n int) {
	if n < 0 {
		panic("table: negative capacity")
	}
	if n > int(^uint(0)>>1)/2 {
		panic("table: capacity too large")
	}
	c := 8
	for c < n {
		c *= 2
	}
	t.slots = make([]Entry[K, V], c)
	t.ctrl = make([]uint8, c)
	t.live = 0
	t.tombs = 0
}

func (t *Table[K, V]) hash(k K) uint64 {
	t.scratch.Reset()
	t.h.Hash(&t.scratch, k)
	return t.scratch.Sum64()
}

// Hash returns the full hash in this table's immutable seed domain.
func (t *Table[K, V]) Hash(k K) uint64 { return t.hash(k) }
func fingerprint(h uint64) uint8       { return occupied | uint8((h^(h>>32))&0x7f) }
func (t *Table[K, V]) find(k K, d uint64) (int, int) {
	mask := len(t.ctrl) - 1
	free := -1
	for n := 0; n < len(t.ctrl); n++ {
		t.probes.Add(1)
		i := int(d+uint64(n)) & mask
		switch c := t.ctrl[i]; {
		case c == empty:
			return -1, choose(free, i)
		case c == deleted:
			if free < 0 {
				free = i
			}
		case c == fingerprint(d) && t.slots[i].Hash == d:
			t.equalCalls.Add(1)
			if t.h.Equal(t.slots[i].Key, k) {
				return i, free
			}
		}
	}
	return -1, free
}

func choose(a, b int) int {
	if a >= 0 {
		return a
	}
	return b
}

func (t *Table[K, V]) rehash(c int) {
	old, ctrl := t.slots, t.ctrl
	if c > len(t.ctrl) {
		t.resizes.Add(1)
	} else {
		t.rebuilds.Add(1)
	}
	t.slots = make([]Entry[K, V], c)
	t.ctrl = make([]uint8, c)
	t.live = 0
	t.tombs = 0
	for i, x := range old {
		if ctrl[i] >= occupied {
			t.place(x)
		}
	}
}

func (t *Table[K, V]) place(x Entry[K, V]) {
	for n := 0; n < len(t.ctrl); n++ {
		i := int(x.Hash+uint64(n)) & (len(t.ctrl) - 1)
		if t.ctrl[i] == empty {
			t.ctrl[i] = fingerprint(x.Hash)
			t.slots[i] = x
			t.live++
			return
		}
	}
	panic("table: full")
}

func (t *Table[K, V]) Get(k K) (V, bool) {
	return t.GetHashed(k, t.hash(k))
}

func (t *Table[K, V]) GetHashed(k K, d uint64) (V, bool) {
	i, _ := t.find(k, d)
	if i < 0 {
		var z V
		return z, false
	}
	return t.slots[i].Value, true
}

func (t *Table[K, V]) GetEntry(k K) (K, V, bool) {
	return t.GetEntryHashed(k, t.hash(k))
}

func (t *Table[K, V]) GetEntryHashed(k K, d uint64) (K, V, bool) {
	i, _ := t.find(k, d)
	if i < 0 {
		var z K
		var v V
		return z, v, false
	}
	x := t.slots[i]
	return x.Key, x.Value, true
}

func (t *Table[K, V]) Set(k K, v V) (V, bool) {
	return t.SetHashed(k, v, t.hash(k))
}

func (t *Table[K, V]) SetHashed(k K, v V, d uint64) (V, bool) {
	i, free := t.find(k, d)
	if i >= 0 {
		old := t.slots[i].Value
		t.slots[i].Value = v
		return old, true
	}
	if t.live+t.tombs+1 > len(t.ctrl)*7/8 {
		t.rehash(len(t.ctrl) * 2)
		_, free = t.find(k, d)
	}
	if free < 0 {
		t.rehash(len(t.ctrl) * 2)
		_, free = t.find(k, d)
	}
	if t.ctrl[free] == deleted {
		t.tombs--
	}
	t.ctrl[free] = fingerprint(d)
	t.slots[free] = Entry[K, V]{d, k, v}
	t.live++
	return *new(V), false
}

func (t *Table[K, V]) Delete(k K) bool {
	return t.DeleteHashed(k, t.hash(k))
}

func (t *Table[K, V]) DeleteHashed(k K, d uint64) bool {
	i, _ := t.find(k, d)
	if i < 0 {
		return false
	}
	t.slots[i] = Entry[K, V]{}
	t.ctrl[i] = deleted
	t.live--
	t.tombs++
	if t.tombs > len(t.ctrl)/4 {
		t.rehash(len(t.ctrl))
	}
	return true
}
func (t *Table[K, V]) Clear()   { clear(t.slots); clear(t.ctrl); t.live = 0; t.tombs = 0 }
func (t *Table[K, V]) Len() int { return t.live }

type metrics struct {
	probes, equalCalls, resizes, rebuilds uint64
}

func (t *Table[K, V]) resetMetrics() {
	t.probes.Store(0)
	t.equalCalls.Store(0)
	t.resizes.Store(0)
	t.rebuilds.Store(0)
}

func (t *Table[K, V]) metrics() metrics {
	return metrics{t.probes.Load(), t.equalCalls.Load(), t.resizes.Load(), t.rebuilds.Load()}
}

func (t *Table[K, V]) All() iter.Seq2[K, V] {
	return func(y func(K, V) bool) {
		for i, c := range t.ctrl {
			if c >= occupied {
				e := t.slots[i]
				if !y(e.Key, e.Value) {
					return
				}
			}
		}
	}
}
