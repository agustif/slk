package userprofile

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	slackclient "github.com/agustif/slk/internal/slack"
)

func TestOpenClose(t *testing.T) {
	m := New()
	if m.IsVisible() {
		t.Fatal("hidden by default")
	}
	m.Open(Profile{UserID: "U1", DisplayName: "Ada"})
	if !m.IsVisible() || m.UserID() != "U1" {
		t.Fatalf("open: visible=%v id=%q", m.IsVisible(), m.UserID())
	}
	m.HandleKey("esc")
	if m.IsVisible() {
		t.Error("esc should close")
	}
}

func TestEnterMessagesOtherUser(t *testing.T) {
	m := New()
	m.Open(Profile{UserID: "U1", DisplayName: "Ada"})
	r := m.HandleKey("enter")
	if r == nil || !r.Message {
		t.Fatalf("enter = %+v, want Message", r)
	}
	if m.IsVisible() {
		t.Error("should close after Message")
	}
}

func TestEnterDoesNotMessageSelf(t *testing.T) {
	m := New()
	m.Open(Profile{UserID: "U1", DisplayName: "Me", IsSelf: true})
	if r := m.HandleKey("enter"); r != nil {
		t.Errorf("self enter = %+v, want nil", r)
	}
	if !m.IsVisible() {
		t.Error("should stay open")
	}
}

func TestSetProfileSameUser(t *testing.T) {
	m := New()
	m.Open(Profile{UserID: "U1", DisplayName: "Ada", Loading: true})
	m.SetProfile(Profile{UserID: "U1", DisplayName: "Ada Lovelace", Title: "Engineer", Loading: false})
	if m.Profile().Title != "Engineer" {
		t.Errorf("title = %q", m.Profile().Title)
	}
	m.SetProfile(Profile{UserID: "U2", DisplayName: "Bob"})
	if m.Profile().UserID != "U1" {
		t.Error("should ignore a different user while open")
	}
}

func TestRenderIncludesFields(t *testing.T) {
	m := New()
	m.Open(Profile{
		UserID:      "U1",
		DisplayName: "Ada",
		RealName:    "Ada Lovelace",
		Title:       "Mathematician",
		Handle:      "ada",
		Status:      slackclient.UserStatus{Text: "crunching numbers", Emoji: ":abacus:"},
		TZ:          "America/New_York",
		TZLabel:     "EDT",
		TZOffset:    -4 * 3600,
		Presence:    "active",
	})
	box := m.renderBox(80)
	for _, want := range []string{"Ada", "Ada Lovelace", "Mathematician", "@ada", "Active", "Message"} {
		if !strings.Contains(box, want) {
			t.Errorf("box missing %q:\n%s", want, box)
		}
	}
}

func TestFormatStatusExpiredHidden(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	st := slackclient.UserStatus{Text: "gone", Emoji: ":wave:", Expiration: now.Unix() - 1}
	if got := formatStatus(st, now); got != "" {
		t.Errorf("expired status = %q, want empty", got)
	}
	st.Expiration = now.Unix() + 60
	if got := formatStatus(st, now); got == "" {
		t.Error("active status should render")
	}
}

func TestBoxSizeMatchesRender(t *testing.T) {
	m := New()
	m.Open(Profile{UserID: "U1", DisplayName: "Ada"})
	w, h := m.BoxSize(80, 24)
	box := m.renderBox(80)
	if gw := lipgloss.Width(box); w != gw {
		t.Errorf("width BoxSize=%d render=%d", w, gw)
	}
	if gh := lipgloss.Height(box); h != gh {
		t.Errorf("height BoxSize=%d render=%d", h, gh)
	}
}

func TestFormatLocalTime(t *testing.T) {
	now := time.Date(2026, 8, 31, 19, 4, 0, 0, time.UTC)
	got := formatLocalTime("America/New_York", "EDT", -4*3600, now)
	if !strings.Contains(got, "Local time") || !strings.Contains(got, "EDT") {
		t.Errorf("local time = %q", got)
	}
	if got := formatLocalTime("", "", 0, now); got != "" {
		t.Errorf("no tz = %q, want empty", got)
	}
}
