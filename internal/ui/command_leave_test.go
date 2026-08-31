package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/sidebar"
)

func seedLeaveChannel(t *testing.T, items []sidebar.ChannelItem, activeID string) *App {
	t.Helper()
	a := NewApp()
	a.activeTeamID = "T1"
	a.view = ViewChannels
	a.SetChannels(items)
	a.activeChannelID = activeID
	return a
}

func TestExecuteCommand_LeaveOpensConfirm(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")

	if cmd := executeCommand(a, "leave"); cmd != nil {
		t.Errorf("expected nil cmd (prompt opens directly), got %T", cmd)
	}
	if !a.confirmPrompt.IsVisible() {
		t.Fatal("expected confirm prompt to be visible")
	}
	if a.mode != ModeConfirm {
		t.Errorf("mode = %v, want ModeConfirm", a.mode)
	}
	if out := a.confirmPrompt.View(80); !strings.Contains(out, "Leave #general?") {
		t.Errorf("confirm copy missing %q:\n%s", "Leave #general?", out)
	}
}

func TestExecuteCommand_LeavePrivateChannelOpensConfirm(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C9", Name: "secret", Type: "private"},
	}, "C9")

	_ = executeCommand(a, "leave")
	if !a.confirmPrompt.IsVisible() {
		t.Fatal("private channels should be leavable")
	}
	if out := a.confirmPrompt.View(80); !strings.Contains(out, "Leave #secret?") {
		t.Errorf("confirm copy missing %q:\n%s", "Leave #secret?", out)
	}
}

func TestExecuteCommand_LeaveConfirmEmitsLeaveChannelMsg(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	_ = executeCommand(a, "leave")

	cmd := handleConfirmMode(a, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from confirm")
	}
	res := cmd()
	lm, ok := res.(LeaveChannelMsg)
	if !ok {
		t.Fatalf("expected LeaveChannelMsg, got %T", res)
	}
	if lm.ID != "C1" || lm.Name != "general" {
		t.Errorf("LeaveChannelMsg = %+v, want C1/general", lm)
	}
	if a.confirmPrompt.IsVisible() {
		t.Error("prompt should be closed after confirm")
	}
	if a.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal after confirm", a.mode)
	}
}

func TestExecuteCommand_LeaveCancelDoesNotLeave(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	_ = executeCommand(a, "leave")

	cmd := handleConfirmMode(a, tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd != nil {
		t.Errorf("expected nil cmd on cancel, got %T from %v", cmd(), cmd)
	}
	if a.confirmPrompt.IsVisible() {
		t.Error("prompt should be closed after cancel")
	}
	if a.activeChannelID != "C1" {
		t.Errorf("activeChannelID = %q, want C1 (still in channel)", a.activeChannelID)
	}
	found := false
	for _, it := range a.sidebar.Items() {
		if it.ID == "C1" {
			found = true
		}
	}
	if !found {
		t.Error("channel should still be in sidebar after cancel")
	}
}

func TestExecuteCommand_LeaveOnDMOpensCloseConfirm(t *testing.T) {
	for _, typ := range []string{"dm", "group_dm", "app"} {
		a := seedLeaveChannel(t, []sidebar.ChannelItem{
			{ID: "D1", Name: "alice", Type: typ},
		}, "D1")
		if cmd := executeCommand(a, "leave"); cmd != nil {
			t.Fatalf("type %s: expected nil cmd (prompt opens), got %T", typ, cmd)
		}
		if !a.confirmPrompt.IsVisible() {
			t.Errorf("type %s: confirm should open for DMs", typ)
		}
		if out := a.confirmPrompt.View(80); !strings.Contains(out, "Close this conversation?") {
			t.Errorf("type %s: confirm copy missing close prompt:\n%s", typ, out)
		}
	}
}

func TestExecuteCommand_LeaveOnThreadsToasts(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	a.view = ViewThreads

	cmd := executeCommand(a, "leave")
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if a.confirmPrompt.IsVisible() {
		t.Error("confirm should not open from Threads view")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "No channel to leave") {
		t.Errorf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_LeaveWithNoChannelToasts(t *testing.T) {
	a := NewApp()
	cmd := executeCommand(a, "leave")
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "No channel to leave") {
		t.Errorf("toast missing:\n%s", out)
	}
}

func TestLeaveChannelMsg_DispatchesLeave(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var calledID, calledName string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		Leave: func(channelID ids.ChannelID, channelName string) tea.Msg {
			calledID, calledName = string(channelID), channelName
			return ChannelLeftMsg{ID: string(channelID), Name: channelName}
		},
	}))

	_, cmd := a.Update(LeaveChannelMsg{ID: "C1", Name: "general"})
	if cmd == nil {
		t.Fatal("expected Leave dispatch cmd")
	}
	res := cmd()
	if _, ok := res.(ChannelLeftMsg); !ok {
		t.Fatalf("expected ChannelLeftMsg, got %T", res)
	}
	if calledID != "C1" || calledName != "general" {
		t.Errorf("Leave(%q, %q), want C1/general", calledID, calledName)
	}
}

func TestChannelLeftMsg_RemovesAndSwitchesToLastVisited(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "C2", Name: "random", Type: "channel"},
	}, "C2")
	a.navHistory.Push("T1", "C1")
	a.navHistory.Push("T1", "C2")

	_, cmd := a.Update(ChannelLeftMsg{ID: "C2", Name: "random"})
	for _, it := range a.sidebar.Items() {
		if it.ID == "C2" {
			t.Fatal("left channel still in sidebar")
		}
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Left #random") {
		t.Fatalf("toast missing Left #random:\n%s", out)
	}
	if cmd == nil {
		t.Fatal("expected switch cmd")
	}
	sel, ok := findChannelSelected(cmd())
	if !ok || sel.ID != "C1" {
		t.Fatalf("want switch to last-visited C1, got ok=%v id=%q", ok, sel.ID)
	}
}

func TestChannelLeftMsg_FallsBackToThreads(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")

	_, cmd := a.Update(ChannelLeftMsg{ID: "C1", Name: "general"})
	if len(a.sidebar.Items()) != 0 {
		t.Fatalf("sidebar still has %d items, want 0", len(a.sidebar.Items()))
	}
	if a.activeChannelID != "" {
		t.Errorf("activeChannelID = %q, want empty after leaving last channel", a.activeChannelID)
	}
	if cmd == nil {
		t.Fatal("expected fallback cmd")
	}
	if !findThreadsActivated(cmd()) {
		t.Fatal("expected ThreadsViewActivatedMsg fallback")
	}
}

func TestChannelLeaveFailedMsg_Toasts(t *testing.T) {
	a := seedLeaveChannel(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")

	_, cmd := a.Update(ChannelLeaveFailedMsg{ID: "C1", Name: "general", Err: errors.New("restricted_action")})
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Failed to leave #general") {
		t.Fatalf("toast missing:\n%s", out)
	}
	found := false
	for _, it := range a.sidebar.Items() {
		if it.ID == "C1" {
			found = true
		}
	}
	if !found {
		t.Error("channel should remain in sidebar on failed leave")
	}
}

func findThreadsActivated(msg tea.Msg) bool {
	if _, ok := msg.(ThreadsViewActivatedMsg); ok {
		return true
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil && findThreadsActivated(c()) {
				return true
			}
		}
	}
	return false
}
