package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/help"
	"github.com/agustif/slk/internal/ui/sidebar"
)

type testSectionsProvider struct {
	ready bool
	secs  []sidebar.SectionMeta
}

func (p testSectionsProvider) Ready() bool { return p.ready }
func (p testSectionsProvider) OrderedSlackSections() []sidebar.SectionMeta {
	return p.secs
}

func readyMoveApp() *App {
	a := NewApp()
	a.activeChannelID = "C1"
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel", Section: "L2"},
	})
	a.sidebar.SetSectionsProvider(testSectionsProvider{
		ready: true,
		secs: []sidebar.SectionMeta{
			{ID: "L1", Name: "Engineering", Type: "standard"},
			{ID: "L2", Name: "Channels", Type: "channels"},
			{ID: "L3", Name: "Starred", Type: "stars"},
			{ID: "L4", Name: "Direct Messages", Type: "direct_messages"},
		},
	})
	return a
}

func TestExecuteCommand_MoveOpensSectionPicker(t *testing.T) {
	a := readyMoveApp()
	_ = executeCommand(a, "move")
	if a.mode != ModeSectionPicker {
		t.Fatalf("mode = %v, want ModeSectionPicker", a.mode)
	}
	if !a.sectionPicker.IsVisible() {
		t.Fatal("picker not visible")
	}
	if a.sectionPicker.Title() != "Move to section" {
		t.Errorf("title = %q", a.sectionPicker.Title())
	}
	items := a.sectionPicker.Items()
	if len(items) != 3 {
		t.Fatalf("items = %+v, want Engineering/Channels/DMs (no Starred)", items)
	}
	if items[0].Label != "Engineering" || items[0].ID != "L1" {
		t.Errorf("item0 = %+v", items[0])
	}
	if items[1].Label != "Channels" || items[1].Detail != "current" {
		t.Errorf("item1 = %+v, want Channels marked current", items[1])
	}
}

func TestExecuteCommand_MoveRequiresActiveChannel(t *testing.T) {
	a := readyMoveApp()
	a.activeChannelID = ""
	cmd := executeCommand(a, "move")
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v", a.mode)
	}
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "No active channel") {
		t.Fatalf("toast = %q", out)
	}
}

func TestExecuteCommand_MoveRequiresSlackSections(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	cmd := executeCommand(a, "move")
	if a.mode == ModeSectionPicker {
		t.Fatal("must not open picker without Slack sections")
	}
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Slack sections not available") {
		t.Fatalf("toast = %q", out)
	}
}

func TestExecuteCommand_MoveByNameAssigns(t *testing.T) {
	a := readyMoveApp()
	var gotCh, gotTo string
	a.SetSectionService(NewSectionService(SectionServiceFuncs{
		Assign: func(channelID ids.ChannelID, sectionID string) tea.Msg {
			gotCh, gotTo = string(channelID), sectionID
			return SectionMovedMsg{ChannelID: gotCh, SectionID: gotTo, Name: "Engineering"}
		},
	}))
	cmd := executeCommand(a, "move Engineering")
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v, want Normal (no picker)", a.mode)
	}
	if currentSectionID(a, "C1") != "L1" {
		t.Errorf("optimistic section = %q", currentSectionID(a, "C1"))
	}
	if cmd == nil {
		t.Fatal("expected assign cmd")
	}
	msg := cmd()
	moved, ok := msg.(SectionMovedMsg)
	if !ok {
		t.Fatalf("msg = %#v", msg)
	}
	if gotCh != "C1" || gotTo != "L1" || moved.Name != "Engineering" {
		t.Errorf("assign ch=%q to=%q msg=%+v", gotCh, gotTo, moved)
	}
}

func TestSectionPicker_EnterAssignsActiveChannel(t *testing.T) {
	a := readyMoveApp()
	var gotTo string
	a.SetSectionService(NewSectionService(SectionServiceFuncs{
		Assign: func(channelID ids.ChannelID, sectionID string) tea.Msg {
			gotTo = sectionID
			return SectionMovedMsg{ChannelID: string(channelID), SectionID: sectionID, Name: "Engineering"}
		},
	}))
	_ = executeCommand(a, "move")
	cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v after enter", a.mode)
	}
	if cmd == nil {
		t.Fatal("expected assign cmd")
	}
	_ = cmd()
	if gotTo != "L1" {
		t.Errorf("assigned to %q, want L1", gotTo)
	}
	if currentSectionID(a, "C1") != "L1" {
		t.Errorf("optimistic section = %q", currentSectionID(a, "C1"))
	}
}

func TestSectionPicker_EscCloses(t *testing.T) {
	a := readyMoveApp()
	_ = executeCommand(a, "move")
	cmd := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Errorf("esc should not dispatch, got %#v", cmd())
	}
	if a.mode != ModeNormal || a.sectionPicker.IsVisible() {
		t.Errorf("mode=%v visible=%v", a.mode, a.sectionPicker.IsVisible())
	}
}

func TestSectionMoveFailed_RevertsOptimisticSection(t *testing.T) {
	a := readyMoveApp()
	a.SetSectionService(NewSectionService(SectionServiceFuncs{
		Assign: func(channelID ids.ChannelID, sectionID string) tea.Msg {
			return SectionMoveFailedMsg{ChannelID: string(channelID), SectionID: sectionID, FromSectionID: "L2", Err: "invalid_auth"}
		},
	}))
	cmd := executeCommand(a, "move Engineering")
	if currentSectionID(a, "C1") != "L1" {
		t.Fatal("expected optimistic patch before the cmd runs")
	}
	_, _ = a.Update(cmd())
	if currentSectionID(a, "C1") != "L2" {
		t.Errorf("after fail, section = %q, want L2", currentSectionID(a, "C1"))
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Move failed") {
		t.Fatalf("toast = %q", out)
	}
}

func TestExecuteCommand_SectionCreates(t *testing.T) {
	a := readyMoveApp()
	var gotName string
	a.SetSectionService(NewSectionService(SectionServiceFuncs{
		Create: func(name string) tea.Msg {
			gotName = name
			return SectionCreatedMsg{ID: "LNEW", Name: name}
		},
	}))
	cmd := executeCommand(a, "section Archive bin")
	if cmd == nil {
		t.Fatal("expected create cmd")
	}
	msg := cmd()
	created, ok := msg.(SectionCreatedMsg)
	if !ok {
		t.Fatalf("msg = %#v", msg)
	}
	if gotName != "Archive bin" || created.ID != "LNEW" {
		t.Errorf("create name=%q msg=%+v", gotName, created)
	}
}

func TestExecuteCommand_SectionRequiresName(t *testing.T) {
	a := readyMoveApp()
	cmd := executeCommand(a, "section")
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Usage: :section") {
		t.Fatalf("toast = %q", out)
	}
}

func TestHelp_ListsMoveAndSectionCommands(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	foundMove, foundSection := false, false
	for _, e := range entries {
		if e.Key == ":move" && e.Desc == "move channel to section" {
			foundMove = true
		}
		if strings.Contains(e.Key, ":section") && strings.Contains(e.Desc, "sidebar section") {
			foundSection = true
		}
	}
	if !foundMove || !foundSection {
		t.Fatalf("help missing :move/:section (move=%v section=%v)", foundMove, foundSection)
	}
}
