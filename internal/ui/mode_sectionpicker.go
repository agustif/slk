// internal/ui/mode_sectionpicker.go
//
// Key handler for ModeSectionPicker: the chooser opened by :move.
// Enter assigns the active channel to the highlighted section.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

func handleSectionPickerMode(a *App, msg tea.KeyMsg) tea.Cmd {
	item, chosen := a.sectionPicker.HandleKey(msg.String())
	if chosen {
		a.SetMode(ModeNormal)
		return moveActiveChannelTo(a, item.ID, item.Label)
	}
	if !a.sectionPicker.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}
