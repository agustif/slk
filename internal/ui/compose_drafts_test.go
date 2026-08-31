package ui

import (
	"testing"

	"github.com/gammons/slk/internal/ui/compose"
	"github.com/gammons/slk/internal/ui/messages"
)

func TestChannelDrafts_SwitchDoesNotLeakAndRestores(t *testing.T) {
	app := NewApp()
	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	app.compose.SetValue("hello a")

	_, _ = app.Update(ChannelSelectedMsg{ID: "CB", Name: "b", Type: "channel"})
	if got := app.compose.Value(); got != "" {
		t.Fatalf("compose leaked into #b: %q", got)
	}

	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	if got := app.compose.Value(); got != "hello a" {
		t.Fatalf("expected #a draft restored, got %q", got)
	}
}

func TestChannelDrafts_RestoresPendingAttachments(t *testing.T) {
	app := NewApp()
	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	app.compose.SetValue("caption")
	app.compose.AddAttachment(compose.PendingAttachment{
		Filename: "shot.png", Path: "/tmp/shot.png", Size: 99,
	})

	_, _ = app.Update(ChannelSelectedMsg{ID: "CB", Name: "b", Type: "channel"})
	if len(app.compose.Attachments()) != 0 {
		t.Fatalf("#b inherited attachments: %+v", app.compose.Attachments())
	}

	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	if app.compose.Value() != "caption" {
		t.Fatalf("text = %q, want caption", app.compose.Value())
	}
	atts := app.compose.Attachments()
	if len(atts) != 1 || atts[0].Filename != "shot.png" {
		t.Fatalf("attachments not restored: %+v", atts)
	}
}

func TestChannelDrafts_EditStashRestoredOnSwitchBack(t *testing.T) {
	app := NewApp()
	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	app.SetCurrentUserID("U_ME")
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1.0", UserID: "U_ME", Text: "my message"},
	})
	app.focusedPanel = PanelMessages
	app.compose.SetValue("draft in progress")
	app.beginEditOfSelected()
	if app.compose.Value() != "my message" {
		t.Fatalf("setup: compose should be the message being edited, got %q", app.compose.Value())
	}

	_, _ = app.Update(ChannelSelectedMsg{ID: "CB", Name: "b", Type: "channel"})
	if app.editing.IsActive() {
		t.Fatal("channel switch should cancel edit")
	}
	if got := app.compose.Value(); got != "" {
		t.Fatalf("edit text leaked into #b: %q", got)
	}

	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	if got := app.compose.Value(); got != "draft in progress" {
		t.Fatalf("expected stashed channel draft, got %q (not the edited message)", got)
	}
}

func TestThreadDrafts_OpenCloseRestores(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	parent1 := messages.MessageItem{TS: "T1", Text: "p1"}
	parent2 := messages.MessageItem{TS: "T2", Text: "p2"}

	_ = app.openThreadPanel(parent1, "C1", "T1")
	app.threadCompose.SetValue("reply to t1")
	app.CloseThread()
	if got := app.threadCompose.Value(); got != "" {
		t.Fatalf("thread compose leaked after close: %q", got)
	}

	_ = app.openThreadPanel(parent2, "C1", "T2")
	if got := app.threadCompose.Value(); got != "" {
		t.Fatalf("t1 draft leaked into t2: %q", got)
	}

	app.CloseThread()
	_ = app.openThreadPanel(parent1, "C1", "T1")
	if got := app.threadCompose.Value(); got != "reply to t1" {
		t.Fatalf("expected t1 draft restored, got %q", got)
	}
}

func TestThreadDrafts_SwitchThreadWithoutClose(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	parent1 := messages.MessageItem{TS: "T1", Text: "p1"}
	parent2 := messages.MessageItem{TS: "T2", Text: "p2"}

	_ = app.openThreadPanel(parent1, "C1", "T1")
	app.threadCompose.SetValue("t1 draft")
	_ = app.openThreadPanel(parent2, "C1", "T2")
	if got := app.threadCompose.Value(); got != "" {
		t.Fatalf("t1 draft leaked into t2: %q", got)
	}
	_ = app.openThreadPanel(parent1, "C1", "T1")
	if got := app.threadCompose.Value(); got != "t1 draft" {
		t.Fatalf("expected t1 draft restored, got %q", got)
	}
}

func TestChannelSwitch_ParksOpenThreadDraft(t *testing.T) {
	app := NewApp()
	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	parent := messages.MessageItem{TS: "T1", Text: "p1"}
	_ = app.openThreadPanel(parent, "CA", "T1")
	app.threadCompose.SetValue("thread draft")
	app.compose.SetValue("channel draft")

	_, _ = app.Update(ChannelSelectedMsg{ID: "CB", Name: "b", Type: "channel"})
	if app.threadVisible {
		t.Fatal("channel switch should close the thread")
	}
	if got := app.compose.Value(); got != "" {
		t.Fatalf("channel draft leaked into #b: %q", got)
	}
	if got := app.threadCompose.Value(); got != "" {
		t.Fatalf("thread compose not parked: %q", got)
	}

	_, _ = app.Update(ChannelSelectedMsg{ID: "CA", Name: "a", Type: "channel"})
	if got := app.compose.Value(); got != "channel draft" {
		t.Fatalf("channel draft not restored: %q", got)
	}
	_ = app.openThreadPanel(parent, "CA", "T1")
	if got := app.threadCompose.Value(); got != "thread draft" {
		t.Fatalf("thread draft not restored after channel round-trip: %q", got)
	}
}

func TestThreadDraftKey_EmptyParts(t *testing.T) {
	if got := threadDraftKey("", "T1"); got != "" {
		t.Fatalf("empty channel: %q", got)
	}
	if got := threadDraftKey("C1", ""); got != "" {
		t.Fatalf("empty ts: %q", got)
	}
	if got := threadDraftKey("C1", "T1"); got != "C1\x00T1" {
		t.Fatalf("got %q", got)
	}
}
