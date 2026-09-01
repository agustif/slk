package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/help"
	"github.com/agustif/slk/internal/ui/sidebar"
)

func seedChannelCmd(t *testing.T, items []sidebar.ChannelItem, activeID string) *App {
	t.Helper()
	a := NewApp()
	a.activeTeamID = "T1"
	a.view = ViewChannels
	a.SetChannels(items)
	a.activeChannelID = activeID
	return a
}

func TestExecuteCommand_CreateNoArgsToasts(t *testing.T) {
	a := NewApp()
	cmd := executeCommand(a, "create")
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Usage: :create") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_CreateDispatchesLowercasedName(t *testing.T) {
	a := NewApp()
	var got string
	var gotPriv bool
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		Create: func(name string, private bool) tea.Msg {
			got = name
			gotPriv = private
			return ChannelCreatedMsg{ID: "C9", Name: name, Private: private}
		},
	}))
	cmd := executeCommand(a, "create SLK-HAR-TEST")
	if cmd == nil {
		t.Fatal("expected dispatch cmd")
	}
	res := cmd()
	if got != "slk-har-test" {
		t.Errorf("Create(%q), want slk-har-test", got)
	}
	cm, ok := res.(ChannelCreatedMsg)
	if !ok {
		t.Fatalf("got %T", res)
	}
	if cm.ID != "C9" || cm.Name != "slk-har-test" || gotPriv || cm.Private {
		t.Errorf("ChannelCreatedMsg = %+v priv=%v", cm, gotPriv)
	}
}

func TestExecuteCommand_CreatePrivateDispatches(t *testing.T) {
	a := NewApp()
	var got string
	var gotPriv bool
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		Create: func(name string, private bool) tea.Msg {
			got, gotPriv = name, private
			return ChannelCreatedMsg{ID: "C8", Name: name, Private: private}
		},
	}))
	cmd := executeCommand(a, "create private slk-har-priv2")
	if cmd == nil {
		t.Fatal("expected dispatch")
	}
	res := cmd()
	if got != "slk-har-priv2" || !gotPriv {
		t.Errorf("Create(%q, %v)", got, gotPriv)
	}
	cm, ok := res.(ChannelCreatedMsg)
	if !ok || !cm.Private {
		t.Fatalf("got %+v", res)
	}
}

func TestChannelCreatedMsg_AddsOpensAndToasts(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")

	_, cmd := a.Update(ChannelCreatedMsg{ID: "C9", Name: "slk-har-test"})
	found := false
	for _, it := range a.sidebar.Items() {
		if it.ID == "C9" && it.Name == "slk-har-test" && it.Type == "channel" {
			found = true
		}
	}
	if !found {
		t.Fatal("created channel missing from sidebar")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Created #slk-har-test") {
		t.Fatalf("toast missing:\n%s", out)
	}
	if cmd == nil {
		t.Fatal("expected select cmd")
	}
	sel, ok := findChannelSelected(cmd())
	if !ok || sel.ID != "C9" {
		t.Fatalf("want ChannelSelectedMsg C9, got ok=%v id=%q", ok, sel.ID)
	}
}

func TestChannelCreateFailedMsg_Toasts(t *testing.T) {
	a := NewApp()
	_, cmd := a.Update(ChannelCreateFailedMsg{Name: "nope", Err: errors.New("name_taken")})
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Create failed") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_InviteNoArgsToasts(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	cmd := executeCommand(a, "invite")
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Usage: :invite") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_InviteOnDMToasts(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm"},
	}, "D1")
	cmd := executeCommand(a, "invite a@b.c")
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "not a DM") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_InviteDispatches(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var gotEmails []string
	var gotID string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		InviteEmails: func(emails []string, channelID ids.ChannelID) tea.Msg {
			gotEmails = append([]string(nil), emails...)
			gotID = string(channelID)
			return ChannelInvitedMsg{Emails: emails, ChannelID: string(channelID)}
		},
	}))
	cmd := executeCommand(a, "invite agusti+123@obvious.ai")
	if cmd == nil {
		t.Fatal("expected dispatch")
	}
	res := cmd()
	if gotID != "C1" || len(gotEmails) != 1 || gotEmails[0] != "agusti+123@obvious.ai" {
		t.Errorf("InviteEmails(%v, %q)", gotEmails, gotID)
	}
	if _, ok := res.(ChannelInvitedMsg); !ok {
		t.Fatalf("got %T", res)
	}
}

func TestExecuteCommand_InviteUserIDDispatches(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var got []string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		InviteUsers: func(userIDs []string, channelID ids.ChannelID) tea.Msg {
			got = append([]string(nil), userIDs...)
			return ChannelInvitedMsg{UserIDs: userIDs, ChannelID: string(channelID)}
		},
	}))
	cmd := executeCommand(a, "invite U0BU3458TTK")
	if cmd == nil {
		t.Fatal("expected dispatch")
	}
	_ = cmd()
	if len(got) != 1 || got[0] != "U0BU3458TTK" {
		t.Errorf("InviteUsers(%v)", got)
	}
}

