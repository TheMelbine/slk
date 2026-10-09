package notify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/beeep"
)

func TestNew_SetsAppName(t *testing.T) {
	New(true, "")
	if beeep.AppName != "slk" {
		t.Errorf("beeep.AppName = %q, want %q", beeep.AppName, "slk")
	}
}

func TestShouldNotify_SelfMessage(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnMention:       true,
		OnDM:            true,
	}
	if ShouldNotify(ctx, "C1", "U1", "hello", "dm") {
		t.Error("should not notify for self-messages")
	}
}

func TestShouldNotify_DM(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnDM:            true,
	}
	if !ShouldNotify(ctx, "C1", "U2", "hello", "dm") {
		t.Error("should notify for DM")
	}
	if !ShouldNotify(ctx, "C1", "U2", "hello", "group_dm") {
		t.Error("should notify for group DM")
	}
}

func TestShouldNotify_DM_Disabled(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnDM:            false,
	}
	if ShouldNotify(ctx, "C1", "U2", "hello", "dm") {
		t.Error("should not notify for DM when OnDM is false")
	}
}

func TestShouldNotify_Mention(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnMention:       true,
	}
	if !ShouldNotify(ctx, "C1", "U2", "hey <@U1> check this", "channel") {
		t.Error("should notify for mention")
	}
	if ShouldNotify(ctx, "C1", "U2", "hey <@U3> check this", "channel") {
		t.Error("should not notify for mention of another user")
	}
}

func TestShouldNotify_Mention_Disabled(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnMention:       false,
	}
	if ShouldNotify(ctx, "C1", "U2", "hey <@U1> check this", "channel") {
		t.Error("should not notify for mention when OnMention is false")
	}
}

func TestShouldNotify_SpecialMentions(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnMention:       true,
	}
	if !ShouldNotify(ctx, "C1", "U2", "hey <!here> check this", "channel") {
		t.Error("should notify for @here mention")
	}
	if !ShouldNotify(ctx, "C1", "U2", "hey <!channel> check this", "channel") {
		t.Error("should notify for @channel mention")
	}
	if !ShouldNotify(ctx, "C1", "U2", "hey <!everyone> check this", "channel") {
		t.Error("should notify for @everyone mention")
	}

	ctxNoMention := ctx
	ctxNoMention.OnMention = false
	if ShouldNotify(ctxNoMention, "C1", "U2", "hey <!here> check this", "channel") {
		t.Error("should not notify for @here when OnMention is false")
	}
}

func TestShouldNotify_Keyword(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnKeyword:       []string{"deploy", "incident"},
	}
	if !ShouldNotify(ctx, "C1", "U2", "starting deploy now", "channel") {
		t.Error("should notify for keyword match")
	}
	if !ShouldNotify(ctx, "C1", "U2", "DEPLOY is done", "channel") {
		t.Error("should notify for case-insensitive keyword match")
	}
	if ShouldNotify(ctx, "C1", "U2", "nothing relevant", "channel") {
		t.Error("should not notify when no keyword matches")
	}
}

func TestShouldNotify_ActiveChannel_Suppressed(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C1",
		IsActiveWS:      true,
		OnDM:            true,
	}
	if ShouldNotify(ctx, "C1", "U2", "hello", "dm") {
		t.Error("should suppress notification for active channel")
	}
}

func TestShouldNotify_InactiveWorkspace_NotSuppressed(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C1",
		IsActiveWS:      false,
		OnDM:            true,
	}
	if !ShouldNotify(ctx, "C1", "U2", "hello", "dm") {
		t.Error("should notify when workspace is inactive even if channel ID matches")
	}
}

func TestShouldNotify_SuppressedByDND(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      false, // would otherwise notify
		OnDM:            true,
		OnMention:       true,
		OnKeyword:       []string{"deploy"},
		IsDND:           true,
	}
	if ShouldNotify(ctx, "C1", "U2", "hey <@U1> deploy", "dm") {
		t.Error("DND should suppress notifications regardless of triggers")
	}
}

