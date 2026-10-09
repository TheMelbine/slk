// internal/ui/command.go
//
// The vi-style ":" command registry (window-management design §5).
//
// executeCommand parses a command line (without the leading ':')
// and dispatches through the commands map — the designated
// extension point for future :commands. Later phases of the
// window-management plan register sp / vsp / q / only here.
package ui

import (
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/wintree"
)

// commandFunc executes a named :command. args holds the
// whitespace-separated tokens after the command name (unused by
// v1 commands, reserved for e.g. ":sp #channel").
type commandFunc func(a *App, args []string) tea.Cmd

// commands maps a command name to its handler. Names are matched
// exactly (no prefix matching); aliases get their own entries.
var commands = map[string]commandFunc{
	"ws":       cmdWorkspaceFinder,
	"sp":       cmdSplit,
	"vsp":      cmdVSplit,
	"q":        cmdCloseWindow,
	"q!":       cmdQuit,
	"qa":       cmdQuit,
	"qa!":      cmdQuit,
	"only":     cmdOnlyWindow,
	"on":       cmdOnlyWindow,
	"activity": cmdActivity,
}

// cmdActivity opens the Activity view (mentions, thread replies,
// reactions to your messages, DMs). The sidebar ◉ Activity row is the
// primary entry point; this command mirrors it for discoverability.
func cmdActivity(a *App, _ []string) tea.Cmd {
	a.sidebar.SelectActivityRow()
	return func() tea.Msg { return ActivityViewActivatedMsg{} }
}

// cmdSplit / cmdVSplit create a stacked / side-by-side split of the
// focused window (window-management design §5).
func cmdSplit(a *App, _ []string) tea.Cmd  { return a.splitWindow(wintree.SplitStacked) }
func cmdVSplit(a *App, _ []string) tea.Cmd { return a.splitWindow(wintree.SplitSideBySide) }

// cmdCloseWindow closes the focused window. In the last window it
// quits, as in vim.
func cmdCloseWindow(a *App, args []string) tea.Cmd {
	if a.wins.Len() <= 1 {
		return cmdQuit(a, args)
	}
	return a.closeWindow()
}

// cmdQuit quits slk without the confirm prompt: a typed :q is
// deliberate in a way a stray q or ctrl+c is not. An upload in progress
// still blocks it.
func cmdQuit(a *App, _ []string) tea.Cmd {
	if a.compose.Uploading() || a.threadCompose.Uploading() {
		return a.uploadToastCmd("Upload in progress", 2*time.Second)
	}
	return tea.Quit
}

// cmdOnlyWindow closes all other windows.
func cmdOnlyWindow(a *App, _ []string) tea.Cmd {
	a.onlyWindow()
	return nil
}

// cmdWorkspaceFinder opens the workspace finder overlay —
// the :command replacement for the finder's old ctrl+w binding.
func cmdWorkspaceFinder(a *App, _ []string) tea.Cmd {
	a.workspaceFinder.Open()
	a.SetMode(ModeWorkspaceFinder)
	return nil
}

// executeCommand parses and runs one command line (without the
// leading ':'). Empty input is a no-op; unknown commands show a
// transient status-bar toast.
func executeCommand(a *App, line string) tea.Cmd {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	fn, ok := commands[fields[0]]
	if !ok {
		// Typed on a Russian layout: ":й" is ":q".
		fn, ok = commands[latinCommand(fields[0])]
	}
	if !ok {
		return toastWithClear(a, "Unknown command: "+fields[0], 2*time.Second)
	}
	return fn(a, fields[1:])
}

// latinCommand maps the letters of a command typed on a ЙЦУКЕН layout
// to the US keys at the same positions.
func latinCommand(name string) string {
	var b strings.Builder
	for _, r := range name {
		if l, ok := cyrillicToLatin[unicode.ToLower(r)]; ok {
			r = l
		}
		b.WriteRune(r)
	}
	return b.String()
}
