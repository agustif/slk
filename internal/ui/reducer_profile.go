package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ui/statusbar"
)

var reduceProfile reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	m, ok := msg.(UserProfileLoadedMsg)
	if !ok {
		return nil, false
	}
	if !a.userProfile.IsVisible() || a.userProfile.UserID() != m.UserID {
		return nil, true
	}
	if m.Err != nil {
		p := a.userProfile.Profile()
		p.Loading = false
		a.userProfile.SetProfile(p)
		a.statusbar.SetToast("Profile lookup failed: " + m.Err.Error())
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusbar.CopiedClearMsg{} }), true
	}
	p := m.Profile
	p.Loading = false
	p.IsSelf = p.UserID == a.currentUserID
	p.AlreadyInDM = a.isDMWith(p.UserID)
	if p.Presence == "" {
		p.Presence = a.sidebar.PresenceForUser(p.UserID)
	}
	a.userProfile.SetProfile(p)
	a.sidebar.UpdateUserStatus(p.UserID, p.Status)
	return nil, true
}
