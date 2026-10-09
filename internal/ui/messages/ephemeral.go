package messages

import "github.com/gammons/slk/internal/ui/styles"

// SubtypeEphemeral marks a message only the user sees, such as a slash
// command's response. slk never caches it, so it is gone after the
// channel reloads, as in Slack.
const SubtypeEphemeral = "ephemeral"

// EphemeralMark is the note after the author line of an ephemeral
// message, "" for any other message.
func EphemeralMark(msg MessageItem) string {
	if msg.Subtype != SubtypeEphemeral {
		return ""
	}
	return " " + styles.Timestamp.Render("· only visible to you")
}
