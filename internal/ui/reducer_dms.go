// internal/ui/reducer_dms.go
//
// Direct Messages inbox reducer for App.Update.
//
//	DMsViewActivatedMsg — user opened Slack's DMs tab: switch the
//	sidebar to the full conversation list (no IsStale) and keep the
//	message pane on a conversation.
package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/agustif/slk/internal/ui/sidebar"
)

var reduceDMs reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case DMsViewActivatedMsg:
		a.setInboxView(ViewDMs)
		a.focusedPanel = PanelSidebar
		// Keep the cursor on the open conversation only when it is a
		// DM. SelectByID on a public channel is a no-op in this nav
		// (or would land on Home), and SelectDMsRow would highlight
		// some other 1:1 while the message pane still shows #general.
		if a.activeChannelID != "" && a.sidebarChannelIsDirect(a.activeChannelID) {
			a.sidebar.SelectByID(a.activeChannelID)
		}
		return a.fetchDMsSnippetsCmd(), true

	case DMsSnippetsMsg:
		if m.TeamID != "" && m.TeamID != a.activeTeamID {
			return nil, true
		}
		if len(m.Snips) == 0 {
			return nil, true
		}
		typeByID := map[string]string{}
		for _, it := range a.sidebar.Items() {
			typeByID[it.ID] = it.Type
		}
		formatted := make(map[string]sidebar.DMSnippet, len(m.Snips))
		for id, s := range m.Snips {
			s.Text = a.formatDMSnippet(s.Text, s.UserID, typeByID[id])
			formatted[id] = s
		}
		a.sidebar.ApplyDMSnippets(formatted)
		return nil, true
	}
	return nil, false
}

func (a *App) sidebarChannelIsDirect(id string) bool {
	if id == "" {
		return false
	}
	for _, it := range a.sidebar.Items() {
		if it.ID == id {
			return channelTypeIsDirect(it.Type)
		}
	}
	return false
}

func (a *App) fetchDMsSnippetsCmd() tea.Cmd {
	if a.dmsSnippets == nil || a.activeTeamID == "" {
		return nil
	}
	ids := a.sidebar.DMChannelIDs()
	if len(ids) == 0 {
		return nil
	}
	fetch := a.dmsSnippets
	return func() tea.Msg { return fetch(ids) }
}

func (a *App) applyLiveDMSnippet(channelID, ts, text, userID string) {
	if channelID == "" {
		return
	}
	typ := ""
	for _, it := range a.sidebar.Items() {
		if it.ID == channelID {
			typ = it.Type
			break
		}
	}
	if typ != "dm" && typ != "group_dm" && typ != "app" {
		return
	}
	a.sidebar.ApplyDMSnippets(map[string]sidebar.DMSnippet{
		channelID: {
			Text:     a.formatDMSnippet(text, userID, typ),
			UserID:   userID,
			Activity: slackTSUnix(ts),
		},
	})
}

func (a *App) formatDMSnippet(text, userID, chType string) string {
	body := strings.Join(strings.Fields(strings.ReplaceAll(text, "\n", " ")), " ")
	if body == "" || chType != "group_dm" {
		return body
	}
	name := a.userNameFor(userID)
	if userID != "" && userID == a.currentUserID {
		name = "You"
	}
	if name == "" || name == userID {
		return body
	}
	return name + ": " + body
}

func slackTSUnix(ts string) int64 {
	if ts == "" {
		return 0
	}
	if i := strings.IndexByte(ts, '.'); i >= 0 {
		ts = ts[:i]
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
