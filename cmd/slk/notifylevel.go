package main

import (
	"context"
	"errors"
	"time"

	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui"
)

// cycleChannelNotifyLevel moves channelID's desktop notification level
// one step along everything → mentions → nothing and applies the prefs
// blob Slack echoes back to the mute store, so ShouldNotify sees the new
// level before the pref_change event arrives.
func cycleChannelNotifyLevel(wctx *WorkspaceContext, channelID string) ui.ChannelNotifyLevelChangedMsg {
	res := ui.ChannelNotifyLevelChangedMsg{ChannelID: channelID}
	store := wctx.MuteStore
	if store == nil || !store.Ready() {
		res.Err = errors.New("notification prefs are not loaded")
		return res
	}
	level := slackclient.NextNotifyLevel(store.DesktopLevel(channelID))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	prefs, err := wctx.Client.SetChannelNotificationLevel(ctx, channelID, level)
	if err != nil {
		res.Err = err
		return res
	}
	res.Level = level
	if prefs != "" && store.ApplyPrefChange("all_notifications_prefs", prefs) && wctx.RTMHandler != nil {
		wctx.RTMHandler.refreshMutedForActive()
	}
	return res
}
