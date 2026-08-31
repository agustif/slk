package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/presencemenu"
)

func handleRemindDurationMode(a *App, msg tea.KeyMsg) tea.Cmd {
	keyStr := msg.String()
	switch msg.Key().Code {
	case tea.KeyEnter:
		keyStr = "enter"
	case tea.KeyEscape:
		keyStr = "esc"
	case tea.KeyUp:
		keyStr = "up"
	case tea.KeyDown:
		keyStr = "down"
	case tea.KeyBackspace:
		keyStr = "backspace"
	}

	result := a.presenceMenu.HandleKey(keyStr)
	if result != nil {
		a.presenceMenu.Close()
		a.SetMode(ModeNormal)
		if result.Action == presencemenu.ActionSnooze && result.SnoozeMinutes > 0 {
			return a.remindSelected(result.SnoozeMinutes)
		}
		return nil
	}
	if !a.presenceMenu.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}
