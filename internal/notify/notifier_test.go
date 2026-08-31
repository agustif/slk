package notify

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestShouldNotify_SuppressedByQuietHours(t *testing.T) {
	ctx := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      false, // would otherwise notify
		OnDM:            true,
		OnMention:       true,
		OnKeyword:       []string{"deploy"},
		IsQuiet:         true,
	}
	if ShouldNotify(ctx, "C1", "U2", "hey <@U1> deploy", "dm") {
		t.Error("quiet hours should suppress notifications regardless of triggers")
	}
}

func clock(hour, min int) time.Time {
	return time.Date(2026, 6, 15, hour, min, 0, 0, time.Local)
}

func TestInQuietHours(t *testing.T) {
	tests := []struct {
		name string
		spec string
		now  time.Time
		want bool
	}{
		// same-day window 09:00-17:00 (start inclusive, end exclusive)
		{"in-window", "09:00-17:00", clock(12, 0), true},
		{"in-window start", "09:00-17:00", clock(9, 0), true},
		{"in-window last minute", "09:00-17:00", clock(16, 59), true},
		{"out-of-window end", "09:00-17:00", clock(17, 0), false},
		{"out-of-window before", "09:00-17:00", clock(8, 59), false},
		{"out-of-window evening", "09:00-17:00", clock(22, 0), false},

		// overnight wrap 22:00-08:00
		{"overnight evening", "22:00-08:00", clock(22, 0), true},
		{"overnight late", "22:00-08:00", clock(23, 30), true},
		{"overnight midnight", "22:00-08:00", clock(0, 0), true},
		{"overnight early", "22:00-08:00", clock(7, 59), true},
		{"overnight end", "22:00-08:00", clock(8, 0), false},
		{"overnight daytime", "22:00-08:00", clock(12, 0), false},
		{"overnight before start", "22:00-08:00", clock(21, 59), false},

		{"empty", "", clock(23, 0), false},
		{"whitespace", "   ", clock(23, 0), false},
		{"invalid garbage", "nope", clock(23, 0), false},
		{"invalid missing end", "22:00", clock(23, 0), false},
		{"invalid extra part", "22:00-08:00-09:00", clock(23, 0), false},
		{"invalid hour", "25:00-08:00", clock(23, 0), false},
		{"invalid minute", "22:60-08:00", clock(23, 0), false},
		{"invalid end hour", "22:00-24:00", clock(23, 0), false},
		{"invalid unpadded", "9:00-17:00", clock(12, 0), false},
		{"zero-width", "08:00-08:00", clock(8, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InQuietHours(tt.spec, tt.now); got != tt.want {
				t.Errorf("InQuietHours(%q, %s) = %v, want %v", tt.spec, tt.now.Format("15:04"), got, tt.want)
			}
		})
	}
}

func TestShouldNotify_QuietHoursWindow(t *testing.T) {
	base := NotifyContext{
		CurrentUserID:   "U1",
		ActiveChannelID: "C_OTHER",
		IsActiveWS:      true,
		OnDM:            true,
	}

	in := base
	in.IsQuiet = InQuietHours("22:00-08:00", clock(23, 0))
	if ShouldNotify(in, "C1", "U2", "hello", "dm") {
		t.Error("in-window quiet hours should suppress")
	}

	out := base
	out.IsQuiet = InQuietHours("22:00-08:00", clock(12, 0))
	if !ShouldNotify(out, "C1", "U2", "hello", "dm") {
		t.Error("out-of-window quiet hours should still notify")
	}

	empty := base
	empty.IsQuiet = InQuietHours("", clock(23, 0))
	if !ShouldNotify(empty, "C1", "U2", "hello", "dm") {
		t.Error("empty quiet_hours is disabled")
	}

	invalid := base
	invalid.IsQuiet = InQuietHours("not-a-window", clock(23, 0))
	if !ShouldNotify(invalid, "C1", "U2", "hello", "dm") {
		t.Error("invalid quiet_hours is disabled, not a crash")
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
