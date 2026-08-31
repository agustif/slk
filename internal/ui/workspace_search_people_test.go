package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/agustif/slk/internal/ui/searchresults"
	"github.com/agustif/slk/internal/ui/sidebar"
)

func peopleSearchTestApp(t *testing.T) (*App, *[]searchresults.Kind) {
	t.Helper()
	app := searchTestApp(t)
	var kinds []searchresults.Kind
	app.SetSearchService(NewSearchService(SearchServiceFuncs{
		SearchWorkspace: func(req WorkspaceSearchRequest) tea.Msg {
			kinds = append(kinds, req.Kind)
			item := searchresults.Item{ChannelID: "C2", ChannelName: "ops", Text: "msg"}
			switch req.Kind {
			case searchresults.KindFiles:
				item = searchresults.Item{Kind: searchresults.KindFiles, FileName: "a.pdf", Text: "a.pdf"}
			case searchresults.KindPeople:
				item = searchresults.Item{Kind: searchresults.KindPeople, UserID: "U1", UserName: "Alice Smith", Text: "alice"}
			}
			return WorkspaceSearchResultsMsg{
				Query: req.Query, Kind: req.Kind, Page: req.Page, Gen: req.Gen,
				Items: []searchresults.Item{item}, Total: 1,
			}
		},
	}))
	return app, &kinds
}

func openPeopleTab(t *testing.T, app *App) {
	t.Helper()
	app.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if app.searchResults.Kind() != searchresults.KindPeople {
		t.Fatalf("kind = %v, want People", app.searchResults.Kind())
	}
}

func typeIntoPeople(a *App, s string) []peopleSearchDebounceMsg {
	var emitted []peopleSearchDebounceMsg
	for _, r := range s {
		a.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		emitted = append(emitted, peopleSearchDebounceMsg{
			query: a.searchResults.Query(),
			gen:   a.pendingPeopleSearchGen,
		})
	}
	return emitted
}

func TestWorkspaceSearchTabCyclesToPeople(t *testing.T) {
	app, kinds := peopleSearchTestApp(t)
	app.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, m := range drainCmd(cmd) {
		app.Update(m)
	}
	_, cmd = app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	for _, m := range drainCmd(cmd) {
		app.Update(m)
	}
	_, cmd = app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	for _, m := range drainCmd(cmd) {
		app.Update(m)
	}
	if len(*kinds) != 3 || (*kinds)[0] != searchresults.KindMessages || (*kinds)[1] != searchresults.KindFiles || (*kinds)[2] != searchresults.KindPeople {
		t.Fatalf("kinds = %v, want [Messages Files People]", *kinds)
	}
	if app.searchResults.Kind() != searchresults.KindPeople {
		t.Fatal("active tab should be People")
	}
	sel, ok := app.searchResults.Selected()
	if !ok || sel.UserID != "U1" {
		t.Fatalf("people result = %+v ok=%v", sel, ok)
	}
}

func TestWorkspaceSearchPersonSelectOpensDM(t *testing.T) {
	app := searchTestApp(t)
	cap := &capturedOpenConv{}
	app.SetChannelService(NewChannelService(ChannelServiceFuncs{
		OpenConversation: func(userIDs []string, requestID uint64) tea.Cmd {
			cap.calls = append(cap.calls, openConvCall{UserIDs: userIDs, RequestID: requestID})
			return nil
		},
	}))
	app.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	app.searchResults.HandleKey("tab")
	app.searchResults.HandleKey("tab")
	app.searchResults.HandleKey("q")
	app.searchResults.HandleKey("enter")
	app.searchResults.SetResults([]searchresults.Item{{
		Kind: searchresults.KindPeople, UserID: "U9", UserName: "Ada", Text: "ada",
	}}, 1)

	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = drainCmd(cmd)
	if len(cap.calls) != 1 {
		t.Fatalf("OpenConversation calls = %d, want 1", len(cap.calls))
	}
	if len(cap.calls[0].UserIDs) != 1 || cap.calls[0].UserIDs[0] != "U9" {
		t.Errorf("opened DM with %v, want [U9]", cap.calls[0].UserIDs)
	}
	if cap.calls[0].RequestID == 0 {
		t.Error("expected non-zero request ID")
	}
	if app.mode != ModeNormal || app.searchResults.IsVisible() {
		t.Fatal("modal not closed")
	}
}

