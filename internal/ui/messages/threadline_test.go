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
	ts := func(d time.Duration) string { return strconv.FormatInt(now.Add(-d).Unix(), 10) + ".000100" }
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{20 * time.Second, "just now"},
		{time.Minute, "1 minute ago"},
		{59 * time.Minute, "59 minutes ago"},
		{2*time.Hour + 50*time.Minute, "2 hours ago"},
		{49 * time.Hour, "2 days ago"},
		{70 * 24 * time.Hour, "2 months ago"},
		{400 * 24 * time.Hour, "1 year ago"},
	}
	for _, c := range cases {
		if got := lastReplyLabel(ts(c.ago), now); got != c.want {
			t.Errorf("lastReplyLabel(-%v) = %q, want %q", c.ago, got, c.want)
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
	want := "[U1] [U3] [U4] 5 replies  Last reply 30 minutes ago"
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

func TestRefreshReplyAges_RerendersLabel(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	SetNowFunc(func() time.Time { return now })
	t.Cleanup(func() { SetNowFunc(nil) })
	latest := strconv.FormatInt(now.Add(-5*time.Minute).Unix(), 10) + ".000100"
	m := New([]MessageItem{{TS: "1790000000.000100", UserName: "a", Text: "hi", ReplyCount: 2, LatestReply: latest}}, "general")
	if v := ansi.Strip(m.View(20, 80)); !strings.Contains(v, "Last reply 5 minutes ago") {
		t.Fatalf("initial label missing:\n%s", v)
	}
	now = now.Add(2 * time.Hour)
	if v := ansi.Strip(m.View(20, 80)); !strings.Contains(v, "5 minutes ago") {
		t.Fatalf("expected the cached label before refresh:\n%s", v)
	}
	m.RefreshReplyAges()
	if v := ansi.Strip(m.View(20, 80)); !strings.Contains(v, "Last reply 2 hours ago") {
		t.Fatalf("label not refreshed:\n%s", v)
	}
}