func TestExecuteCommand_ManagerOpensConfirm(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	if cmd := executeCommand(a, "manager U0BU3458TTK"); cmd != nil {
		t.Errorf("expected nil cmd, got %T", cmd)
	}
	if !a.confirmPrompt.IsVisible() {
		t.Fatal("expected confirm")
	}
	if out := a.confirmPrompt.View(80); !strings.Contains(out, "U0BU3458TTK") || !strings.Contains(out, "Channel Manager") {
		t.Errorf("confirm missing:\n%s", out)
	}
}

func TestExecuteCommand_ManagerDispatchesAddManagers(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var gotUsers []string
	var gotID string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		AddManagers: func(channelID ids.ChannelID, userIDs []string) tea.Msg {
			gotID = string(channelID)
			gotUsers = append([]string(nil), userIDs...)
			return ChannelManagersAddedMsg{ChannelID: string(channelID), Channel: "general", UserIDs: userIDs}
		},
	}))
	if cmd := executeCommand(a, "manager U0BU3458TTK"); cmd != nil {
		t.Fatalf("expected nil, got %T", cmd)
	}
	cmd := a.handleConfirmMode(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected confirm cmd")
	}
	res := cmd()
	m, ok := res.(AddChannelManagersMsg)
	if !ok || m.ChannelID != "C1" || len(m.UserIDs) != 1 || m.UserIDs[0] != "U0BU3458TTK" {
		t.Fatalf("confirm msg = %T %+v", res, res)
	}
	_, cmd = a.Update(res)
	if cmd == nil {
		t.Fatal("expected AddManagers dispatch")
	}
	out := cmd()
	if gotID != "C1" || len(gotUsers) != 1 || gotUsers[0] != "U0BU3458TTK" {
		t.Errorf("AddManagers(%q, %v)", gotID, gotUsers)
	}
	if _, ok := out.(ChannelManagersAddedMsg); !ok {
		t.Fatalf("got %T", out)
	}
}

func TestExecuteCommand_KickOpensConfirm(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	if cmd := executeCommand(a, "kick U0BU3458TTK"); cmd != nil {
		t.Errorf("expected nil cmd, got %T", cmd)
	}
	if !a.confirmPrompt.IsVisible() {
		t.Fatal("expected confirm")
	}
	if out := a.confirmPrompt.View(80); !strings.Contains(out, "U0BU3458TTK") {
		t.Errorf("confirm missing user:\n%s", out)
	}
}

func TestChannelInvitedMsg_ToastsEmail(t *testing.T) {
	a := NewApp()
	_, cmd := a.Update(ChannelInvitedMsg{Emails: []string{"agusti+123@obvious.ai"}, ChannelID: "C1"})
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Invited agusti+123@obvious.ai") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_NotifyUsageToasts(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	cmd := executeCommand(a, "notify")
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Usage: :notify") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_NotifyMentionsDispatches(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var gotID, gotLevel string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		SetNotifyLevel: func(channelID ids.ChannelID, level string) tea.Msg {
			gotID, gotLevel = string(channelID), level
			return ChannelNotifySetMsg{ChannelID: string(channelID), Level: level}
		},
	}))
	cmd := executeCommand(a, "notify mentions")
	if cmd == nil {
		t.Fatal("expected dispatch")
	}
	_ = cmd()
	if gotID != "C1" || gotLevel != slackclient.NotifyMentions {
		t.Errorf("SetNotifyLevel(%q, %q)", gotID, gotLevel)
	}
}

func TestChannelNotifySetMsg_ToastsMentions(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	_, cmd := a.Update(ChannelNotifySetMsg{ChannelID: "C1", Level: slackclient.NotifyMentions})
	if cmd == nil {
		t.Fatal("expected toast")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "#general: mentions only") {
		t.Fatalf("toast missing:\n%s", out)
	}
}

func TestHelp_ListsCreateInviteNotify(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	want := map[string]string{
		":create": "create [private] channel",
		":invite": "invite email or U",
		":notify": "all / mentions",
		":kick":    "remove member",
		":manager": "make Channel Manager",
	}
	for _, e := range entries {
		if desc, ok := want[e.Key]; ok && strings.Contains(e.Desc, desc) {
			delete(want, e.Key)
		}
	}
	if len(want) > 0 {
		t.Fatalf("help missing %v", want)
	}
}

func TestExecuteCommand_NotifyAllDispatchesEverything(t *testing.T) {
	a := seedChannelCmd(t, []sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	}, "C1")
	var got string
	a.SetChannelService(NewChannelService(ChannelServiceFuncs{
		SetNotifyLevel: func(_ ids.ChannelID, level string) tea.Msg {
			got = level
			return ChannelNotifySetMsg{ChannelID: "C1", Level: level}
		},
	}))
	cmd := executeCommand(a, "notify all")
	if cmd == nil {
		t.Fatal("expected dispatch")
	}
	_ = cmd()
	if got != slackclient.NotifyEverything {
		t.Errorf("level = %q, want %q", got, slackclient.NotifyEverything)
	}
}
