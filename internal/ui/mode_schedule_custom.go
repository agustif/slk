// internal/ui/mode_schedule_custom.go
//
// Numeric-minutes input for scheduled send, cloned from the presence
// custom-snooze prompt. Esc returns to the duration menu.
package ui

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
)

func handleScheduleCustomMode(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.Key().Code {
	case tea.KeyEscape:
		a.scheduleCustomBuf = ""
		a.SetMode(ModeScheduleMenu)
		return nil
	case tea.KeyEnter:
		mins, err := strconv.Atoi(a.scheduleCustomBuf)
		if err != nil || mins <= 0 {
			return toastWithClear(a, "Invalid schedule duration", 2*time.Second)
		}
		return a.confirmSchedule(time.Now().Add(time.Duration(mins) * time.Minute))
	case tea.KeyBackspace:
		if a.scheduleCustomBuf != "" {
			a.scheduleCustomBuf = a.scheduleCustomBuf[:len(a.scheduleCustomBuf)-1]
		}
		return nil
	}
	s := msg.String()
	if len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
		a.scheduleCustomBuf += s
	}
	return nil
}
