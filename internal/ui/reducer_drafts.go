package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/draftsview"
)

type pendingDraftOpen struct {
	channelID string
	threadTS  string
}

var reduceDrafts reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case DraftsViewActivatedMsg:
		_ = m
		a.setInboxView(ViewDrafts)
		a.focusedPanel = PanelMessages
		a.draftsView.SetBadge(a.sidebar.DraftsUnreadCount())
		if cmd := a.fetchDraftsCmd(); cmd != nil {
			return tea.Batch(cmd, a.fetchDraftsCountCmd()), true
		}
		return a.fetchDraftsCountCmd(), true

	case DraftsListLoadedMsg:
		if m.TeamID != a.activeTeamID || m.Gen != a.draftsGen {
			return nil, true
		}
		if m.Err != nil {
			a.draftsView.SetLoading(false)
			a.draftsView.SetError("drafts.list failed — " + m.Err.Error())
			return nil, true
		}
		if m.Filter == draftsview.FilterScheduled {
			items := make([]draftsview.Item, 0, len(m.Sched))
			for _, s := range m.Sched {
				it := draftsview.ItemFromScheduled(s)
				a.decorateDraftItem(&it)
				items = append(items, it)
			}
			a.draftsView.SetItems(items)
			a.draftsView.SetScheduledCount(len(items))
			return nil, true
		}
		items := make([]draftsview.Item, 0, len(m.Drafts))
		for _, d := range m.Drafts {
			it := draftsview.ItemFromDraft(d)
			a.decorateDraftItem(&it)
			items = append(items, it)
		}
		a.draftsView.SetPage(items, m.NextTS, m.Append)
		return nil, true

	case DraftsCountsMsg:
		if m.TeamID != "" && m.TeamID != a.activeTeamID {
			return nil, true
		}
		a.sidebar.SetDraftsUnreadCount(m.Count)
		a.draftsView.SetBadge(m.Count)
		return nil, true

	case DraftDeletedMsg:
		if m.Err != nil {
			return toastWithClear(a, "Delete failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		return tea.Batch(toastWithClear(a, "Deleted", 2*time.Second), a.fetchDraftsCmd(), a.fetchDraftsCountCmd()), true
	}
	return nil, false
}

func (a *App) decorateDraftItem(it *draftsview.Item) {
	if name, ok := a.channelNames[it.ChannelID]; ok {
		it.ChannelName = name
	}
	if _, t, ok := a.channels.Lookup(ids.ChannelID(it.ChannelID)); ok {
		it.ChannelType = t
	}
}

func (a *App) fetchDraftsCmd() tea.Cmd {
	if a.activeTeamID == "" {
		return nil
	}
	a.draftsGen++
	gen := a.draftsGen
	a.draftsView.SetLoading(true)
	a.draftsView.SetError("")
	drafts := a.drafts
	team := ids.TeamID(a.activeTeamID)
	if a.draftsView.Filter() == draftsview.FilterScheduled {
		return func() tea.Msg { return drafts.ListScheduled(team, gen) }
	}
	return func() tea.Msg { return drafts.List(team, gen, "") }
}

func (a *App) fetchDraftsMoreCmd() tea.Cmd {
	if a.activeTeamID == "" || !a.draftsView.HasMore() {
		return nil
	}
	a.draftsGen++
	gen := a.draftsGen
	drafts := a.drafts
	team := ids.TeamID(a.activeTeamID)
	cursor := a.draftsView.NextTS()
	return func() tea.Msg { return drafts.List(team, gen, cursor) }
}

func (a *App) fetchDraftsCountCmd() tea.Cmd {
	if a.activeTeamID == "" {
		return nil
	}
	drafts := a.drafts
	team := ids.TeamID(a.activeTeamID)
	return func() tea.Msg { return drafts.Count(team) }
}

func (a *App) openSelectedDraftCmd() tea.Cmd {
	it, ok := a.draftsView.SelectedItem()
	if !ok || it.ChannelID == "" {
		return nil
	}
	if it.Kind == draftsview.KindDraft {
		key := slackclient.DraftKeyFor(it.ChannelID, it.ThreadTS)
		if it.ThreadTS != "" {
			a.threadCompose.ReplaceTextDraft(key, it.Text)
		} else {
			a.compose.ReplaceTextDraft(key, it.Text)
		}
		a.pendingDraftOpen = &pendingDraftOpen{channelID: it.ChannelID, threadTS: it.ThreadTS}
	}
	name, chType, found := a.channels.Lookup(ids.ChannelID(it.ChannelID))
	if !found {
		name, chType = it.ChannelName, it.ChannelType
	}
	id := it.ChannelID
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}

func (a *App) completePendingDraftOpen(selectedID string) tea.Cmd {
	p := a.pendingDraftOpen
	if p == nil {
		return nil
	}
	if selectedID != p.channelID {
		a.pendingDraftOpen = nil
		return nil
	}
	if p.channelID != a.activeChannelID {
		return nil
	}
	a.pendingDraftOpen = nil
	if p.threadTS != "" {
		cmd := a.openThreadForPermalink(p.channelID, p.threadTS)
		a.focusedPanel = PanelThread
		a.SetMode(ModeInsert)
		return tea.Batch(cmd, a.threadCompose.Focus())
	}
	a.focusedPanel = PanelMessages
	a.SetMode(ModeInsert)
	return a.compose.Focus()
}

func (a *App) deleteSelectedDraftCmd() tea.Cmd {
	it, ok := a.draftsView.SelectedItem()
	if !ok {
		return toastWithClear(a, "No draft selected", 2*time.Second)
	}
	drafts := a.drafts
	if it.Kind == draftsview.KindScheduled {
		ch, id := ids.ChannelID(it.ChannelID), it.ID
		kind := it.Kind
		return func() tea.Msg {
			err := drafts.DeleteScheduled(ch, id)
			return DraftDeletedMsg{Kind: kind, ID: id, Err: err}
		}
	}
	id, ts := it.ID, it.LastUpdatedTS
	key := slackclient.DraftKeyFor(it.ChannelID, it.ThreadTS)
	a.clearComposeDraft(key)
	return func() tea.Msg {
		err := drafts.Delete(id, ts)
		return DraftDeletedMsg{Kind: draftsview.KindDraft, ID: id, Err: err}
	}
}
