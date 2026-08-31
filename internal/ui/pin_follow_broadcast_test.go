package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
)

func TestTogglePin_FromMessagesPane(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.view = ViewChannels
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1.0", UserName: "alice", Text: "hi"},
	})
	var gotCh, gotTS string
	var pinCalls, unpinCalls int
	app.SetMessageService(NewMessageService(MessageServiceFuncs{
		Pin: func(channelID ids.ChannelID, ts ids.MessageTS) error {
			pinCalls++
			gotCh, gotTS = string(channelID), string(ts)
			return nil
		},
		Unpin: func(channelID ids.ChannelID, ts ids.MessageTS) error {
			unpinCalls++
			return nil
		},
	}))

	cmd := app.handleNormalMode(tea.KeyPressMsg{Code: 'P', Text: "P"})
	if cmd == nil {
		t.Fatal("expected pin cmd")
	}
	msg, ok := cmd().(PinToggledMsg)
	if !ok {
		t.Fatalf("got %T, want PinToggledMsg", cmd())
	}
	if msg.Err != nil || !msg.Pinned || msg.ChannelID != "C1" || msg.TS != "1.0" {
		t.Fatalf("PinToggledMsg = %+v", msg)
	}
	if pinCalls != 1 || unpinCalls != 0 {
		t.Fatalf("pinCalls=%d unpinCalls=%d", pinCalls, unpinCalls)
	}
	if gotCh != "C1" || gotTS != "1.0" {
		t.Fatalf("pin args channel=%s ts=%s", gotCh, gotTS)
	}

	app.Update(msg)
	got, _ := app.messagepane.SelectedMessage()
	if !got.Pinned {
		t.Fatal("expected message to be marked pinned after PinToggledMsg")
	}
	if !strings.Contains(app.statusbar.View(80), "Pinned") {
		t.Fatalf("expected Pinned toast, got %q", app.statusbar.View(80))
	}

	cmd = app.handleNormalMode(tea.KeyPressMsg{Code: 'P', Text: "P"})
	msg = cmd().(PinToggledMsg)
	if msg.Pinned {
		t.Fatal("second P should unpin")
	}
	if pinCalls != 1 || unpinCalls != 1 {
		t.Fatalf("after unpin pinCalls=%d unpinCalls=%d", pinCalls, unpinCalls)
	}
}

func TestTogglePin_NothingSelectedNoop(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.focusedPanel = PanelMessages
	app.SetMessageService(NewMessageService(MessageServiceFuncs{
		Pin: func(ids.ChannelID, ids.MessageTS) error {
			t.Fatal("Pin must not be called")
			return nil
		},
	}))
	if cmd := app.handleNormalMode(tea.KeyPressMsg{Code: 'P', Text: "P"}); cmd != nil {
		if msg := cmd(); msg != nil {
			t.Fatalf("expected nil, got %T", msg)
		}
	}
}

func TestTogglePin_ErrorToasts(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{{TS: "1.0", Text: "hi"}})
	app.SetMessageService(NewMessageService(MessageServiceFuncs{
		Pin: func(ids.ChannelID, ids.MessageTS) error { return errors.New("boom") },
	}))
	cmd := app.togglePinOfSelected()
	app.Update(cmd())
	if !strings.Contains(app.statusbar.View(80), "Pin failed") {
		t.Fatalf("expected pin-failed toast, got %q", app.statusbar.View(80))
	}
}

func TestToggleFollow_ThreadPanel(t *testing.T) {
	app := NewApp()
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1", Text: "parent"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelThread
	var sub, unsub int
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		IsSubscribed: func(ids.ChannelID, ids.ThreadTS) bool { return false },
		Subscribe: func(channelID ids.ChannelID, threadTS ids.ThreadTS) error {
			sub++
			if string(channelID) != "C1" || string(threadTS) != "P1" {
				t.Fatalf("subscribe args %s %s", channelID, threadTS)
			}
			return nil
		},
		Unsubscribe: func(ids.ChannelID, ids.ThreadTS) error {
			unsub++
			return nil
		},
		ListFetch: func(ids.TeamID) tea.Msg { return nil },
	}))

	cmd := app.handleNormalMode(tea.KeyPressMsg{Code: 't', Text: "t"})
	if cmd == nil {
		t.Fatal("expected follow cmd")
	}
	msg, ok := cmd().(FollowToggledMsg)
	if !ok {
		t.Fatalf("got %T", cmd())
	}
	if !msg.Following || msg.Err != nil {
		t.Fatalf("FollowToggledMsg = %+v", msg)
	}
	if sub != 1 {
		t.Fatalf("subscribe calls = %d", sub)
	}
	app.Update(msg)
	if !app.threadPanel.Following() {
		t.Fatal("thread panel should show following")
	}
	if !strings.Contains(app.statusbar.View(80), "Following thread") {
		t.Fatalf("toast = %q", app.statusbar.View(80))
	}

	cmd = app.handleNormalMode(tea.KeyPressMsg{Code: 't', Text: "t"})
	msg = cmd().(FollowToggledMsg)
	if msg.Following {
		t.Fatal("second t should unfollow")
	}
	app.Update(msg)
	if app.threadPanel.Following() {
		t.Fatal("expected unfollowed")
	}
	if !strings.Contains(app.statusbar.View(80), "Unfollowing") {
		t.Fatalf("toast = %q", app.statusbar.View(80))
	}
	if unsub != 1 {
		t.Fatalf("unsubscribe calls = %d", unsub)
	}
}

