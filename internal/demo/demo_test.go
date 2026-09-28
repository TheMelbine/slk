package demo

import (
	"strings"
	"testing"

	"github.com/gammons/slk/internal/ui"
)

func TestNewRejectsUnknownScenario(t *testing.T) {
	if _, err := New("nope", testNow); err == nil || !strings.Contains(err.Error(), "hero") {
		t.Fatalf("New(nope) err = %v, want the list of scenarios", err)
	}
}

func TestStartupMsgs(t *testing.T) {
	d := testDemo(t)
	msgs := d.StartupMsgs()
	if len(msgs) != 3 {
		t.Fatalf("got %d startup messages, want 3", len(msgs))
	}
	lumen, ok := msgs[0].(ui.WorkspaceReadyMsg)
	if !ok || !lumen.InitialActive || lumen.TeamID != teamLumen || lumen.LastChannelID != chGeneral ||
		lumen.Theme != "tokyo night" || lumen.UserID != uAlex || len(lumen.Channels) == 0 {
		t.Errorf("first = %#v", msgs[0])
	}
	if dw, ok := msgs[1].(ui.WorkspaceReadyMsg); !ok || dw.InitialActive || dw.TeamID != teamDriftwood {
		t.Errorf("second = %#v", msgs[1])
	}
	if c, ok := msgs[2].(ui.ConnectionStateMsg); !ok || c.State != 1 {
		t.Errorf("third = %#v, want connected", msgs[2])
	}
	if d.InitialTheme() != "tokyo night" {
		t.Errorf("InitialTheme = %q", d.InitialTheme())
	}
}

func TestInstallWiresAnApp(t *testing.T) {
	testDemo(t).Install(ui.NewApp()) // must not panic; cmd/slk's smoke test renders it
}
