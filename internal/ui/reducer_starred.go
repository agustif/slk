package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/starredview"
)

var reduceStarred reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case StarredViewActivatedMsg:
		_ = m
		a.setInboxView(ViewStarred)
		a.focusedPanel = PanelMessages
		if cmd := a.fetchStarredMessagesCmd(); cmd != nil {
			a.starredView.SetLoading(true)
			a.starredView.SetError("")
			return cmd, true
		}
		return nil, true
	}
	return nil, false
}

func (a *App) applyStarredInbox(items []slackclient.StarredMessage, fileIDs []string, files []slackclient.FileInfo) {
	rows := a.decorateStarredItems(items)
	byID := make(map[string]slackclient.FileInfo, len(files))
	for _, f := range files {
		if f.ID != "" {
			byID[f.ID] = f
		}
	}
	if len(fileIDs) == 0 {
		for _, f := range files {
			if f.ID != "" {
				fileIDs = append(fileIDs, f.ID)
			}
		}
	}
	for _, id := range fileIDs {
		if id == "" {
			continue
		}
		it := starredview.Item{FileID: id}
		if f, ok := byID[id]; ok {
			it.FileTitle = f.DisplayName()
			it.Filetype = f.Filetype
			it.FileMode = f.Mode
		}
		rows = append(rows, it)
	}
	a.starredView.SetItems(rows)
	a.sidebar.SetStarredCount(len(items))
}

func (a *App) decorateStarredItems(in []slackclient.StarredMessage) []starredview.Item {
	out := make([]starredview.Item, 0, len(in))
	for _, it := range in {
		item := starredview.Item{StarredMessage: it}
		if name, ok := a.channelNames[it.ChannelID]; ok {
			item.ChannelName = name
		}
		if _, t, ok := a.channels.Lookup(ids.ChannelID(it.ChannelID)); ok {
			item.ChannelType = t
		}
		if it.UserID != "" {
			if it.UserID == a.currentUserID {
				item.AuthorName = "me"
			} else if name, ok := a.userNames[it.UserID]; ok && name != "" {
				item.AuthorName = name
			}
		}
		out = append(out, item)
	}
	return out
}

func (a *App) openSelectedStarredCmd() tea.Cmd {
	it, ok := a.starredView.SelectedItem()
	if !ok || it.ChannelID == "" || it.TS == "" {
		return nil
	}
	name, chType, found := a.channels.Lookup(ids.ChannelID(it.ChannelID))
	if !found {
		name = it.ChannelName
		chType = it.ChannelType
		if name == "" {
			name = it.ChannelID
		}
	}
	a.pendingLinkNav = &pendingLinkNav{
		channelID: it.ChannelID,
		messageTS: it.TS,
	}
	id := it.ChannelID
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}

func (a *App) handleStarredEnter() tea.Cmd {
	return a.openSelectedStarredCmd()
}
