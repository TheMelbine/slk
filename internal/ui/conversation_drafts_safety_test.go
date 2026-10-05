package ui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/core"
	"github.com/gammons/slk/internal/ui/sidebar"
	"github.com/gammons/slk/internal/ui/wintree"
)

// TestDraftSafety_ChannelPickerSwitchKeepsDraft pins that an open
// channel picker cannot smuggle text into a channel the user switched
// to: switching away mid-query saves the draft under the original
// channel (not the destination), sends nothing, and restores the
// original draft on return.
func TestDraftSafety_ChannelPickerSwitchKeepsDraft(t *testing.T) {
	a := newTestApp(t, withActiveTeam("T1"), withWindowSize(200, 60))

	a.SetInitialChannel("C1", "alpha", nil)
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "alpha", Type: "channel"},
		{ID: "C2", Name: "beta", Type: "channel"},
	})

	_, _ = a.Update(keyPress('i'))
	for _, r := range "hello #" {
		a.Update(keyPress(r))
	}

	if !a.compose.IsChannelActive() {
		t.Fatal("precondition: typing '#' did not open the channel picker")
	}

	// Switch to C2 while the picker is open. The draft must not follow.
	a.Update(ChannelSelectedMsg{ID: "C2", Name: "beta", Type: "channel"})

	// Enter on the (empty) destination sends nothing and the picker is gone.
	a.Update(keyCode(tea.KeyEnter))
	if got := a.compose.Value(); got != "" {
		t.Errorf("destination C2 draft = %q, want empty", got)
	}
	if a.compose.IsChannelActive() {
		t.Error("channel picker survived the channel switch")
	}

	// Returning to C1 restores the pre-switch draft.
	a.Update(ChannelSelectedMsg{ID: "C1", Name: "alpha", Type: "channel"})
	if got := a.compose.Value(); got != "hello #" {
		t.Errorf("returning to C1: draft = %q, want %q", got, "hello #")
	}
}

// TestDraftSafety_UploadInFlightBlocksWindowOps pins that an in-flight
// upload freezes window management (a no-op, not a channel switch), so
// the uploading window's caption cannot be stranded or resurrected,
// and a failed upload preserves the source text + attachments.
func TestDraftSafety_UploadInFlightBlocksWindowOps(t *testing.T) {
	a := newTestApp(t, withActiveTeam("T1"), withWindowSize(200, 60))

	a.SetInitialChannel("C1", "alpha", nil)
	a.Update(ChannelSelectedMsg{ID: "C1", Name: "alpha", Type: "channel"})

	first := a.focusedWin

	// Split off a second window and bind it to C2 with its own draft.
	a.splitWindow(wintree.SplitSideBySide)
	a.Update(ChannelSelectedMsg{ID: "C2", Name: "beta", Type: "channel"})
	a.compose.SetValue("other draft")
	second := a.focusedWin

	// Back to C1 with a caption + attachment queued for upload.
	a.focusWindow(first)
	const caption = "upload caption"
	a.compose.SetValue(caption)
	a.compose.AddAttachment(core.PendingAttachment{Filename: "shot.png", Bytes: []byte("x")})

	// An unrelated thread draft that must not be disturbed.
	a.threadCompose.SetValue("thread keep")

	a.setUploaderForTest(func(channelID, threadTS, captionText string, attachments []core.PendingAttachment) tea.Cmd {
		return nil
	})
	a.submitWithAttachments(&a.compose)
	if !a.compose.Uploading() {
		t.Fatal("precondition: main compose did not enter the uploading state")
	}

	// Window churn while uploading is refused wholesale: focus cannot
	// move, no window opens or closes, and the caption is untouched.
	a.focusWindow(second)
	a.closeWindow()
	a.splitWindow(wintree.SplitSideBySide)
	a.onlyWindow()

	if a.focusedWin != first {
		t.Errorf("focused window changed during upload: got %v, want %v", a.focusedWin, first)
	}
	if got := a.wins.Len(); got != 2 {
		t.Errorf("window count changed during upload: got %d, want 2", got)
	}
	if got := a.compose.Value(); got != caption {
		t.Errorf("caption changed during upload: got %q, want %q", got, caption)
	}

	// Successful upload clears only the source composer.
	a.Update(UploadResultMsg{})
	if got := a.compose.Value(); got != "" {
		t.Errorf("main draft after success = %q, want empty", got)
	}
	if got := a.compose.Attachments(); len(got) != 0 {
		t.Errorf("main attachments after success = %d, want 0", len(got))
	}
	if got := a.threadCompose.Value(); got != "thread keep" {
		t.Errorf("thread draft clobbered by main upload: got %q, want %q", got, "thread keep")
	}

	// The other window's draft is still intact.
	a.focusWindow(second)
	if got := a.compose.Value(); got != "other draft" {
		t.Errorf("second window draft = %q, want %q", got, "other draft")
	}

	// Returning to the source window must not resurrect the cleared caption.
	a.focusWindow(first)
	if got := a.compose.Value(); got != "" {
		t.Errorf("caption resurrected on return: got %q, want empty", got)
	}

	// A failed upload preserves the source caption + attachment.
	a.compose.SetValue("retry caption")
	a.compose.AddAttachment(core.PendingAttachment{Filename: "retry.png", Bytes: []byte("y")})
	a.submitWithAttachments(&a.compose)
	a.Update(UploadResultMsg{Err: errors.New("network")})

	if got := a.compose.Value(); got != "retry caption" {
		t.Errorf("caption after failed upload = %q, want %q", got, "retry caption")
	}
	if got := len(a.compose.Attachments()); got != 1 {
		t.Errorf("attachments after failed upload = %d, want 1", got)
	}
	if a.compose.Uploading() {
		t.Error("still uploading after failure; retry would be refused")
	}
}

