package main

import (
	"github.com/agustif/slk/internal/cache"
	"github.com/agustif/slk/internal/config"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/channelfinder"
	"github.com/agustif/slk/internal/ui/sidebar"
	"github.com/slack-go/slack"
)

// mergeClientDMsIntoWorkspace appends client.dms conversations that
// boot / users.conversations missed. Existing rows are unchanged.
// New 1:1s are Closed so Home still hides them; the DMs tab lists them.
// Last-message text is not taken from this response — snippets still
// use cache + conversations.history limit=1.
func mergeClientDMsIntoWorkspace(wctx *WorkspaceContext, db *cache.DB, cfg config.Config, teamID string, dms slackclient.ClientDMs) []sidebar.ChannelItem {
	seen := map[string]bool{}
	if wctx != nil {
		for _, it := range wctx.Channels {
			seen[it.ID] = true
		}
	}
	var added []sidebar.ChannelItem
	add := func(id string, isIM, isMpIM bool) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ch := slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:     id,
					IsIM:   isIM,
					IsMpIM: isMpIM,
					IsOpen: !isIM,
				},
			},
			IsMember: true,
		}
		var item sidebar.ChannelItem
		var finder channelfinder.Item
		if wctx != nil {
			item, finder = buildChannelItem(ch, wctx, cfg, teamID)
		} else {
			typ := "dm"
			if isMpIM {
				typ = "group_dm"
			}
			item = sidebar.ChannelItem{ID: id, Type: typ, Closed: isIM}
		}
		if item.Name == "" {
			item.Name = id
		}
		if finder.ID == "" {
			finder = channelfinder.Item{ID: item.ID, Name: item.Name, Type: item.Type, Joined: true}
		} else if finder.Name == "" {
			finder.Name = item.Name
		}
		if wctx != nil {
			if db != nil {
				upsertChannelInDB(db, ch, item.Type, teamID)
			}
			wctx.Channels = append(wctx.Channels, item)
			if wctx.LastVisitedByChannel != nil {
				finder.LastVisited = wctx.LastVisitedByChannel[id]
			}
			wctx.FinderItems = append(wctx.FinderItems, finder)
		}
		added = append(added, item)
	}
	for _, im := range dms.IMs {
		add(im.ID, true, false)
	}
	for _, g := range dms.MPIMs {
		add(g.ID, false, true)
	}
	return added
}
