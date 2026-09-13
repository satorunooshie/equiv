package cache

import (
	"hash/maphash"
	"math"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeClock) Now() time.Time          { f.mu.Lock(); defer f.mu.Unlock(); return f.t }
func (f *fakeClock) Advance(d time.Duration) { f.mu.Lock(); f.t = f.t.Add(d); f.mu.Unlock() }

func TestPolicyAndExpiration(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: LRU, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Set(3, 3)
	if c.Contains(2) {
		t.Fatal("LRU did not evict least recently used entry")
	}
	c.SetTTL(4, 4, time.Millisecond)
	time.Sleep(3 * time.Millisecond)
	if c.Contains(4) {
		t.Fatal("expired entry remained visible")
	}
}

func TestInvalidConfig(t *testing.T) {
	if _, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{}); e == nil {
		t.Fatal("expected invalid capacity")
	}
}

func TestNewRejectsNilHasherImmediately(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil Hasher panic")
		}
	}()
	_, _ = New[int, int](nil, Config[int, int]{MaxEntries: 1, Shards: 1})
}

func TestObserverMayReenter(t *testing.T) {
	var c *Cache[int, int]
	var e error
	c, e = New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 2, Observer: func(Event[int, int]) {
		if c != nil {
			c.Len()
		}
	}})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
}

func TestPerEntryTTISurvivesTouch(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.SetExpiration(1, 1, 0, 30*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if !c.Touch(1) {
		t.Fatal("Touch failed")
	}
	time.Sleep(10 * time.Millisecond)
	if !c.Contains(1) {
		t.Fatal("Touch shortened per-entry TTI")
	}
}

func TestPeekDoesNotRefreshRecencyOrTTI(t *testing.T) {
	c, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		Policy: LRU, MaxEntries: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	if _, ok := c.Peek(1); !ok {
		t.Fatal("Peek missed resident entry")
	}
	c.Set(3, 3)
	if c.Contains(1) || !c.Contains(2) {
		t.Fatal("Peek changed LRU recency")
	}

	f := &fakeClock{t: time.Unix(200, 0)}
	tti, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 2})
	if err != nil {
		t.Fatal(err)
	}
	tti.clock = f
	tti.SetExpiration(1, 1, 0, 10*time.Second)
	f.Advance(9 * time.Second)
	if _, ok := tti.Peek(1); !ok {
		t.Fatal("Peek unexpectedly expired entry")
	}
	f.Advance(2 * time.Second)
	if _, ok := tti.Peek(1); ok {
		t.Fatal("Peek refreshed TTI")
	}
}

func TestSLRUHitProtectsEntry(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: SLRU, MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	for i := 1; i <= 4; i++ {
		c.Set(i, i)
	}
	c.Get(1)
	c.Set(5, 5)
	if c.Contains(1) == false {
		t.Fatal("protected entry was evicted")
	}
}

func TestWTinyLFURejectsColdCandidate(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: WTinyLFU, MaxEntries: 1})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Get(1)
	if c.Set(2, 2) {
		t.Fatal("cold candidate was admitted")
	}
	if !c.Contains(1) {
		t.Fatal("hot entry was evicted")
	}
}

func TestARCConfigurationAndTrace(t *testing.T) {
	if _, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: ARC, MaxWeight: 10, Weigher: func(int, int) uint64 { return 1 }}); e == nil {
		t.Fatal("weighted ARC must be rejected")
	}
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: ARC, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Set(3, 3)
	if !c.Contains(1) {
		t.Fatal("ARC evicted frequently used entry")
	}
}

func TestTwoQPromotesReReferencedEntry(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: TwoQ, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Set(3, 3)
	if !c.Contains(1) {
		t.Fatal("2Q evicted re-referenced entry")
	}
}

func TestWeigherRunsOutsideCacheLock(t *testing.T) {
	var c *Cache[int, int]
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxWeight: 4, Weigher: func(k, v int) uint64 {
		if c != nil {
			c.Len()
		}
		return 1
	}})
	if e != nil {
		t.Fatal(e)
	}
	if !c.Set(1, 1) {
		t.Fatal("Set rejected")
	}
}

func TestFakeClockExpirationIsDeterministic(t *testing.T) {
	f := &fakeClock{t: time.Unix(100, 0)}
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	c.clock = f
	c.SetTTL(1, 1, 10*time.Second)
	f.Advance(9 * time.Second)
	if !c.Contains(1) {
		t.Fatal("entry expired too early")
	}
	f.Advance(2 * time.Second)
	if c.Contains(1) {
		t.Fatal("entry did not expire")
	}
	c.SetUntil(2, 2, f.t.Add(time.Hour))
	f.Advance(2 * time.Hour)
	if c.Contains(2) {
		t.Fatal("absolute expiry failed")
	}
}

func TestPastAbsoluteExpirationIsImmediate(t *testing.T) {
	f := &fakeClock{t: time.Unix(100, 0)}
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	c.clock = f
	if !c.SetUntil(1, 1, f.t.Add(-time.Second)) {
		t.Fatal("SetUntil rejected")
	}
	if _, ok := c.Get(1); ok {
		t.Fatal("past deadline remained resident")
	}
}

func TestNegativeTTLIsImmediate(t *testing.T) {
	f := &fakeClock{t: time.Unix(100, 0)}
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	c.clock = f
	c.SetTTL(1, 1, -time.Second)
	if _, ok := c.Get(1); ok {
		t.Fatal("negative TTL remained resident")
	}
}