func TestShouldNotify_SuppressedByMute(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      false, // would otherwise notify
		OnDM:            true,
		OnMention:       true,
		OnKeyword:       []string{"deploy"},
		IsMuted:         true,
	}
	if ShouldNotify(ctx, "C1", "U2", "hey <@U1> deploy", "dm") {
		t.Error("a muted conversation should suppress notifications regardless of triggers")
	}
}

func TestStripSlackMarkup(t *testing.T) {
	userNames := map[string]string{
		"U123": "Alice",
		"U456": "Bob",
	}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "hello world", "hello world"},
		{"known user mention", "hey <@U123>", "hey @Alice"},
		{"unknown user mention falls back to ID", "hey <@U999>", "hey @U999"},
		{"multiple user mentions", "<@U123> and <@U456>", "@Alice and @Bob"},
		{"channel mention", "see <#C123|general>", "see #general"},
		{"link with label", "visit <https://example.com|Example>", "visit Example"},
		{"bare link", "visit <https://example.com>", "visit https://example.com"},
		{"labeled mailto link", "ping <mailto:foo@bar.com|foo@bar.com>", "ping foo@bar.com"},
		{"bare mailto link", "email <mailto:foo@bar.com>", "email foo@bar.com"},
		{"broadcast here", "<!here> heads up", "@here heads up"},
		{"broadcast channel", "<!channel> heads up", "@channel heads up"},
		{"broadcast everyone", "<!everyone> heads up", "@everyone heads up"},
		{"subteam mention", "ping <!subteam^S123|@platform> please", "ping @platform please"},
		{"markup chars stripped", "*bold* and _italic_ and ~strike~", "bold and italic and strike"},
		{"code", "`code`", "code"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripSlackMarkup(tt.input, userNames)
			if result != tt.expected {
				t.Errorf("StripSlackMarkup(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripSlackMarkup_NilUserNames(t *testing.T) {
	// Nil map should not panic; mentions fall back to user ID.
	result := StripSlackMarkup("hi <@U123>", nil)
	if result != "hi @U123" {
		t.Errorf("got %q, want %q", result, "hi @U123")
	}
}

func TestStripSlackMarkup_Truncation(t *testing.T) {
	long := ""
	for i := 0; i < 120; i++ {
		long += "a"
	}
	result := StripSlackMarkup(long, nil)
	if len(result) > 103 {
		t.Errorf("expected truncation, got length %d", len(result))
	}
	if result[len(result)-3:] != "..." {
		t.Error("expected ... suffix")
	}
}

func TestNotify_RunsNotifyCommand(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	n := New(true, "printf '%s\\n%s' \"$SLK_TITLE\" \"$SLK_BODY\" >"+out)
	if err := n.Notify("the title", "the body"); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading notify_command output: %v", err)
	}
	if want := "the title\nthe body"; string(got) != want {
		t.Errorf("notify_command received %q, want %q", got, want)
	}
}

func TestNotify_DisabledSkipsCommand(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	n := New(false, "touch "+out)
	if err := n.Notify("t", "b"); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("disabled notifier must not run notify_command")
	}
}

// Title/body reach the command through the environment, never interpolated into
// the command string, so a message body cannot inject a second shell command.
func TestNotify_CommandBodyIsNotInjected(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	pwned := filepath.Join(dir, "pwned")
	n := New(true, "printf '%s' \"$SLK_BODY\" >"+out)
	if err := n.Notify("title", "; touch "+pwned); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if _, err := os.Stat(pwned); !os.IsNotExist(err) {
		t.Error("message body was able to inject a shell command")
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading notify_command output: %v", err)
	}
	if want := "; touch " + pwned; string(got) != want {
		t.Errorf("body not passed literally: got %q, want %q", got, want)
	}
}

