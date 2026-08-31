package sidebar

import (
	"strings"
	"testing"
	"time"

	slackclient "github.com/gammons/slk/internal/slack"
)

func TestDMStatusSuffix_ActiveEmoji(t *testing.T) {
	m := New([]ChannelItem{{ID: "D1", Name: "alice", Type: "dm", DMUserID: "U1"}})
	m.nowFn = func() time.Time { return time.Unix(1_700_000_000, 0) }
	m.UpdateUserStatus("U1", slackclient.UserStatus{Emoji: ":pizza:", Text: "lunch"})
	suf, w := m.dmStatusSuffix(m.items[0], 20)
	if w == 0 || suf == "" {
		t.Fatalf("expected a status suffix, got %q w=%d", suf, w)
	}
	if !strings.Contains(suf, " ") {
		t.Errorf("suffix should be spaced off the name: %q", suf)
	}
}

func TestDMStatusSuffix_ExpiredHidden(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	m := New([]ChannelItem{{ID: "D1", Name: "alice", Type: "dm", DMUserID: "U1"}})
	m.nowFn = func() time.Time { return now }
	m.UpdateUserStatus("U1", slackclient.UserStatus{
		Emoji:      ":palm_tree:",
		Text:       "ooo",
		Expiration: now.Unix() - 1,
	})
	if suf, w := m.dmStatusSuffix(m.items[0], 20); suf != "" || w != 0 {
		t.Errorf("expired status suffix = %q w=%d, want empty", suf, w)
	}
}

func TestDMStatusSuffix_TextWhenNoEmoji(t *testing.T) {
	m := New([]ChannelItem{{ID: "D1", Name: "alice", Type: "dm", DMUserID: "U1"}})
	m.UpdateUserStatus("U1", slackclient.UserStatus{Text: "focus"})
	suf, w := m.dmStatusSuffix(m.items[0], 30)
	if w == 0 || !strings.Contains(stripANSI(suf), "focus") {
		t.Errorf("text suffix = %q w=%d", suf, w)
	}
}

func TestDMStatusSuffix_SkipsChannels(t *testing.T) {
	m := New([]ChannelItem{{ID: "C1", Name: "general", Type: "channel"}})
	m.UpdateUserStatus("U1", slackclient.UserStatus{Text: "hi"})
	if suf, w := m.dmStatusSuffix(m.items[0], 20); suf != "" || w != 0 {
		t.Errorf("channel suffix = %q w=%d", suf, w)
	}
}

func TestSetUserStatuses_ReplacesMap(t *testing.T) {
	m := New([]ChannelItem{{ID: "D1", Name: "alice", Type: "dm", DMUserID: "U1"}})
	m.SetUserStatuses(map[string]slackclient.UserStatus{
		"U1": {Text: "one"},
	})
	st, ok := m.StatusForUser("U1")
	if !ok || st.Text != "one" {
		t.Fatalf("got %+v ok=%v", st, ok)
	}
	m.ResetUserStatuses()
	if _, ok := m.StatusForUser("U1"); ok {
		t.Error("ResetUserStatuses should drop the map")
	}
}

func stripANSI(s string) string {
	out := make([]rune, 0, len(s))
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				in = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
