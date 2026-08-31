package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/help"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/schedulemenu"
)

func TestParseScheduleSpec(t *testing.T) {
	now := time.Date(2026, 8, 31, 15, 4, 0, 0, time.Local)
	tests := []struct {
		spec    string
		want    time.Time
		wantErr bool
	}{
		{spec: "20m", want: now.Add(20 * time.Minute)},
		{spec: "1h", want: now.Add(time.Hour)},
		{spec: "1H", want: now.Add(time.Hour)},
		{spec: "45", want: now.Add(45 * time.Minute)},
		{spec: "tomorrow", want: time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)},
		{spec: "", wantErr: true},
		{spec: "0", wantErr: true},
		{spec: "bogon", wantErr: true},
		{spec: "-5m", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseScheduleSpec(tt.spec, now)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseScheduleSpec(%q) err=nil, want error", tt.spec)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseScheduleSpec(%q): %v", tt.spec, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("parseScheduleSpec(%q) = %v, want %v", tt.spec, got, tt.want)
		}
	}
}

func TestTomorrowMorning(t *testing.T) {
	now := time.Date(2026, 8, 31, 15, 4, 0, 0, time.Local)
	got := tomorrowMorning(now)
	want := time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("tomorrowMorning = %v, want %v", got, want)
	}
}

func TestHandleInsertMode_CtrlGOpensScheduleMenu(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.focusedPanel = PanelMessages
	app.SetMode(ModeInsert)
	app.compose.SetValue("hello")

	cmd := app.handleInsertMode(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Fatalf("opening overlay should not emit a cmd, got %T", cmd)
	}
	if app.mode != ModeScheduleMenu {
		t.Fatalf("mode = %v, want ModeScheduleMenu", app.mode)
	}
	if !app.scheduleMenu.IsVisible() {
		t.Fatal("expected schedule overlay visible")
	}
	if app.compose.Value() != "hello" {
		t.Fatalf("compose should be unchanged until confirm, got %q", app.compose.Value())
	}
}

func TestHandleInsertMode_CtrlGEmptyToasts(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.SetMode(ModeInsert)
	_ = app.handleInsertMode(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if app.mode != ModeInsert {
		t.Fatalf("mode = %v, want ModeInsert", app.mode)
	}
	if out := app.statusbar.View(120); !strings.Contains(out, "Nothing to schedule") {
		t.Fatalf("expected empty-compose toast, got:\n%s", out)
	}
}

func TestScheduleMenu_Pick20mEmitsScheduleMessage(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.focusedPanel = PanelMessages
	app.SetMode(ModeInsert)
	app.compose.SetValue("hello")

	_ = handleInsertMode(app, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	before := time.Now()
	cmd := handleScheduleMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected ScheduleMessageMsg cmd")
	}
	msg := cmd()
	sm, ok := msg.(ScheduleMessageMsg)
	if !ok {
		t.Fatalf("expected ScheduleMessageMsg, got %T", msg)
	}
	if sm.ChannelID != "C1" || sm.Text != "hello" || sm.ThreadTS != "" {
		t.Fatalf("msg = %+v", sm)
	}
	delta := sm.PostAt.Sub(before)
	if delta < 19*time.Minute || delta > 21*time.Minute {
		t.Fatalf("PostAt delta = %v, want ~20m", delta)
	}
	if app.compose.Value() != "" {
		t.Fatalf("compose should be cleared, got %q", app.compose.Value())
	}
	if app.mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", app.mode)
	}
}

func TestScheduleMenu_EscapeRestoresInsert(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.SetMode(ModeInsert)
	app.compose.SetValue("hello")
	app.compose.Focus()
	_ = handleInsertMode(app, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	_ = handleScheduleMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEscape})
	if app.mode != ModeInsert {
		t.Fatalf("mode = %v, want ModeInsert", app.mode)
	}
	if app.compose.Value() != "hello" {
		t.Fatalf("compose = %q, want hello", app.compose.Value())
	}
	if app.scheduleMenu.IsVisible() {
		t.Fatal("overlay should be closed")
	}
}

func TestExecuteCommand_Schedule20m(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	a.compose.SetValue("later")
	before := time.Now()
	cmd := executeCommand(a, "schedule 20m")
	if cmd == nil {
		t.Fatal("expected ScheduleMessageMsg cmd")
	}
	msg := cmd()
	sm, ok := msg.(ScheduleMessageMsg)
	if !ok {
		t.Fatalf("expected ScheduleMessageMsg, got %T", msg)
	}
	if sm.Text != "later" || sm.ChannelID != "C1" {
		t.Fatalf("msg = %+v", sm)
	}
	delta := sm.PostAt.Sub(before)
	if delta < 19*time.Minute || delta > 21*time.Minute {
		t.Fatalf("PostAt delta = %v, want ~20m", delta)
	}
	if a.compose.Value() != "" {
		t.Fatalf("compose should be cleared, got %q", a.compose.Value())
	}
}

