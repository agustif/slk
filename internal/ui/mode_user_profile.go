package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/userprofile"
)

func handleUserProfileMode(a *App, msg tea.KeyMsg) tea.Cmd {
	keyStr := msg.String()
	switch msg.Key().Code {
	case tea.KeyEnter:
		keyStr = "enter"
	case tea.KeyEscape:
		keyStr = "esc"
	}

	userID := a.userProfile.UserID()
	result := a.userProfile.HandleKey(keyStr)
	if result != nil && result.Message && userID != "" {
		a.SetMode(ModeNormal)
		if a.isDMWith(userID) {
			return nil
		}
		a.newMessageInFlightID++
		a.newMessageCancelled = false
		return a.channels.OpenConversation([]string{userID}, a.newMessageInFlightID)
	}
	if !a.userProfile.IsVisible() {
		a.SetMode(ModeNormal)
	}
	return nil
}

func (a *App) openUserProfile() tea.Cmd {
	userID, display := a.selectedMessageUser()
	if userID == "" {
		return nil
	}
	st, _ := a.sidebar.StatusForUser(userID)
	p := userprofile.Profile{
		UserID:      userID,
		DisplayName: display,
		Status:      st,
		Presence:    a.sidebar.PresenceForUser(userID),
		IsSelf:      userID == a.currentUserID,
		AlreadyInDM: a.isDMWith(userID),
		Loading:     true,
	}
	if p.DisplayName == "" {
		p.DisplayName = a.userNames[userID]
	}
	a.userProfile.Open(p)
	a.SetMode(ModeUserProfile)
	if a.profileFetchFn == nil {
		p.Loading = false
		a.userProfile.SetProfile(p)
		return nil
	}
	return a.profileFetchFn(userID)
}

func (a *App) selectedMessageUser() (userID, display string) {
	if a.focusedPanel == PanelThread {
		if reply := a.threadPanel.SelectedReply(); reply != nil {
			return reply.UserID, reply.UserName
		}
	}
	if msg, ok := a.messagepane.SelectedMessage(); ok {
		return msg.UserID, msg.UserName
	}
	return "", ""
}

func (a *App) isDMWith(userID string) bool {
	if userID == "" || a.activeChannelID == "" {
		return false
	}
	for _, it := range a.sidebar.Items() {
		if it.ID == a.activeChannelID && it.DMUserID == userID {
			return true
		}
	}
	return false
}
