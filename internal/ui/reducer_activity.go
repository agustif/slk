// internal/ui/reducer_activity.go
//
// Activity-inbox reducer for App.Update.
//
//	ActivityViewActivatedMsg — user opened the Activity list:
//	  switch view + focus, kick a feed fetch.
//	ActivityFeedLoadedMsg    — activity.feed returned: push items,
//	  drop stale generations / other workspaces.
//	ActivityCountsMsg        — client.counts activity_v2 badge.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/cache"
	"github.com/agustif/slk/internal/ids"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/activityview"
)

// activityMessageCache is the cache-first parent-quote / reaction
// lookup used to decorate Activity cards. *cache.DB satisfies it.
type activityMessageCache interface {
	GetMessage(channelID, ts string) (cache.Message, error)
	GetReactions(messageTS, channelID string) ([]cache.ReactionRow, error)
}

var reduceActivity reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case ActivityViewActivatedMsg:
		_ = m
		a.setInboxView(ViewActivity)
		a.focusedPanel = PanelMessages
		a.activityView.SetUnreadBadge(a.sidebar.ActivityUnreadCount())
		var batch []tea.Cmd
		if a.activeTeamID != "" {
			activity := a.activity
			team := ids.TeamID(a.activeTeamID)
			batch = append(batch, func() tea.Msg { return activity.FetchViews(team) })
		}
		if cmd := a.fetchActivityCmd(); cmd != nil {
			batch = append(batch, cmd)
		}
		return tea.Batch(batch...), true

	case ActivityViewsLoadedMsg:
		if m.TeamID != a.activeTeamID {
			return nil, true
		}
		if m.Err != nil || len(m.Views) == 0 {
			return nil, true
		}
		a.activityView.SetViews(m.Views)
		return a.fetchActivityCmd(), true

	case ActivityFeedLoadedMsg:
		if m.TeamID != a.activeTeamID || m.Gen != a.activityGen {
			return nil, true
		}
		if m.Err != nil {
			a.activityView.SetLoading(false)
			a.activityView.SetError("activity.feed failed — " + m.Err.Error())
			return nil, true
		}
		a.activityView.SetItems(a.decorateActivityItems(m.Items))
		return nil, true

	case ActivityCountsMsg:
		if m.TeamID != "" && m.TeamID != a.activeTeamID {
			return nil, true
		}
		a.sidebar.SetActivityUnreadCount(m.Unread)
		a.activityView.SetUnreadBadge(m.Unread)
		return nil, true
	}
	return nil, false
}

func (a *App) fetchActivityCmd() tea.Cmd {
	if a.activeTeamID == "" {
		return nil
	}
	a.activityGen++
	gen := a.activityGen
	q := a.activityView.Query()
	limit := a.activityCfg.Limit
	if limit < 1 {
		limit = 50
	}
	a.activityView.SetLoading(true)
	a.activityView.SetError("")
	activity := a.activity
	team := ids.TeamID(a.activeTeamID)
	return func() tea.Msg {
		return activity.FetchFeed(team, ActivityFeedQuery{
			Filter:       q.Filter,
			Types:        q.Types,
			Sort:         q.Sort,
			UnreadOnly:   q.UnreadOnly,
			PriorityOnly: q.PriorityOnly,
			Limit:        limit,
			Gen:          gen,
		})
	}
}

func (a *App) decorateActivityItems(in []slackclient.ActivityItem) []activityview.Item {
	out := make([]activityview.Item, len(in))
	for i, it := range in {
		item := activityview.Item{ActivityItem: it}
		if name, ok := a.channelNames[it.ChannelID]; ok {
			item.ChannelName = name
		}
		if _, t, ok := a.channels.Lookup(ids.ChannelID(it.ChannelID)); ok {
			item.ChannelType = t
		}
		if it.ActorID != "" {
			if it.ActorID == a.currentUserID {
				item.ActorName = "me"
			} else if name, ok := a.userNames[it.ActorID]; ok && name != "" {
				item.ActorName = name
			}
		}
		if it.Text != "" {
			item.ParentText = it.Text
		}
		a.decorateActivityParent(&item)
		out[i] = item
	}
	return out
}

func (a *App) decorateActivityParent(item *activityview.Item) {
	if a.activityCache == nil || item.ChannelID == "" || item.MessageTS == "" {
		return
	}
	if msg, err := a.activityCache.GetMessage(item.ChannelID, item.MessageTS); err == nil && !msg.IsDeleted {
		item.ParentText = msg.Text
	}
	rs, err := a.activityCache.GetReactions(item.MessageTS, item.ChannelID)
	if err != nil {
		return
	}
	item.ReactionsKnown = true
	for _, r := range rs {
		for _, uid := range r.UserIDs {
			if uid != a.currentUserID {
				continue
			}
			item.OwnReactions = append(item.OwnReactions, r.Emoji)
			if r.Emoji == item.Reaction {
				item.HasReacted = true
			}
			break
		}
	}
}

func (a *App) openSelectedActivityCmd() tea.Cmd {
	it, ok := a.activityView.SelectedItem()
	if !ok || it.ChannelID == "" {
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
		messageTS: it.MessageTS,
		threadTS:  it.ThreadTS,
	}
	id := it.ChannelID
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}
