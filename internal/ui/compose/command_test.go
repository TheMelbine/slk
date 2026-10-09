package compose

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/commandpicker"
)

func commandCompose() Model {
	m := New("general")
	m.SetCommands([]commandpicker.Command{{Name: "/remind"}, {Name: "/rename"}, {Name: "/shrug"}})
	m.SetWidth(80)
	m.Focus()
	return m
}

func TestCommandPicker_OpensOnLeadingSlash(t *testing.T) {
	m := typeText(commandCompose(), "/re")
	if !m.IsCommandActive() {
		t.Fatal("picker not open after /re")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.IsCommandActive() {
		t.Error("picker still open after Tab")
	}
	if got := m.Value(); got != "/rename " {
		t.Errorf("value = %q, want %q", got, "/rename ")
	}
	m = typeText(m, "ops")
	if got := m.Value(); got != "/rename ops" {
		t.Errorf("value after typing args = %q", got)
	}
}

func TestCommandPicker_NotMidMessage(t *testing.T) {
	m := typeText(commandCompose(), "a/b")
	if m.IsCommandActive() {
		t.Error("picker opened for a slash inside text")
	}
	m = typeText(New("general"), "/")
	if m.IsCommandActive() {
		t.Error("picker opened with no commands loaded")
	}
}

func TestCommandPicker_SpaceCloses(t *testing.T) {
	m := typeText(commandCompose(), "/shrug ")
	if m.IsCommandActive() {
		t.Error("picker still open after a space")
	}
}

func TestSlashCommand(t *testing.T) {
	cases := []struct {
		draft, name, args string
		isCommand, known  bool
	}{
		{"/remind me to stretch in 1h", "/remind", "me to stretch in 1h", true, true},
		{"/shrug", "/shrug", "", true, true},
		{"/shrug\nsecond line", "/shrug", "second line", true, true},
		{"/remnd me", "/remnd", "me", true, false},
		{"/", "", "", false, false},
		{" /remind me", "", "", false, false},
		{"see /remind", "", "", false, false},
	}
	for _, c := range cases {
		m := commandCompose()
		m.SetValue(c.draft)
		name, args, isCommand, known := m.SlashCommand()
		if name != c.name || args != c.args || isCommand != c.isCommand || known != c.known {
			t.Errorf("%q: got (%q, %q, %v, %v), want (%q, %q, %v, %v)", c.draft, name, args, isCommand, known, c.name, c.args, c.isCommand, c.known)
		}
	}
}
