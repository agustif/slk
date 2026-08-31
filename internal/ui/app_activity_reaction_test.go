package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/ids"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/activityview"
)

func activityReactionApp(t *testing.T) *App {
	t.Helper()
	a := NewApp()
	a.width = 160
	a.height = 30
	a.view = ViewActivity
	a.focusedPanel = PanelMessages
	a.currentUserID = "U-self"
	a.activityView.SetItems([]activityview.Item{{
		ActivityItem: slackclient.ActivityItem{
			Key: "r1", Type: "message_reaction",
			ChannelID: "C1", MessageTS: "1700000001.000000",
			ActorID: "U1", Reaction: "eyes", FeedTS: "1700000001.000000",
		},
		ChannelName: "eng", ChannelType: "channel", ActorName: "Alice",
		ParentText: "hello parent",
	}})
	_ = a.View()
	return a
}

func findActivityReactionHit(t *testing.T, a *App) (x, y int, emoji string) {
	t.Helper()
	maxPaneY := a.height - 2
	maxPaneX := a.layout.MsgEnd() - a.layout.SidebarEnd() - 2
	for paneY := 0; paneY < maxPaneY; paneY++ {
		for paneX := 0; paneX < maxPaneX; paneX++ {
			if e, ok := a.activityView.HitTestReaction(paneY, paneX); ok {
				return a.layout.SidebarEnd() + 1 + paneX, paneY + 1, e
			}
		}
	}
	t.Fatal("no activity reaction hit found after render")
	return 0, 0, ""
}

func TestApp_ActivityFeedDecoratesParentQuoteAndHasReacted(t *testing.T) {
	db, err := cache.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertWorkspace(cache.Workspace{ID: "T1", Name: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertMessage(cache.Message{
		TS: "1.0", ChannelID: "C1", WorkspaceID: "T1", UserID: "U1", Text: "cached parent",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertReaction("1.0", "C1", "eyes", []string{"U-self"}, 1); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.activeTeamID = "T1"
	app.SetCurrentUserID("U-self")
	app.SetActivityCache(db)
	app.activityGen = 1
	_, _ = app.Update(ActivityFeedLoadedMsg{
		TeamID: "T1",
		Gen:    1,
		Items: []slackclient.ActivityItem{{
			Key: "k1", Type: "message_reaction", ChannelID: "C1", MessageTS: "1.0",
			ActorID: "U1", Reaction: "eyes",
		}},
	})
	it, ok := app.activityView.SelectedItem()
	if !ok {
		t.Fatal("expected a selected activity item")
	}
	if it.ParentText != "cached parent" {
		t.Errorf("ParentText = %q, want cached parent", it.ParentText)
	}
	if !it.ReactionsKnown {
		t.Error("ReactionsKnown should be true after a successful GetReactions")
	}
	if !it.HasReacted {
		t.Error("HasReacted should be true when cache lists current user")
	}
}

func TestApp_ActivityClickReactionAddsWhenUnknown(t *testing.T) {
	a := activityReactionApp(t)
	var added, removed int
	a.SetReactionService(NewReactionService(
		func(channelID ids.ChannelID, ts ids.MessageTS, emoji string) error {
			added++
			if string(channelID) != "C1" || string(ts) != "1700000001.000000" || emoji != "eyes" {
				t.Errorf("add(%s, %s, %s)", channelID, ts, emoji)
			}
			return nil
		},
		func(channelID ids.ChannelID, ts ids.MessageTS, emoji string) error {
			removed++
			return nil
		},
		nil, nil,
	))

	x, y, emoji := findActivityReactionHit(t, a)
	if emoji != "eyes" {
		t.Fatalf("hit emoji %q, want eyes", emoji)
	}
	_, cmd := a.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("expected a tea.Cmd for reaction click")
	}
	_ = drainBatch(cmd)
	if added != 1 {
		t.Errorf("added=%d, want 1 (reactions unknown → Add)", added)
	}
	if removed != 0 {
		t.Errorf("removed=%d, want 0", removed)
	}
	it, _ := a.activityView.SelectedItem()
	if !it.HasReacted {
		t.Error("optimistic HasReacted should be true after Add")
	}
}

func TestApp_ActivityClickReactionRemovesWhenOwn(t *testing.T) {
	a := activityReactionApp(t)
	it, _ := a.activityView.SelectedItem()
	it.ReactionsKnown = true
	it.HasReacted = true
	a.activityView.SetItems([]activityview.Item{it})
	_ = a.View()

	var added, removed int
	a.SetReactionService(NewReactionService(
		func(channelID ids.ChannelID, ts ids.MessageTS, emoji string) error {
			added++
			return nil
		},
		func(channelID ids.ChannelID, ts ids.MessageTS, emoji string) error {
			removed++
			return nil
		},
		nil, nil,
	))

	x, y, _ := findActivityReactionHit(t, a)
	_, cmd := a.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("expected a tea.Cmd")
	}
	_ = drainBatch(cmd)
	if added != 0 || removed != 1 {
		t.Errorf("added=%d removed=%d, want 0/1", added, removed)
	}
}

func TestApp_ActivityClickBodyOpensItem(t *testing.T) {
	a := activityReactionApp(t)
	x, y, _ := findActivityReactionHit(t, a)
	paneY := y - 1
	bodyPaneX := 2
	if _, ok := a.activityView.HitTestReaction(paneY, bodyPaneX); ok {
		t.Fatal("column 2 unexpectedly hit the emoji; layout assumption broken")
	}
	kind := a.activityView.ClickAt(paneY, bodyPaneX)
	if kind != activityview.ClickItem {
		t.Fatalf("body click = %v, want ClickItem (screen=%d,%d paneX=2)", kind, x, y)
	}
}

func TestApp_ActivityROpensPicker(t *testing.T) {
	a := activityReactionApp(t)
	cmd := handleNormalMode(a, tea.KeyPressMsg{Code: 'r', Text: "r"})
	_ = cmd
	if a.mode != ModeReactionPicker {
		t.Errorf("mode = %v, want ModeReactionPicker", a.mode)
	}
	if a.reactionPicker.ChannelID() != "C1" || a.reactionPicker.MessageTS() != "1700000001.000000" {
		t.Errorf("picker target (%s, %s)", a.reactionPicker.ChannelID(), a.reactionPicker.MessageTS())
	}
}

func TestApp_ActivityRightClickOpensPicker(t *testing.T) {
	a := activityReactionApp(t)
	x, y, _ := findActivityReactionHit(t, a)
	_, cmd := a.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
	_ = cmd
	if a.mode != ModeReactionPicker {
		t.Errorf("right-click mode = %v, want ModeReactionPicker", a.mode)
	}
	if a.reactionPicker.ChannelID() != "C1" {
		t.Errorf("picker channel %q", a.reactionPicker.ChannelID())
	}
}
