package main

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/debuglog"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui"
	"github.com/gammons/slk/internal/ui/commandpicker"
)

type commandLister interface {
	ListCommands(ctx context.Context) ([]slackclient.SlashCommand, error)
}

type commandRunner interface {
	RunCommand(ctx context.Context, channelID, command, text, threadTS string) error
}

// loadSlashCommands fetches the workspace's slash commands for the "/"
// picker. On failure the picker stays off for this workspace and every
// draft is sent as a message.
func loadSlashCommands(ctx context.Context, client commandLister, teamID string, send func(tea.Msg)) {
	list, err := client.ListCommands(ctx)
	if err != nil {
		debuglog.General("slash commands %s: %v", teamID, err)
		return
	}
	cmds := make([]commandpicker.Command, len(list))
	for i, c := range list {
		cmds[i] = commandpicker.Command{Name: c.Name, Desc: c.Desc, Usage: c.Usage, AppName: c.AppName}
	}
	send(ui.SlashCommandsMsg{TeamID: teamID, Commands: cmds})
}

// runSlashCommand runs one command for ChannelService.RunCommand.
func runSlashCommand(ctx context.Context, client commandRunner, channelID, threadTS, command, text string) ui.SlashCommandRanMsg {
	err := client.RunCommand(ctx, channelID, command, text, threadTS)
	if err != nil {
		debuglog.General("slash command %s in %s: %v", command, channelID, err)
	}
	return ui.SlashCommandRanMsg{Command: command, Err: err}
}
