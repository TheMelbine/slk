package main

import "sync"

// userNameStore is a workspace's user ID -> display name cache on the
// engine side (cmd/slk). It replaces a plain map[string]string that was
// shared, as one map object, between the UI goroutine and every
// background path: history fetchers running as bubbletea Cmds, the
// WebSocket event loop, and the unresolved-DM sweep.
//
// That sharing was a "fatal error: concurrent map read and map write"
// waiting to happen, and on 2026-10-02 it happened: fetchThreadReplies
// memoized a cached name into the map from its Cmd goroutine while
// the Threads view rendered mentions from the same map. Reproduced in
// one process by TestFetchThreadReplies_RacesThreadsViewRender.
//
// The rule now:
//   - Engine code reads and writes names only through this store, from
//     any goroutine.
//   - The UI never sees the store. It is handed a Snapshot (a private
//     copy) in WorkspaceReadyMsg / WorkspaceSwitchedMsg, and names
//     learned later reach it as UserResolvedMsg, applied on the UI
//     goroutine. So a write here that wants to be visible live must
//     also send UserResolvedMsg.
//
// A nil *userNameStore reads as empty and drops writes, so helpers whose
// name source is optional (tests, cache-only renders) can pass nil.
type userNameStore struct {
	mu    sync.RWMutex
	names map[string]string
}

// newUserNameStore returns a store seeded with a copy of seed (which
// may be nil).
func newUserNameStore(seed map[string]string) *userNameStore {
	names := make(map[string]string, len(seed))
	for id, name := range seed {
		names[id] = name
	}
	return &userNameStore{names: names}
}

// Get returns the display name recorded for userID.
func (s *userNameStore) Get(userID string) (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	name, ok := s.names[userID]
	return name, ok
}

// Set records a display name for userID.
func (s *userNameStore) Set(userID, name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.names[userID] = name
	s.mu.Unlock()
}

// Snapshot returns an independent copy of every recorded name. The
// caller owns the result; nothing else holds a reference to it. This is
// the only way a name map leaves the store, and it is how the UI gets
// one.
func (s *userNameStore) Snapshot() map[string]string {
	if s == nil {
		return map[string]string{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.names))
	for id, name := range s.names {
		out[id] = name
	}
	return out
}
