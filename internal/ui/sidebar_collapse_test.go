package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/core"
	"github.com/gammons/slk/internal/ui/sidebar"
)

// collapseStore is an in-memory stand-in for the sidebar_collapse table.
type collapseStore map[string]map[string]bool

func (s collapseStore) funcs() core.ChannelServiceFuncs {
	return core.ChannelServiceFuncs{
		SectionCollapse: func(team string) map[string]bool { return s[team] },
		SaveSectionCollapse: func(team, key string, collapsed bool) {
			if s[team] == nil {
				s[team] = map[string]bool{}
			}
			s[team][key] = collapsed
		},
	}
}

var collapseChannels = []sidebar.ChannelItem{
	{ID: "C1", Name: "general", Type: "channel"},
	{ID: "D1", Name: "alice", Type: "dm"},
}

func TestSectionCollapse_SurvivesRestart(t *testing.T) {
	store := collapseStore{}
	a := NewApp()
	setChannelFuncsForTest(a, store.funcs())
	_, cmd := a.Update(WorkspaceReadyMsg{TeamID: "T1", TeamName: "Test", Channels: collapseChannels, InitialActive: true})
	drainBatch(cmd)

	// Collapse Direct Messages from its header.
	for i := 0; i < 20; i++ {
		a.sidebar.MoveUp()
	}
	for i := 0; i < 20; i++ {
		if name, ok := a.sidebar.IsSectionHeaderSelected(); ok && name == "Direct Messages" {
			break
		}
		a.sidebar.MoveDown()
	}
	cmd, ok := a.toggleSectionCollapse()
	if !ok {
		t.Fatal("cursor never reached the Direct Messages header")
	}
	drainBatch(cmd)
	if store["T1"]["Direct Messages"] != true {
		t.Fatalf("toggle not saved: %v", store)
	}

	// A fresh App (restart) restores it, and restoring the last channel,
	// which sits in that section, does not expand it.
	b := NewApp()
	setChannelFuncsForTest(b, store.funcs())
	_, cmd = b.Update(WorkspaceReadyMsg{TeamID: "T1", TeamName: "Test", Channels: collapseChannels, LastChannelID: "D1", InitialActive: true})
	drainBatch(cmd)
	if !b.sidebar.IsCollapsed("Direct Messages") {
		t.Error("Direct Messages expanded after restart")
	}
}

func TestSectionCollapse_PerWorkspace(t *testing.T) {
	store := collapseStore{"T1": {"Direct Messages": true}}
	a := NewApp()
	setChannelFuncsForTest(a, store.funcs())
	_, cmd := a.Update(WorkspaceReadyMsg{TeamID: "T1", TeamName: "One", Channels: collapseChannels, InitialActive: true})
	drainBatch(cmd)
	if !a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("T1 state not applied")
	}
	_, cmd = a.Update(WorkspaceSwitchedMsg{TeamID: "T2", TeamName: "Two", Channels: collapseChannels})
	drainBatch(cmd)
	if a.sidebar.IsCollapsed("Direct Messages") {
		t.Error("T1's collapsed section leaked into T2")
	}
}

var _ tea.Msg = WorkspaceReadyMsg{}
