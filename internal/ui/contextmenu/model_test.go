package contextmenu

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func sampleItems() []Item {
	return []Item{
		{Label: "Add reaction", Action: ActionAddReaction, Enabled: true},
		{Label: "Reply in thread", Action: ActionReplyInThread, Enabled: true},
		{Label: "Copy permalink", Action: ActionCopyPermalink, Enabled: true},
		{Label: "Download file", Action: ActionDownloadFile, Enabled: false},
		{Label: "Open links", Action: ActionOpenLinks, Enabled: false},
		{Label: "Edit message", Action: ActionEdit, Enabled: false},
		{Label: "Delete message", Action: ActionDelete, Enabled: false},
		{Label: "Mark unread", Action: ActionMarkUnread, Enabled: true},
		{Label: "List reactions", Action: ActionListReactions, Enabled: false},
	}
}

func TestHandleKey_EnterReturnsFirstEnabled(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	it := m.HandleKey("enter")
	if it == nil {
		t.Fatal("enter on first enabled row should return an item")
	}
	if it.Action != ActionAddReaction {
		t.Errorf("got %q, want add_reaction", it.Action)
	}
	if m.IsVisible() {
		t.Error("menu should close after enter")
	}
}

func TestHandleKey_EscCloses(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	if m.HandleKey("esc") != nil {
		t.Fatal("esc should not return an item")
	}
	if m.IsVisible() {
		t.Error("menu should close on esc")
	}
}

func TestHandleKey_JKSkipsDisabled(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	// 0 Add, 1 Reply, 2 Permalink, 3-6 disabled, 7 Mark unread
	m.HandleKey("j")
	m.HandleKey("j")
	m.HandleKey("j")
	if m.selected != 7 {
		t.Errorf("j should skip disabled rows, selected=%d want 7", m.selected)
	}
	it := m.HandleKey("enter")
	if it == nil || it.Action != ActionMarkUnread {
		t.Fatalf("expected mark_unread, got %+v", it)
	}
}

func TestHandleKey_EnterOnDisabledIsNoop(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	m.selected = 3 // Download file, disabled
	if m.HandleKey("enter") != nil {
		t.Fatal("enter on disabled row should not activate")
	}
	if !m.IsVisible() {
		t.Error("menu should stay open")
	}
}

func TestBoxSizeMatchesRender(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	w, h := m.BoxSize(80, 24)
	box := m.renderBox(80)
	if gw := lipgloss.Width(box); w != gw {
		t.Errorf("BoxSize width = %d, rendered width = %d", w, gw)
	}
	if gh := lipgloss.Height(box); h != gh {
		t.Errorf("BoxSize height = %d, rendered height = %d", h, gh)
	}
}

func TestBoxOriginCenteredWithoutAnchor(t *testing.T) {
	m := New()
	m.Open(sampleItems())
	w, h := m.BoxSize(80, 24)
	x, y := m.BoxOrigin(80, 24)
	if want := (80 - w) / 2; x != want {
		t.Errorf("origin x = %d, want %d", x, want)
	}
	if want := (24 - h) / 2; y != want {
		t.Errorf("origin y = %d, want %d", y, want)
	}
}

func TestBoxOriginNearAnchor(t *testing.T) {
	m := New()
	m.OpenAt(sampleItems(), 10, 5)
	x, y := m.BoxOrigin(80, 24)
	if x != 10 || y != 5 {
		t.Errorf("origin = (%d,%d), want (10,5)", x, y)
	}
}
