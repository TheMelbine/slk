package main

import (
	"context"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/core"
	"github.com/gammons/slk/internal/demo"
	emojiwidth "github.com/gammons/slk/internal/emoji"
	"github.com/gammons/slk/internal/ui"
	"github.com/gammons/slk/internal/ui/styles"
)

// runDemo runs slk against the made-up workspaces in internal/demo. It is
// the recording mode behind the hidden --demo flag: no config, tokens,
// cache, network or config writes. See
// docs/superpowers/specs/2026-09-27-demo-mode-design.md.
func runDemo(scenario string) error {
	d, err := demo.New(scenario, time.Now())
	if err != nil {
		return err
	}
	styles.Apply(d.InitialTheme(), core.Theme{})
	emojiwidth.SetImageMode(false, 2)

	p := tea.NewProgram(newDemoApp(d))
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.Run(ctx, p.Send)
	}()
	// Send blocks until the program's event loop is running, so the
	// workspaces "connect" from their own goroutine, as in run().
	go func() {
		for _, msg := range d.StartupMsgs() {
			p.Send(msg)
		}
	}()
	_, err = p.Run()
	cancel()
	wg.Wait()
	return err
}

// newDemoApp builds the demo's App. It takes no config, token store or
// cache handle, so demo mode cannot reach the user's real data.
func newDemoApp(d *demo.Demo) *ui.App {
	app := ui.NewApp()
	applyUISettings(app, uiSettings{
		TimestampFormat:  demo.TimestampFormat,
		TypingIndicators: true,
		MouseWheelLines:  3,
		ColoredUsernames: true,
	})
	d.Install(app)
	return app
}