func TestToggleFollow_IgnoredWhenThreadNotFocused(t *testing.T) {
	app := NewApp()
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelMessages
	app.SetThreadService(NewThreadService(ThreadServiceFuncs{
		Subscribe: func(ids.ChannelID, ids.ThreadTS) error {
			t.Fatal("must not subscribe when messages pane is focused")
			return nil
		},
	}))
	if cmd := app.handleNormalMode(tea.KeyPressMsg{Code: 't', Text: "t"}); cmd != nil {
		t.Fatalf("expected no-op, got %T", cmd())
	}
}

func TestFollowToggled_RemovesFromThreadsView(t *testing.T) {
	app := NewApp()
	app.threadsView.SetSummaries([]cache.ThreadSummary{
		{ChannelID: "C1", ThreadTS: "P1", ParentText: "hello"},
	})
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.Update(FollowToggledMsg{ChannelID: "C1", ThreadTS: "P1", Following: false})
	if n := len(app.threadsView.Summaries()); n != 0 {
		t.Fatalf("unfollowed thread should drop from list, got %d", n)
	}
}

func TestHandleInsertMode_CtrlEnterBroadcastsThreadReply(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelThread
	app.SetMode(ModeInsert)
	app.threadCompose.SetValue("also to channel")

	cmd := app.handleInsertMode(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected send cmd")
	}
	msg, ok := cmd().(SendThreadReplyMsg)
	if !ok {
		t.Fatalf("got %T, want SendThreadReplyMsg", cmd())
	}
	if !msg.Broadcast {
		t.Fatal("Ctrl+Enter should set Broadcast")
	}
	if msg.Text != "also to channel" || msg.ChannelID != "C1" || msg.ThreadTS != "P1" {
		t.Fatalf("msg = %+v", msg)
	}
}

func TestHandleInsertMode_PlainEnterIsReplyOnly(t *testing.T) {
	app := NewApp()
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelThread
	app.SetMode(ModeInsert)
	app.threadCompose.SetValue("reply only")

	msg := app.handleInsertMode(tea.KeyPressMsg{Code: tea.KeyEnter})()
	got, ok := msg.(SendThreadReplyMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	if got.Broadcast {
		t.Fatal("plain Enter must not broadcast")
	}
}

func TestInsertMode_ThreadComposeShowsBroadcastHint(t *testing.T) {
	app := NewApp()
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelThread
	app.SetMode(ModeInsert)
	view := ansi.Strip(app.statusbar.View(160))
	if !strings.Contains(view, "ctrl+enter") {
		t.Fatalf("expected ctrl+enter hint in status bar, got %q", view)
	}
	app.SetMode(ModeNormal)
	view = ansi.Strip(app.statusbar.View(160))
	if strings.Contains(view, "ctrl+enter also send") {
		t.Fatalf("hint should clear in normal mode, got %q", view)
	}
}

func TestSendThreadReplyBroadcast_InstantDisplayChannelCopy(t *testing.T) {
	app := NewApp()
	app.SetCurrentUserID("USELF")
	app.activeChannelID = "C1"
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "P1", UserName: "alice", Text: "parent"},
	})
	parent := messages.MessageItem{TS: "P1"}
	app.threadPanel.SetThread(parent, nil, "C1", "P1")
	app.threadVisible = true

	app.Update(SendThreadReplyMsg{
		ChannelID: "C1",
		ThreadTS:  "P1",
		Text:      "broadcast me",
		Broadcast: true,
	})
	if got := app.threadPanel.ReplyCount(); got != 1 {
		t.Fatalf("thread replies = %d, want 1", got)
	}
	msgs := app.messagepane.Messages()
	if len(msgs) != 2 {
		t.Fatalf("channel messages = %d, want 2 (parent + broadcast)", len(msgs))
	}
	if msgs[1].Subtype != "thread_broadcast" {
		t.Fatalf("subtype = %q, want thread_broadcast", msgs[1].Subtype)
	}

	localTS := app.threadPanel.Replies()[0].TS
	app.Update(ThreadReplySentMsg{
		ChannelID: "C1",
		ThreadTS:  "P1",
		LocalTS:   localTS,
		Message: messages.MessageItem{
			TS: "2.0", UserID: "USELF", UserName: "you",
			Text: "broadcast me", ThreadTS: "P1", Subtype: "thread_broadcast",
		},
	})
	msgs = app.messagepane.Messages()
	found := false
	for _, m := range msgs {
		if m.TS == "2.0" && m.Subtype == "thread_broadcast" {
			found = true
		}
		if strings.HasPrefix(m.TS, "local:") {
			t.Fatalf("placeholder still in channel pane: %s", m.TS)
		}
	}
	if !found {
		t.Fatal("expected swapped broadcast copy in channel pane")
	}
}

func TestPinToggledMsg_NoopServiceSilent(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{{TS: "1.0", Text: "hi"}})
	cmd := app.togglePinOfSelected()
	if cmd == nil {
		t.Fatal("cmd")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("unwired pin should no-op, got %T %+v", msg, msg)
	}
}

func TestApp_PinToggledShowsToast(t *testing.T) {
	a := NewApp()
	a.Update(PinToggledMsg{ChannelID: "C1", TS: "1.0", Pinned: true})
	if !strings.Contains(a.statusbar.View(80), "Pinned") {
		t.Fatalf("toast = %q", a.statusbar.View(80))
	}
}
