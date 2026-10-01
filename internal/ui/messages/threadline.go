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
// "▣▣ 3 replies  Last reply today at 3:04 PM". miniAvatar may be nil
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

// lastReplyLabel formats a Slack ts relative to now the way Slack's
// thread line does: "today at 3:04 PM", "yesterday at 9:12 AM",
// "on Sep 30" beyond that. Empty or unparseable ts gives "".
func lastReplyLabel(ts string, now time.Time) string {
	t, ok := tsTime(ts)
	if !ok {
		return ""
	}
	t = t.In(now.Location())
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	today := time.Date(y2, m2, d2, 0, 0, 0, 0, now.Location())
	day := time.Date(y1, m1, d1, 0, 0, 0, 0, now.Location())
	switch days := int(today.Sub(day).Hours() / 24); {
	case days == 0:
		return "today at " + t.Format("3:04 PM")
	case days == 1:
		return "yesterday at " + t.Format("3:04 PM")
	case y1 == y2:
		return "on " + t.Format("Jan 2")
	default:
		return "on " + t.Format("Jan 2, 2006")
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
