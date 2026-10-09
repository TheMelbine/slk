// internal/ui/reducer_slash.go
//
// Slash commands: the per-workspace command list for the "/" pickers,
// running a command typed in a composer, and the ephemeral messages
// commands answer with.
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/compose"
	"github.com/gammons/slk/internal/ui/messages"
)

var reduceSlash reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case SlashCommandsMsg:
		a.slashCommands[m.TeamID] = m.Commands
		if m.TeamID == a.activeTeamID {
			a.applySlashCommands()
		}
		return nil, true

	case RunSlashCommandMsg:
		channels := a.channels
		if channels == nil {
			return nil, true
		}
		return func() tea.Msg {
			return channels.RunCommand(ids.ChannelID(m.ChannelID), m.ThreadTS, m.Command, m.Text)
		}, true

	case SlashCommandRanMsg:
		if m.Err != nil {
			return toastWithClear(a, m.Command+" failed: "+m.Err.Error(), 4*time.Second), true
		}
		return nil, true

	case EphemeralMessageMsg:
		item := m.Message
		item.Subtype = messages.SubtypeEphemeral
		reply := item.ThreadTS != "" && item.ThreadTS != item.TS
		if !reply {
			for _, mm := range a.modelsForChannel(m.ChannelID) {
				mm.AppendMessage(cloneMessageItem(item))
			}
		} else if a.threadVisible && m.ChannelID == a.threadPanel.ChannelID() && item.ThreadTS == a.threadPanel.ThreadTS() {
			a.threadPanel.AddReply(item)
		}
		return nil, true
	}
	return nil, false
}

// applySlashCommands hands the active workspace's commands to both
// composers.
func (a *App) applySlashCommands() {
	cmds := a.slashCommands[a.activeTeamID]
	a.compose.SetCommands(cmds)
	a.threadCompose.SetCommands(cmds)
}

// runSlashCommand handles a draft that starts with "/name": a known
// command runs; an unknown one, or any while the list has not loaded,
// is refused with a toast and the draft stays, so a typo never lands
// in the channel. It reports false for any other draft, which the
// caller sends as a message.
func (a *App) runSlashCommand(c *compose.Model, channelID, threadTS string) (tea.Cmd, bool) {
	name, args, isCommand, known := c.SlashCommand()
	if !isCommand || channelID == "" {
		return nil, false
	}
	if !known {
		text := name + " is not a valid command. Start with a space to send it as text"
		if len(a.slashCommands[a.activeTeamID]) == 0 {
			text = "Slash commands have not loaded. Start with a space to send it as text"
		}
		return toastWithClear(a, text, 4*time.Second), true
	}
	args = c.TranslateMentionsForSend(args)
	c.Reset()
	a.exitInsertAfterSend()
	return func() tea.Msg {
		return RunSlashCommandMsg{ChannelID: channelID, ThreadTS: threadTS, Command: name, Text: args}
	}, true
}