func TestShouldNotify_FollowedThread(t *testing.T) {
	base := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C1",
		IsActiveWS:      true,
		OnThread:        true,
		ThreadFollowed:  true,
	}
	// A reply is not visible in the channel feed, so viewing the channel
	// does not suppress it; neither does a muted channel.
	muted := base
	muted.IsMuted = true
	if !ShouldNotify(muted, "C1", "U2", "reply", "channel") {
		t.Error("reply in a followed thread should notify, even in the active, muted channel")
	}

	open := base
	open.ThreadOpen = true
	if ShouldNotify(open, "C1", "U2", "reply", "channel") {
		t.Error("should not notify while the thread is on screen")
	}

	if ShouldNotify(base, "C1", "U1", "reply", "channel") {
		t.Error("should not notify for own reply")
	}

	dnd := base
	dnd.IsDND = true
	if ShouldNotify(dnd, "C1", "U2", "reply", "channel") {
		t.Error("should not notify during DND")
	}

	off := base
	off.OnThread = false
	if ShouldNotify(off, "C1", "U2", "reply", "channel") {
		t.Error("on_thread = false should leave the reply to the other rules")
	}

	unfollowed := base
	unfollowed.ThreadFollowed = false
	unfollowed.ActiveChannelID = "C_OTHER"
	if ShouldNotify(unfollowed, "C1", "U2", "reply", "channel") {
		t.Error("reply in an unfollowed thread should not notify")
	}
}

func TestShouldNotify_ChannelLevel(t *testing.T) {
	base := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnMention:       true,
		OnDM:            true,
		OnKeyword:       []string{"deploy"},
	}
	mention := "hey <@U1> look"
	plain := "nothing special"

	nothing := base
	nothing.ChannelLevel = LevelNothing
	if ShouldNotify(nothing, "C1", "U2", mention, "channel") {
		t.Error("level nothing: mention should not notify")
	}
	if ShouldNotify(nothing, "C1", "U2", "deploy now", "channel") {
		t.Error("level nothing: keyword should not notify")
	}

	everything := base
	everything.ChannelLevel = LevelEverything
	if !ShouldNotify(everything, "C1", "U2", plain, "channel") {
		t.Error("level everything: plain message should notify")
	}
	everything.ActiveChannelID = "C1"
	if ShouldNotify(everything, "C1", "U2", plain, "channel") {
		t.Error("level everything: the channel on screen should stay quiet")
	}

	mentions := base
	mentions.ChannelLevel = LevelMentions
	mentions.OnMention = false // the channel's own choice wins over the global toggle
	if ShouldNotify(mentions, "C1", "U2", plain, "channel") {
		t.Error("level mentions_dms: plain message should not notify")
	}
	if !ShouldNotify(mentions, "C1", "U2", mention, "channel") {
		t.Error("level mentions_dms: mention should notify")
	}
	if !ShouldNotify(mentions, "C1", "U2", "deploy now", "channel") {
		t.Error("level mentions_dms: keyword should notify")
	}

	// A followed thread still notifies in a channel set to nothing.
	thread := nothing
	thread.OnThread = true
	thread.ThreadFollowed = true
	if !ShouldNotify(thread, "C1", "U2", plain, "channel") {
		t.Error("level nothing: followed thread reply should notify")
	}
}

func TestNotifyWithImage_RunsHelper(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "args")
	helper := filepath.Join(dir, "slk-notifier")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >" + out + "\n"
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("__CFBundleIdentifier", "com.example.term")
	n := &Notifier{enabled: true, helper: helper}
	if err := n.NotifyWithImage("Alice", "hi", "/tmp/a.png"); err != nil {
		t.Fatalf("NotifyWithImage: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := "--title\nAlice\n--body\nhi\n--focus-title\nslk\n--image\n/tmp/a.png\n--activate\ncom.example.term\n"
	if string(got) != want {
		t.Errorf("helper args:\n%s\nwant:\n%s", got, want)
	}
}

func TestNotifyWithImage_CommandSeesImage(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	n := New(true, "printf '%s' \"$SLK_IMAGE\" >"+out)
	if err := n.NotifyWithImage("t", "b", "/tmp/a.png"); err != nil {
		t.Fatalf("NotifyWithImage: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "/tmp/a.png" {
		t.Errorf("$SLK_IMAGE = %q, want /tmp/a.png", got)
	}
}
