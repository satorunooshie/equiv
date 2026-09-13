package concurrent

import (
	"hash/maphash"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/satorunooshie/equiv"
)

type countingHasher struct{ calls atomic.Int64 }

func (h *countingHasher) Hash(out *maphash.Hash, v int) {
	h.calls.Add(1)
	var b [8]byte
	for i := range b {
		b[i] = byte(uint64(v) >> uint(8*i))
	}
	out.Write(b[:])
}
func (*countingHasher) Equal(a, b int) bool { return a == b }

func TestOperationsHashOnceBeforeShardRouting(t *testing.T) {
	h := &countingHasher{}
	m, e := NewMap[int, int](h, WithShards(4))
	if e != nil {
		t.Fatal(e)
	}
	m.Set(1, 1)
	if _, ok := m.Get(1); !ok {
		t.Fatal("stored value missing")
	}
	if got := h.calls.Load(); got != 2 {
		t.Fatalf("Hasher.Hash calls = %d, want 2", got)
	}
}

func TestConcurrentIteratorMayMutateContainer(t *testing.T) {
	m, e := NewMap[int, int](maphash.ComparableHasher[int]{}, WithShards(4))
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		m.Set(i, i)
	}
	seen := 0
	for k := range m.Keys() {
		seen++
		m.Delete(k)
	}
	if seen == 0 || m.Len() != 0 {
		t.Fatalf("seen=%d len=%d", seen, m.Len())
	}
}

func TestConcurrentSnapshotIsIndependent(t *testing.T) {
	m, _ := NewMap[int, int](maphash.ComparableHasher[int]{}, WithShards(2))
	m.Set(1, 1)
	s := m.Snapshot()
	m.Set(1, 2)
	if v, _ := s.Get(1); v != 1 {
		t.Fatal("snapshot changed")
	}
	var _ *equiv.Map[int, int] = s
}

func TestConcurrentMutations(t *testing.T) {
	m, _ := NewMap[int, int](maphash.ComparableHasher[int]{}, WithShards(8))
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				k := (i + g) % 64
				m.Set(k, i)
				m.Get(k)
				if i%3 == 0 {
					m.Delete(k)
				}
			}
		}(g)
	}
	wg.Wait()
	if m.Len() < 0 {
		t.Fatal("negative length")
	}
}
