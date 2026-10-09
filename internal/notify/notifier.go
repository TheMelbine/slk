// Package notify provides desktop notification support.
package notify

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gammons/slk/internal/mention"
	"github.com/gammons/slk/internal/slackfmt"
	"github.com/gen2brain/beeep"
)

// Notifier sends OS-level desktop notifications.
type Notifier struct {
	enabled bool
	command string
	// helper is the slk-notifier binary inside the macOS app bundle
	// built from macos/notifier. Empty when it is not installed or
	// off macOS.
	helper string
	// clickSocket is the unix socket slk listens on for notification
	// clicks; the helper writes Message.Click to it. Empty: clicks only
	// bring the terminal back.
	clickSocket string
}

// Message is one desktop notification.
type Message struct {
	Title string
	Body  string
	// Image is a picture file to attach, usually the sender's avatar.
	Image string
	// Click is the payload the helper writes to the click socket when
	// the user clicks the notification.
	Click string
}

// SetClickSocket sets the unix socket path the helper reports clicks to.
// Call it before the first notification.
func (n *Notifier) SetClickSocket(path string) {
	n.clickSocket = path
}

// HasHelper reports whether notifications go through slk-notifier, the
// only path that reports clicks.
func (n *Notifier) HasHelper() bool {
	return n.enabled && n.command == "" && n.helper != ""
}

// helperPaths lists where New looks for the slk-notifier app bundle.
func helperPaths() []string {
	const bin = "slk-notifier.app/Contents/MacOS/slk-notifier"
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Applications", bin))
	}
	return append(paths, filepath.Join("/Applications", bin))
}

