// internal/ui/date.go
//
// Jump-to-date: parse a local YYYY-MM-DD[ HH:MM], convert to a Slack
// ts, FetchAround, and land the message pane on the nearest message.
package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/messages"
)

const jumpDateUsage = "Invalid date (YYYY-MM-DD or YYYY-MM-DD HH:MM)"

func cmdDate(a *App, args []string) tea.Cmd {
	if len(args) == 0 {
		return a.openDateMenu()
	}
	return a.jumpToDateSpec(strings.Join(args, " "))
}

func (a *App) openDateMenu() tea.Cmd {
	if !a.inChannelView() || a.activeChannelID == "" {
		return toastWithClear(a, "Jump to date works in a channel or DM", 2*time.Second)
	}
	a.dateMenu.Open()
	a.SetMode(ModeDateMenu)
	return nil
}

func (a *App) jumpToDateSpec(spec string) tea.Cmd {
	t, err := parseJumpDate(spec, time.Local)
	if err != nil {
		return toastWithClear(a, jumpDateUsage, 3*time.Second)
	}
	return a.jumpToDate(t)
}

func (a *App) jumpToDate(t time.Time) tea.Cmd {
	if !a.inChannelView() || a.activeChannelID == "" {
		return toastWithClear(a, "Jump to date works in a channel or DM", 2*time.Second)
	}
	ts := slackTSFromTime(t)
	channels := a.channels
	chID := ids.ChannelID(a.activeChannelID)
	return func() tea.Msg {
		msg := channels.FetchAround(chID, ids.MessageTS(ts))
		if msg == nil {
			return MessagesAroundLoadedMsg{
				ChannelID: string(chID),
				TargetTS:  ts,
				JumpDate:  true,
				Err:       errors.New("history fetch failed"),
			}
		}
		if m, ok := msg.(MessagesAroundLoadedMsg); ok {
			m.JumpDate = true
			if m.ChannelID == "" {
				m.ChannelID = string(chID)
			}
			if m.TargetTS == "" {
				m.TargetTS = ts
			}
			return m
		}
		return msg
	}
}

func reduceJumpDateLoaded(a *App, m MessagesAroundLoadedMsg) tea.Cmd {
	if m.Err != nil {
		return func() tea.Msg { return ToastMsg{Text: "Failed to jump to date"} }
	}
	if len(m.Messages) == 0 {
		return func() tea.Msg { return ToastMsg{Text: "No messages around that date"} }
	}
	a.messagepane.SetMessages(a.applyStarredFlags(m.ChannelID, m.Messages))
	if ts := nearestMessageTS(m.Messages, m.TargetTS); ts != "" {
		a.messagepane.SelectByTS(ts)
	}
	return nil
}

// parseJumpDate accepts YYYY-MM-DD (local midnight) or YYYY-MM-DD HH:MM
// (local time). loc nil means time.Local.
func parseJumpDate(spec string, loc *time.Location) (time.Time, error) {
	spec = strings.Join(strings.Fields(spec), " ")
	if spec == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	if loc == nil {
		loc = time.Local
	}
	var (
		t      time.Time
		err    error
		layout string
	)
	switch {
	case len(spec) == len("2006-01-02"):
		layout = "2006-01-02"
		t, err = time.ParseInLocation(layout, spec, loc)
	default:
		layout = "2006-01-02 15:04"
		t, err = time.ParseInLocation(layout, spec, loc)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date")
	}
	if t.Format(layout) != spec {
		return time.Time{}, fmt.Errorf("invalid date")
	}
	return t, nil
}

func slackTSFromTime(t time.Time) string {
	return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
}

// nearestMessageTS picks the first message at or after target, or the
// newest message if the whole window is older. msgs is ascending by TS.
func nearestMessageTS(msgs []messages.MessageItem, target string) string {
	if len(msgs) == 0 {
		return ""
	}
	for _, m := range msgs {
		if m.TS >= target {
			return m.TS
		}
	}
	return msgs[len(msgs)-1].TS
}
