package ui

import (
	"strings"
	"testing"

	"github.com/gammons/slk/internal/ui/messages"
)

func TestExecuteCommand_EmptyIsNoop(t *testing.T) {
	a := NewApp()
	if cmd := executeCommand(a, "   "); cmd != nil {
		t.Fatal("empty command line should be a no-op")
	}
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", a.mode)
	}
}

func TestExecuteCommand_UnknownShowsToast(t *testing.T) {
	a := NewApp()
	cmd := executeCommand(a, "bogus")
	if cmd == nil {
		t.Fatal("unknown command should return the toast-clear cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Unknown command: bogus") {
		t.Fatalf("expected unknown-command toast, got:\n%s", out)
	}
}

func TestExecuteCommand_WSOpensWorkspaceFinder(t *testing.T) {
	a := NewApp()
	_ = executeCommand(a, "ws")
	if a.mode != ModeWorkspaceFinder {
		t.Fatalf("mode = %v, want ModeWorkspaceFinder", a.mode)
	}
}

func TestParseRemindDuration(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"20m", 20},
		{"1h", 60},
		{"2d", 2880},
		{"45", 45},
	}
	for _, c := range cases {
		got, err := parseRemindDuration(c.in)
		if err != nil {
			t.Errorf("parseRemindDuration(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseRemindDuration(%q) = %d, want %d", c.in, got, c.want)
		}
	}
	if _, err := parseRemindDuration("nope"); err == nil {
		t.Error("want error for nope")
	}
}

func TestExecuteCommand_RemindNoArgsOpensMenu(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	a.focusedPanel = PanelMessages
	a.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1.0", UserName: "alice", Text: "hi"},
	})
	_ = executeCommand(a, "remind")
	if a.mode != ModeRemindDuration {
		t.Fatalf("mode = %v, want ModeRemindDuration", a.mode)
	}
}

func TestExecuteCommand_TrimsAndIgnoresArgs(t *testing.T) {
	a := NewApp()
	_ = executeCommand(a, "  ws   extra  ")
	if a.mode != ModeWorkspaceFinder {
		t.Fatalf("mode = %v, want ModeWorkspaceFinder", a.mode)
	}
}
