package datemenu

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestModel_EnterReturnsQuery(t *testing.T) {
	m := New()
	m.Open()
	if !m.IsVisible() {
		t.Fatal("expected visible")
	}
	for _, r := range "2024-01-15" {
		m.HandleKey(string(r))
	}
	if m.Query() != "2024-01-15" {
		t.Fatalf("query = %q", m.Query())
	}
	r := m.HandleKey("enter")
	if r == nil || r.Query != "2024-01-15" {
		t.Fatalf("enter = %+v", r)
	}
	if !m.IsVisible() {
		t.Error("enter should leave the overlay open for the caller to close")
	}
}

func TestModel_SpaceAppends(t *testing.T) {
	m := New()
	m.Open()
	m.HandleKey("2")
	m.HandleKey("0")
	m.HandleKey("2")
	m.HandleKey("4")
	m.HandleKey("-")
	m.HandleKey("0")
	m.HandleKey("1")
	m.HandleKey("-")
	m.HandleKey("1")
	m.HandleKey("5")
	m.HandleKey("space")
	m.HandleKey("1")
	m.HandleKey("4")
	m.HandleKey(":")
	m.HandleKey("3")
	m.HandleKey("0")
	if m.Query() != "2024-01-15 14:30" {
		t.Fatalf("query = %q", m.Query())
	}
}

func TestModel_EscapeCloses(t *testing.T) {
	m := New()
	m.Open()
	m.HandleKey("2")
	if r := m.HandleKey("esc"); r != nil {
		t.Errorf("esc result = %+v, want nil", r)
	}
	if m.IsVisible() {
		t.Error("expected closed after esc")
	}
	if m.Query() != "" {
		t.Errorf("query = %q after close", m.Query())
	}
}

func TestModel_BackspaceEdits(t *testing.T) {
	m := New()
	m.Open()
	m.HandleKey("2")
	m.HandleKey("0")
	m.HandleKey("backspace")
	if m.Query() != "2" {
		t.Fatalf("query = %q, want 2", m.Query())
	}
}

func TestBoxSizeMatchesRender(t *testing.T) {
	m := New()
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

func TestViewOverlay_HiddenIsBackground(t *testing.T) {
	m := New()
	if got := m.ViewOverlay(40, 12, "bg"); got != "bg" {
		t.Errorf("hidden overlay = %q, want background", got)
	}
	m.Open()
	got := m.ViewOverlay(80, 24, "bg")
	if !strings.Contains(got, "Jump to date") {
		t.Errorf("visible overlay missing title:\n%s", got)
	}
}

func TestClickRowNeverHits(t *testing.T) {
	m := New()
	m.Open()
	if m.ClickRow(80, 24, 4) {
		t.Error("date overlay has no list rows")
	}
}