func TestLFUAgesFrequencies(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: LFU, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	for i := 0; i < 4096; i++ {
		c.Get(1)
	}
	x, _ := c.m.Get(1)
	if x.freq >= 4096 {
		t.Fatalf("frequency was not aged: %d", x.freq)
	}
}

func TestLFUTieBreaksByRecency(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: LFU, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Get(2)
	c.Set(3, 3)
	if c.Contains(1) {
		t.Fatal("LFU tie did not evict oldest entry")
	}
}

func TestMRUGetUpdatesRecency(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: MRU, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Set(3, 3)
	if c.Contains(1) {
		t.Fatal("MRU did not evict the most recently used entry")
	}
	if !c.Contains(2) {
		t.Fatal("MRU evicted the older entry")
	}
}

func TestFIFOAndLRUTraces(t *testing.T) {
	t.Run("FIFO", func(t *testing.T) {
		c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: FIFO, MaxEntries: 2})
		if e != nil {
			t.Fatal(e)
		}
		c.Set(1, 1)
		c.Set(2, 2)
		c.Get(1)
		c.Set(3, 3)
		if c.Contains(1) || !c.Contains(2) {
			t.Fatal("FIFO read changed insertion order")
		}
	})
	t.Run("LRU", func(t *testing.T) {
		c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: LRU, MaxEntries: 2})
		if e != nil {
			t.Fatal(e)
		}
		c.Set(1, 1)
		c.Set(2, 2)
		c.Get(1)
		c.Set(3, 3)
		if !c.Contains(1) || c.Contains(2) {
			t.Fatal("LRU did not retain the recently used entry")
		}
	})
}

func TestClockSecondChanceTrace(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: Clock, MaxEntries: 2})
	if e != nil {
		t.Fatal(e)
	}
	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Set(3, 3)
	if c.Len() > 2 || !c.Contains(1) {
		t.Fatal("CLOCK failed to give referenced entry a second chance")
	}
}

func TestWTinyLFUHasWindowAndMainSegments(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{Policy: WTinyLFU, MaxEntries: 100})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		c.Set(i, i)
	}
	window := 0
	main := 0
	for k, x := range c.m.All() {
		_ = k
		if x.segment == 0 {
			window++
		} else {
			main++
		}
	}
	if window == 0 || main == 0 {
		t.Fatalf("window=%d main=%d", window, main)
	}
}

func TestWeightedSegmentPoliciesUseWeightBudgets(t *testing.T) {
	weigher := func(_ int, v int) uint64 { return uint64(v) }
	s, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		Policy: SLRU, MaxWeight: 10, Weigher: weigher,
	})
	if e != nil {
		t.Fatal(e)
	}
	for i := 1; i <= 5; i++ {
		s.Set(i, 2)
	}
	if s.Weight() > 10 {
		t.Fatalf("SLRU exceeded weight budget: %d", s.Weight())
	}
	w, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		Policy: WTinyLFU, MaxWeight: 100, Weigher: weigher,
	})
	if e != nil {
		t.Fatal(e)
	}
	for i := 1; i <= 100; i++ {
		w.Set(i, 1)
	}
	if w.Weight() > 100 {
		t.Fatalf("W-TinyLFU exceeded weight budget: %d", w.Weight())
	}
}

func TestOversizedWeightEmitsRejection(t *testing.T) {
	events := make(chan Event[int, int], 1)
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		MaxWeight: 2,
		Weigher:   func(_ int, v int) uint64 { return uint64(v) },
		Observer:  func(ev Event[int, int]) { events <- ev },
	})
	if e != nil {
		t.Fatal(e)
	}
	if c.Set(1, 3) {
		t.Fatal("oversized value was accepted")
	}
	if got := c.Stats().Rejections; got != 1 {
		t.Fatalf("rejections=%d, want 1", got)
	}
	ev := <-events
	if ev.Type != EventReject || ev.Key != 1 || ev.Weight != 3 {
		t.Fatalf("unexpected rejection event: %+v", ev)
	}
}

func TestWeightSumDoesNotOverflowAtUint64Limit(t *testing.T) {
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		MaxWeight: math.MaxUint64,
		Weigher:   func(int, int) uint64 { return math.MaxUint64 },
	})
	if e != nil {
		t.Fatal(e)
	}
	if !c.Set(1, 1) || !c.Set(2, 2) {
		t.Fatal("max-weight entries were not accepted")
	}
	if c.Weight() != math.MaxUint64 || c.Len() != 1 {
		t.Fatalf("weight=%d len=%d, want one resident at max weight", c.Weight(), c.Len())
	}
}

func TestWeightedUpdateEvictsOtherEntriesBeforeUpdatedEntry(t *testing.T) {
	c, err := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		MaxWeight: 10,
		Weigher:   func(_ int, v int) uint64 { return uint64(v) },
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Set(1, 6)
	c.Set(2, 4)
	if !c.Set(1, 10) {
		t.Fatal("weighted update rejected")
	}
	if v, ok := c.Get(1); !ok || v != 10 {
		t.Fatalf("updated entry=(%d,%v)", v, ok)
	}
	if c.Contains(2) || c.Weight() != 10 || c.Len() != 1 {
		t.Fatalf("entries after update: len=%d weight=%d contains2=%v", c.Len(), c.Weight(), c.Contains(2))
	}
}
