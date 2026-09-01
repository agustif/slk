// internal/ui/mode_channel_finder.go
//
// Channel-finder mode key handler (Phase 5h).
//
// Forwards normalised keys to the channel-finder overlay. On a
// result:
//   - Synthetic "threads" destination -> activate the threads
//     view (ThreadsViewActivatedMsg).
//   - Already-joined channel -> select it (ChannelSelectedMsg).
//   - Not yet joined -> fire a Join via the channels service;
//     ChannelJoinedMsg (reducer_channels) folds it into the
//     sidebar and dispatches the ChannelSelectedMsg.
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/searchresults"
)

func handleChannelFinderMode(a *App, msg tea.KeyMsg) tea.Cmd {
	// Map tea.KeyMsg to string for the finder.
	before := a.channelFinder.Query()
	result := a.channelFinder.HandleKey(normalizeFinderKey(msg))
	if result != nil {
		a.channelFinder.Close()
		a.SetMode(ModeNormal)
		// Synthetic destinations (e.g. Threads view) live alongside
		// channels in the finder but route to a view activation
		// rather than a channel switch.
		if result.Type == "threads" {
			return func() tea.Msg { return ThreadsViewActivatedMsg{} }
		}
		if result.Type == "activity" {
			return func() tea.Msg { return ActivityViewActivatedMsg{} }
		}
		if result.Type == "later" {
			return func() tea.Msg { return LaterViewActivatedMsg{} }
		}
		if result.Type == "dms" {
			return func() tea.Msg { return DMsViewActivatedMsg{} }
		}
		if result.Type == "drafts" {
			return func() tea.Msg { return DraftsViewActivatedMsg{} }
		}
		if result.Type == "unreads" {
			return func() tea.Msg { return UnreadsViewActivatedMsg{} }
		}
		if result.Type == "starred" {
			return func() tea.Msg { return StarredViewActivatedMsg{} }
		}
		if result.Type == "search" {
			q := result.ID
			a.searchResults.OpenQuery(q)
			a.SetMode(ModeWorkspaceSearch)
			if a.searchResults.StartSearch() {
				return workspaceSearchCmd(a, 1)
			}
			return nil
		}
		if result.Type == "user" {
			uid := result.UserID
			if uid == "" {
				uid = result.ID
			}
			if uid == "" {
				return toastWithClear(a, "No user to message", 2*time.Second)
			}
			a.newMessageInFlightID++
			a.newMessageCancelled = false
			reqID := a.newMessageInFlightID
			return a.channels.OpenConversation([]string{uid}, reqID)
		}
		if result.Type == "message" {
			if result.ID == "" || result.TS == "" {
				return nil
			}
			a.pendingLinkNav = &pendingLinkNav{
				channelID: result.ID,
				messageTS: result.TS,
			}
			id, name, typ := result.ID, result.Name, "channel"
			if n, t, ok := a.channels.Lookup(ids.ChannelID(result.ID)); ok {
				name, typ = n, t
			}
			return func() tea.Msg {
				return ChannelSelectedMsg{ID: id, Name: name, Type: typ}
			}
		}
		if result.Type == "file" {
			return selectSearchFile(searchresults.Item{
				Kind:      searchresults.KindFiles,
				FileID:    result.FileID,
				FileName:  result.Name,
				FileURL:   result.FileURL,
				Permalink: result.FileURL,
			})
		}
		// Already-joined: switch immediately. Not joined: kick off
		// a join command; ChannelJoinedMsg will fold the channel
		// into the sidebar and switch to it.
		if result.Joined {
			a.sidebar.SelectByID(result.ID)
			return func() tea.Msg {
				return ChannelSelectedMsg{ID: result.ID, Name: result.Name, Type: result.Type}
			}
		}
		channels := a.channels
		id, name := ids.ChannelID(result.ID), result.Name
		return func() tea.Msg {
			return channels.Join(id, name)
		}
	}

	// Check if finder closed itself (Esc).
	if !a.channelFinder.IsVisible() {
		a.SetMode(ModeNormal)
		return nil
	}

	// The local filter has already run inside HandleKey, so the list
	// on screen is up to date before anything touches the network.
	// Only the server query is deferred.
	if after := a.channelFinder.Query(); after != before {
		return a.scheduleChannelSearch(after)
	}
	return nil
}
