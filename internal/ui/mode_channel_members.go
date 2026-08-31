// internal/ui/mode_channel_members.go
//
// Channel-members mode key handler. Forwards normalised keys to the
// members overlay. Enter on a row dispatches OpenConversation (the
// same conversations.open path as Ctrl+N) for that one user.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

func handleChannelMembersMode(a *App, msg tea.KeyMsg) tea.Cmd {
	result := a.channelMembers.HandleKey(normalizeFinderKey(msg))
	if result != nil {
		a.channelMembers.Close()
		a.SetMode(ModeNormal)
		a.newMessageInFlightID++
		a.newMessageCancelled = false
		reqID := a.newMessageInFlightID
		return a.channels.OpenConversation([]string{result.UserID}, reqID)
	}
	if !a.channelMembers.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}