// TestDraftSafety_SendTargetsCaptureChannel pins that a queued send
// resolves to the channel it was composed in, even if the active
// channel switches before the command runs, and that clearing the
// composer is durable across a switch away and back.
func TestDraftSafety_SendTargetsCaptureChannel(t *testing.T) {
	a := newTestApp(t, withActiveTeam("T1"), withWindowSize(200, 60))

	a.SetInitialChannel("C1", "alpha", nil)
	a.Update(ChannelSelectedMsg{ID: "C1", Name: "alpha", Type: "channel"})

	a.compose.SetValue("send me")
	a.SetMode(ModeInsert)
	cmd := a.handleInsertMode(keyCode(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter with text produced no send command")
	}

	// The channel changes before the send command is executed.
	a.Update(ChannelSelectedMsg{ID: "C2", Name: "beta", Type: "channel"})

	msg, ok := cmd().(SendMessageMsg)
	if !ok {
		t.Fatalf("send command returned %T, want SendMessageMsg", cmd())
	}
	if msg.ChannelID != "C1" {
		t.Errorf("send target = %q, want %q (composed-in channel)", msg.ChannelID, "C1")
	}
	if msg.Text != "send me" {
		t.Errorf("send text = %q, want %q", msg.Text, "send me")
	}

	// Back on C1 the composer is empty: the send consumed the draft.
	a.Update(ChannelSelectedMsg{ID: "C1", Name: "alpha", Type: "channel"})
	if got := a.compose.Value(); got != "" {
		t.Errorf("C1 draft after send = %q, want empty", got)
	}

	// Ctrl+U clears the composer, and the empty draft stays empty
	// across a switch away and back.
	a.compose.SetValue("clear me")
	a.SetMode(ModeInsert)
	a.Update(keyMod('u', tea.ModCtrl))
	a.Update(ChannelSelectedMsg{ID: "C2", Name: "beta", Type: "channel"})
	a.Update(ChannelSelectedMsg{ID: "C1", Name: "alpha", Type: "channel"})
	if got := a.compose.Value(); got != "" {
		t.Errorf("cleared draft resurrected after round trip: got %q, want empty", got)
	}
}
