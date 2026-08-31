package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/channelfinder"
	"github.com/agustif/slk/internal/ui/messages"
)

func TestShare_PickerPostsPermalinkAndToasts(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	a.SetChannelFinderItems([]channelfinder.Item{
		{ID: "C2", Name: "random", Type: "channel", Joined: true, LastVisited: 500},
	})

	sel, ok := a.messagepane.SelectedMessage()
	if !ok {
		t.Fatal("expected a selected message")
	}
	const permalink = "https://example.slack.com/archives/C1/p1000000001000200"
	var sentCh, sentText string
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Permalink: func(ctx context.Context, channelID ids.ChannelID, ts ids.MessageTS) (string, error) {
			if string(channelID) != "C1" || string(ts) != sel.TS {
				t.Errorf("permalink args channel=%s ts=%s, want C1/%s", channelID, ts, sel.TS)
			}
			return permalink, nil
		},
		Send: func(channelID ids.ChannelID, text string) tea.Msg {
			sentCh, sentText = string(channelID), text
			return MessageSentMsg{
				ChannelID: string(channelID),
				Message:   messages.MessageItem{TS: "9.0", Text: text},
			}
		},
	}))

	if cmd := a.openSharePicker(); cmd != nil {
		t.Fatalf("openSharePicker returned cmd %T", cmd)
	}
	if a.mode != ModeShare {
		t.Fatalf("mode = %v, want ModeShare", a.mode)
	}
	filtered := a.channelFinder.FilteredItems()
	if len(filtered) != 1 || filtered[0].ID != "C2" {
		t.Fatalf("share picker rows = %+v, want only C2", filtered)
	}

	_, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.mode != ModeNormal {
		t.Errorf("mode after pick = %v, want ModeNormal", a.mode)
	}
	if a.channelFinder.IsVisible() {
		t.Error("finder should close after pick")
	}
	if cmd == nil {
		t.Fatal("expected share cmd")
	}

	var shared MessageSharedMsg
	foundShared := false
	for _, msg := range drainBatch(cmd) {
		if m, ok := msg.(MessageSharedMsg); ok {
			foundShared = true
			shared = m
		}
	}
	if !foundShared {
		t.Fatal("expected MessageSharedMsg")
	}
	if sentCh != "C2" || sentText != permalink {
		t.Fatalf("Send channel=%q text=%q, want C2 + permalink", sentCh, sentText)
	}
	if shared.DestName != "random" {
		t.Errorf("DestName = %q, want random", shared.DestName)
	}

	a.Update(shared)
	if !strings.Contains(a.statusbar.View(80), "Shared to #random") {
		t.Fatalf("toast = %q, want Shared to #random", a.statusbar.View(80))
	}
}

func TestShare_FromThreadPane(t *testing.T) {
	a := NewApp()
	a.width = 120
	a.height = 30
	parent := messages.MessageItem{TS: "1.0", UserName: "alice", Text: "parent"}
	a.threadPanel.SetThread(parent, []messages.MessageItem{
		parent,
		{TS: "2.0", UserName: "bob", Text: "reply"},
	}, "C9", "1.0")
	a.threadVisible = true
	a.focusedPanel = PanelThread
	for i := 0; i < 8; i++ {
		sel := a.threadPanel.SelectedReply()
		if sel != nil && sel.TS == "2.0" {
			break
		}
		a.threadPanel.MoveDown()
	}
	if sel := a.threadPanel.SelectedReply(); sel == nil || sel.TS != "2.0" {
		t.Fatalf("could not select reply; got %+v", a.threadPanel.SelectedReply())
	}
	a.SetChannelFinderItems([]channelfinder.Item{
		{ID: "D1", Name: "Alice", Type: "dm", Joined: true, LastVisited: 100},
	})

	const permalink = "https://example.slack.com/archives/C9/p2000000002000200"
	var gotCh, gotTS, sentCh, sentText string
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Permalink: func(ctx context.Context, channelID ids.ChannelID, ts ids.MessageTS) (string, error) {
			gotCh, gotTS = string(channelID), string(ts)
			return permalink, nil
		},
		Send: func(channelID ids.ChannelID, text string) tea.Msg {
			sentCh, sentText = string(channelID), text
			return nil
		},
	}))

	_ = a.openSharePicker()
	_, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected share cmd")
	}
	_ = drainBatch(cmd)
	if gotCh != "C9" || gotTS != "2.0" {
		t.Fatalf("permalink lookup channel=%s ts=%s, want C9/2.0", gotCh, gotTS)
	}
	if sentCh != "D1" || sentText != permalink {
		t.Fatalf("Send channel=%q text=%q", sentCh, sentText)
	}
}

func TestShare_ErrorToasts(t *testing.T) {
	a := NewApp()
	a.Update(MessageShareFailedMsg{Reason: "boom"})
	if !strings.Contains(a.statusbar.View(80), "Share failed: boom") {
		t.Fatalf("toast = %q", a.statusbar.View(80))
	}

	a = newTestAppWithMessages(t)
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	a.SetChannelFinderItems([]channelfinder.Item{
		{ID: "C2", Name: "random", Type: "channel", Joined: true},
	})
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Permalink: func(context.Context, ids.ChannelID, ids.MessageTS) (string, error) {
			return "", errors.New("nope")
		},
		Send: func(ids.ChannelID, string) tea.Msg {
			t.Fatal("Send must not run when permalink fails")
			return nil
		},
	}))
	_ = a.openSharePicker()
	_, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected fail cmd")
	}
	msg := cmd()
	fail, ok := msg.(MessageShareFailedMsg)
	if !ok {
		t.Fatalf("got %T, want MessageShareFailedMsg", msg)
	}
	a.Update(fail)
	if !strings.Contains(a.statusbar.View(80), "Share failed") {
		t.Fatalf("toast = %q", a.statusbar.View(80))
	}
}

func TestShare_EscClosesWithoutSend(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Send: func(ids.ChannelID, string) tea.Msg {
			t.Fatal("Send must not run on Esc")
			return nil
		},
	}))
	_ = a.openSharePicker()
	_, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatalf("Esc should not post, got %T", cmd)
	}
	if a.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal", a.mode)
	}
	if a.channelFinder.IsVisible() || a.channelFinder.ShareMode() {
		t.Error("finder should close and leave share mode")
	}
}

func TestShare_CommandOpensPicker(t *testing.T) {
	a := newTestAppWithMessages(t)
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	_ = executeCommand(a, "share")
	if a.mode != ModeShare {
		t.Fatalf("mode = %v, want ModeShare", a.mode)
	}
}

func TestShare_NothingSelectedNoop(t *testing.T) {
	a := NewApp()
	a.focusedPanel = PanelMessages
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Send: func(ids.ChannelID, string) tea.Msg {
			t.Fatal("Send must not run")
			return nil
		},
	}))
	if cmd := a.openSharePicker(); cmd != nil {
		t.Fatalf("expected nil, got %T", cmd)
	}
	if a.mode != ModeNormal {
		t.Errorf("mode = %v, want ModeNormal", a.mode)
	}
}

func TestShareDestLabel(t *testing.T) {
	if got := shareDestLabel("random", "channel"); got != "#random" {
		t.Errorf("channel = %q", got)
	}
	if got := shareDestLabel("Alice", "dm"); got != "Alice" {
		t.Errorf("dm = %q", got)
	}
}
