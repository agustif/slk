package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/sidebar"
	"github.com/agustif/slk/internal/ui/userprofile"
)

func TestOpenUserProfile_FromSelectedMessage(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.activeTeamID = "T1"
	app.currentUserID = "USELF"
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1.0", UserID: "U1", UserName: "Ada", Text: "hello"},
	})
	app.sidebar.UpdateUserStatus("U1", slackclient.UserStatus{Text: "focus", Emoji: ":dart:"})

	_ = app.handleNormalMode(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if app.mode != ModeUserProfile {
		t.Fatalf("mode = %v, want ModeUserProfile", app.mode)
	}
	if !app.userProfile.IsVisible() {
		t.Fatal("profile overlay should be visible")
	}
	p := app.userProfile.Profile()
	if p.UserID != "U1" || p.DisplayName != "Ada" {
		t.Errorf("profile = %+v", p)
	}
	if p.Status.Text != "focus" {
		t.Errorf("cached status = %+v", p.Status)
	}
	if p.IsSelf {
		t.Error("U1 should not be marked self")
	}
}

func TestOpenUserProfile_NoMessageIsNoop(t *testing.T) {
	app := NewApp()
	app.focusedPanel = PanelMessages
	if cmd := app.handleNormalMode(tea.KeyPressMsg{Code: 'p', Text: "p"}); cmd != nil {
		t.Errorf("cmd = %v, want nil", cmd)
	}
	if app.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal", app.mode)
	}
}

func TestUserProfileLoadedMsg_FillsOverlay(t *testing.T) {
	app := NewApp()
	app.currentUserID = "USELF"
	app.userProfile.Open(userprofile.Profile{UserID: "U1", DisplayName: "Ada", Loading: true})
	app.SetMode(ModeUserProfile)

	_, _ = app.Update(UserProfileLoadedMsg{
		UserID: "U1",
		Profile: userprofile.Profile{
			UserID:      "U1",
			DisplayName: "Ada Lovelace",
			RealName:    "Ada Lovelace",
			Title:       "Engineer",
			Handle:      "ada",
			Status:      slackclient.UserStatus{Text: "coding"},
			TZ:          "Europe/London",
			Presence:    "active",
		},
	})
	p := app.userProfile.Profile()
	if p.Loading {
		t.Error("loading should clear")
	}
	if p.Title != "Engineer" || p.Handle != "ada" {
		t.Errorf("filled profile = %+v", p)
	}
	st, ok := app.sidebar.StatusForUser("U1")
	if !ok || st.Text != "coding" {
		t.Errorf("sidebar status = %+v ok=%v", st, ok)
	}
}

func TestPresenceMenu_SetStatusOpensInput(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24
	app.presenceMenu.OpenWith("WS", "active", false, time.Time{})
	app.SetMode(ModePresenceMenu)
	app.presenceMenu.HandleKey("s")
	app.presenceMenu.HandleKey("e")
	app.presenceMenu.HandleKey("t")
	_ = handlePresenceMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if app.mode != ModePresenceSetStatus {
		t.Fatalf("mode = %v, want ModePresenceSetStatus", app.mode)
	}
	if !app.statusInput.IsVisible() {
		t.Fatal("set-status overlay should be visible")
	}
}

func TestClearStatus_Optimistic(t *testing.T) {
	app := NewApp()
	app.currentUserID = "USELF"
	app.sidebar.UpdateUserStatus("USELF", slackclient.UserStatus{Text: "old"})
	var got slackclient.UserStatus
	app.SetCustomStatusSetter(func(st slackclient.UserStatus) { got = st })
	app.presenceMenu.OpenWith("WS", "active", false, time.Time{})
	app.SetMode(ModePresenceMenu)
	for _, r := range "clear" {
		app.presenceMenu.HandleKey(string(r))
	}
	_ = handlePresenceMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got.Text != "" || got.Emoji != "" {
		t.Errorf("clear callback = %+v", got)
	}
	st, _ := app.sidebar.StatusForUser("USELF")
	if st.Text != "" {
		t.Errorf("optimistic clear left %+v", st)
	}
}

func TestUserProfile_MessageOpensConversation(t *testing.T) {
	app, cap := newApp_WithOpenConvCapture(t)
	app.width = 80
	app.height = 24
	app.userProfile.Open(userprofile.Profile{UserID: "U1", DisplayName: "Ada"})
	app.SetMode(ModeUserProfile)
	cmd := handleUserProfileMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		_ = cmd()
	}
	if len(cap.calls) != 1 || len(cap.calls[0].UserIDs) != 1 || cap.calls[0].UserIDs[0] != "U1" {
		t.Errorf("OpenConversation calls = %+v", cap.calls)
	}
}

func TestUserProfile_AlreadyInDMSkipsOpen(t *testing.T) {
	app, cap := newApp_WithOpenConvCapture(t)
	app.activeChannelID = "D1"
	app.SetChannels([]sidebar.ChannelItem{{ID: "D1", Type: "dm", DMUserID: "U1", Name: "Ada"}})
	app.userProfile.Open(userprofile.Profile{UserID: "U1", DisplayName: "Ada", AlreadyInDM: true})
	app.SetMode(ModeUserProfile)
	cmd := handleUserProfileMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("already-in-DM should not open, cmd=%v", cmd)
	}
	if len(cap.calls) != 0 {
		t.Errorf("OpenConversation calls = %+v", cap.calls)
	}
}
