package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/sidebar"
)

func TestParseJumpDate(t *testing.T) {
	loc := time.FixedZone("CST", -6*3600)
	tests := []struct {
		in      string
		want    time.Time
		wantErr bool
	}{
		{in: "2024-01-15", want: time.Date(2024, 1, 15, 0, 0, 0, 0, loc)},
		{in: "2024-01-15 14:30", want: time.Date(2024, 1, 15, 14, 30, 0, 0, loc)},
		{in: "  2024-06-01  09:05  ", want: time.Date(2024, 6, 1, 9, 5, 0, 0, loc)},
		{in: "2024-02-29", want: time.Date(2024, 2, 29, 0, 0, 0, 0, loc)},
		{in: "", wantErr: true},
		{in: "2024/01/15", wantErr: true},
		{in: "01-15-2024", wantErr: true},
		{in: "2024-13-01", wantErr: true},
		{in: "2024-02-30", wantErr: true},
		{in: "2023-02-29", wantErr: true},
		{in: "2024-01-15 25:00", wantErr: true},
		{in: "2024-01-15 14", wantErr: true},
		{in: "2024-01-15T14:30", wantErr: true},
		{in: "not-a-date", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseJumpDate(tt.in, loc)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseJumpDate(%q) err=nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseJumpDate(%q): %v", tt.in, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("parseJumpDate(%q) = %v, want %v", tt.in, got, tt.want)
		}
		if got.Location() != loc {
			t.Errorf("parseJumpDate(%q) location = %v, want %v", tt.in, got.Location(), loc)
		}
	}
}

func TestSlackTSFromTime(t *testing.T) {
	loc := time.FixedZone("UTC-5", -5*3600)
	tm := time.Date(2024, 6, 15, 0, 0, 0, 0, loc)
	got := slackTSFromTime(tm)
	want := fmt.Sprintf("%d.000000", tm.Unix())
	if got != want {
		t.Fatalf("slackTSFromTime = %q, want %q", got, want)
	}
	withTime := time.Date(2024, 6, 15, 14, 30, 0, 0, loc)
	got = slackTSFromTime(withTime)
	want = fmt.Sprintf("%d.000000", withTime.Unix())
	if got != want {
		t.Fatalf("slackTSFromTime(HH:MM) = %q, want %q", got, want)
	}
}

func TestNearestMessageTS(t *testing.T) {
	msgs := []messages.MessageItem{
		{TS: "1700000001.000000"},
		{TS: "1700000005.000000"},
		{TS: "1700000010.000000"},
	}
	if got := nearestMessageTS(nil, "1"); got != "" {
		t.Errorf("empty = %q", got)
	}
	if got := nearestMessageTS(msgs, "1700000005.000000"); got != "1700000005.000000" {
		t.Errorf("exact = %q", got)
	}
	if got := nearestMessageTS(msgs, "1700000003.000000"); got != "1700000005.000000" {
		t.Errorf("first after = %q", got)
	}
	if got := nearestMessageTS(msgs, "1700000000.000000"); got != "1700000001.000000" {
		t.Errorf("before all = %q", got)
	}
	if got := nearestMessageTS(msgs, "1700000099.000000"); got != "1700000010.000000" {
		t.Errorf("after all = %q", got)
	}
}

func seedJumpChannel(t *testing.T) *App {
	t.Helper()
	a := NewApp()
	a.view = ViewChannels
	a.activeChannelID = "C1"
	a.SetChannels([]sidebar.ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
	})
	a.messagepane.SetMessages([]messages.MessageItem{
		{TS: "100.000000", Text: "keep-me"},
	})
	return a
}

func TestExecuteCommand_DateNoArgsOpensOverlay(t *testing.T) {
	a := seedJumpChannel(t)
	if cmd := executeCommand(a, "date"); cmd != nil {
		t.Fatalf("expected nil cmd, got %T", cmd)
	}
	if a.mode != ModeDateMenu {
		t.Fatalf("mode = %v, want ModeDateMenu", a.mode)
	}
	if !a.dateMenu.IsVisible() {
		t.Fatal("expected date overlay visible")
	}
}

func TestExecuteCommand_JumpAliasOpensOverlay(t *testing.T) {
	a := seedJumpChannel(t)
	_ = executeCommand(a, "jump")
	if a.mode != ModeDateMenu {
		t.Fatalf("mode = %v, want ModeDateMenu", a.mode)
	}
}

func TestExecuteCommand_DateOutsideConversationToasts(t *testing.T) {
	views := []View{ViewThreads, ViewActivity, ViewLater, ViewDrafts, ViewUnreads}
	for _, v := range views {
		a := seedJumpChannel(t)
		a.view = v
		cmd := executeCommand(a, "date")
		if cmd == nil {
			t.Fatalf("view %v: expected toast cmd", v)
		}
		if a.dateMenu.IsVisible() {
			t.Errorf("view %v: overlay should not open", v)
		}
		if out := a.statusbar.View(120); !strings.Contains(out, "Jump to date works in a channel or DM") {
			t.Errorf("view %v: toast missing:\n%s", v, out)
		}
	}
}

