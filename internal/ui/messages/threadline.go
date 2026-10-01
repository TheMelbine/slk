package messages

import (
	"fmt"
	"strings"
	"time"

	"github.com/gammons/slk/internal/ui/styles"
)

// threadLineMaxAvatars caps the participant avatars drawn on a thread
// line, newest participant first.
const threadLineMaxAvatars = 3

// noteReplyUser returns users with userID moved (or added) to the front,
// newest participant first. It always returns a fresh slice so items
// shared across window models never alias. An empty userID is a no-op
// copy.
func noteReplyUser(users []string, userID string) []string {
	out := make([]string, 0, len(users)+1)
	if userID != "" {
		out = append(out, userID)
	}
	for _, u := range users {
		if u != userID {
			out = append(out, u)
		}
	}
	return out
}

// renderThreadLine draws the line under a thread parent: participant
// avatars, the reply count and when the last reply landed, e.g.
// "▣▣ 3 replies  Last reply 2 hours ago". miniAvatar may be nil
// or return "" for avatars that are not loaded yet; those are skipped.
func renderThreadLine(msg MessageItem, miniAvatar func(userID string) string, now time.Time) string {
	var b strings.Builder
	if miniAvatar != nil {
		drawn := 0
		for _, uid := range msg.ReplyUsers {
			if drawn == threadLineMaxAvatars {
				break
			}
			if av := miniAvatar(uid); av != "" {
				b.WriteString(av)
				b.WriteString(" ")
				drawn++
			}
		}
	}
	word := "replies"
	if msg.ReplyCount == 1 {
		word = "reply"
	}
	b.WriteString(styles.ThreadIndicator.Render(fmt.Sprintf("%d %s", msg.ReplyCount, word)))
	if when := lastReplyLabel(msg.LatestReply, now); when != "" {
		b.WriteString(styles.Timestamp.Render("  Last reply " + when))
	}
	return b.String()
}

// lastReplyLabel says how long before now the ts was, in its largest
// whole unit: "just now", "5 minutes ago", "2 hours ago", "3 days ago",
// "2 months ago", "1 year ago". Empty or unparseable ts gives "".
// The label ages while the line sits in the render cache;
// RefreshReplyAges re-renders thread lines once a minute.
func lastReplyLabel(ts string, now time.Time) string {
	t, ok := tsTime(ts)
	if !ok {
		return ""
	}
	d := now.Sub(t)
	unit := func(n int, name string) string {
		if n != 1 {
			name += "s"
		}
		return fmt.Sprintf("%d %s ago", n, name)
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return unit(int(d/time.Minute), "minute")
	case d < 24*time.Hour:
		return unit(int(d/time.Hour), "hour")
	case d < 30*24*time.Hour:
		return unit(int(d/(24*time.Hour)), "day")
	case d < 365*24*time.Hour:
		return unit(int(d/(30*24*time.Hour)), "month")
	default:
		return unit(int(d/(365*24*time.Hour)), "year")
	}
}

// tsTime parses the seconds part of a Slack "1700000000.000100" ts.
func tsTime(ts string) (time.Time, bool) {
	sec, _, _ := strings.Cut(ts, ".")
	if sec == "" {
		return time.Time{}, false
	}
	var n int64
	if _, err := fmt.Sscan(sec, &n); err != nil || n <= 0 {
		return time.Time{}, false
	}
	return time.Unix(n, 0), true
}
