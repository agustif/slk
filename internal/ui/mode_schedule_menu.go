// internal/ui/mode_schedule_menu.go
//
// Duration-picker key handler for scheduled send. Enter confirms a
// preset (20m/1h/…) or opens the custom-minutes sub-mode. Esc restores
// the mode the overlay was opened from (insert or normal).
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/schedulemenu"
)

func handleScheduleMenuMode(a *App, msg tea.KeyMsg) tea.Cmd {
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

	result := a.scheduleMenu.HandleKey(keyStr)
	if result != nil {
		if result.Action == schedulemenu.ActionCustom {
			a.scheduleCustomBuf = ""
			a.SetMode(ModeScheduleCustom)
			return nil
		}
		return a.confirmSchedule(postAtFromResult(*result, time.Now()))
	}
	if !a.scheduleMenu.IsVisible() {
		a.closeScheduleMenu()
	}
	return nil
}
