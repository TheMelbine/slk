package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/ui"
)

// threadFollower is the one Slack call toggleThreadFollow makes.
type threadFollower interface {
	SetThreadSubscription(ctx context.Context, channelID, threadTS, lastRead string, subscribe bool) error
}

// toggleThreadFollow follows the thread when the local cache says the
// user does not follow it, and unfollows it otherwise. On success it
// writes the new state to thread_subscriptions, so a second press does
// not depend on the WS echo having arrived.
//
// A new subscription starts read up to now: following a thread should
// not mark its existing replies unread. Unfollowing keeps the cursor.
func toggleThreadFollow(client threadFollower, db *cache.DB, teamID, channelID, threadTS string, now time.Time) ui.ThreadFollowToggledMsg {
	res := ui.ThreadFollowToggledMsg{TeamID: teamID, ChannelID: channelID, ThreadTS: threadTS}
	lastRead, following, err := db.ThreadSubscriptionState(teamID, channelID, threadTS)
	if err != nil {
		res.Err = err
		return res
	}
	follow := !following
	if follow {
		lastRead = fmt.Sprintf("%d.%06d", now.Unix(), now.Nanosecond()/1000)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.SetThreadSubscription(ctx, channelID, threadTS, lastRead, follow); err != nil {
		res.Err = err
		return res
	}
	if err := db.UpsertThreadSubscription(teamID, channelID, threadTS, lastRead, follow); err != nil {
		res.Err = err
		return res
	}
	res.Following = follow
	return res
}