func TestExecuteCommand_DateInDMsOpensOverlay(t *testing.T) {
	a := seedJumpChannel(t)
	a.view = ViewDMs
	_ = executeCommand(a, "date")
	if a.mode != ModeDateMenu {
		t.Fatalf("mode = %v, want ModeDateMenu", a.mode)
	}
}

func TestExecuteCommand_DateInvalidArgToasts(t *testing.T) {
	a := seedJumpChannel(t)
	cmd := executeCommand(a, "date nope")
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if a.dateMenu.IsVisible() {
		t.Error("overlay should not open on a bad arg")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Invalid date") {
		t.Errorf("toast missing:\n%s", out)
	}
}

func TestExecuteCommand_DateWithArgFetchesAround(t *testing.T) {
	a := seedJumpChannel(t)
	loc := time.Local
	want, err := parseJumpDate("2024-06-15", loc)
	if err != nil {
		t.Fatal(err)
	}
	wantTS := slackTSFromTime(want)
	var gotCh ids.ChannelID
	var gotTS ids.MessageTS
	setChannelFetchAroundForTest(a, func(channelID ids.ChannelID, ts ids.MessageTS) tea.Msg {
		gotCh, gotTS = channelID, ts
		return MessagesAroundLoadedMsg{
			ChannelID: string(channelID),
			TargetTS:  string(ts),
			Messages: []messages.MessageItem{
				{TS: "1718400001.000000", Text: "around"},
			},
		}
	})
	cmd := executeCommand(a, "date 2024-06-15")
	if cmd == nil {
		t.Fatal("expected fetch cmd")
	}
	msg := cmd()
	m, ok := msg.(MessagesAroundLoadedMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	if !m.JumpDate {
		t.Fatal("expected JumpDate on wrapped FetchAround result")
	}
	if gotCh != "C1" {
		t.Errorf("FetchAround channel = %q, want C1", gotCh)
	}
	if string(gotTS) != wantTS {
		t.Errorf("FetchAround ts = %q, want %q", gotTS, wantTS)
	}
	if m.TargetTS != wantTS {
		t.Errorf("TargetTS = %q, want %q", m.TargetTS, wantTS)
	}
}

func TestExecuteCommand_JumpWithTimeArgFetchesAround(t *testing.T) {
	a := seedJumpChannel(t)
	want, err := parseJumpDate("2024-06-15 14:30", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	wantTS := slackTSFromTime(want)
	var gotTS ids.MessageTS
	setChannelFetchAroundForTest(a, func(channelID ids.ChannelID, ts ids.MessageTS) tea.Msg {
		gotTS = ts
		return MessagesAroundLoadedMsg{ChannelID: string(channelID), TargetTS: string(ts)}
	})
	cmd := executeCommand(a, "jump 2024-06-15 14:30")
	if cmd == nil {
		t.Fatal("expected fetch cmd")
	}
	msg := cmd()
	m, ok := msg.(MessagesAroundLoadedMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	if !m.JumpDate {
		t.Fatal("expected JumpDate")
	}
	if string(gotTS) != wantTS {
		t.Errorf("FetchAround ts = %q, want %q", gotTS, wantTS)
	}
}

func TestMessagesAroundLoaded_JumpDateSelectsNearest(t *testing.T) {
	a := seedJumpChannel(t)
	a.Update(MessagesAroundLoadedMsg{
		ChannelID: "C1",
		TargetTS:  "1700000004.000000",
		JumpDate:  true,
		Messages: []messages.MessageItem{
			{TS: "1700000001.000000", Text: "older"},
			{TS: "1700000005.000000", Text: "after"},
			{TS: "1700000009.000000", Text: "newer"},
		},
	})
	sel, ok := a.messagepane.SelectedMessage()
	if !ok || sel.TS != "1700000005.000000" {
		t.Fatalf("selected = %+v ok=%v, want 1700000005.000000", sel, ok)
	}
	if a.activeChannelID != "C1" {
		t.Errorf("activeChannelID = %q, want C1", a.activeChannelID)
	}
}

func TestMessagesAroundLoaded_JumpDateEmptyToastsAndKeepsBuffer(t *testing.T) {
	a := seedJumpChannel(t)
	_, cmd := a.Update(MessagesAroundLoadedMsg{
		ChannelID: "C1",
		TargetTS:  "1700000004.000000",
		JumpDate:  true,
		Messages:  []messages.MessageItem{},
	})
	found := false
	for _, m := range drainCmd(cmd) {
		if tm, ok := m.(ToastMsg); ok && tm.Text == "No messages around that date" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected no-messages toast")
	}
	sel, ok := a.messagepane.SelectedMessage()
	if !ok || sel.Text != "keep-me" {
		t.Fatalf("buffer replaced on empty jump: %+v ok=%v", sel, ok)
	}
}

func TestMessagesAroundLoaded_JumpDateErrToastsAndKeepsBuffer(t *testing.T) {
	a := seedJumpChannel(t)
	_, cmd := a.Update(MessagesAroundLoadedMsg{
		ChannelID: "C1",
		TargetTS:  "1700000004.000000",
		JumpDate:  true,
		Err:       errors.New("network"),
	})
	found := false
	for _, m := range drainCmd(cmd) {
		if tm, ok := m.(ToastMsg); ok && tm.Text == "Failed to jump to date" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected network-error toast")
	}
	sel, ok := a.messagepane.SelectedMessage()
	if !ok || sel.Text != "keep-me" {
		t.Fatalf("buffer replaced on error: %+v ok=%v", sel, ok)
	}
}

func TestMessagesAroundLoaded_JumpDateStaleChannelIgnored(t *testing.T) {
	a := seedJumpChannel(t)
	a.Update(MessagesAroundLoadedMsg{
		ChannelID: "C9",
		TargetTS:  "1700000004.000000",
		JumpDate:  true,
		Messages: []messages.MessageItem{
			{TS: "1700000005.000000", Text: "other"},
		},
	})
	sel, ok := a.messagepane.SelectedMessage()
	if !ok || sel.Text != "keep-me" {
		t.Fatalf("stale jump replaced buffer: %+v ok=%v", sel, ok)
	}
}

func TestHandleNormalMode_JOpensDateMenu(t *testing.T) {
	a := seedJumpChannel(t)
	cmd := handleNormalMode(a, tea.KeyPressMsg{Code: 'J', Text: "J"})
	if cmd != nil {
		t.Fatalf("opening overlay should not emit a cmd, got %T", cmd)
	}
	if a.mode != ModeDateMenu {
		t.Fatalf("mode = %v, want ModeDateMenu", a.mode)
	}
}

func TestHandleDateMenuMode_EnterJumps(t *testing.T) {
	a := seedJumpChannel(t)
	var gotTS ids.MessageTS
	setChannelFetchAroundForTest(a, func(channelID ids.ChannelID, ts ids.MessageTS) tea.Msg {
		gotTS = ts
		return MessagesAroundLoadedMsg{ChannelID: string(channelID), TargetTS: string(ts)}
	})
	_ = executeCommand(a, "date")
	for _, r := range "2024-06-15" {
		_ = handleDateMenuMode(a, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	cmd := handleDateMenuMode(a, tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", a.mode)
	}
	if a.dateMenu.IsVisible() {
		t.Fatal("overlay should close on a valid date")
	}
	if cmd == nil {
		t.Fatal("expected fetch cmd")
	}
	msg := cmd()
	m, ok := msg.(MessagesAroundLoadedMsg)
	if !ok || !m.JumpDate {
		t.Fatalf("got %#v", msg)
	}
	want := slackTSFromTime(mustParseJumpDate(t, "2024-06-15"))
	if string(gotTS) != want {
		t.Errorf("ts = %q, want %q", gotTS, want)
	}
}

func TestHandleDateMenuMode_InvalidEnterToastsAndStays(t *testing.T) {
	a := seedJumpChannel(t)
	_ = executeCommand(a, "date")
	for _, r := range "nope" {
		_ = handleDateMenuMode(a, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	cmd := handleDateMenuMode(a, tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.mode != ModeDateMenu {
		t.Fatalf("mode = %v, want ModeDateMenu", a.mode)
	}
	if !a.dateMenu.IsVisible() {
		t.Fatal("overlay should stay open on a bad parse")
	}
	if cmd == nil {
		t.Fatal("expected toast cmd")
	}
	if out := a.statusbar.View(120); !strings.Contains(out, "Invalid date") {
		t.Errorf("toast missing:\n%s", out)
	}
}

func TestHandleDateMenuMode_EscapeCancels(t *testing.T) {
	a := seedJumpChannel(t)
	_ = executeCommand(a, "date")
	cmd := handleDateMenuMode(a, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatalf("esc should be a no-op cmd, got %T", cmd)
	}
	if a.mode != ModeNormal {
		t.Fatalf("mode = %v, want ModeNormal", a.mode)
	}
	if a.dateMenu.IsVisible() {
		t.Fatal("overlay should close on esc")
	}
}

func mustParseJumpDate(t *testing.T, spec string) time.Time {
	t.Helper()
	tm, err := parseJumpDate(spec, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}