func findHelper() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	for _, p := range helperPaths() {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// New creates a Notifier. If enabled is false, Notify is a no-op. When command
// is non-empty, Notify runs it in place of the built-in OS notification, which
// lets you route notifications through your own tooling (a terminal
// multiplexer's notifier, terminal-notifier, mako, etc.).
func New(enabled bool, command string) *Notifier {
	// beeep's default app name is "DefaultAppName"; brand the notification
	// as slk so the notification daemon attributes it correctly.
	beeep.AppName = "slk"
	return &Notifier{enabled: enabled, command: command, helper: findHelper()}
}

// Notify delivers a notification with the given title and body. It returns nil
// when notifications are disabled. If a notify_command is configured it runs in
// place of the built-in OS notification; otherwise beeep shows the OS one.
func (n *Notifier) Notify(title, body string) error {
	return n.Send(Message{Title: title, Body: body})
}

// Send is Notify with a picture and a click payload. notify_command sees
// the picture as $SLK_IMAGE. On macOS with slk-notifier installed the
// helper posts the notification under its own icon, attaches the picture
// and reports a click; beeep ignores both.
func (n *Notifier) Send(m Message) error {
	if !n.enabled {
		return nil
	}
	if n.command != "" {
		return n.runCommand(m)
	}
	if n.helper != "" {
		return n.runHelper(m)
	}
	return beeep.Notify(m.Title, m.Body, "")
}

// runHelper posts through slk-notifier. --activate names the terminal
// app slk runs in and --focus-title the start of slk's window title, so
// a click on the notification brings back the tab slk runs in.
func (n *Notifier) runHelper(m Message) error {
	args := []string{"--title", m.Title, "--body", m.Body, "--focus-title", "slk"}
	if m.Image != "" {
		args = append(args, "--image", m.Image)
	}
	if term := os.Getenv("__CFBundleIdentifier"); term != "" {
		args = append(args, "--activate", term)
	}
	if n.clickSocket != "" && m.Click != "" {
		args = append(args, "--click-socket", n.clickSocket, "--click", m.Click)
	}
	return exec.Command(n.helper, args...).Run()
}

// runCommand runs the configured notify_command via `sh -c`, exposing the
// notification's title, body and image path as $SLK_TITLE, $SLK_BODY and
// $SLK_IMAGE. They are passed through the environment rather than interpolated into the command string, so
// arbitrary message text (e.g. a body containing "; rm -rf ~") cannot inject
// shell syntax. Notify is already called from its own goroutine, so a
// synchronous Run — which also reaps the child — is fine.
func (n *Notifier) runCommand(m Message) error {
	cmd := exec.Command("sh", "-c", n.command)
	cmd.Env = append(os.Environ(), "SLK_TITLE="+m.Title, "SLK_BODY="+m.Body, "SLK_IMAGE="+m.Image)
	return cmd.Run()
}

// NotifyContext holds the state needed to evaluate notification triggers.
type NotifyContext struct {
	CurrentUserID   string
	ActiveChannelID string
	IsActiveWS      bool
	OnMention       bool
	OnDM            bool
	OnKeyword       []string
	IsDND           bool // when true, ShouldNotify always returns false
	IsMuted         bool // when true (conversation is muted), ShouldNotify always returns false
	OnThread        bool // notify on replies in followed threads
	ThreadFollowed  bool // the message is a reply in a thread the user follows
	ThreadOpen      bool // that thread is on screen right now
	// ChannelLevel is the desktop level the user set on this channel in
	// Slack (LevelEverything, LevelMentions, LevelNothing). Empty means
	// the channel was never customized and OnMention/OnDM/OnKeyword
	// decide. A set level replaces those switches for this channel.
	ChannelLevel string
}

// Per-channel desktop levels, named as Slack's all_notifications_prefs
// names them.
const (
	LevelEverything = "everything"
	LevelMentions   = "mentions_dms"
	LevelNothing    = "nothing"
)

// ShouldNotify returns true if a message should trigger a desktop notification.
func ShouldNotify(ctx NotifyContext, channelID, userID, text, channelType string) bool {
	// Never notify for own messages
	if userID == ctx.CurrentUserID {
		return false
	}

	// Suppress entirely while DND/snoozed.
	if ctx.IsDND {
		return false
	}

	// A reply in a followed thread notifies unless the thread is on screen.
	// Checked before mute and the active-channel rule: following a thread is
	// the narrower, later choice, and a reply is not visible in the channel
	// feed anyway.
	if ctx.OnThread && ctx.ThreadFollowed {
		return !ctx.ThreadOpen
	}

	// Suppress notifications from a muted conversation — a muted channel or DM
	// is silent, matching Slack.
	if ctx.IsMuted || ctx.ChannelLevel == LevelNothing {
		return false
	}

	// Suppress if viewing this channel on the active workspace
	if ctx.IsActiveWS && channelID == ctx.ActiveChannelID {
		return false
	}

	// A level the user picked for this channel in Slack replaces the
	// global on_* switches.
	switch ctx.ChannelLevel {
	case LevelEverything:
		return true
	case LevelMentions:
		return mention.InText(text, ctx.CurrentUserID) || matchesKeyword(text, ctx.OnKeyword)
	}

	// Check DM trigger
	if ctx.OnDM && (channelType == "dm" || channelType == "group_dm") {
		return true
	}

	// Check mention trigger. The predicate lives in internal/mention so
	// the sidebar's mention badge applies the same rule; the policy
	// around it (OnMention, DND, mute, active channel) stays here.
	if ctx.OnMention && mention.InText(text, ctx.CurrentUserID) {
		return true
	}

	// Check keyword triggers
	return matchesKeyword(text, ctx.OnKeyword)
}

// matchesKeyword reports whether text contains any of the configured
// keywords, case-insensitively.
func matchesKeyword(text string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	lower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// StripSlackMarkup converts Slack-formatted text to plain text suitable for
// OS notification bodies. User mentions are resolved against userNames; if
// a user ID is missing from the map (or the map is nil) the raw user ID is
// used as a fallback. Output is truncated to 100 characters (by rune) with a
// "..." suffix.
func StripSlackMarkup(text string, userNames map[string]string) string {
	return StripSlackMarkupWithUserGroups(text, userNames, nil)
}

// StripSlackMarkupWithUserGroups is StripSlackMarkup with a workspace-scoped
// Slack usergroup map for resolving bare <!subteam^SID> tokens. The markup
// stripping itself lives in internal/slackfmt; this adds the notification
// body's length cap.
func StripSlackMarkupWithUserGroups(text string, userNames, userGroups map[string]string) string {
	text = slackfmt.StripMarkupWithUserGroups(text, userNames, userGroups)

	// Cap by rune, not byte, so a multibyte body (e.g. Japanese) isn't
	// sliced mid-rune into invalid UTF-8.
	if r := []rune(text); len(r) > 100 {
		text = string(r[:100]) + "..."
	}
	return text
}
