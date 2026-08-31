package ui

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/help"
)

func membersTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()
	app.activeChannelID = "C1"
	app.view = ViewChannels
	app.compose.SetChannel("general")
	app.compose.SetActiveChannel("C1")
	app.SetUserNames(map[string]string{
		"U1": "Alice",
		"U2": "Bob",
		"U3": "Carla",
	})
	app.SetGuestUsers(map[string]bool{"U3": true})
	app.sidebar.UpdatePresenceByUser("U2", "active")
	return app
}

func TestOpenChannelMembers_UsesMembershipSnapshot(t *testing.T) {
	app := membersTestApp(t)
	fetched := 0
	app.setChannelMembershipFetcherForTest(func(ids.ChannelID) { fetched++ })
	app.SetChannelMembership("C1", []string{"U1", "U2", "U3"})

	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})

	if app.mode != ModeChannelMembers {
		t.Fatalf("mode = %v, want ModeChannelMembers", app.mode)
	}
	if !app.channelMembers.IsVisible() {
		t.Fatal("overlay should be visible")
	}
	if fetched != 0 {
		t.Errorf("snapshot was loaded; fetch count = %d, want 0", fetched)
	}
	got := app.channelMembers.FilteredMembers()
	if len(got) != 3 {
		t.Fatalf("members = %d, want 3", len(got))
	}
	byID := map[string]int{}
	for i, m := range got {
		byID[m.ID] = i
	}
	if _, ok := byID["U1"]; !ok {
		t.Error("missing U1")
	}
	bob := got[byID["U2"]]
	if bob.Presence != "active" {
		t.Errorf("Bob presence = %q, want active (from presence map)", bob.Presence)
	}
	carla := got[byID["U3"]]
	if !carla.IsGuest {
		t.Error("Carla should be flagged guest")
	}
}

func TestOpenChannelMembers_FetchesWhenSnapshotMissing(t *testing.T) {
	app := membersTestApp(t)
	var mu sync.Mutex
	var fetched []string
	done := make(chan struct{}, 1)
	app.setChannelMembershipFetcherForTest(func(channelID ids.ChannelID) {
		mu.Lock()
		fetched = append(fetched, string(channelID))
		mu.Unlock()
		done <- struct{}{}
	})

	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})

	if app.mode != ModeChannelMembers {
		t.Fatalf("mode = %v, want ModeChannelMembers", app.mode)
	}
	if !app.channelMembers.Loading() {
		t.Error("expected loading state when snapshot is missing")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("MembershipFetch was not invoked")
	}
	mu.Lock()
	got := append([]string(nil), fetched...)
	mu.Unlock()
	if len(got) != 1 || got[0] != "C1" {
		t.Errorf("fetched %v, want [C1]", got)
	}

	_, _ = app.Update(ChannelMembershipMsg{ChannelID: "C1", MemberIDs: []string{"U1", "U2"}})
	if app.channelMembers.Loading() {
		t.Error("loading should clear when membership arrives")
	}
	if got := len(app.channelMembers.Members()); got != 2 {
		t.Errorf("members after fetch = %d, want 2", got)
	}
}

func TestOpenChannelMembers_NoopOnThreadsView(t *testing.T) {
	app := membersTestApp(t)
	app.view = ViewThreads
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	if app.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal", app.mode)
	}
	if app.channelMembers.IsVisible() {
		t.Error("overlay should not open on Threads view")
	}
}

func TestOpenChannelMembers_NoopWithoutChannel(t *testing.T) {
	app := NewApp()
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	if app.channelMembers.IsVisible() {
		t.Error("overlay should not open with no active channel")
	}
}

func TestChannelMembers_EnterOpensDM(t *testing.T) {
	app := membersTestApp(t)
	app.SetChannelMembership("C1", []string{"U1", "U2"})
	cap := &capturedOpenConv{}
	fns := channelFuncsForTest(app)
	fns.OpenConversation = func(userIDs []string, requestID uint64) tea.Cmd {
		cap.calls = append(cap.calls, openConvCall{UserIDs: userIDs, RequestID: requestID})
		return nil
	}
	app.SetChannelService(NewChannelService(fns))

	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	_ = handleChannelMembersMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})

	if app.channelMembers.IsVisible() {
		t.Error("overlay should close on Enter")
	}
	if app.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal after Enter", app.mode)
	}
	if len(cap.calls) != 1 {
		t.Fatalf("OpenConversation calls = %d, want 1", len(cap.calls))
	}
	if len(cap.calls[0].UserIDs) != 1 {
		t.Fatalf("userIDs = %v, want one id", cap.calls[0].UserIDs)
	}
	// Empty-query sort is by display name: Alice, Bob. Selected starts at 0 = Alice.
	if cap.calls[0].UserIDs[0] != "U1" {
		t.Errorf("opened DM with %s, want U1", cap.calls[0].UserIDs[0])
	}
	if cap.calls[0].RequestID == 0 {
		t.Error("expected non-zero request ID")
	}
}

func TestChannelMembers_EscCloses(t *testing.T) {
	app := membersTestApp(t)
	app.SetChannelMembership("C1", []string{"U1"})
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	_ = handleChannelMembersMode(app, tea.KeyPressMsg{Code: tea.KeyEscape})
	if app.channelMembers.IsVisible() {
		t.Error("esc should close overlay")
	}
	if app.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal", app.mode)
	}
}

func TestChannelMembers_FilterFromModeHandler(t *testing.T) {
	app := membersTestApp(t)
	app.SetChannelMembership("C1", []string{"U1", "U2", "U3"})
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	_ = handleChannelMembersMode(app, tea.KeyPressMsg{Code: 'b', Text: "b"})
	got := app.channelMembers.FilteredMembers()
	if len(got) != 1 || got[0].ID != "U2" {
		t.Errorf("filter 'b' = %v, want [U2]", got)
	}
}

func TestChannelMembers_HelpListsI(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	found := false
	for _, e := range entries {
		if e.Key == "I" && e.Desc == "channel members" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("help entries missing {Key: \"I\", Desc: \"channel members\"}")
	}
}

func TestChannelMembershipMsg_UpdatesOpenOverlay(t *testing.T) {
	app := membersTestApp(t)
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'I', Text: "I"})
	_, _ = app.Update(ChannelMembershipMsg{ChannelID: "C1", MemberIDs: []string{"U2"}})
	got := app.channelMembers.Members()
	if len(got) != 1 || got[0].ID != "U2" {
		t.Errorf("overlay members = %v, want [U2]", got)
	}
}

func TestChannelMembershipMsg_SetsHeaderCount(t *testing.T) {
	app := membersTestApp(t)
	app.width = 80
	app.height = 24
	app.messagepane.SetChannel("general", "")
	_, _ = app.Update(ChannelMembershipMsg{ChannelID: "C1", MemberIDs: []string{"U1", "U2", "U3"}})
	out := ansi.Strip(app.messagepane.View(10, 40))
	if !strings.Contains(out, "· 3") {
		t.Errorf("header missing member count, got:\n%s", out)
	}
}

func TestLowercaseI_StillEntersInsert(t *testing.T) {
	app := membersTestApp(t)
	app.SetChannelMembership("C1", []string{"U1"})
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'i', Text: "i"})
	if app.mode != ModeInsert {
		t.Fatalf("lowercase i should enter insert, got %v", app.mode)
	}
	if app.channelMembers.IsVisible() {
		t.Error("lowercase i must not open the members overlay")
	}
}
