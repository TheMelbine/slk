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

// usShifted is the US-layout shifted form of the non-letter keys.
var usShifted = map[rune]rune{
	'[': '{', ']': '}', ';': ':', '\'': '"', ',': '<', '.': '>', '`': '~',
	'/': '?', '\\': '|', '-': '_', '=': '+',
	'1': '!', '2': '@', '3': '#', '4': '$', '5': '%',
	'6': '^', '7': '&', '8': '*', '9': '(', '0': ')',
}

// ruPunct maps the ASCII punctuation a Russian (PC) layout produces to
// the US key at the same position, as the US character it types there:
// `.` is the `/` key, shift+that key gives `,`, which is US `?`.
var ruPunct = map[rune]rune{'.': '/', ',': '?'}

// layoutHint updates the "non-Latin layout active" guess from msg: a
// typed letter decides it, anything else keeps the previous guess.
func layoutHint(msg tea.KeyMsg, prev bool) bool {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return prev
	}
	for _, r := range press.Text {
		if unicode.IsLetter(r) {
			return r >= 0x80
		}
	}
	return prev
}

// physicalKey rewrites a key press typed on a Russian layout into the
// US key at the same physical position, so vim-style bindings keep
// working: `о` acts as `j`, `Ж` as `:`, `.` as `/`, ctrl+е as ctrl+t.
//
// Printable keys are rewritten only when printable is true; callers pass
// false in modes where the key is text the user is typing. Ctrl and alt
// chords are always rewritten. ASCII punctuation is rewritten only when
// nonLatin says a Cyrillic layout is active (see layoutHint), since `.`
// on a US layout must stay `.`.
//
// The physical key comes from Key.BaseCode when the terminal reports it
// (kitty keyboard protocol with alternate keys, see App.View), and from
// the ЙЦУКЕН tables otherwise.
func physicalKey(msg tea.KeyMsg, printable, nonLatin bool) tea.KeyMsg {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok || press.Code > unicode.MaxRune {
		return msg
	}
	chord := press.Mod.Contains(tea.ModCtrl) || press.Mod.Contains(tea.ModAlt)
	if !chord && !printable {
		return msg
	}

	var base, text rune
	switch {
	case press.Code >= 0x80:
		base = press.BaseCode
		if base == 0 || base >= 0x80 {
			base = cyrillicToLatin[unicode.ToLower(press.Code)]
		}
	case nonLatin && !chord && press.Text != "" && unicode.IsPunct(press.Code):
		switch {
		case press.BaseCode != 0 && press.BaseCode < 0x80:
			base = press.BaseCode
		case ruPunct[press.Code] != 0:
			// Legacy input: Text is already the shifted character.
			text = ruPunct[press.Code]
			base = text
			if base == '?' {
				base = '/'
			}
		}
	}
	if base == 0 || base == press.Code && text == 0 && press.Code < 0x80 {
		return msg
	}
	base = unicode.ToLower(base)

	upper := unicode.IsUpper(press.Code) || press.Mod.Contains(tea.ModShift)
	for _, r := range press.Text {
		if unicode.IsUpper(r) {
			upper = true
		}
	}

	out := press
	out.Code = base
	out.BaseCode = base
	out.ShiftedCode = 0
	if press.Text != "" {
		if text == 0 {
			text = base
			if upper {
				if s, ok := usShifted[base]; ok {
					text = s
				} else {
					text = unicode.ToUpper(base)
				}
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
