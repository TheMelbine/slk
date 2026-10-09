package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/wintree"
)

func TestExecuteCommand_EmptyIsNoop(t *testing.T) {
	a := NewApp()
	if cmd := executeCommand(a, "   "); cmd != nil {
		t.Fatal("empty command line should be a no-op")
	}
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", a.mode)
	}
}

func TestExecuteCommand_UnknownShowsToast(t *testing.T) {
	a := NewApp()
	cmd := executeCommand(a, "bogus")
	if cmd == nil {
		t.Fatal("unknown command should return the toast-clear cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Unknown command: bogus") {
		t.Fatalf("expected unknown-command toast, got:\n%s", out)
	}
}

func TestExecuteCommand_WSOpensWorkspaceFinder(t *testing.T) {
	a := NewApp()
	_ = executeCommand(a, "ws")
	if a.mode != ModeWorkspaceFinder {
		t.Fatalf("mode = %v, want ModeWorkspaceFinder", a.mode)
	}
}

func TestExecuteCommand_TrimsAndIgnoresArgs(t *testing.T) {
	a := NewApp()
	_ = executeCommand(a, "  ws   extra  ")
	if a.mode != ModeWorkspaceFinder {
		t.Fatalf("mode = %v, want ModeWorkspaceFinder", a.mode)
	}
}

func TestExecuteCommand_QInLastWindowQuits(t *testing.T) {
	for _, line := range []string{"q", "q!", "qa", "й", "йф"} {
		a := newWideTestApp(t)
		cmd := executeCommand(a, line)
		if cmd == nil {
			t.Fatalf(":%s returned no cmd", line)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf(":%s did not quit", line)
		}
	}
}

func TestExecuteCommand_QClosesSplitFirst(t *testing.T) {
	a := newWideTestApp(t)
	_ = a.splitWindow(wintree.SplitSideBySide)
	if a.wins.Len() != 2 {
		t.Fatalf("precondition: Len = %d, want 2", a.wins.Len())
	}
	if cmd := executeCommand(a, "q"); cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal(":q quit with two windows open")
		}
	}
	if a.wins.Len() != 1 {
		t.Errorf("Len = %d after :q, want 1", a.wins.Len())
	}
}
