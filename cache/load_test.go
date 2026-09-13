package cache

import (
	"context"
	"errors"
	"hash/maphash"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadInvalidatedBySet(t *testing.T) {
	c, e := NewSync[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	done := make(chan struct{})
	var loaded int
	go func() {
		loaded, _ = c.GetOrLoad(context.Background(), 1, func(context.Context, int) (int, error) { calls.Add(1); close(started); <-release; return 10, nil })
		close(done)
	}()
	<-started
	c.Set(1, 99)
	close(release)
	<-done
	if v, _ := c.Get(1); v != 99 {
		t.Fatalf("stale load overwrote explicit value: %d", v)
	}
	if loaded != 99 {
		t.Fatalf("old generation returned stale value: %d", loaded)
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected loader calls")
	}
	time.Sleep(time.Millisecond)
}

func TestLoadInvalidatedByClearStartsFreshGeneration(t *testing.T) {
	c, err := NewSync[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if err != nil {
		t.Fatal(err)
	}
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})
	first := make(chan int, 1)
	second := make(chan int, 1)
	var calls atomic.Int32
	loader := func(ctx context.Context, _ int) (int, error) {
		n := calls.Add(1)
		if n == 1 {
			close(firstStarted)
			<-releaseFirst
			return 10, nil
		}
		close(secondStarted)
		<-releaseSecond
		return 20, nil
	}
	go func() {
		v, _ := c.GetOrLoad(context.Background(), 1, loader)
		first <- v
	}()
	<-firstStarted
	c.Clear()
	go func() {
		v, _ := c.GetOrLoad(context.Background(), 1, loader)
		second <- v
	}()
	<-secondStarted
	close(releaseFirst)
	if got := <-first; got != 10 {
		t.Fatalf("first generation returned %d", got)
	}
	if _, ok := c.Get(1); ok {
		t.Fatal("invalidated generation became resident")
	}
	close(releaseSecond)
	if got := <-second; got != 20 {
		t.Fatalf("second generation returned %d", got)
	}
	if got, ok := c.Get(1); !ok || got != 20 {
		t.Fatalf("fresh generation not resident: %d, %v", got, ok)
	}
	if calls.Load() != 2 {
		t.Fatalf("loader calls=%d, want 2", calls.Load())
	}
}

type collisionLoadHasher struct{}

func (collisionLoadHasher) Hash(h *maphash.Hash, _ int) { h.WriteByte(1) }
func (collisionLoadHasher) Equal(a, b int) bool         { return a == b }
func TestLoadCoalescingChecksEqualAfterHash(t *testing.T) {
	c, e := NewSync[int, int](collisionLoadHasher{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	other := make(chan struct{})
	var calls atomic.Int32
	go func() {
		c.GetOrLoad(context.Background(), 1, func(context.Context, int) (int, error) { calls.Add(1); close(started); <-release; return 1, nil })
	}()
	<-started
	go func() {
		c.GetOrLoad(context.Background(), 2, func(context.Context, int) (int, error) { calls.Add(1); close(other); return 2, nil })
	}()
	<-other
	close(release)
	if calls.Load() != 2 {
		t.Fatalf("collision incorrectly coalesced: %d", calls.Load())
	}
}

func TestLoaderWeigherRunsOutsideCacheLock(t *testing.T) {
	var c *Cache[int, int]
	c, e := New[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		MaxWeight: 4,
		Weigher: func(k, v int) uint64 {
			if c != nil {
				c.Len()
			}
			return 1
		},
	})
	if e != nil {
		t.Fatal(e)
	}
	v, e := c.GetOrLoad(context.Background(), 1, func(context.Context, int) (int, error) {
		return 42, nil
	})
	if e != nil || v != 42 {
		t.Fatalf("load = %d, %v", v, e)
	}
	if got, ok := c.Get(1); !ok || got != 42 {
		t.Fatalf("loaded value missing: %d, %v", got, ok)
	}
}

func TestFirstWaiterCancellationDoesNotCancelSharedLoad(t *testing.T) {
	c, e := NewSync[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{MaxEntries: 4})
	if e != nil {
		t.Fatal(e)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	loader := func(ctx context.Context, _ int) (int, error) {
		close(started)
		select {
		case <-release:
			return 7, nil
		case <-ctx.Done():
			t.Fatalf("shared loader was canceled: %v", ctx.Err())
			return 0, ctx.Err()
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		_, err := c.GetOrLoad(ctx, 1, loader)
		first <- err
	}()
	<-started
	cancel()
	if err := <-first; err == nil {
		t.Fatal("first waiter did not observe cancellation")
	}
	second := make(chan int, 1)
	go func() {
		v, err := c.GetOrLoad(context.Background(), 1, loader)
		if err != nil {
			t.Errorf("second waiter: %v", err)
		}
		second <- v
	}()
	deadline := time.After(time.Second)
	for c.Stats().LoadCoalesced == 0 {
		select {
		case <-deadline:
			t.Fatal("second waiter did not coalesce")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(release)
	if v := <-second; v != 7 {
		t.Fatalf("second waiter got %d", v)
	}
}

func TestSharedLoadTimeoutBoundsLoaderContext(t *testing.T) {
	c, err := NewSync[int, int](maphash.ComparableHasher[int]{}, Config[int, int]{
		MaxEntries:  2,
		LoadTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetOrLoad(context.Background(), 1, func(ctx context.Context, _ int) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("load error=%v, want deadline exceeded", err)
	}
}
