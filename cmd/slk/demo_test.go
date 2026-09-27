package main

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/gammons/slk/internal/demo"
	"github.com/gammons/slk/internal/ui"
)

// deliverDemo feeds msg to app, then runs the commands it returns,
// recursively, as Bubble Tea would. A command that does not answer
// within 50ms is a timer (spinner, typing expiry) and is skipped.
func deliverDemo(t *testing.T, app *ui.App, msg tea.Msg, depth int) {
	t.Helper()
	if msg == nil || depth > 12 {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, cmd := range batch {
			runDemoCmd(t, app, cmd, depth+1)
		}
		return
	}
	_, cmd := app.Update(msg)
	runDemoCmd(t, app, cmd, depth+1)
}

func runDemoCmd(t *testing.T, app *ui.App, cmd tea.Cmd, depth int) {
	t.Helper()
	if cmd == nil {
		return
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		deliverDemo(t, app, msg, depth)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDemoAppRendersItsFirstFrame(t *testing.T) {
	d, err := demo.New("hero", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	app := newDemoApp(d)
	deliverDemo(t, app, tea.WindowSizeMsg{Width: 120, Height: 40}, 0)
	for _, msg := range d.StartupMsgs() {
		deliverDemo(t, app, msg, 0)
	}
	frame := ansi.Strip(app.View().Content)
	for _, want := range []string{"general", "engineering", "Lunch order goes out"} {
		if !strings.Contains(frame, want) {
			t.Errorf("first demo frame is missing %q:\n%s", want, frame)
		}
	}
}

func TestRunDemoRejectsUnknownScenario(t *testing.T) {
	err := runDemo("nope")
	if err == nil || !strings.Contains(err.Error(), "hero") {
		t.Fatalf("runDemo(nope) = %v, want an error listing the scenarios", err)
	}
}