func TestWorkspaceSearchPeopleDebounceEmptyQueryIssuesNoSearch(t *testing.T) {
	// edge.UsersSearch returns early on an empty query, but the caller
	// must not queue one either: backspacing to empty is how a People
	// search session normally ends, and it is exactly when a pending
	// tick for the last non-empty query would fire.
	var searched []string
	app := searchTestApp(t)
	app.SetSearchService(NewSearchService(SearchServiceFuncs{
		SearchWorkspace: func(req WorkspaceSearchRequest) tea.Msg {
			searched = append(searched, req.Query)
			return WorkspaceSearchResultsMsg{Query: req.Query, Kind: req.Kind, Page: req.Page, Gen: req.Gen}
		},
	}))
	openPeopleTab(t, app)

	emitted := typeIntoPeople(app, "a")
	app.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	stale := emitted[0]

	if _, cmd := app.Update(stale); cmd != nil {
		for _, out := range drainCmd(cmd) {
			_ = out
		}
	}
	if _, cmd := app.Update(peopleSearchDebounceMsg{query: "", gen: app.pendingPeopleSearchGen}); cmd != nil {
		for _, out := range drainCmd(cmd) {
			_ = out
		}
	}
	if len(searched) != 0 {
		t.Errorf("empty query issued %d searches (%v); want none", len(searched), searched)
	}
}

func TestWorkspaceSearchPeopleTypingBurstIssuesOneSearch(t *testing.T) {
	var searched []string
	app := searchTestApp(t)
	app.SetSearchService(NewSearchService(SearchServiceFuncs{
		SearchWorkspace: func(req WorkspaceSearchRequest) tea.Msg {
			searched = append(searched, req.Query)
			if req.CurrentChannel != "C1" {
				t.Errorf("CurrentChannel = %q, want C1", req.CurrentChannel)
			}
			return WorkspaceSearchResultsMsg{
				Query: req.Query, Kind: req.Kind, Page: req.Page, Gen: req.Gen,
				Items: []searchresults.Item{{Kind: searchresults.KindPeople, UserID: "U1", UserName: "Alice", Text: "alice"}},
				Total: 1,
			}
		},
	}))
	openPeopleTab(t, app)

	emitted := typeIntoPeople(app, "ali")
	if len(emitted) != 3 {
		t.Fatalf("typed 3 keys, scheduled %d debounce ticks", len(emitted))
	}
	for _, m := range emitted {
		if _, cmd := app.Update(m); cmd != nil {
			for _, out := range drainCmd(cmd) {
				app.Update(out)
			}
		}
	}
	if len(searched) != 1 {
		t.Fatalf("three keystrokes issued %d searches (%v); want 1", len(searched), searched)
	}
	if searched[0] != "ali" {
		t.Errorf("searched %q; want the final query \"ali\"", searched[0])
	}
}

func TestWorkspaceSearchPeoplePresenceFromSidebar(t *testing.T) {
	app := searchTestApp(t)
	app.sidebar.SetItems([]sidebar.ChannelItem{{ID: "D1", Type: "dm", DMUserID: "U1", Presence: "active"}})
	app.sidebar.UpdatePresenceByUser("U1", "active")
	app.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	app.searchResults.HandleKey("tab")
	app.searchResults.HandleKey("tab")
	app.searchResults.HandleKey("a")
	app.searchResults.HandleKey("enter")
	app.Update(WorkspaceSearchResultsMsg{
		Query: "a", Kind: searchresults.KindPeople, Gen: app.searchResults.Gen(),
		Items: []searchresults.Item{{Kind: searchresults.KindPeople, UserID: "U1", UserName: "Alice", Text: "alice"}},
		Total: 1,
	})
	sel, ok := app.searchResults.Selected()
	if !ok || sel.Presence != "active" {
		t.Fatalf("presence = %+v ok=%v, want active from sidebar (no extra API)", sel, ok)
	}
}
