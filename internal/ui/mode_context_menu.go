// internal/ui/mode_context_menu.go
//
// Context-menu mode key handler. Forwards normalised keys to the
// message-actions overlay. Enter on an enabled row closes the menu
// and runs the matching existing action (reaction picker, thread,
// permalink, …). Esc / overlay Close drops back to Normal.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

func handleContextMenuMode(a *App, msg tea.KeyMsg) tea.Cmd {
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
	}

	item := a.contextMenu.HandleKey(keyStr)
	if item != nil {
		a.SetMode(ModeNormal)
		return a.dispatchContextMenuAction(item.Action)
	}
	if !a.contextMenu.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}
