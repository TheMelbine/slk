package text

import "testing"

func TestIsQueryRune(t *testing.T) {
	for _, ok := range []string{"a", "Z", "ж", "Я", "é", " ", "#"} {
		if !IsQueryRune(ok) {
			t.Errorf("IsQueryRune(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "enter", "ctrl+n", "ab", "\x1b", "\n"} {
		if IsQueryRune(bad) {
			t.Errorf("IsQueryRune(%q) = true, want false", bad)
		}
	}
}

func TestTrimLastRune(t *testing.T) {
	cases := map[string]string{"": "", "a": "", "abc": "ab", "общ": "об", "aж": "a"}
	for in, want := range cases {
		if got := TrimLastRune(in); got != want {
			t.Errorf("TrimLastRune(%q) = %q, want %q", in, got, want)
		}
	}
}
