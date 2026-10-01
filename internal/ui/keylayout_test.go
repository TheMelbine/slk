package ui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestPhysicalKey(t *testing.T) {
	keys := DefaultKeyMap()
	cases := []struct {
		name string
		in   tea.KeyPressMsg
		want key.Binding
	}{
		{"table: о is j", tea.KeyPressMsg{Code: 'о', Text: "о"}, keys.Down},
		{"table: л is k", tea.KeyPressMsg{Code: 'л', Text: "л"}, keys.Up},
		{"table: legacy uppercase П is G", tea.KeyPressMsg{Code: 'П', Text: "П"}, keys.Bottom},
		{"table: shift+ж is :", tea.KeyPressMsg{Code: 'ж', Mod: tea.ModShift, Text: "Ж"}, keys.CommandMode},
		{"table: ctrl+е is ctrl+t", tea.KeyPressMsg{Code: 'е', Mod: tea.ModCtrl}, keys.FuzzyFinder},
		{"table: ctrl+ъ is ctrl+]", tea.KeyPressMsg{Code: 'ъ', Mod: tea.ModCtrl}, keys.ToggleThread},
		{"basecode wins", tea.KeyPressMsg{Code: 'ш', BaseCode: 'i', Text: "ш"}, keys.InsertMode},
		{"basecode shifted", tea.KeyPressMsg{Code: 'й', BaseCode: 'q', Mod: tea.ModShift, Text: "Й"}, keys.QuitConfirm},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := physicalKey(tc.in, true)
			if !key.Matches(got, tc.want) {
				t.Fatalf("physicalKey(%q) = %q, want a match for %v", tc.in.String(), got.String(), tc.want.Keys())
			}
		})
	}
}

func TestPhysicalKey_LeavesTextAlone(t *testing.T) {
	in := tea.KeyPressMsg{Code: 'о', BaseCode: 'j', Text: "о"}
	if got := physicalKey(in, false); got.String() != "о" {
		t.Fatalf("text-entry key rewritten to %q", got.String())
	}
	ascii := tea.KeyPressMsg{Code: 'j', Text: "j"}
	if got := physicalKey(ascii, true); got != tea.KeyMsg(ascii) {
		t.Fatalf("ASCII key changed: %#v", got)
	}
	// Chords are rewritten even while typing: ctrl+г clears compose like ctrl+u.
	if got := physicalKey(tea.KeyPressMsg{Code: 'г', Mod: tea.ModCtrl}, false); got.String() != "ctrl+u" {
		t.Fatalf("ctrl+г = %q, want ctrl+u", got.String())
	}
}

func TestHandleKey_RussianLayoutEntersInsertMode(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.SetMode(ModeNormal)
	_ = a.handleKey(tea.KeyPressMsg{Code: 'ш', Text: "ш"})
	if a.mode != ModeInsert {
		t.Fatalf("mode = %v, want insert", a.mode)
	}
	_ = a.handleKey(tea.KeyPressMsg{Code: 'ш', Text: "ш"})
	if v := a.compose.Value(); v != "ш" {
		t.Fatalf("compose = %q, want the Cyrillic letter typed as-is", v)
	}
}
