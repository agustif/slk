package ui

import (
	tea "charm.land/bubbletea/v2"

	slackclient "github.com/agustif/slk/internal/slack"
)

func handlePresenceSetStatusMode(a *App, msg tea.KeyMsg) tea.Cmd {
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
	case tea.KeyTab:
		keyStr = "tab"
	}

	result := a.statusInput.HandleKey(keyStr)
	if result != nil {
		a.SetMode(ModeNormal)
		st := slackclient.UserStatus{
			Text:       result.StatusText,
			Emoji:      result.StatusEmoji,
			Expiration: result.StatusExpiration,
		}
		a.applyOwnCustomStatus(st)
		if a.setCustomStatusFn != nil {
			a.setCustomStatusFn(st)
		}
		return nil
	}
	if !a.statusInput.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}

func (a *App) applyOwnCustomStatus(st slackclient.UserStatus) {
	if a.currentUserID == "" {
		return
	}
	a.sidebar.UpdateUserStatus(a.currentUserID, st)
}
