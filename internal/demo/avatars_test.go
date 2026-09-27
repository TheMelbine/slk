package demo

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/gammons/slk/internal/avatar"
)

func TestAvatarFootprintMatchesTheRealAvatars(t *testing.T) {
	if avatarCols != avatar.AvatarCols || avatarRows != avatar.AvatarRows {
		t.Fatalf("demo avatars are %dx%d, real ones %dx%d", avatarCols, avatarRows, avatar.AvatarCols, avatar.AvatarRows)
	}
	lines := strings.Split(renderAvatar("U0PRIYA", "Priya Shah"), "\n")
	if len(lines) != avatarRows {
		t.Fatalf("avatar has %d lines, want %d", len(lines), avatarRows)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != avatarCols {
			t.Errorf("line %d is %d cells wide, want %d: %q", i, w, avatarCols, l)
		}
	}
	if !strings.Contains(ansi.Strip(lines[0]), "PS") {
		t.Errorf("top row %q does not carry the initials", ansi.Strip(lines[0]))
	}
}

func TestAvatarIsStablePerUser(t *testing.T) {
	if renderAvatar("U1", "A B") != renderAvatar("U1", "A B") {
		t.Fatal("same user rendered differently")
	}
}

func TestInitials(t *testing.T) {
	for name, want := range map[string]string{
		"Priya Shah":    "PS",
		"Finn O'Brien":  "FO",
		"deploybot":     "DE",
		"X":             "X ",
		"":              "??",
		"Ana María Paz": "AM",
	} {
		if got := initials(name); got != want {
			t.Errorf("initials(%q) = %q, want %q", name, got, want)
		}
	}
}
