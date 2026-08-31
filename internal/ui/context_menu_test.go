package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ui/channelfinder"
	"github.com/agustif/slk/internal/ui/contextmenu"
	"github.com/agustif/slk/internal/ui/messages"
)

func openContextMenuOnMessage(t *testing.T) *App {
	t.Helper()
	a := newTestAppWithMessages(t)
	// openPickerFor no-ops without a channel; the fixture must look like
	// a real open conversation so Add reaction can hand off to the picker.
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	pressX := a.layout.sidebarEnd + 2
	pressY := 4
	_, _ = a.Update(tea.MouseClickMsg{X: pressX, Y: pressY, Button: tea.MouseRight})
	if a.mode != ModeContextMenu {
		t.Fatalf("right-click mode = %v, want ModeContextMenu", a.mode)
	}
	if !a.contextMenu.IsVisible() {
		t.Fatal("context menu should be visible after right-click")
	}
	return a
}

func TestMessageContextMenu_RightClickOpens(t *testing.T) {
	_ = openContextMenuOnMessage(t)
}

func TestMessageContextMenu_EscCloses(t *testing.T) {
	a := openContextMenuOnMessage(t)
	_, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if a.mode != ModeNormal {
		t.Errorf("mode = %v after Esc, want ModeNormal", a.mode)
	}
	if a.contextMenu.IsVisible() {
		t.Error("context menu should close on Esc")
	}
}

func TestMessageContextMenu_IncludesLaterPinFollow(t *testing.T) {
	a := openContextMenuOnMessage(t)
	got := map[contextmenu.ActionID]contextmenu.Item{}
	for _, it := range a.contextMenu.Items() {
		got[it.Action] = it
	}
	for _, want := range []contextmenu.ActionID{
		contextmenu.ActionSaveForLater,
		contextmenu.ActionRemind,
		contextmenu.ActionShare,
		contextmenu.ActionPin,
		contextmenu.ActionStar,
		contextmenu.ActionFollowThread,
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing action %q", want)
		}
	}
	if it := got[contextmenu.ActionFollowThread]; it.Enabled {
		t.Error("Follow thread should be disabled in the channel pane")
	}
	if it := got[contextmenu.ActionSaveForLater]; it.Label != "Save for later" {
		t.Errorf("later label = %q, want Save for later", it.Label)
	}
}

func TestMessageContextMenu_EnterAddReactionOpensPicker(t *testing.T) {
	a := openContextMenuOnMessage(t)
	if got := a.contextMenu.Selected(); got != 0 {
		t.Fatalf("precondition: selected=%d, want 0 (Add reaction)", got)
	}
	items := a.contextMenu.Items()
	if len(items) == 0 || items[0].Action != contextmenu.ActionAddReaction {
		t.Fatalf("first item = %+v, want Add reaction", items)
	}
	_, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.mode != ModeReactionPicker {
		t.Errorf("mode = %v after Enter, want ModeReactionPicker", a.mode)
	}
	if !a.reactionPicker.IsVisible() {
		t.Error("reaction picker should be visible")
	}
	if a.contextMenu.IsVisible() {
		t.Error("context menu should close when opening the reaction picker")
	}
}

func TestMessageContextMenu_XKeyOpens(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.focusedPanel = PanelMessages
	_, _ = a.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if a.mode != ModeContextMenu {
		t.Errorf("mode = %v after x, want ModeContextMenu", a.mode)
	}
	if !a.contextMenu.IsVisible() {
		t.Error("context menu should be visible after x")
	}
}

func TestMessageContextMenu_XKeyFromThreadPane(t *testing.T) {
	a := NewApp()
	a.width = 120
	a.height = 30
	parent := messages.MessageItem{TS: "1.0", UserName: "alice", Text: "parent"}
	a.threadPanel.SetThread(parent, []messages.MessageItem{
		parent,
		{TS: "2.0", UserName: "bob", Text: "reply"},
	}, "C1", "1.0")
	a.threadVisible = true
	a.focusedPanel = PanelThread
	_, _ = a.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if a.mode != ModeContextMenu {
		t.Errorf("mode = %v after x on thread, want ModeContextMenu", a.mode)
	}
	if !a.contextMenu.IsVisible() {
		t.Error("context menu should open from the thread pane")
	}
	var followEnabled bool
	for _, it := range a.contextMenu.Items() {
		if it.Action == contextmenu.ActionFollowThread {
			followEnabled = it.Enabled
		}
	}
	if !followEnabled {
		t.Error("Follow thread should be enabled in the thread pane")
	}
}

func TestMessageContextMenu_ShareOpensFinder(t *testing.T) {
	a := openContextMenuOnMessage(t)
	a.SetChannelFinderItems([]channelfinder.Item{
		{ID: "C2", Name: "random", Type: "channel", Joined: true, LastVisited: 100},
	})
	found := false
	for _, it := range a.contextMenu.Items() {
		if it.Action == contextmenu.ActionShare && it.Enabled {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Share missing from message actions menu")
	}
	cmd := a.dispatchContextMenuAction(contextmenu.ActionShare)
	if cmd != nil {
		t.Fatalf("openSharePicker should be sync, got cmd %T", cmd)
	}
	if a.mode != ModeShare {
		t.Errorf("mode = %v, want ModeShare", a.mode)
	}
	if !a.channelFinder.IsVisible() || !a.channelFinder.ShareMode() {
		t.Fatal("share picker should be visible in share mode")
	}
	if !strings.Contains(a.channelFinder.View(80), "Share to...") {
		t.Errorf("finder title missing Share to...:\n%s", a.channelFinder.View(80))
	}
}

func TestMessageContextMenu_LeftClickDoesNotOpen(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.focusedPanel = PanelMessages
	pressX := a.layout.sidebarEnd + 2
	pressY := 4
	_, _ = a.Update(tea.MouseClickMsg{X: pressX, Y: pressY, Button: tea.MouseLeft})
	if a.mode == ModeContextMenu || a.contextMenu.IsVisible() {
		t.Fatal("left-click must not open the context menu")
	}
}

func TestMessageContextMenu_OutsideClickDismisses(t *testing.T) {
	a := openContextMenuOnMessage(t)
	_, _ = a.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	if a.mode != ModeNormal {
		t.Errorf("mode = %v after outside click, want ModeNormal", a.mode)
	}
	if a.contextMenu.IsVisible() {
		t.Error("context menu should close on outside click")
	}
}
