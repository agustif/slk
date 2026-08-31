package main

import (
	"strconv"
	"strings"

	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/config"
	"github.com/gammons/slk/internal/slackfmt"
	"github.com/gammons/slk/internal/ui/channelfinder"
	"github.com/gammons/slk/internal/ui/sidebar"
	"github.com/slack-go/slack"
)

// isGroupDM reports a multi-person DM. Slack sometimes omits is_mpim
// on users.conversations rows while still using the mpdm-* name.
func isGroupDM(ch slack.Channel) bool {
	return ch.IsMpIM || strings.HasPrefix(ch.Name, "mpdm-")
}

// buildChannelItem converts a Slack conversation into the sidebar
// ChannelItem + finder Item shape used everywhere in slk. Pure function:
// reads from wctx for name/presence resolution, returns the constructed
// sidebar item plus a parallel finder entry. The caller decides whether
// to append, upsert, or persist.
//
// Extracted from the workspace-bootstrap loop in initWorkspace so that
// mid-session conversation events (mpim_open / im_created / group_joined /
// channel_joined) can produce identical items.
//
// The returned finder Item uses the same display name as the sidebar item
// (e.g. "alice" for a DM, the formatted participant list for a group DM)
// because that's what the bootstrap loop has always passed to the finder
// and what the finder's filter/render code expects.
func buildChannelItem(ch slack.Channel, wctx *WorkspaceContext, cfg config.Config, teamID string) (sidebar.ChannelItem, channelfinder.Item) {
	chType := "channel"
	if ch.IsIM {
		// Slack returns the same is_im=true for human DMs and app DMs;
		// the only differentiator is the peer user's IsBot/IsAppUser
		// flag, which we look up via the cache-seeded BotUserIDs set.
		// Unknown peers default to "dm" and are reclassified later by
		// the resolveUser path.
		if wctx.BotUserIDs[ch.User] {
			chType = "app"
		} else {
			chType = "dm"
		}
	} else if isGroupDM(ch) {
		chType = "group_dm"
	} else if ch.IsPrivate {
		chType = "private"
	}

	displayName := ch.Name
	if ch.IsIM {
		if resolved, ok := wctx.UserNames[ch.User]; ok {
			displayName = resolved
		} else {
			displayName = ch.User
		}
	} else if isGroupDM(ch) {
		displayName = slackfmt.FormatMPDMName(ch.Name, func(h string) string {
			return wctx.UserNamesByHandle[h]
		})
	}

	section := ""
	if wctx.SectionStore != nil && wctx.SectionStore.Ready() {
		if id, ok := wctx.SectionStore.SectionForChannel(ch.ID); ok {
			section = id
		}
	}
	var sectionOrder int
	var channelOrder int
	if section == "" {
		var matchedOrder int
		section, matchedOrder = cfg.MatchSectionAndOrder(teamID, ch.Name)
		if section != "" {
			sectionOrder = cfg.SectionOrder(teamID, section)
			channelOrder = matchedOrder
		}
	}

	muted := false
	if wctx.MuteStore != nil {
		muted = wctx.MuteStore.IsMuted(ch.ID)
	}

	item := sidebar.ChannelItem{
		ID:           ch.ID,
		Name:         displayName,
		Type:         chType,
		Topic:        ch.Topic.Value,
		Section:      section,
		SectionOrder: sectionOrder,
		ChannelOrder: channelOrder,
		IsMuted:      muted,
		LastVisited:  wctx.LastVisitedByChannel[ch.ID],
		LastActivity: lastActivityUnix(ch),
	}
	if wctx.SectionStore != nil {
		item.IsStarred = wctx.SectionStore.IsStarred(ch.ID)
	}
	if ch.IsIM {
		item.DMUserID = ch.User
		item.Closed = !ch.IsOpen
	} else if isGroupDM(ch) {
		item.DMUserID = groupDMPeerID(ch, wctx)
	}
	if wctx.VIPStore != nil {
		// Prefer the sidebar peer id (DMUserID) so MPIMs match
		// refreshVIPForActive; ch.User is empty on group DMs.
		item.IsVIP = wctx.VIPStore.IsVIP(item.DMUserID) || wctx.VIPStore.IsVIP(ch.ID)
	}

	finderItem := channelfinder.Item{
		ID:       ch.ID,
		Name:     displayName,
		Type:     chType,
		Presence: item.Presence,
		Joined:   true,
	}
	return item, finderItem
}

// lastActivityUnix is Slack's last-message stamp for recency sort
// (the official DMs tab). users.conversations puts it on Latest;
// userBoot IMs use a Latest stub from `updated`.
func groupDMPeerID(ch slack.Channel, wctx *WorkspaceContext) string {
	self := ""
	if wctx != nil {
		self = wctx.UserID
	}
	for _, uid := range ch.Members {
		if uid != "" && uid != self {
			return uid
		}
	}
	if wctx == nil || wctx.UserIDsByHandle == nil {
		return ""
	}
	for _, h := range slackfmt.ParseMPDMHandles(ch.Name) {
		if id := wctx.UserIDsByHandle[h]; id != "" && id != self {
			return id
		}
	}
	return ""
}

func fillGroupDMAvatars(items []sidebar.ChannelItem, db *cache.DB, teamID, selfID string) {
	if db == nil {
		return
	}
	for i := range items {
		if items[i].Type != "group_dm" || items[i].DMUserID != "" {
			continue
		}
		members, err := db.ListChannelMembers(teamID, items[i].ID)
		if err != nil {
			continue
		}
		for _, uid := range members {
			if uid != "" && uid != selfID {
				items[i].DMUserID = uid
				break
			}
		}
	}
}

func lastActivityUnix(ch slack.Channel) int64 {
	if ch.Latest == nil {
		return 0
	}
	ts := ch.Latest.Timestamp
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

// upsertChannelInDB writes the channel to the SQLite cache. Separated from
// buildChannelItem so the latter stays a pure function.
func upsertChannelInDB(db *cache.DB, ch slack.Channel, chType string, teamID string) {
	db.UpsertChannel(cache.Channel{
		ID:          ch.ID,
		WorkspaceID: teamID,
		Name:        ch.Name,
		Type:        chType,
		Topic:       ch.Topic.Value,
		IsMember:    ch.IsMember,
	})
}
