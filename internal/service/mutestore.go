package service

import (
	"context"
	"fmt"
	"sync"

	slk "github.com/gammons/slk/internal/slack"
)

// MutedChannelsClient is the subset of slk.Client MuteStore needs.
// Defined as an interface so tests can pass fakes.
type MutedChannelsClient interface {
	// GetChannelNotificationPrefs returns the per-channel notification
	// prefs (mute state, desktop level) of the authenticated user. The
	// data lives in the user's Slack prefs blob; it is not exposed
	// per-channel via conversations.list.
	GetChannelNotificationPrefs(ctx context.Context) (map[string]slk.ChannelNotifyPrefs, error)
}

// MuteStore is the per-workspace authoritative cache of the
// authenticated user's per-channel notification prefs: which channels
// are muted and which desktop level each one has. Populated on
// bootstrap from the users.prefs.get REST call and kept fresh by
// pref_change WebSocket events (see ApplyPrefChange).
//
// All public methods are safe for concurrent use.
//
// Mirrors SectionStore in shape — same Bootstrap / Ready / Apply*
// lifecycle, same nil-safety expectations from callers.
type MuteStore struct {
	mu    sync.RWMutex
	ready bool
	prefs map[string]slk.ChannelNotifyPrefs
}

// NewMuteStore returns an empty store. Reports Ready()==false until
// Bootstrap completes successfully.
func NewMuteStore() *MuteStore {
	return &MuteStore{prefs: map[string]slk.ChannelNotifyPrefs{}}
}

// Ready reports whether the store has successfully bootstrapped at
// least once. Callers should treat !Ready as "assume nothing is muted"
// (the conservative default — better to show an unread dot we should
// have suppressed than to hide one the user wanted to see).
func (s *MuteStore) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// Bootstrap fetches the notification prefs and replaces any prior
// state atomically. Returns an error without mutating state if the
// fetch fails.
func (s *MuteStore) Bootstrap(ctx context.Context, client MutedChannelsClient) error {
	prefs, err := client.GetChannelNotificationPrefs(ctx)
	if err != nil {
		return fmt.Errorf("fetching notification prefs: %w", err)
	}
	next := make(map[string]slk.ChannelNotifyPrefs, len(prefs))
	for id, p := range prefs {
		if id == "" {
			continue
		}
		next[id] = p
	}
	s.mu.Lock()
	s.prefs = next
	s.ready = true
	s.mu.Unlock()
	return nil
}

// IsMuted reports whether the given channel is currently muted by the
// authenticated user. Returns false when the store is not ready (the
// conservative default: don't claim a channel is muted unless we know).
func (s *MuteStore) IsMuted(channelID string) bool {
	if channelID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return false
	}
	return s.prefs[channelID].Muted
}

// DesktopLevel returns the channel's desktop notification level as
// Slack names it ("everything", "mentions_dms", "nothing"), or "" when
// the user never customized the channel or the store is not ready. An
// empty level means the caller's own defaults apply.
func (s *MuteStore) DesktopLevel(channelID string) string {
	if channelID == "" {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return ""
	}
	return s.prefs[channelID].Desktop
}

// MutedChannels returns a snapshot of every channel ID currently
// recorded as muted. Empty when the store is not ready. The returned
// slice is a copy and safe to mutate.
func (s *MuteStore) MutedChannels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return nil
	}
	var out []string
	for id, p := range s.prefs {
		if p.Muted {
			out = append(out, id)
		}
	}
	return out
}

// ApplyPrefChange applies a pref_change WebSocket event. Two prefs are
// acted on:
//
//   - all_notifications_prefs: Slack's current (per-channel) home for
//     mute state and desktop levels. value is a JSON-encoded string
//     with channels[id].muted and channels[id].desktop keys;
//     ParseAllNotificationsPrefs decodes it.
//   - muted_channels: legacy flat list, comma-separated. Kept for
//     back-compat in case Slack still ships it on some workspaces.
//
// Slack ships the full updated payload on every change for both prefs,
// so this is a wholesale replace, not an incremental delta.
//
// Returns true when the prefs actually changed, so callers can decide
// whether to trigger a sidebar refresh.
func (s *MuteStore) ApplyPrefChange(name, value string) bool {
	var next map[string]slk.ChannelNotifyPrefs
	switch name {
	case "muted_channels":
		next = slk.MergeChannelNotificationPrefs(value, "")
	case "all_notifications_prefs":
		next = slk.ParseAllNotificationsPrefs(value)
		if next == nil {
			next = map[string]slk.ChannelNotifyPrefs{}
		}
	default:
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Mark ready even on the first pref_change we see — the event
	// carries the full authoritative list, so it's a valid bootstrap
	// on its own.
	changed := !s.ready || !samePrefs(s.prefs, next)
	s.prefs = next
	s.ready = true
	return changed
}

// samePrefs reports whether a and b hold the same prefs for the same
// channel IDs.
func samePrefs(a, b map[string]slk.ChannelNotifyPrefs) bool {
	if len(a) != len(b) {
		return false
	}
	for id, p := range a {
		q, ok := b[id]
		if !ok || p != q {
			return false
		}
	}
	return true
}
