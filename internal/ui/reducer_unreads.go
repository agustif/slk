package ui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/unreadsview"
)

var reduceUnreads reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case UnreadsViewActivatedMsg:
		_ = m
		a.setInboxView(ViewUnreads)
		a.focusedPanel = PanelMessages
		if cmd := a.fetchUnreadsCmd(); cmd != nil {
			return cmd, true
		}
		return nil, true

	case UnreadsListLoadedMsg:
		if m.TeamID != a.activeTeamID || m.Gen != a.unreadsGen {
			return nil, true
		}
		if m.Err != nil {
			a.unreadsView.SetLoading(false)
			a.unreadsView.SetError("unreads failed — " + m.Err.Error())
			return nil, true
		}
		a.unreadsView.SetSidebarOrder(a.sidebar.ChannelIDsInOrder())
		a.unreadsView.SetBlocks(a.decorateUnreadBlocks(m.Blocks))
		return nil, true

	case UnreadsMarkedMsg:
		if m.Err != nil {
			action := "Mark as read"
			if m.Undo {
				action = "Undo"
			}
			return toastWithClear(a, action+" failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		if m.Undo {
			a.unreadsView.UndoBlock(m.ChannelID)
			return toastWithClear(a, "Mark as read undone", 2*time.Second), true
		}
		a.unreadsView.MarkBlockRead(m.ChannelID)
		return toastWithClear(a, "Marked as read", 2*time.Second), true
	}
	return nil, false
}

func (a *App) decorateUnreadBlocks(in []unreadsview.Block) []unreadsview.Block {
	type meta struct {
		Type    string
		Starred bool
		VIP     bool
	}
	byID := map[string]meta{}
	for _, it := range a.sidebar.AllItems() {
		byID[it.ID] = meta{Type: it.Type, Starred: it.IsStarred, VIP: it.IsVIP}
	}
	out := make([]unreadsview.Block, len(in))
	copy(out, in)
	for i := range out {
		b := &out[i]
		if name, ok := a.channelNames[b.ChannelID]; ok {
			b.ChannelName = name
		}
		if _, t, ok := a.channels.Lookup(ids.ChannelID(b.ChannelID)); ok {
			b.ChannelType = t
		}
		if info, ok := byID[b.ChannelID]; ok {
			b.IsStarred = info.Starred
			b.IsVIP = info.VIP
			if b.ChannelType == "" {
				b.ChannelType = info.Type
			}
		}
		for j := range b.Messages {
			msg := &b.Messages[j]
			if msg.UserID == "" {
				continue
			}
			if msg.UserID == a.currentUserID {
				msg.UserName = "me"
			} else if name, ok := a.userNames[msg.UserID]; ok && name != "" {
				msg.UserName = name
			}
		}
	}
	return out
}

func (a *App) persistUnreadsSortCmd() tea.Cmd {
	order := a.unreadsView.Sort()
	svc := a.unreads
	return func() tea.Msg {
		if err := svc.SetSortOrder(order); err != nil {
			if errors.Is(err, errServiceNoop) {
				return nil
			}
			return ToastMsg{Text: "Could not save Unreads sort: " + truncateReason(err.Error(), 40)}
		}
		return nil
	}
}

func (a *App) persistUnreadsFilterCmd() tea.Cmd {
	filter := a.unreadsView.Filter()
	svc := a.unreads
	return func() tea.Msg {
		if err := svc.SetFilter(filter); err != nil {
			if errors.Is(err, errServiceNoop) {
				return nil
			}
			return ToastMsg{Text: "Could not save Unreads filter: " + truncateReason(err.Error(), 40)}
		}
		return nil
	}
}

func (a *App) applyUnreadsPrefs(sortOrder, filter string) {
	a.unreadsView.SetSort(sortOrder)
	a.unreadsView.SetFilter(filter)
}

func (a *App) fetchUnreadsCmd() tea.Cmd {
	if a.activeTeamID == "" {
		return nil
	}
	a.unreadsGen++
	gen := a.unreadsGen
	a.unreadsView.SetLoading(true)
	a.unreadsView.SetError("")
	unreads := a.unreads
	team := ids.TeamID(a.activeTeamID)
	return func() tea.Msg { return unreads.List(team, gen) }
}

func (a *App) openSelectedUnreadCmd() tea.Cmd {
	b, msg, ok := a.unreadsView.SelectedMessage()
	if !ok || b.ChannelID == "" || msg.TS == "" {
		return nil
	}
	name, chType, found := a.channels.Lookup(ids.ChannelID(b.ChannelID))
	if !found {
		name, chType = b.ChannelName, b.ChannelType
		if name == "" {
			name = b.ChannelID
		}
	}
	a.pendingLinkNav = &pendingLinkNav{
		channelID: b.ChannelID,
		messageTS: msg.TS,
	}
	id := b.ChannelID
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}

func (a *App) markSelectedUnreadCmd() tea.Cmd {
	b, ok := a.unreadsView.SelectedBlock()
	if !ok || b.ChannelID == "" {
		return nil
	}
	unreads := a.unreads
	ch := ids.ChannelID(b.ChannelID)
	if b.MarkedRead {
		ts := ids.MessageTS(b.LastRead)
		return func() tea.Msg {
			err := unreads.MarkUnread(ch, ts)
			return UnreadsMarkedMsg{ChannelID: b.ChannelID, TS: string(ts), LastRead: b.LastRead, Undo: true, Err: err}
		}
	}
	if b.LatestTS == "" {
		return toastWithClear(a, "No messages to mark read", 2*time.Second)
	}
	ts := ids.MessageTS(b.LatestTS)
	last := b.LastRead
	return func() tea.Msg {
		err := unreads.MarkRead(ch, ts)
		return UnreadsMarkedMsg{ChannelID: b.ChannelID, TS: string(ts), LastRead: last, Err: err}
	}
}

func (a *App) handleUnreadsEnter() tea.Cmd {
	if a.unreadsView.SelectedIsHeader() {
		return a.markSelectedUnreadCmd()
	}
	return a.openSelectedUnreadCmd()
}
