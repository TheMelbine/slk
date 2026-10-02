package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gammons/slk/internal/cache"
)

type fakeFollower struct {
	lastRead  string
	subscribe bool
	calls     int
	err       error
}

func (f *fakeFollower) SetThreadSubscription(_ context.Context, _, _, lastRead string, subscribe bool) error {
	f.calls++
	f.lastRead, f.subscribe = lastRead, subscribe
	return f.err
}

func followTestDB(t *testing.T) *cache.DB {
	t.Helper()
	db, err := cache.New(":memory:")
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.UpsertWorkspace(cache.Workspace{ID: "T1", Name: "Test", Domain: "test"}); err != nil {
		t.Fatalf("UpsertWorkspace: %v", err)
	}
	return db
}

func TestToggleThreadFollow_FollowsThenUnfollows(t *testing.T) {
	db := followTestDB(t)
	client := &fakeFollower{}
	now := time.Unix(1700000000, 123456000)

	res := toggleThreadFollow(client, db, "T1", "C1", "100.000001", now)
	if res.Err != nil || !res.Following {
		t.Fatalf("first toggle = %+v, want Following", res)
	}
	if !client.subscribe || client.lastRead != "1700000000.123456" {
		t.Errorf("follow sent subscribe=%v last_read=%q, want true and now", client.subscribe, client.lastRead)
	}
	lastRead, active, _ := db.ThreadSubscriptionState("T1", "C1", "100.000001")
	if !active || lastRead != "1700000000.123456" {
		t.Errorf("cache after follow = (%q, %v)", lastRead, active)
	}

	res = toggleThreadFollow(client, db, "T1", "C1", "100.000001", now.Add(time.Hour))
	if res.Err != nil || res.Following {
		t.Fatalf("second toggle = %+v, want unfollowed", res)
	}
	if client.subscribe || client.lastRead != "1700000000.123456" {
		t.Errorf("unfollow sent subscribe=%v last_read=%q, want false and the kept cursor", client.subscribe, client.lastRead)
	}
	if _, active, _ = db.ThreadSubscriptionState("T1", "C1", "100.000001"); active {
		t.Error("cache still says followed after unfollow")
	}
}

func TestToggleThreadFollow_SlackErrorLeavesCacheAlone(t *testing.T) {
	db := followTestDB(t)
	client := &fakeFollower{err: errors.New("boom")}

	res := toggleThreadFollow(client, db, "T1", "C1", "100.000001", time.Now())
	if res.Err == nil {
		t.Fatal("want the Slack error in the result")
	}
	if _, active, _ := db.ThreadSubscriptionState("T1", "C1", "100.000001"); active {
		t.Error("cache says followed though Slack rejected the call")
	}
}
