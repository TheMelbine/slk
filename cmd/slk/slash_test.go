package main

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui"
	"github.com/slack-go/slack"
)

type fakeSlash struct {
	list    []slackclient.SlashCommand
	listErr error
	runErr  error
	ran     []string
}

func (f *fakeSlash) ListCommands(context.Context) ([]slackclient.SlashCommand, error) {
	return f.list, f.listErr
}

func (f *fakeSlash) RunCommand(_ context.Context, channelID, command, text, threadTS string) error {
	f.ran = append(f.ran, channelID, command, text, threadTS)
	return f.runErr
}

func TestLoadSlashCommands_SendsList(t *testing.T) {
	f := &fakeSlash{list: []slackclient.SlashCommand{{Name: "/remind", Desc: "Set a reminder", AppName: ""}}}
	var got []tea.Msg
	loadSlashCommands(context.Background(), f, "T1", func(m tea.Msg) { got = append(got, m) })
	if len(got) != 1 {
		t.Fatalf("sent %d msgs", len(got))
	}
	m, ok := got[0].(ui.SlashCommandsMsg)
	if !ok || m.TeamID != "T1" || len(m.Commands) != 1 || m.Commands[0].Name != "/remind" || m.Commands[0].Desc != "Set a reminder" {
		t.Errorf("msg = %#v", got[0])
	}
}

func TestLoadSlashCommands_ErrorSendsNothing(t *testing.T) {
	f := &fakeSlash{listErr: errors.New("boom")}
	loadSlashCommands(context.Background(), f, "T1", func(m tea.Msg) { t.Errorf("sent %#v", m) })
}

func TestRunSlashCommand(t *testing.T) {
	f := &fakeSlash{runErr: errors.New("invalid_command")}
	got := runSlashCommand(context.Background(), f, "C1", "1.0", "/remind", "help")
	if got.Command != "/remind" || got.Err == nil {
		t.Errorf("result = %#v", got)
	}
	if want := []string{"C1", "/remind", "help", "1.0"}; len(f.ran) != 4 || f.ran[0] != want[0] || f.ran[1] != want[1] || f.ran[2] != want[2] || f.ran[3] != want[3] {
		t.Errorf("ran = %v", f.ran)
	}
}

func TestOnEphemeralMessage_SendsWithoutCaching(t *testing.T) {
	db := newTestDB(t)
	sender := &captureSender{}
	h := &rtmEventHandler{program: sender, db: db, userNames: newUserNameStore(nil), workspaceID: "T1"}
	h.OnEphemeralMessage("C1", "USLACKBOT", "9.0", "Need some help?", "", slack.Blocks{}, nil, "B01", "")
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d msgs", len(sender.sent))
	}
	m, ok := sender.sent[0].(ui.EphemeralMessageMsg)
	if !ok || m.ChannelID != "C1" || m.Message.UserName != "Slackbot" || m.Message.Text != "Need some help?" {
		t.Errorf("msg = %#v", sender.sent[0])
	}
	cached, err := db.GetMessages("C1", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 0 {
		t.Error("ephemeral message was cached")
	}
}

func TestOnEphemeralMessage_InactiveWorkspaceDropped(t *testing.T) {
	sender := &captureSender{}
	h := &rtmEventHandler{program: sender, isActive: func() bool { return false }}
	h.OnEphemeralMessage("C1", "U1", "9.0", "x", "", slack.Blocks{}, nil, "", "")
	if len(sender.sent) != 0 {
		t.Errorf("sent %v for an inactive workspace", sender.sent)
	}
}
