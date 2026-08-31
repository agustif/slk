// internal/ui/mode_workspace_search.go
//
// Workspace-search mode key handler: the ctrl+f modal.
//
// Forwards normalized keys to the searchresults overlay and
// translates its actions: Submit / Load more dispatch search.messages,
// search.files, or edge.UsersSearch via the SearchService (Kind/Page/Gen
// stamped so a stale page is dropped). Select on a message hit navigates
// via pendingLinkNav; Select on a file hit downloads through filedl when
// a URL is present, otherwise opens the permalink; Select on a person
// opens a DM via ChannelService.OpenConversation. People-tab query
// edits are debounced (~300 ms) rather than searched per keystroke.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/searchresults"
)

func handleWorkspaceSearchMode(a *App, msg tea.KeyMsg) tea.Cmd {
	keyStr := normalizeFinderKey(msg)
	before := a.searchResults.Query()
	switch action := a.searchResults.HandleKey(keyStr); action {
	case searchresults.ActionClose:
		a.SetMode(ModeNormal)
		return nil
	case searchresults.ActionSubmit:
		return workspaceSearchCmd(a, 1)
	case searchresults.ActionLoadMore:
		return workspaceSearchCmd(a, a.searchResults.Page()+1)
	case searchresults.ActionSelect:
		item, ok := a.searchResults.Selected()
		kind := a.searchResults.Kind()
		a.searchResults.Close()
		a.SetMode(ModeNormal)
		if !ok {
			return nil
		}
		if kind == searchresults.KindPeople {
			return selectSearchPerson(a, item)
		}
		if kind == searchresults.KindFiles {
			return selectSearchFile(item)
		}
		return selectSearchMessage(a, item)
	}
	if a.searchResults.Kind() == searchresults.KindPeople {
		if after := a.searchResults.Query(); after != before {
			return a.schedulePeopleSearch(after)
		}
	}
	return nil
}

func workspaceSearchCmd(a *App, page int) tea.Cmd {
	// If the service never answers (e.g. the noop service returns a
	// nil msg), the modal isn't stuck: backspace drops the widget
	// back to the input state and Esc closes it.
	req := WorkspaceSearchRequest{
		Query:          a.searchResults.Query(),
		Kind:           a.searchResults.Kind(),
		Page:           page,
		Gen:            a.searchResults.Gen(),
		CurrentChannel: a.activeChannelID,
	}
	search := a.searchSvc
	return func() tea.Msg { return search.SearchWorkspace(req) }
}

func selectSearchPerson(a *App, item searchresults.Item) tea.Cmd {
	if item.UserID == "" {
		return func() tea.Msg { return ToastMsg{Text: "No user to message"} }
	}
	a.newMessageInFlightID++
	a.newMessageCancelled = false
	reqID := a.newMessageInFlightID
	return a.channels.OpenConversation([]string{item.UserID}, reqID)
}

func selectSearchFile(item searchresults.Item) tea.Cmd {
	name := item.FileName
	if name == "" {
		name = item.Text
	}
	if item.FileURL != "" {
		att := messages.Attachment{
			Kind:        "file",
			Name:        name,
			URL:         item.Permalink,
			DownloadURL: item.FileURL,
			FileID:      item.FileID,
			Size:        item.FileSize,
		}
		return func() tea.Msg { return DownloadFileMsg{Attachment: att} }
	}
	if item.Permalink != "" {
		url := item.Permalink
		return func() tea.Msg { return OpenLinkMsg{URL: url} }
	}
	return func() tea.Msg { return ToastMsg{Text: "No download URL for " + name} }
}

func selectSearchMessage(a *App, item searchresults.Item) tea.Cmd {
	if item.ChannelID == a.activeChannelID {
		a.pendingLinkNav = &pendingLinkNav{
			channelID: item.ChannelID,
			messageTS: item.TS,
			threadTS:  item.ThreadTS,
		}
		return a.completePendingLinkNav(a.activeChannelID, true)
	}
	// Slack search also returns hits in public channels the user
	// hasn't joined. A Lookup miss is the not-a-member signal at
	// this layer: navigating there would fail with not_in_channel
	// and strand the user in an empty pane, so don't navigate —
	// tell them how to join instead.
	name, chType, ok := a.channels.Lookup(ids.ChannelID(item.ChannelID))
	if !ok {
		chName := item.ChannelName
		return func() tea.Msg {
			return ToastMsg{Text: "Not a member of #" + chName + " — join via ctrl+t to view"}
		}
	}
	a.pendingLinkNav = &pendingLinkNav{
		channelID: item.ChannelID,
		messageTS: item.TS,
		threadTS:  item.ThreadTS,
	}
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: item.ChannelID, Name: name, Type: chType}
	}
}
