// internal/ui/mode_share.go
//
// Share-destination mode: the channel-finder overlay with synthetic
// views and unjoined channels hidden. Enter posts the stored
// message's permalink to the chosen conversation; Esc cancels.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

func handleShareMode(a *App, msg tea.KeyMsg) tea.Cmd {
	result := a.channelFinder.HandleKey(normalizeFinderKey(msg))
	if result != nil {
		cmd := a.shareMessageTo(result)
		a.closeSharePicker()
		return cmd
	}
	if !a.channelFinder.IsVisible() {
		a.closeSharePicker()
	}
	return nil
}
