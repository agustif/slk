package presencemenu

import (
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestParseStatusInput(t *testing.T) {
	tests := []struct {
		in, emoji, text string
	}{
		{"", "", ""},
		{"  lunch  ", "", "lunch"},
		{":pizza: lunch", ":pizza:", "lunch"},
		{":pizza:", ":pizza:", ""},
		{":calendar_pad: in a meeting", ":calendar_pad:", "in a meeting"},
		{":notclosed lunch", "", ":notclosed lunch"},
		{": : empty name", "", ": : empty name"},
		{":ok-hand: hi", ":ok-hand:", "hi"},
	}
	for _, tc := range tests {
		emoji, text := ParseStatusInput(tc.in)
		if emoji != tc.emoji || text != tc.text {
			t.Errorf("ParseStatusInput(%q) = (%q, %q), want (%q, %q)",
				tc.in, emoji, text, tc.emoji, tc.text)
		}
	}
}

func TestStatusExpirationUnix(t *testing.T) {
	now := time.Date(2026, 8, 31, 15, 4, 0, 0, time.FixedZone("X", -4*3600))
	if got := StatusExpirationUnix("30m", now); got != now.Add(30*time.Minute).Unix() {
		t.Errorf("30m = %d", got)
	}
	if got := StatusExpirationUnix("1h", now); got != now.Add(time.Hour).Unix() {
		t.Errorf("1h = %d", got)
	}
	if got := StatusExpirationUnix("4h", now); got != now.Add(4*time.Hour).Unix() {
		t.Errorf("4h = %d", got)
	}
	if got := StatusExpirationUnix("none", now); got != 0 {
		t.Errorf("none = %d, want 0", got)
	}
	wantToday := time.Date(2026, 9, 1, 0, 0, 0, 0, now.Location()).Unix()
	if got := StatusExpirationUnix("today", now); got != wantToday {
		t.Errorf("today = %d, want %d", got, wantToday)
	}
}

func TestSetStatusModel_EnterCommits(t *testing.T) {
	m := NewSetStatus()
	m.Open()
	if !m.IsVisible() {
		t.Fatal("expected visible")
	}
	for _, r := range ":pizza: focus" {
		m.HandleKey(string(r))
	}
	// Default duration is Don't clear.
	if m.DurationKind() != "none" {
		t.Errorf("default duration = %q, want none", m.DurationKind())
	}
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionSetStatus {
		t.Fatalf("result = %+v", r)
	}
	if r.StatusEmoji != ":pizza:" || r.StatusText != "focus" {
		t.Errorf("status = %q %q", r.StatusEmoji, r.StatusText)
	}
	if r.StatusExpiration != 0 {
		t.Errorf("expiration = %d, want 0", r.StatusExpiration)
	}
	if m.IsVisible() {
		t.Error("should close on enter")
	}
}

func TestSetStatusModel_EmptyEnterDoesNotCommit(t *testing.T) {
	m := NewSetStatus()
	m.Open()
	if r := m.HandleKey("enter"); r != nil {
		t.Errorf("empty enter = %+v, want nil", r)
	}
	if !m.IsVisible() {
		t.Error("should stay open")
	}
}

func TestSetStatusModel_DurationCycle(t *testing.T) {
	m := NewSetStatus()
	m.Open()
	m.HandleKey("up") // from last (none) to today
	if m.DurationKind() != "today" {
		t.Errorf("after up = %q, want today", m.DurationKind())
	}
	m.HandleKey("down")
	if m.DurationKind() != "none" {
		t.Errorf("after down = %q, want none", m.DurationKind())
	}
}

func TestSetStatusModel_BoxSizeMatchesRender(t *testing.T) {
	m := NewSetStatus()
	m.Open()
	w, h := m.BoxSize(80, 24)
	box := m.renderBox(80)
	if gw := lipgloss.Width(box); w != gw {
		t.Errorf("BoxSize width = %d, rendered width = %d", w, gw)
	}
	if gh := lipgloss.Height(box); h != gh {
		t.Errorf("BoxSize height = %d, rendered height = %d", h, gh)
	}
}

func TestSetStatusModel_ClickRow(t *testing.T) {
	m := NewSetStatus()
	m.Open()
	if !m.ClickRow(80, 24, setStatusListTopOffset) {
		t.Fatal("first duration row should hit")
	}
	if m.DurationKind() != "30m" {
		t.Errorf("kind = %q, want 30m", m.DurationKind())
	}
	if m.ClickRow(80, 24, setStatusListTopOffset-1) {
		t.Error("above list should miss")
	}
}
