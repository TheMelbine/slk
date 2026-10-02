package main

import (
	"regexp"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/ui"
)

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
//     copy) in WorkspaceReadyMsg / WorkspaceSwitchedMsg. Once NotifyFrom
//     has been called with that snapshot, every Set that adds or
//     changes a name is reported to the notifier, which sends it to the
//     UI as UserResolvedMsg. So callers just Set; reaching the UI is the
//     store's job, not something each caller has to remember.
//
// A nil *userNameStore reads as empty and drops writes, so helpers whose
// name source is optional (tests, cache-only renders) can pass nil.
type userNameStore struct {
	mu    sync.RWMutex
	names map[string]string
	// notify, once set by NotifyFrom, is told of every Set that adds or
	// changes a name. Called outside mu.
	notify func(userID, name string)
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

// Set records a display name for userID and, if it is new or changed
// and a notifier is installed, reports it.
func (s *userNameStore) Set(userID, name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	old, had := s.names[userID]
	s.names[userID] = name
	notify := s.notify
	s.mu.Unlock()
	if notify != nil && (!had || old != name) {
		notify(userID, name)
	}
}

// NotifyFrom installs notify and immediately reports every name that
// differs from base, the snapshot the UI was last handed. That closes
// the gap between taking a snapshot and the UI applying it: a name Set
// in between is in neither the snapshot nor (if the UI drops messages
// for a workspace it has not switched to yet) a message, so it is
// reported here instead. Call it once the UI has applied base; calling
// it again (each workspace switch) just repeats the reconciliation.
func (s *userNameStore) NotifyFrom(base map[string]string, notify func(userID, name string)) {
	if s == nil {
		return
	}
	type entry struct{ id, name string }
	s.mu.Lock()
	s.notify = notify
	var gap []entry
	for id, name := range s.names {
		if old, ok := base[id]; !ok || old != name {
			gap = append(gap, entry{id, name})
		}
	}
	s.mu.Unlock()
	for _, e := range gap {
		notify(e.id, e.name)
	}
}

// uiNameNotifier is the notifier main.go installs: it reports a name to
// the UI as UserResolvedMsg. Two properties matter:
//
//   - The send is asynchronous. bubbletea's Program.Send blocks until
//     the Update loop takes the message, and Set is called from inside
//     that loop (ReadCache / CacheRead run synchronously in reducers,
//     and record SQLite hits). A synchronous send from there would wait
//     on itself forever.
//   - The TeamID is the store's workspace. A message for a workspace
//     that is not active is dropped by the reducer; that is fine,
//     because NotifyFrom re-reports the gap when the workspace is
//     switched to.
func uiNameNotifier(teamID string, send func(tea.Msg)) func(userID, name string) {
	return func(userID, name string) {
		go send(ui.UserResolvedMsg{TeamID: teamID, UserID: userID, DisplayName: name})
	}
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

// mentionIDRe matches a Slack user mention, <@U…>, capturing the ID.
// Same shape slackfmt and the renderer use.
var mentionIDRe = regexp.MustCompile(`<@([A-Z0-9]+)>`)

// MentionedNames returns the recorded names of just the users mentioned
// in text, as a fresh map the caller owns. For one-off rendering (a
// desktop notification) where copying the whole store per call would
// be wasteful.
func (s *userNameStore) MentionedNames(text string) map[string]string {
	out := map[string]string{}
	for _, m := range mentionIDRe.FindAllStringSubmatch(text, -1) {
		if name, ok := s.Get(m[1]); ok {
			out[m[1]] = name
		}
	}
	return out
}
