package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/statusbar"
)

func pressYank(a *App) tea.Cmd {
	return handleNormalMode(a, tea.KeyPressMsg{Code: 'y', Text: "y"})
}

func setupYankApp(t *testing.T, body string) (*App, *string) {
	t.Helper()
	app := NewApp()
	var copied string
	app.SetClipboardWriter(func(text string) tea.Cmd {
		copied = text
		return nil
	})
	app.activeChannelID = "C123"
	app.focusedPanel = PanelMessages
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1700000001.000200", UserName: "alice", Text: body},
	})
	app.setPermalinkFetcherForTest(func(ctx context.Context, channelID ids.ChannelID, ts ids.MessageTS) (string, error) {
		t.Fatal("yy must not fetch a permalink")
		return "https://example.slack.com/archives/C123/p1700000001000200", nil
	})
	return app, &copied
}

func TestYY_CopiesMessageTextNotPermalink(t *testing.T) {
	const body = "hello from slack"
	app, copied := setupYankApp(t, body)

	cmd1 := pressYank(app)
	if cmd1 == nil {
		t.Fatal("first y should arm the prefix (timeout tick)")
	}
	if *copied != "" {
		t.Fatalf("single y yanked %q; want no clipboard write", *copied)
	}
	if app.pendingPrefix != 'y' {
		t.Fatalf("pendingPrefix = %q, want 'y'", app.pendingPrefix)
	}

	cmd2 := pressYank(app)
	if cmd2 == nil {
		t.Fatal("expected yank cmd from yy")
	}
	if app.pendingPrefix != 0 {
		t.Fatal("pendingPrefix must clear after completing yy")
	}
	if !drainForMessageCopied(t, cmd2()) {
		t.Fatal("expected statusbar.MessageCopiedMsg in batch")
	}
	if !strings.Contains(*copied, body) {
		t.Errorf("clipboard %q missing message text %q", *copied, body)
	}
	if strings.Contains(*copied, "https://") || strings.Contains(*copied, "/archives/") {
		t.Errorf("clipboard looks like a permalink: %q", *copied)
	}
	if strings.Contains(*copied, "\x1b") {
		t.Errorf("clipboard still contains ANSI: %q", *copied)
	}
}

func TestYY_PrefixTimeoutDoesNotYank(t *testing.T) {
	app, copied := setupYankApp(t, "hello from slack")

	_ = pressYank(app)
	if app.pendingPrefix != 'y' {
		t.Fatal("expected pendingPrefix='y' after first y")
	}
	gen := app.prefixGen
	app.Update(prefixTimeoutMsg{prefix: 'y', gen: gen})
	if app.pendingPrefix != 0 {
		t.Fatal("prefixTimeoutMsg must clear pendingPrefix")
	}
	if *copied != "" {
		t.Fatalf("prefix timeout yanked %q", *copied)
	}

	_ = pressYank(app)
	if *copied != "" {
		t.Fatalf("y after timeout yanked %q; should only re-arm", *copied)
	}
	if app.pendingPrefix != 'y' {
		t.Fatal("y after timeout should re-arm the prefix")
	}
}

func TestYY_OtherKeyCancelsPrefixWithoutYank(t *testing.T) {
	app, copied := setupYankApp(t, "hello from slack")
	_ = pressYank(app)
	_ = handleNormalMode(app, tea.KeyPressMsg{Code: 'j', Text: "j"})
	if app.pendingPrefix != 0 {
		t.Fatal("pendingPrefix must clear after a non-y key")
	}
	if *copied != "" {
		t.Fatalf("y then j yanked %q", *copied)
	}
}

func TestApp_MessageCopiedMsgShowsToast(t *testing.T) {
	a := NewApp()
	_, cmd := a.Update(statusbar.MessageCopiedMsg{})
	if !strings.Contains(a.statusbar.View(80), "Copied message") {
		t.Fatalf("expected 'Copied message' toast; got %q", a.statusbar.View(80))
	}
	if cmd == nil {
		t.Fatal("expected a clear-tick cmd")
	}
}

func drainForMessageCopied(t *testing.T, msg tea.Msg) bool {
	t.Helper()
	switch v := msg.(type) {
	case statusbar.MessageCopiedMsg:
		return true
	case tea.BatchMsg:
		for _, c := range v {
			if c == nil {
				continue
			}
			if drainForMessageCopied(t, c()) {
				return true
			}
		}
	}
	return false
}
