package main

import (
	"context"
	"errors"
	"time"

	"github.com/gammons/slk/internal/ui"
)

// toggleChannelStar stars channelID when the section store does not
// list it under Starred, and unstars it otherwise. On success it
// re-bootstraps the store, since stars.list is the only source for the
// Starred section's members, and refreshes the sidebar.
func toggleChannelStar(wctx *WorkspaceContext, channelID string) ui.ChannelStarToggledMsg {
	res := ui.ChannelStarToggledMsg{ChannelID: channelID}
	store := wctx.SectionStore
	if store == nil || !store.Ready() {
		res.Err = errors.New("sidebar sections are not loaded")
		return res
	}
	star := !store.IsStarred(channelID)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := wctx.Client.SetChannelStar(ctx, channelID, star); err != nil {
		res.Err = err
		return res
	}
	res.Starred = star
	if err := store.Bootstrap(ctx, wctx.Client); err != nil {
		res.Err = err
		return res
	}
	if wctx.RTMHandler != nil {
		wctx.RTMHandler.refreshSectionsForActive()
	}
	return res
}
