package text

import (
	"unicode"
	"unicode/utf8"
)

// IsQueryRune reports whether key, a key string as the pickers receive
// it, is a single printable character that belongs in a type-to-filter
// query. Letters of any script count, so Cyrillic names are searchable;
// named keys ("enter", "ctrl+n") and controls do not.
func IsQueryRune(key string) bool {
	r, size := utf8.DecodeRuneInString(key)
	if size == 0 || size != len(key) || r == utf8.RuneError {
		return false
	}
	return unicode.IsPrint(r)
}

// TrimLastRune removes the last character of s. A query is edited with
// this rather than a byte slice so a multi-byte character is removed
// whole.
func TrimLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
