// internal/ui/mode_date.go
//
// Typed-date overlay for jump-to-date. Esc cancels. Enter parses the
// buffer as YYYY-MM-DD[ HH:MM] and FetchAround-s; a bad parse toasts
// and keeps the overlay open.
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func handleDateMenuMode(a *App, msg tea.KeyMsg) tea.Cmd {
	keyStr := msg.String()
	switch msg.Key().Code {
	case tea.KeyEnter:
		keyStr = "enter"
	case tea.KeyEscape:
		keyStr = "esc"
	case tea.KeyBackspace:
		keyStr = "backspace"
	case tea.KeySpace:
		keyStr = " "
	}

	result := a.dateMenu.HandleKey(keyStr)
	if result != nil {
		t, err := parseJumpDate(result.Query, time.Local)
		if err != nil {
			return toastWithClear(a, jumpDateUsage, 2*time.Second)
		}
		a.dateMenu.Close()
		a.SetMode(ModeNormal)
		return a.jumpToDate(t)
	}
	if !a.dateMenu.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}
