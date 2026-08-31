// internal/ui/reducer_mouse_test.go
package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ui/help"
	"github.com/agustif/slk/internal/ui/sidebar"
	"github.com/agustif/slk/internal/ui/wintree"
)

// TestReduceMouseWheel_ScrollsActiveModal verifies that when a modal
// overlay is open, a mouse-wheel notch scrolls the items inside the
// modal (advancing its selection by mouseWheelLines) instead of
// scrolling the panel under the cursor on the main tab behind it.
func TestReduceMouseWheel_ScrollsActiveModal(t *testing.T) {
	app := NewApp()

	// Populate the help modal with enough rows to scroll through.
	entries := make([]help.Entry, 0, 20)
	for i := 0; i < 20; i++ {
		entries = append(entries, help.Entry{Key: "k", Desc: "desc"})
	}
	app.help.SetEntries(entries)
	app.help.Open()
	app.SetMode(ModeHelp)

	if got := app.help.Selected(); got != 0 {
		t.Fatalf("precondition: help selection should start at 0, got %d", got)
	}

	// A wheel-down notch (X anywhere on screen) should move the modal
	// selection down by mouseWheelLines (default 3), not touch panels.
	reduceMouseWheel(app, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5})
	if got, want := app.help.Selected(), app.mouseWheelLines; got != want {
		t.Fatalf("wheel down: help selection = %d, want %d", got, want)
	}

	// A wheel-up notch should move the selection back up, clamping at 0.
	reduceMouseWheel(app, tea.MouseWheelMsg{Button: tea.MouseWheelUp, X: 5})
	if got := app.help.Selected(); got != 0 {
		t.Fatalf("wheel up: help selection = %d, want 0", got)
	}
}

// TestReduceMouseWheel_NoModalLeavesModalUntouched is a guard that the
// modal-routing branch only fires when a modal mode is active: with the
// app in normal mode, a wheel notch must not advance the (open) help
// modal's selection through the modal path.
func TestReduceMouseWheel_NoModalLeavesModalUntouched(t *testing.T) {
	app := NewApp()

	entries := make([]help.Entry, 0, 20)
	for i := 0; i < 20; i++ {
		entries = append(entries, help.Entry{Key: "k", Desc: "desc"})
	}
	app.help.SetEntries(entries)
	// Note: NOT opening the modal / not setting ModeHelp; mode stays Normal.

	reduceMouseWheel(app, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5})
	if got := app.help.Selected(); got != 0 {
		t.Fatalf("normal mode: help selection should stay 0, got %d", got)
	}
}

// TestReduceMouseClick_IgnoredInMessagesRegionWhenSplit pins the
// interim Phase 2 guard: with multiple windows, PanelAt still maps
// the whole messages region to the single live pane, so a click on a
// placeholder window would begin a drag selection in the live channel
// at bogus coordinates. Clicks (and therefore drags, which can only
// start from the click branch) must be swallowed until per-window
// mouse routing lands in Phase 4.
func TestReduceMouseClick_IgnoredInMessagesRegionWhenSplit(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.width = 200 // wide enough for two ≥42-col windows (wintree.MinWidth)
	if cmd := a.splitWindow(wintree.SplitSideBySide); cmd != nil {
		t.Fatal("split refused at width 200")
	}
	_ = a.View() // recompute layout bands for the split layout

	pressX := a.layout.sidebarEnd + 2
	pressY := 4
	_, _ = a.Update(tea.MouseClickMsg{X: pressX, Y: pressY, Button: tea.MouseLeft})
	_, _ = a.Update(tea.MouseMotionMsg{X: pressX + 10, Y: pressY + 1, Button: tea.MouseLeft})
	_, _ = a.Update(tea.MouseReleaseMsg{X: pressX + 10, Y: pressY + 1, Button: tea.MouseLeft})
	if a.messagepane.HasSelection() {
		t.Fatal("split mode: click+drag in the messages region must not start a selection")
	}
}

