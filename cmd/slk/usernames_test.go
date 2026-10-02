package main

import (
	"fmt"
	"sync"
	"testing"
)

// TestUserNameStore_SnapshotIsIndependent pins the property the
// 2026-10-02 crash fix rests on: the map handed to the UI is a copy,
// not the store's backing map. If Snapshot ever returns the live map
// again, background writes become UI-goroutine races again.
func TestUserNameStore_SnapshotIsIndependent(t *testing.T) {
	s := newUserNameStore(map[string]string{"U1": "Alice"})

	snap := s.Snapshot()
	s.Set("U2", "Bob")
	if _, ok := snap["U2"]; ok {
		t.Fatal("a Set after Snapshot leaked into the snapshot: the UI and the store share a map")
	}

	snap["U3"] = "Carol"
	if _, ok := s.Get("U3"); ok {
		t.Fatal("a write to the snapshot leaked into the store: the UI and the store share a map")
	}

	if name, ok := s.Get("U1"); !ok || name != "Alice" {
		t.Errorf("Get(U1) = (%q, %v), want (\"Alice\", true)", name, ok)
	}
}

// TestUserNameStore_ConcurrentAccess runs Set, Get and Snapshot from
// many goroutines at once. Under -race a store without its lock fails
// here; without -race Go's map checks usually do.
func TestUserNameStore_ConcurrentAccess(t *testing.T) {
	s := newUserNameStore(nil)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				id := fmt.Sprintf("U%d_%d", g, i)
				s.Set(id, "name")
				_, _ = s.Get(id)
				_ = s.Snapshot()
			}
		}(g)
	}
	wg.Wait()
	if got := len(s.Snapshot()); got != 8*200 {
		t.Errorf("len(Snapshot()) = %d, want %d", got, 8*200)
	}
}

// TestUserNameStore_NilIsEmpty: several history helpers document that
// their name source may be absent. A nil store reads as empty and
// ignores writes instead of panicking.
func TestUserNameStore_NilIsEmpty(t *testing.T) {
	var s *userNameStore
	if _, ok := s.Get("U1"); ok {
		t.Error("nil store Get returned ok")
	}
	s.Set("U1", "Alice") // must not panic
	if snap := s.Snapshot(); snap == nil || len(snap) != 0 {
		t.Errorf("nil store Snapshot() = %v, want empty non-nil map", snap)
	}
}
