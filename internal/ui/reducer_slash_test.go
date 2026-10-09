package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/core"
	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/commandpicker"
	"github.com/gammons/slk/internal/ui/messages"
)

func slashTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestAppWithMessages(t)
	a.activeTeamID = "T1"
	a.activeChannelID = "C1"
	a.Update(SlashCommandsMsg{TeamID: "T1", Commands: []commandpicker.Command{{Name: "/remind"}}})
	return a
}

func typeKeys(a *App, s string) {
	for _, r := range s {
		a.handleKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestSlash_KnownCommandRuns(t *testing.T) {
	a := slashTestApp(t)
	var got []string
	setChannelFuncsForTest(a, core.ChannelServiceFuncs{
		RunCommand: func(ch ids.ChannelID, threadTS, command, text string) core.Msg {
			got = []string{string(ch), threadTS, command, text}
			return SlashCommandRanMsg{Command: command}
		},
	})
	a.SetMode(ModeInsert)
	a.compose.Focus()
	a.compose.SetValue("/remind me to stretch in 1h")
	cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, m := range drainBatch(cmd) {
		if run, ok := m.(RunSlashCommandMsg); ok {
			_, next := a.Update(run)
			drainBatch(next)
		}
		if _, ok := m.(SendMessageMsg); ok {
			t.Fatal("a known command was sent as a message")
		}
	}
	if strings.Join(got, "|") != "C1||/remind|me to stretch in 1h" {
		t.Fatalf("RunCommand got %q", got)
	}
	if a.compose.Value() != "" {
		t.Errorf("compose not cleared: %q", a.compose.Value())
	}
}

func TestSlash_UnknownCommandStaysInCompose(t *testing.T) {
	for _, draft := range []string{"/remnd me", "/usr/bin is a path"} {
		a := slashTestApp(t)
		a.SetMode(ModeInsert)
		a.compose.Focus()
		a.compose.SetValue(draft)
		cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		for _, m := range drainBatch(cmd) {
			switch m.(type) {
			case RunSlashCommandMsg, SendMessageMsg:
				t.Fatalf("%q: dispatched %T", draft, m)
			}
		}
		if a.compose.Value() != draft {
			t.Errorf("%q: compose = %q, want the draft kept", draft, a.compose.Value())
		}
		if out := a.statusbar.View(160); !strings.Contains(out, "is not a valid command") {
			t.Errorf("%q: no warning toast:\n%s", draft, out)
		}
	}
}

func TestSlash_LeadingSpaceSendsText(t *testing.T) {
	a := slashTestApp(t)
	a.SetMode(ModeInsert)
	a.compose.Focus()
	a.compose.SetValue(" /usr/bin is a path")
	cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	sent := false
	for _, m := range drainBatch(cmd) {
		if _, ok := m.(SendMessageMsg); ok {
			sent = true
		}
	}
	if !sent {
		t.Error("a draft with a leading space was not sent as a message")
	}
}

func TestSlash_NotLoadedRefuses(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.activeTeamID = "T1"
	a.activeChannelID = "C1"
	a.SetMode(ModeInsert)
	a.compose.Focus()
	a.compose.SetValue("/remind me")
	cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, m := range drainBatch(cmd) {
		if _, ok := m.(SendMessageMsg); ok {
			t.Fatal("a slash command was sent as text before commands loaded")
		}
	}
	if out := a.statusbar.View(160); !strings.Contains(out, "have not loaded") {
		t.Errorf("no warning toast:\n%s", out)
	}
}

func TestSlash_PickerOpensInInsertMode(t *testing.T) {
	a := slashTestApp(t)
	a.SetMode(ModeInsert)
	a.compose.Focus()
	typeKeys(a, "/re")
	if !a.compose.IsCommandActive() {
		t.Fatal("command picker not open")
	}
	a.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if a.compose.IsCommandActive() || a.mode != ModeInsert {
		t.Errorf("Esc: picker=%v mode=%v, want closed picker, still insert", a.compose.IsCommandActive(), a.mode)
	}
}

func TestSlash_FailureToasts(t *testing.T) {
	a := slashTestApp(t)
	a.Update(SlashCommandRanMsg{Command: "/remind", Err: errors.New("invalid_command")})
	if out := a.statusbar.View(120); !strings.Contains(out, "/remind failed: invalid_command") {
		t.Errorf("no failure toast:\n%s", out)
	}
}

func TestSlash_EphemeralAppendsWithoutReadState(t *testing.T) {
	a := slashTestApp(t)
	before := len(a.messagepane.Messages())
	a.Update(EphemeralMessageMsg{ChannelID: "C1", Message: messages.MessageItem{TS: "9.0", UserName: "Slackbot", Text: "Need some help?"}})
	msgs := a.messagepane.Messages()
	if len(msgs) != before+1 {
		t.Fatalf("messages = %d, want %d", len(msgs), before+1)
	}
	last := msgs[len(msgs)-1]
	if last.Subtype != messages.SubtypeEphemeral || last.Text != "Need some help?" {
		t.Errorf("last = %+v", last)
	}
	a.Update(EphemeralMessageMsg{ChannelID: "C_OTHER", Message: messages.MessageItem{TS: "9.1", Text: "elsewhere"}})
	if len(a.messagepane.Messages()) != before+1 {
		t.Error("ephemeral for another channel landed in the pane")
	}
}

func TestSlash_CommandsFollowActiveWorkspace(t *testing.T) {
	a := slashTestApp(t)
	a.Update(SlashCommandsMsg{TeamID: "T2", Commands: []commandpicker.Command{{Name: "/other"}}})
	a.compose.SetValue("/other x")
	if _, _, _, known := a.compose.SlashCommand(); known {
		t.Error("another workspace's command reached the active composer")
	}
	a.compose.SetValue("/remind x")
	if _, _, _, known := a.compose.SlashCommand(); !known {
		t.Error("active workspace's command not recognized")
	}
}
