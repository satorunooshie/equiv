package concurrent

import (
	"hash/maphash"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	for i := range 100 {
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

type snapshotBlockingHasher struct {
	block     atomic.Bool
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
	blockOnce sync.Once
}

func (h *snapshotBlockingHasher) Hash(out *maphash.Hash, v int) {
	if h.block.Load() {
		h.blockOnce.Do(func() {
			h.startOnce.Do(func() { close(h.started) })
			<-h.release
		})
	}
	var b [8]byte
	for i := range b {
		b[i] = byte(uint64(v) >> uint(8*i))
	}
	out.Write(b[:])
}

func (*snapshotBlockingHasher) Equal(a, b int) bool { return a == b }

func TestSnapshotExcludesConcurrentWriters(t *testing.T) {
	h := &snapshotBlockingHasher{started: make(chan struct{}), release: make(chan struct{})}
	m, err := NewMap[int, int](h, WithShards(2))
	if err != nil {
		t.Fatal(err)
	}
	m.Set(1, 1)
	h.block.Store(true)

	done := make(chan struct{})
	go func() {
		m.Snapshot()
		close(done)
	}()
	<-h.started

	writerDone := make(chan struct{})
	go func() {
		m.Set(2, 2)
		close(writerDone)
	}()
	select {
	case <-writerDone:
		t.Fatal("writer was not excluded during snapshot")
	case <-time.After(10 * time.Millisecond):
	}
	close(h.release)
	<-done
	<-writerDone
}

func TestConcurrentMutations(t *testing.T) {
	m, _ := NewMap[int, int](maphash.ComparableHasher[int]{}, WithShards(8))
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range 1000 {
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

func TestMapReadHelpersAndClear(t *testing.T) {
	m, err := NewMap[string, int](maphash.ComparableHasher[string]{}, WithShards(2))
	if err != nil {
		t.Fatal(err)
	}
	if v, existed := m.GetOrSet("a", 1); existed || v != 1 {
		t.Fatalf("first GetOrSet=(%d,%v)", v, existed)
	}
	if v, existed := m.GetOrSet("a", 2); !existed || v != 1 {
		t.Fatalf("second GetOrSet=(%d,%v)", v, existed)
	}
	if k, v, ok := m.GetEntry("a"); !ok || k != "a" || v != 1 {
		t.Fatalf("GetEntry=(%q,%d,%v)", k, v, ok)
	}
	var values []int
	for v := range m.Values() {
		values = append(values, v)
	}
	if len(values) != 1 || values[0] != 1 {
		t.Fatalf("values=%v", values)
	}
	m.Clear()
	if m.Len() != 0 {
		t.Fatal("clear did not empty map")
	}
}

func TestConcurrentSetLifecycle(t *testing.T) {
	s, err := NewSet[string](maphash.ComparableHasher[string]{}, WithShards(2))
	if err != nil {
		t.Fatal(err)
	}
	if !s.Insert("a") || s.Insert("a") || !s.Contains("a") {
		t.Fatal("set insert/contains failed")
	}
	if v, ok := s.Lookup("a"); !ok || v != "a" {
		t.Fatalf("lookup=(%q,%v)", v, ok)
	}
	clone := s.Snapshot()
	s.Delete("a")
	if !clone.Contains("a") || s.Contains("a") {
		t.Fatal("set snapshot or delete failed")
	}
	s.Clear()
	if s.Len() != 0 {
		t.Fatal("set clear failed")
	}
}

func TestInvalidShardOptions(t *testing.T) {
	h := maphash.ComparableHasher[int]{}
	for _, n := range []int{0, 3, -2} {
		if _, err := NewMap[int, int](h, WithShards(n)); err == nil {
			t.Fatalf("WithShards(%d) was accepted", n)
		}
		if _, err := NewSet[int](h, WithShards(n)); err == nil {
			t.Fatalf("set WithShards(%d) was accepted", n)
		}
	}
}

func TestConcurrentMapHandlesFullHashCollisions(t *testing.T) {
	m, err := NewMap[int, int](constantHasher{}, WithShards(4))
	if err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		m.Set(i, i*i)
	}
	if m.Len() != 100 {
		t.Fatalf("Len=%d, want 100", m.Len())
	}
	for i := range 100 {
		if v, ok := m.Get(i); !ok || v != i*i {
			t.Fatalf("Get(%d)=(%d,%v)", i, v, ok)
		}
	}
	for i := 0; i < 100; i += 2 {
		if !m.Delete(i) {
			t.Fatalf("Delete(%d) failed", i)
		}
	}
	if _, ok := m.Get(0); m.Len() != 50 || ok {
		t.Fatalf("collision delete corrupted map: len=%d", m.Len())
	}
	if _, ok := m.Get(1); !ok {
		t.Fatal("collision delete removed unrelated entry")
	}
}

type constantHasher struct{}

func (constantHasher) Hash(h *maphash.Hash, _ int) { h.WriteByte(1) }
func (constantHasher) Equal(a, b int) bool         { return a == b }
