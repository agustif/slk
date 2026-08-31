package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/help"
	"github.com/gammons/slk/internal/ui/sidebar"
)

func setupMuteSidebar(t *testing.T, a *App, items []sidebar.ChannelItem) {
	t.Helper()
	a.SetChannels(items)
	if len(items) > 0 {
		a.sidebar.SelectByID(items[0].ID)
	}
}

func TestToggleMute_SidebarFocusedMutesSelectedChannel(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelSidebar
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "Eng"},
		{ID: "C2", Name: "random", Type: "channel", Section: "Eng"},
	})
	a.sidebar.SelectByID("C2")
	a.activeChannelID = "C1"

	cmd := a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("`m` should dispatch a persist cmd")
	}
	item, ok := a.sidebar.SelectedItem()
	if !ok || item.ID != "C2" {
		t.Fatalf("sidebar selection drifted: %+v ok=%v", item, ok)
	}
	if !item.IsMuted {
		t.Fatal("expected C2 muted after toggle")
	}
	if !strings.Contains(a.statusbar.View(80), "Muted #random") {
		t.Fatalf("expected 'Muted #random' toast; got %q", a.statusbar.View(80))
	}
}

func TestToggleMute_MessagesFocusedMutesActiveChannel(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "Eng"},
		{ID: "C2", Name: "random", Type: "channel", Section: "Eng"},
	})
	a.sidebar.SelectByID("C2")
	a.activeChannelID = "C1"

	_ = a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	var mutedC1, mutedC2 bool
	for _, it := range a.sidebar.Items() {
		if it.ID == "C1" {
			mutedC1 = it.IsMuted
		}
		if it.ID == "C2" {
			mutedC2 = it.IsMuted
		}
	}
	if !mutedC1 {
		t.Fatal("expected active channel C1 muted")
	}
	if mutedC2 {
		t.Fatal("selected-but-unfocused C2 should not mute")
	}
	if !strings.Contains(a.statusbar.View(80), "Muted #general") {
		t.Fatalf("expected 'Muted #general' toast; got %q", a.statusbar.View(80))
	}
}

func TestToggleMute_SidebarHeaderFallsThroughToActiveChannel(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelSidebar
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "Eng"},
	})
	a.activeChannelID = "C1"
	a.sidebar.SelectActivityRow()
	if _, ok := a.sidebar.SelectedItem(); ok {
		t.Fatal("precondition: cursor should not be on a channel")
	}

	_ = a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	var muted bool
	for _, it := range a.sidebar.Items() {
		if it.ID == "C1" {
			muted = it.IsMuted
		}
	}
	if !muted {
		t.Fatal("Activity-row `m` should mute the active channel")
	}
}

func TestToggleMute_UnmutesWhenAlreadyMuted(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelSidebar
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "C1", Name: "noisy", Type: "channel", Section: "Eng", IsMuted: true},
	})
	a.sidebar.SelectByID("C1")

	_ = a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	item, ok := a.sidebar.SelectedItem()
	if !ok || item.IsMuted {
		t.Fatalf("expected unmute, got %+v ok=%v", item, ok)
	}
	if !strings.Contains(a.statusbar.View(80), "Unmuted #noisy") {
		t.Fatalf("expected 'Unmuted #noisy' toast; got %q", a.statusbar.View(80))
	}
}

func TestToggleMute_DMToastOmitsHash(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelSidebar
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", Section: "Direct Messages"},
	})
	a.sidebar.SelectByID("D1")

	_ = a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if !strings.Contains(a.statusbar.View(80), "Muted alice") {
		t.Fatalf("expected 'Muted alice' toast; got %q", a.statusbar.View(80))
	}
	if strings.Contains(a.statusbar.View(80), "#alice") {
		t.Fatalf("DM toast should not use #: %q", a.statusbar.View(80))
	}
}

func TestToggleMute_NoTargetNoOps(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.activeChannelID = ""

	cmd := a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd != nil {
		t.Fatal("expected no-op with no channel")
	}
}

func TestToggleMute_DispatchesSetMuted(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelSidebar
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "Eng"},
	})
	a.sidebar.SelectByID("C1")

	var gotID ids.ChannelID
	var gotMuted bool
	a.setChannelMutedForTest(func(channelID ids.ChannelID, muted bool) tea.Msg {
		gotID, gotMuted = channelID, muted
		return ChannelMutedMsg{ChannelID: string(channelID), Muted: muted}
	})

	cmd := a.handleNormalMode(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("expected persist cmd")
	}
	if !invokeMutePersist(t, cmd) {
		t.Fatal("SetMuted was not invoked from the returned cmd")
	}
	if gotID != "C1" || !gotMuted {
		t.Fatalf("SetMuted(%q, %v), want (C1, true)", gotID, gotMuted)
	}
}

func TestChannelMutedMsg_ErrorRollsBackAndToasts(t *testing.T) {
	a := NewApp()
	setupMuteSidebar(t, a, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "Eng", IsMuted: true},
	})

	_, cmd := a.Update(ChannelMutedMsg{ChannelID: "C1", Muted: true, Err: errors.New("boom")})
	if cmd == nil {
		t.Fatal("expected failure toast cmd")
	}
	var stillMuted bool
	for _, it := range a.sidebar.Items() {
		if it.ID == "C1" {
			stillMuted = it.IsMuted
		}
	}
	if stillMuted {
		t.Fatal("sidebar should roll back mute on write error")
	}
	if !strings.Contains(a.statusbar.View(80), "Mute failed") {
		t.Fatalf("expected Mute failed toast; got %q", a.statusbar.View(80))
	}
}

func TestHelp_ListsMuteToggle(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	for _, e := range entries {
		if e.Key == "m" && e.Desc == "mute channel" {
			return
		}
	}
	t.Fatal("help entries missing {Key: \"m\", Desc: \"mute channel\"}")
}

func TestMuteToastFormat(t *testing.T) {
	if got := muteToast(true, "general", "channel"); got != "Muted #general" {
		t.Errorf("channel mute: %q", got)
	}
	if got := muteToast(false, "ops", "private"); got != "Unmuted #ops" {
		t.Errorf("private unmute: %q", got)
	}
	if got := muteToast(true, "alice", "dm"); got != "Muted alice" {
		t.Errorf("dm mute: %q", got)
	}
}

// invokeMutePersist runs cmd, recursing into BatchMsg, and returns
// true if ChannelService.SetMuted ran. tea.Tick (toast auto-clear)
// is skipped via a short timeout so the test does not wait 2s.
func invokeMutePersist(t *testing.T, cmd tea.Cmd) bool {
	t.Helper()
	if cmd == nil {
		return false
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(100 * time.Millisecond):
		return false
	}
	switch m := msg.(type) {
	case ChannelMutedMsg:
		return true
	case tea.BatchMsg:
		for _, sub := range m {
			if invokeMutePersist(t, sub) {
				return true
			}
		}
	}
	return false
}