func TestExecuteCommand_Schedule1h(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	a.compose.SetValue("later")
	before := time.Now()
	cmd := executeCommand(a, "schedule 1h")
	sm := cmd().(ScheduleMessageMsg)
	delta := sm.PostAt.Sub(before)
	if delta < 59*time.Minute || delta > 61*time.Minute {
		t.Fatalf("PostAt delta = %v, want ~1h", delta)
	}
}

func TestExecuteCommand_ScheduleNoArgsOpensMenu(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	a.compose.SetValue("hello")
	_ = executeCommand(a, "schedule")
	if a.mode != ModeScheduleMenu {
		t.Fatalf("mode = %v, want ModeScheduleMenu", a.mode)
	}
}

func TestExecuteCommand_ScheduleEmptyToasts(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	_ = executeCommand(a, "schedule 20m")
	if out := a.statusbar.View(120); !strings.Contains(out, "Nothing to schedule") {
		t.Fatalf("expected empty-compose toast, got:\n%s", out)
	}
}

func TestExecuteCommand_ScheduleInvalidToasts(t *testing.T) {
	a := NewApp()
	a.activeChannelID = "C1"
	a.compose.SetValue("hello")
	_ = executeCommand(a, "schedule bogon")
	if out := a.statusbar.View(120); !strings.Contains(out, "Invalid schedule") {
		t.Fatalf("expected invalid-duration toast, got:\n%s", out)
	}
}

func TestScheduleMenu_ThreadReplySetsThreadTS(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.threadPanel.SetThread(messages.MessageItem{TS: "P1"}, nil, "C1", "P1")
	app.threadVisible = true
	app.focusedPanel = PanelThread
	app.SetMode(ModeInsert)
	app.threadCompose.SetValue("reply later")

	_ = handleInsertMode(app, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	cmd := handleScheduleMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	sm := cmd().(ScheduleMessageMsg)
	if sm.ThreadTS != "P1" || sm.ChannelID != "C1" || sm.Text != "reply later" {
		t.Fatalf("msg = %+v", sm)
	}
	if app.threadCompose.Value() != "" {
		t.Fatalf("thread compose should be cleared, got %q", app.threadCompose.Value())
	}
}

func TestScheduleCustom_Minutes(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.compose.SetValue("hello")
	_ = app.openScheduleMenu()
	for i := 0; i < 6; i++ {
		_ = handleScheduleMenuMode(app, tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	_ = handleScheduleMenuMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if app.mode != ModeScheduleCustom {
		t.Fatalf("mode = %v, want ModeScheduleCustom", app.mode)
	}
	_ = handleScheduleCustomMode(app, tea.KeyPressMsg{Code: '4', Text: "4"})
	_ = handleScheduleCustomMode(app, tea.KeyPressMsg{Code: '5', Text: "5"})
	before := time.Now()
	cmd := handleScheduleCustomMode(app, tea.KeyPressMsg{Code: tea.KeyEnter})
	sm := cmd().(ScheduleMessageMsg)
	delta := sm.PostAt.Sub(before)
	if delta < 44*time.Minute || delta > 46*time.Minute {
		t.Fatalf("PostAt delta = %v, want ~45m", delta)
	}
}

func TestReduceScheduleMessage_SuccessToast(t *testing.T) {
	a := NewApp()
	postAt := time.Date(2026, 8, 31, 15, 4, 0, 0, time.Local)
	a.SetMessageService(NewMessageService(MessageServiceFuncs{
		Schedule: func(channelID ids.ChannelID, threadTS ids.ThreadTS, text string, at time.Time) tea.Msg {
			if channelID != "C1" || text != "hi" || !at.Equal(postAt) {
				t.Errorf("Schedule(%q, %q, %q, %v)", channelID, threadTS, text, at)
			}
			return MessageScheduledMsg{ChannelID: string(channelID), PostAt: at}
		},
	}))
	cmd, handled := reduceSend.Handle(a, ScheduleMessageMsg{ChannelID: "C1", Text: "hi", PostAt: postAt})
	if !handled || cmd == nil {
		t.Fatalf("handled=%v cmd=%v", handled, cmd)
	}
	cmd2, handled2 := reduceSend.Handle(a, cmd())
	if !handled2 {
		t.Fatal("expected MessageScheduledMsg to be handled")
	}
	_ = cmd2
	if out := a.statusbar.View(120); !strings.Contains(out, "Scheduled for 3:04 PM") {
		t.Fatalf("expected schedule toast, got:\n%s", out)
	}
}

func TestHelp_ListsScheduleBinding(t *testing.T) {
	entries := help.FromKeyMap(DefaultKeyMap())
	found := false
	for _, e := range entries {
		if e.Key == "ctrl+g / :schedule" && e.Desc == "schedule message" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("help entries missing schedule binding")
	}
}

func TestPostAtFromResult_Tomorrow(t *testing.T) {
	now := time.Date(2026, 8, 31, 15, 4, 0, 0, time.Local)
	got := postAtFromResult(schedulemenu.Result{Action: schedulemenu.ActionTomorrowMorning}, now)
	want := time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("tomorrow = %v, want %v", got, want)
	}
}