func TestRegisterSidebarHeaderClick_DoubleClickToggles(t *testing.T) {
	a := NewApp()
	if a.registerSidebarHeaderClick("Channels") {
		t.Fatal("first click should not toggle")
	}
	if !a.registerSidebarHeaderClick("Channels") {
		t.Fatal("second click within the window should toggle")
	}
	if a.registerSidebarHeaderClick("Channels") {
		t.Fatal("third click starts a new sequence and must not toggle")
	}
}

func TestRegisterSidebarHeaderClick_DifferentHeaderIsNotDouble(t *testing.T) {
	a := NewApp()
	a.registerSidebarHeaderClick("Channels")
	if a.registerSidebarHeaderClick("Direct Messages") {
		t.Fatal("switching headers is not a double-click")
	}
}

func TestRegisterSidebarHeaderClick_ExpiredWindow(t *testing.T) {
	a := NewApp()
	a.registerSidebarHeaderClick("Channels")
	a.sidebarHeaderClick.at = time.Now().Add(-time.Second)
	if a.registerSidebarHeaderClick("Channels") {
		t.Fatal("click after the double-click window should not toggle")
	}
}

func clickSidebarSectionHeader(t *testing.T, a *App, header string) {
	t.Helper()
	_ = a.View()
	w := a.sidebar.Width()
	h := a.layout.PageHeight(PanelSidebar)
	if h < 12 {
		h = 40
	}
	view := a.sidebar.View(h, w)
	y := -1
	for i, l := range strings.Split(view, "\n") {
		if !strings.Contains(l, header) {
			continue
		}
		if strings.Contains(l, "▸") || strings.Contains(l, "▾") {
			y = i
			break
		}
	}
	if y < 0 {
		t.Fatalf("section header %q not found in sidebar:\n%s", header, view)
	}
	x := a.layout.RailWidth() + 2
	reduceMouseClick(a, tea.MouseClickMsg{X: x, Y: y + 1, Button: tea.MouseLeft})
}

func TestReduceMouseClick_DoubleClickSectionHeaderTogglesCollapse(t *testing.T) {
	a := NewApp()
	a.width = 120
	a.height = 30
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "D1", Name: "alice", Type: "dm"},
	})
	if a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("precondition: Direct Messages starts expanded")
	}

	clickSidebarSectionHeader(t, a, "Direct Messages")
	name, ok := a.sidebar.IsSectionHeaderSelected()
	if !ok || name != "Direct Messages" {
		t.Fatalf("first click should select the header, got name=%q ok=%v", name, ok)
	}
	if a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("single click must not collapse the section")
	}

	clickSidebarSectionHeader(t, a, "Direct Messages")
	if !a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("double-click should collapse Direct Messages")
	}
	out := a.sidebar.View(20, 30)
	if strings.Contains(out, "alice") {
		t.Errorf("collapsed DM section should hide alice:\n%s", out)
	}

	clickSidebarSectionHeader(t, a, "Direct Messages")
	if !a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("third click starts a new sequence and must not expand")
	}
	clickSidebarSectionHeader(t, a, "Direct Messages")
	if a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("second double-click should expand Direct Messages again")
	}
}

func TestReduceMouseClick_BlankRowDoesNotFinishHeaderDoubleClick(t *testing.T) {
	a := NewApp()
	a.width = 120
	a.height = 30
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "D1", Name: "alice", Type: "dm"},
	})
	clickSidebarSectionHeader(t, a, "Direct Messages")
	_ = a.View()
	x := a.layout.RailWidth() + 2
	// Synth rows occupy 0-4; a blank spacer is inserted after Drafts.
	reduceMouseClick(a, tea.MouseClickMsg{X: x, Y: 6, Button: tea.MouseLeft})
	clickSidebarSectionHeader(t, a, "Direct Messages")
	if a.sidebar.IsCollapsed("Direct Messages") {
		t.Fatal("a spacer click must reset the double-click pair")
	}
}
