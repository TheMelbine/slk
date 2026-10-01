package ui

import (
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// cyrillicToLatin maps a lowercase Cyrillic letter to the US key at the
// same physical position on a ЙЦУКЕН keyboard (Russian, plus the
// Ukrainian letters that differ). Used when the terminal does not
// report Key.BaseCode.
var cyrillicToLatin = map[rune]rune{
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y',
	'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p', 'х': '[', 'ъ': ']',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h',
	'о': 'j', 'л': 'k', 'д': 'l', 'ж': ';', 'э': '\'',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n',
	'ь': 'm', 'б': ',', 'ю': '.', 'ё': '`',
	'і': 's', 'ї': ']', 'є': '\'', 'ґ': '`',
}

// usShifted is the US-layout shifted form of the non-letter keys
// cyrillicToLatin can produce.
var usShifted = map[rune]rune{
	'[': '{', ']': '}', ';': ':', '\'': '"', ',': '<', '.': '>', '`': '~',
}

// physicalKey rewrites a key press typed on a non-Latin layout into the
// US key at the same physical position, so vim-style bindings keep
// working with a Russian layout active: `о` acts as `j`, `Ж` as `:`,
// ctrl+е as ctrl+t.
//
// Printable keys are rewritten only when printable is true; callers pass
// false in modes where the key is text the user is typing. Ctrl and alt
// chords are always rewritten, since nobody types Cyrillic text with
// them. Keys that are already ASCII pass through unchanged.
//
// The physical key comes from Key.BaseCode when the terminal reports it
// (kitty keyboard protocol with alternate keys, see App.View), and from
// the ЙЦУКЕН table otherwise.
func physicalKey(msg tea.KeyMsg, printable bool) tea.KeyMsg {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok || press.Code < 0x80 || press.Code > unicode.MaxRune {
		return msg
	}
	chord := press.Mod.Contains(tea.ModCtrl) || press.Mod.Contains(tea.ModAlt)
	if !chord && !printable {
		return msg
	}

	upper := unicode.IsUpper(press.Code) || press.Mod.Contains(tea.ModShift)
	for _, r := range press.Text {
		if unicode.IsUpper(r) {
			upper = true
		}
	}

	base := press.BaseCode
	if base == 0 || base >= 0x80 {
		var found bool
		base, found = cyrillicToLatin[unicode.ToLower(press.Code)]
		if !found {
			return msg
		}
	}
	base = unicode.ToLower(base)

	out := press
	out.Code = base
	out.BaseCode = base
	out.ShiftedCode = 0
	if press.Text != "" {
		text := base
		if upper {
			if s, ok := usShifted[base]; ok {
				text = s
			} else {
				text = unicode.ToUpper(base)
			}
		}
		out.Text = string(text)
	}
	return out
}

// textEntryMode reports whether printable keys in mode are text the
// user is typing (compose, prompts, type-to-filter pickers) rather than
// commands. physicalKey leaves printable keys alone in these modes.
func textEntryMode(m Mode) bool {
	switch m {
	case ModeNormal, ModeConfirm, ModeReactionsView, ModeLinkPicker, ModeUserProfile:
		return false
	}
	return true
}
