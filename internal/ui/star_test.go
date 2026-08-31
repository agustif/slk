package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/ui/help"
	"github.com/gammons/slk/internal/ui/sidebar"
)

func TestHelp_ListsStarBinding(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	found := false
	for _, e := range entries {
		if e.Key == "*" && e.Desc == "star/unstar channel" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("help entries missing {Key: \"*\", Desc: \"star/unstar channel\"}")
	}
}

func TestNormalMode_StarTogglesSelectedSidebarChannel(t *testing.T) {
	a := NewApp()
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	})
	a.sidebar.SelectByID("C1")
	var gotID string
	a.SetStarToggler(func(id string) (bool, []sidebar.ChannelItem, bool) {
		gotID = id
		return true, nil, true
	})

	a.handleNormalMode(tea.KeyPressMsg{Code: '*', Text: "*"})
	if gotID != "C1" {
		t.Fatalf("star toggle id = %q, want C1", gotID)
	}
	if !strings.Contains(a.statusbar.View(80), "Starred #general") {
		t.Fatalf("toast = %q, want Starred #general", a.statusbar.View(80))
	}
}

func TestNormalMode_StarUsesActiveChannelWhenMessagesFocused(t *testing.T) {
	a := NewApp()
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "C2", Name: "random", Type: "channel"},
	})
	a.sidebar.SelectByID("C2")
	a.focusedPanel = PanelMessages
	a.activeChannelID = "C1"
	var gotID string
	a.SetStarToggler(func(id string) (bool, []sidebar.ChannelItem, bool) {
		gotID = id
		return false, nil, true
	})

	a.handleNormalMode(tea.KeyPressMsg{Code: '*', Text: "*"})
	if gotID != "C1" {
		t.Fatalf("star toggle id = %q, want C1 (active channel, not sidebar cursor)", gotID)
	}
	if !strings.Contains(a.statusbar.View(80), "Unstarred #general") {
		t.Fatalf("toast = %q, want Unstarred #general", a.statusbar.View(80))
	}
}

func TestNormalMode_StarNoChannelToasts(t *testing.T) {
	a := NewApp()
	a.SetStarToggler(func(id string) (bool, []sidebar.ChannelItem, bool) {
		t.Fatalf("star toggle should not run; got id %q", id)
		return false, nil, true
	})
	a.handleNormalMode(tea.KeyPressMsg{Code: '*', Text: "*"})
	if !strings.Contains(a.statusbar.View(80), "No channel selected") {
		t.Fatalf("toast = %q, want No channel selected", a.statusbar.View(80))
	}
}
