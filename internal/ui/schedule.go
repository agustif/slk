// internal/ui/schedule.go
//
// Helpers for compose scheduled-send: duration parsing, overlay open /
// cancel, and confirm that clears compose and emits ScheduleMessageMsg.
package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/compose"
	"github.com/gammons/slk/internal/ui/schedulemenu"
)

const maxScheduleAhead = 120 * 24 * time.Hour

func (a *App) scheduleUsingThread() bool {
	return a.focusedPanel == PanelThread && a.threadVisible
}

func (a *App) scheduleTarget() *compose.Model {
	if a.scheduleUsingThread() {
		return &a.threadCompose
	}
	return &a.compose
}

func (a *App) scheduleChannelAndThread() (channelID, threadTS string) {
	if a.scheduleUsingThread() {
		return a.threadPanel.ChannelID(), a.threadPanel.ThreadTS()
	}
	return a.activeChannelID, ""
}

func (a *App) schedulePrecheck() string {
	if a.editing.IsActive() {
		return "Can't schedule an edit"
	}
	target := a.scheduleTarget()
	if target.Uploading() {
		return "Upload in progress"
	}
	if len(target.Attachments()) > 0 {
		return "Can't schedule attachments"
	}
	if target.Value() == "" {
		return "Nothing to schedule"
	}
	ch, _ := a.scheduleChannelAndThread()
	if ch == "" {
		return "No channel selected"
	}
	return ""
}

func (a *App) openScheduleMenu() tea.Cmd {
	if msg := a.schedulePrecheck(); msg != "" {
		return toastWithClear(a, msg, 2*time.Second)
	}
	a.scheduleReturnMode = a.mode
	if a.scheduleReturnMode != ModeInsert {
		a.scheduleReturnMode = ModeNormal
	}
	a.scheduleCustomBuf = ""
	a.scheduleMenu.Open()
	a.SetMode(ModeScheduleMenu)
	return nil
}

func (a *App) closeScheduleMenu() {
	a.scheduleMenu.Close()
	a.scheduleCustomBuf = ""
	mode := a.scheduleReturnMode
	a.scheduleReturnMode = ModeNormal
	a.SetMode(mode)
	if mode == ModeInsert {
		if a.scheduleUsingThread() {
			a.threadCompose.Focus()
		} else {
			a.compose.Focus()
		}
	}
}

func postAtFromResult(r schedulemenu.Result, now time.Time) time.Time {
	switch r.Action {
	case schedulemenu.ActionTomorrowMorning:
		return tomorrowMorning(now)
	default:
		return now.Add(r.Duration)
	}
}

func tomorrowMorning(now time.Time) time.Time {
	loc := now.Location()
	return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1)
}

func parseScheduleSpec(spec string, now time.Time) (time.Time, error) {
	spec = strings.TrimSpace(strings.ToLower(spec))
	if spec == "" {
		return time.Time{}, fmt.Errorf("empty duration")
	}
	if spec == "tomorrow" {
		return tomorrowMorning(now), nil
	}
	if mins, err := strconv.Atoi(spec); err == nil {
		if mins <= 0 {
			return time.Time{}, fmt.Errorf("duration must be positive")
		}
		return now.Add(time.Duration(mins) * time.Minute), nil
	}
	d, err := time.ParseDuration(spec)
	if err != nil || d <= 0 {
		return time.Time{}, fmt.Errorf("invalid duration")
	}
	return now.Add(d), nil
}

func validatePostAt(postAt, now time.Time) error {
	if !postAt.After(now) {
		return fmt.Errorf("Time must be in the future")
	}
	if postAt.After(now.Add(maxScheduleAhead)) {
		return fmt.Errorf("Time must be within 120 days")
	}
	return nil
}

func (a *App) confirmSchedule(postAt time.Time) tea.Cmd {
	if err := validatePostAt(postAt, time.Now()); err != nil {
		return toastWithClear(a, err.Error(), 2*time.Second)
	}
	if msg := a.schedulePrecheck(); msg != "" {
		return toastWithClear(a, msg, 2*time.Second)
	}
	target := a.scheduleTarget()
	text := target.TranslateMentionsForSend(target.Value())
	channelID, threadTS := a.scheduleChannelAndThread()
	target.Reset()
	a.scheduleMenu.Close()
	a.scheduleCustomBuf = ""
	a.exitInsertAfterSend()
	return func() tea.Msg {
		return ScheduleMessageMsg{
			ChannelID: channelID,
			ThreadTS:  threadTS,
			Text:      text,
			PostAt:    postAt,
		}
	}
}
