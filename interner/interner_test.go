package interner

import (
	"hash/maphash"
	"runtime"
	"testing"
	"time"

	"github.com/satorunooshie/equiv/hashers"
)

func TestStrongReturnsCanonicalValueAndSupportsLifecycle(t *testing.T) {
	i := NewStrong[string](maphash.ComparableHasher[string]{})
	a := i.Intern(string([]byte("shared")))
	b := i.Intern(string([]byte("shared")))
	if a != b || i.Len() != 1 {
		t.Fatalf("canonical=(%q,%q), len=%d", a, b, i.Len())
	}
	if v, ok := i.Lookup("shared"); !ok || v != a {
		t.Fatalf("lookup=(%q,%v), want (%q,true)", v, ok, a)
	}
	if !i.Delete("shared") || i.Len() != 0 {
		t.Fatal("delete did not remove canonical value")
	}
	if _, ok := i.Lookup("shared"); ok {
		t.Fatal("deleted value is still interned")
	}
	i.Intern("one")
	i.Intern("two")
	seen := 0
	for range i.All() {
		seen++
	}
	if seen != 2 {
		t.Fatalf("All visited %d values", seen)
	}
	i.Clear()
	if i.Len() != 0 {
		t.Fatal("clear did not empty strong interner")
	}
}

func TestWeakReturnsLiveCanonicalPointer(t *testing.T) {
	i := NewWeak[string](hashers.Deref(maphash.ComparableHasher[string]{}))
	first := "shared"
	canonical := i.Intern(&first)
	probe := "shared"
	found, ok := i.Lookup(&probe)
	if !ok || found != canonical || i.LenApprox() != 1 {
		t.Fatalf("lookup=(%p,%v), canonical=%p len=%d", found, ok, canonical, i.LenApprox())
	}
	if removed := i.Sweep(); removed != 0 {
		t.Fatalf("sweep removed a live value: %d", removed)
	}
	seen := 0
	for p := range i.All() {
		if p == canonical {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("weak All visited %d canonical pointers", seen)
	}
	second := "other"
	i.Intern(&second)
	if _, ok := i.Lookup(&probe); !ok {
		t.Fatal("existing weak entry disappeared after another intern")
	}
	missing := "missing"
	if _, ok := i.Lookup(&missing); ok {
		t.Fatal("missing weak value was found")
	}
}

func TestWeakRejectsNilPointers(t *testing.T) {
	i := NewWeak[string](hashers.Deref(maphash.ComparableHasher[string]{}))
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil pointer panic")
		}
	}()
	i.Intern(nil)
}

func TestWeakLookupRejectsNilPointers(t *testing.T) {
	i := NewWeak[string](hashers.Deref(maphash.ComparableHasher[string]{}))
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil pointer panic")
		}
	}()
	i.Lookup(nil)
}

func TestWeakSweepRemovesCollectedEntries(t *testing.T) {
	i := NewWeak[string](hashers.Deref(maphash.ComparableHasher[string]{}))
	ready := make(chan struct{})
	func() {
		value := "collect-me"
		runtime.SetFinalizer(&value, func(*string) { close(ready) })
		i.Intern(&value)
	}()
	for range 20 {
		runtime.GC()
		select {
		case <-ready:
			if i.Sweep() != 1 || i.LenApprox() != 0 {
				t.Fatal("collected weak entry was not swept")
			}
			return
		case <-time.After(time.Millisecond):
		}
	}
	t.Skip("runtime did not collect weak entry during test window")
}
