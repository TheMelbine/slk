package messages

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestNoteReplyUser(t *testing.T) {
	in := []string{"U1", "U2", "U3"}
	got := noteReplyUser(in, "U3")
	if strings.Join(got, ",") != "U3,U1,U2" {
		t.Fatalf("got %v, want U3 moved to front", got)
	}
	if strings.Join(in, ",") != "U1,U2,U3" {
		t.Fatalf("input mutated: %v", in)
	}
	if got := noteReplyUser(nil, "U9"); strings.Join(got, ",") != "U9" {
		t.Fatalf("got %v, want [U9]", got)
	}
}

func TestLastReplyLabel(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	ts := func(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) + ".000100" }
	cases := []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 10, 2, 9, 5, 0, 0, time.Local), "today at 9:05 AM"},
		{time.Date(2026, 10, 1, 23, 50, 0, 0, time.Local), "yesterday at 11:50 PM"},
		{time.Date(2026, 9, 3, 12, 0, 0, 0, time.Local), "on Sep 3"},
		{time.Date(2025, 12, 31, 12, 0, 0, 0, time.Local), "on Dec 31, 2025"},
	}
	for _, c := range cases {
		if got := lastReplyLabel(ts(c.at), now); got != c.want {
			t.Errorf("lastReplyLabel(%v) = %q, want %q", c.at, got, c.want)
		}
	}
	if got := lastReplyLabel("", now); got != "" {
		t.Errorf("empty ts gave %q", got)
	}
}

func TestRenderThreadLine(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	latest := strconv.FormatInt(time.Date(2026, 10, 2, 14, 30, 0, 0, time.Local).Unix(), 10) + ".000200"
	msg := MessageItem{ReplyCount: 5, ReplyUsers: []string{"U1", "U2", "U3", "U4"}, LatestReply: latest}
	mini := func(uid string) string {
		if uid == "U2" {
			return "" // not loaded yet: skipped
		}
		return "[" + uid + "]"
	}
	got := ansi.Strip(renderThreadLine(msg, mini, now))
	want := "[U1] [U3] [U4] 5 replies  Last reply today at 2:30 PM"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := ansi.Strip(renderThreadLine(MessageItem{ReplyCount: 1}, nil, now)); got != "1 reply" {
		t.Fatalf("bare line = %q", got)
	}
}

func TestIncrementReplyCountTracksLatestReplier(t *testing.T) {
	m := New([]MessageItem{{TS: "100.000001", ReplyCount: 1, ReplyUsers: []string{"U1"}, LatestReply: "100.000002"}}, "general")
	m.IncrementReplyCount("100.000001", "100.000009", "U2")
	got := m.messages[0]
	if got.ReplyCount != 2 || strings.Join(got.ReplyUsers, ",") != "U2,U1" || got.LatestReply != "100.000009" {
		t.Fatalf("after reply: count=%d users=%v latest=%s", got.ReplyCount, got.ReplyUsers, got.LatestReply)
	}
}
