// internal/ui/reducer_channels.go
//
// Channel-lifecycle reducer for App.Update (Phase 4j).
//
// Owns the Update arms that drive the channel-selection
// lifecycle and channel-list mutations:
//
//	ChannelSelectedMsg            - user picked a channel: reset
//	                                view state, mark visit,
//	                                dispatch by cache freshness
//	                                tier (fresh / verify-in-bg /
//	                                spinner).
//	MessagesLoadedMsg             - initial messages fetch landed:
//	                                replace pane contents (nil =
//	                                network failure, keep cache).
//	OlderMessagesLoadedMsg        - history backfill landed:
//	                                prepend (anchor-validated: dropped
//	                                if the buffer was replaced
//	                                mid-flight).
//	ChannelMarkedRemoteMsg        - WS echo of a remote mark:
//	                                apply locally.
//	ChannelMarkedReadMsg          - optimistic mark-read echo:
//	                                refresh sidebar read state.
//	ChannelMembershipMsg          - membership fetch landed:
//	                                push to the cache used by
//	                                mention picker / DM resolution.
//	ChannelJoinedMsg              - finder-driven join succeeded:
//	                                add to sidebar + open it.
//	ChannelJoinFailedMsg          - finder-driven join failed:
//	                                log warning (toast TBD).
//	ChannelCreatedMsg             - :create succeeded: add + open + toast.
//	ChannelCreateFailedMsg        - :create API failed: toast.
//	ChannelInvitedMsg             - :invite succeeded: toast.
//	ChannelInviteFailedMsg        - :invite API failed: toast.
//	ChannelNotifySetMsg           - :notify result: toast (or error).
//	LeaveChannelMsg               - user confirmed :leave: dispatch
//	                                ChannelService.Leave.
//	ChannelLeftMsg                - leave succeeded: drop from sidebar,
//	                                switch to last-visited or Threads,
//	                                toast.
//	ChannelLeaveFailedMsg         - leave API failed: toast.
//	ChannelMutedMsg               - mute write result: roll back
//	                                optimistic sidebar mute on error.
//	ChannelChromeMsg              - bookmarks/pins fetch landed:
//	                                update the message-pane header row.
//	channelSearchDebounceMsg      - finder typing paused: issue one
//	                                channels/search for the query
//	                                the user stopped on.
//	RemoteChannelsFoundMsg        - that search answered: merge the
//	                                non-joined matches into the
//	                                finder, unless superseded.
//
// Free reducer (not controller-absorbed): these arms cooperate on
// the sidebar, messagepane, statusbar, channelFinder, navHistory,
// editController, threadPanel close, compose state, and the
// channels service. No single existing controller owns all of
// that cross-section.
//
// Helpers (cancelEdit, CloseThread, clearSelections, SetChannels,
// SetChannelMembership, notifyReadStateChanged, applyChannelMark,
// uploadToastCmd, userNameFor, nowFormatted) stay on App; the
// reducer calls them via `a`.
//
// Inbound image/avatar arms (imgrender.ImageReadyMsg,
// imgrender.ImageFailedMsg, messages.AvatarReadyMsg) are NOT here:
// they're cross-cutting asset-loading echoes that touch messagepane
// + threadPanel caches but have nothing to do with channel
// lifecycle. They go in reducer_io.go (Phase 4l).
package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/debuglog"
	"github.com/agustif/slk/internal/ids"
	"github.com/agustif/slk/internal/ui/sidebar"
)

var reduceChannels reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case ChannelSelectedMsg:
		cmd, fetchFired := reduceChannelSelected(a, m)
		// Permalink completion. !fetchFired means no MessagesLoadedMsg
		// is coming (tier-1 fresh cache, or the upload-guard early
		// return), so complete authoritatively now — otherwise the
		// pending nav would leak and could later hijack the selection.
		// When a fetch WAS fired, defer to the authoritative
		// MessagesLoadedMsg completion and only do a best-effort select
		// here.
		if nav := a.completePendingLinkNav(m.ID, !fetchFired); nav != nil {
			cmd = tea.Batch(cmd, nav)
		}
		if extra := a.completePendingDraftOpen(m.ID); extra != nil {
			cmd = tea.Batch(cmd, extra)
		}
		return cmd, true

	case MessagesLoadedMsg:
		// Distinguish the three cases of the fetcher's nil-vs-[]
		// contract:
		//   nil       -> network failure, keep cached render
		//   []        -> channel is genuinely empty, replace with empty
		//   non-empty -> authoritative replace
		var kind string
		switch {
		case m.Messages == nil:
			kind = "nil_keep_cache"
		case len(m.Messages) == 0:
			kind = "empty_replace"
		default:
			kind = "full_replace"
		}
		debuglog.Cache("MessagesLoadedMsg: channel=%s active=%s kind=%s count=%d",
			m.ChannelID, a.activeChannelID, kind, len(m.Messages))
		// Fan out to every window viewing the channel, focused or
		// not (Phase 3).
		models := a.modelsForChannel(m.ChannelID)
		if len(models) == 0 {
			return nil, true // no window views this channel — stale fetch
		}
		if m.ChannelID == a.activeChannelID {
			a.statusbar.SetSyncing(false)
		}
		for _, mm := range models {
			mm.SetLoading(false)
			mm.SetLastReadTS(m.LastReadTS)
			// nil Messages from the fetcher signals network FAILURE,
			// not an empty channel (empty channels return
			// []messages.MessageItem{}). On failure, preserve whatever
			// the cache already rendered so a transient blip doesn't
			// blank a working view. The Slack-side fetcher logs the
			// error before returning nil. cloneMessageItems gives each
			// window its own copy — two same-channel windows must not
			// alias one slice (in-place model writes would cross-leak).
			if m.Messages != nil {
				mm.SetMessages(a.applyStarredFlags(m.ChannelID, cloneMessageItems(m.Messages)))
			}
		}
		// Authoritative permalink completion: this is the freshest
		// data we'll get for this channel. Focused-window semantics
		// (completePendingLinkNav selects in a.messagepane), so keep
		// the legacy active-channel gate.
		if m.ChannelID == a.activeChannelID {
			return a.completePendingLinkNav(m.ChannelID, true), true
		}
		return nil, true

	case OlderMessagesLoadedMsg:
		debuglog.Cache("OlderMessagesLoadedMsg: channel=%s active=%s anchor=%s count=%d",
			m.ChannelID, a.activeChannelID, m.AnchorTS, len(m.Messages))
		// Always clear the per-channel in-flight flag — even when no
		// window views the channel anymore — or scroll-backfill stays
		// permanently disabled after navigating away mid-fetch.
		delete(a.fetchingOlder, m.ChannelID)
		for _, mm := range a.modelsForChannel(m.ChannelID) {
			mm.SetLoading(false)
			if m.AnchorTS != mm.OldestTS() {
				// This window's buffer was replaced mid-flight (e.g. by
				// a jump-to-message FetchAround): the block is anchored
				// to the OLD buffer's oldest message and prepending it
				// would produce an out-of-order/duplicated buffer. Skip
				// just this window; siblings with the original buffer
				// still want the block.
				debuglog.Cache("OlderMessagesLoadedMsg: channel=%s anchor=%s != oldest=%s (buffer replaced, dropping for window)",
					m.ChannelID, m.AnchorTS, mm.OldestTS())
				continue
			}
			mm.PrependMessages(cloneMessageItems(m.Messages))
		}
		return nil, true

	case MessagesAroundLoadedMsg:
		debuglog.Cache("MessagesAroundLoadedMsg: channel=%s active=%s count=%d err=%v jump_date=%v",
			m.ChannelID, a.activeChannelID, len(m.Messages), m.Err, m.JumpDate)
		if m.ChannelID != a.activeChannelID {
			return nil, true // stale: user navigated away
		}
		if m.JumpDate {
			return reduceJumpDateLoaded(a, m), true
		}
		if m.Err != nil || len(m.Messages) == 0 {
			return func() tea.Msg { return ToastMsg{Text: "Failed to load history around message"} }, true
		}
		// Confirm the target is actually in the fetched window BEFORE
		// replacing the buffer: a failed jump must keep the user's
		// current position (spec error table), not strand them in an
		// unrelated window.
		found := false
		for _, msg := range m.Messages {
			if msg.TS == m.TargetTS {
				found = true
				break
			}
		}
		if !found {
			return func() tea.Msg { return ToastMsg{Text: "Message not found in loaded history"} }, true
		}
		a.messagepane.SetMessages(a.applyStarredFlags(m.ChannelID, m.Messages))
		a.messagepane.SelectByTS(m.TargetTS)
		return nil, true

	case ChannelMarkedRemoteMsg:
		a.applyChannelMark(m.ChannelID, m.TS, m.UnreadCount)
		return nil, true

	case ChannelMarkedReadMsg:
		debuglog.Cache("ChannelMarkedReadMsg: channel=%s active=%s (optimistic clear)",
			m.ChannelID, a.activeChannelID)
		a.notifyReadStateChanged()
		return nil, true

	case ChannelMembershipMsg:
		a.SetChannelMembership(m.ChannelID, m.MemberIDs)
		return nil, true

	case ChannelJoinedMsg:
		return openCreatedChannel(a, m.ID, m.Name, "channel"), true

	case ChannelCreatedMsg:
		chType := "channel"
		if m.Private {
			chType = "private"
		}
		toast := toastWithClear(a, "Created #"+m.Name, 2*time.Second)
		return tea.Batch(toast, openCreatedChannel(a, m.ID, m.Name, chType)), true

	case ChannelCreateFailedMsg:
		return toastWithClear(a, fmt.Sprintf("Create failed: %v", m.Err), 3*time.Second), true

	case ChannelInvitedMsg:
		return toastWithClear(a, inviteToast(m), 2*time.Second), true

	case ChannelInviteFailedMsg:
		return toastWithClear(a, "Invite failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true

	case KickUserMsg:
		channels := a.channels
		chID, userID := ids.ChannelID(m.ChannelID), m.UserID
		return func() tea.Msg { return channels.Kick(chID, userID) }, true

	case ChannelKickedMsg:
		return toastWithClear(a, "Removed "+m.UserID+" from #"+m.Channel, 2*time.Second), true

	case ChannelKickFailedMsg:
		return toastWithClear(a, "Kick failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true

	case AddChannelManagersMsg:
		channels := a.channels
		chID, users := ids.ChannelID(m.ChannelID), m.UserIDs
		return func() tea.Msg { return channels.AddManagers(chID, users) }, true

	case ChannelManagersAddedMsg:
		who := strings.Join(m.UserIDs, ", ")
		if len(m.UserIDs) == 0 {
			who = "member"
		}
		return toastWithClear(a, "Made "+who+" a Channel Manager of #"+m.Channel, 2*time.Second), true

	case ChannelManagersAddFailedMsg:
		return toastWithClear(a, "Channel Manager failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true

	case ChannelNotifySetMsg:
		if m.Err != nil {
			return toastWithClear(a, "Notify failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		label := notifyPrefLabel(m.Level)
		name := m.ChannelID
		for _, it := range a.sidebar.Items() {
			if it.ID == m.ChannelID {
				name = it.Name
				break
			}
		}
		return toastWithClear(a, "#"+name+": "+label, 2*time.Second), true

	case ChannelJoinFailedMsg:
		// Nothing fancy yet -- could surface a status-bar toast
		// in future.
		log.Printf("warning: failed to join channel %s: %v", m.Name, m.Err)
		return nil, true

	case LeaveChannelMsg:
		channels := a.channels
		id, name := ids.ChannelID(m.ID), m.Name
		return func() tea.Msg { return channels.Leave(id, name) }, true

	case CloseChannelMsg:
		channels := a.channels
		id, name := ids.ChannelID(m.ID), m.Name
		return func() tea.Msg { return channels.Close(id, name) }, true

	case ChannelClosedMsg:
		return reduceChannelClosed(a, m), true

	case ChannelCloseFailedMsg:
		return toastWithClear(a, fmt.Sprintf("Failed to close %s: %v", m.Name, m.Err), 3*time.Second), true

	case ChannelReopenedMsg:
		items := a.sidebar.Items()
		out := make([]sidebar.ChannelItem, len(items))
		copy(out, items)
		for i := range out {
			if out[i].ID == m.ID {
				out[i].Closed = false
			}
		}
		a.SetChannels(out)
		return nil, true

	case ChannelReopenFailedMsg:
		return nil, true

	case ChannelLeftMsg:
		return reduceChannelLeft(a, m), true

	case ChannelLeaveFailedMsg:
		log.Printf("warning: failed to leave channel %s: %v", m.Name, m.Err)
		return toastWithClear(a, fmt.Sprintf("Failed to leave #%s: %v", m.Name, m.Err), 3*time.Second), true

	case ChannelMutedMsg:
		if m.Err != nil {
			a.sidebar.SetMuted(m.ChannelID, !m.Muted)
			return toastWithClear(a, "Mute failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		return nil, true

	case channelSearchDebounceMsg:
		// The typing has stopped. Anything older than the current
		// generation is a keystroke the user has already typed past.
		if m.gen != a.pendingChannelSearchGen || m.query == "" {
			return nil, true
		}
		channels := a.channels
		team, gen, query := a.activeTeamID, m.gen, m.query
		omni := a.channelFinder.Omni()
		cur := a.activeChannelID
		recent := a.sidebar.ChannelIDsInOrder()
		return func() tea.Msg {
			msg := RemoteChannelsFoundMsg{
				TeamID: team,
				Query:  query,
				Gen:    gen,
				Items:  channels.SearchRemote(query),
			}
			if omni {
				msg.Omni = channels.SearchOmni(query, cur, recent)
			}
			return msg
		}, true

	case ChannelChromeMsg:
		if m.ChannelID != a.activeChannelID {
			return nil, true
		}
		if m.Gen != 0 && m.Gen != a.chromeGen {
			return nil, true
		}
		for _, mm := range a.modelsForChannel(m.ChannelID) {
			mm.SetHeaderChrome(m.Bookmarks, m.Pins)
		}
		return nil, true

	case RemoteChannelsFoundMsg:
		// Drop answers to a superseded query or to a workspace the
		// user has since switched away from: SetBrowseable replaces
		// the whole non-joined set, so a late arrival would visibly
		// overwrite the list under the cursor.
		if m.TeamID != a.activeTeamID || m.Gen != a.pendingChannelSearchGen {
			return nil, true
		}
		a.channelFinder.SetBrowseable(m.Items)
		if a.channelFinder.Omni() {
			a.channelFinder.SetOmniHits(m.Omni)
		}
		return nil, true
	}
	return nil, false
}

// openCreatedChannel adds id to the sidebar (if missing), marks it
// joined in the finder, and returns a ChannelSelectedMsg to open it.
func openCreatedChannel(a *App, id, name, chType string) tea.Cmd {
	if chType == "" {
		chType = "channel"
	}
	newItem := sidebar.ChannelItem{
		ID:   id,
		Name: name,
		Type: chType,
	}
	items := a.sidebar.Items()
	alreadyInSidebar := false
	for _, it := range items {
		if it.ID == id {
			alreadyInSidebar = true
			break
		}
	}
	if !alreadyInSidebar {
		items = append(items, newItem)
		a.SetChannels(items)
	}
	a.channelFinder.MarkJoined(id)
	a.sidebar.SelectByID(id)
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}

func inviteToast(m ChannelInvitedMsg) string {
	if len(m.UserIDs) == 1 {
		return "Invited " + m.UserIDs[0]
	}
	if len(m.UserIDs) > 1 {
		return fmt.Sprintf("Invited %d members", len(m.UserIDs))
	}
	if len(m.Emails) == 1 {
		return "Invited " + m.Emails[0]
	}
	return fmt.Sprintf("Invited %d people", len(m.Emails))
}

func notifyPrefLabel(level string) string {
	switch level {
	case "mentions_dms":
		return "mentions only"
	case "everything":
		return "all new posts"
	default:
		return level
	}
}

func reduceChannelClosed(a *App, m ChannelClosedMsg) tea.Cmd {
	items := a.sidebar.Items()
	out := make([]sidebar.ChannelItem, len(items))
	copy(out, items)
	for i := range out {
		if out[i].ID == m.ID {
			out[i].Closed = true
		}
	}
	a.SetChannels(out)
	toast := toastWithClear(a, "Closed "+m.Name, 2*time.Second)
	if a.view == ViewDMs {
		return toast
	}
	if a.activeChannelID == m.ID {
		return tea.Batch(toast, a.switchAfterLeave(m.ID))
	}
	return toast
}

// reduceChannelLeft removes the left channel from the sidebar and
// finder, then switches to last-visited (nav history) or Threads.
func reduceChannelLeft(a *App, m ChannelLeftMsg) tea.Cmd {
	items := a.sidebar.Items()
	next := make([]sidebar.ChannelItem, 0, len(items))
	for _, it := range items {
		if it.ID != m.ID {
			next = append(next, it)
		}
	}
	a.SetChannels(next)
	a.channelFinder.MarkUnjoined(m.ID)
	if a.lastChannelByTeam[a.activeTeamID] == m.ID {
		delete(a.lastChannelByTeam, a.activeTeamID)
	}
	a.CloseThread()

	toast := toastWithClear(a, "Left #"+m.Name, 2*time.Second)
	return tea.Batch(toast, a.switchAfterLeave(m.ID))
}

// switchAfterLeave picks the destination after leaving leftID:
// the most recent still-joined nav-history channel, or Threads.
func (a *App) switchAfterLeave(leftID string) tea.Cmd {
	remaining := make(map[string]sidebar.ChannelItem, len(a.sidebar.Items()))
	for _, it := range a.sidebar.Items() {
		remaining[it.ID] = it
	}
	if stack := a.navHistory.Stack(a.activeTeamID); stack != nil {
		for i := stack.cursor; i >= 0; i-- {
			id := stack.entries[i]
			if id == leftID {
				continue
			}
			if ch, ok := remaining[id]; ok {
				a.sidebar.SelectByID(ch.ID)
				return func() tea.Msg {
					return ChannelSelectedMsg{ID: ch.ID, Name: ch.Name, Type: ch.Type}
				}
			}
		}
	}
	a.activeChannelID = ""
	a.sidebar.SetActiveChannelID("")
	a.sidebar.SelectThreadsRow()
	return func() tea.Msg { return ThreadsViewActivatedMsg{} }
}

// retargetActiveChannel points the App's active-channel context
// (activeChannelID, typing throttle, compose targets, statusbar) at
// the given channel and fires the async membership fetch. Factored
// out of reduceChannelSelected so window-focus changes — which swap
// per-window models without re-running channel selection (Phase 3)
// — can reuse it. Selection-only semantics (nav-history push,
// RecordVisit, thread close, tiered cache load, mark-read) stay in
// the reducer.
func (a *App) retargetActiveChannel(id, name, chType string) {
	a.swapChannelCompose(id)
	a.activeChannelID = id
	a.typingOut.ResetThrottle() // reset typing throttle for new channel
	a.compose.SetChannel(name)
	a.compose.SetActiveChannel(id)
	a.threadCompose.SetActiveChannel(id)
	// Fire the membership fetcher on a fresh goroutine so it can't
	// block the Update loop. Fire-and-forget -- results arrive
	// later via ChannelMembershipMsg. main.go's MembershipFetch
	// closure ultimately calls Membership.EnsureFresh which invokes
	// bubbletea Program.Send via pushSnapshot, and bubbletea v2's
	// program channel is unbuffered: a Send from inside Update
	// would deadlock waiting for the same goroutine to receive.
	// See manager.go's EnsureFresh docs and the deadlock-regression
	// test in app_test.go.
	{
		channels := a.channels
		channelID := ids.ChannelID(id)
		go channels.MembershipFetch(channelID)
	}
	a.statusbar.SetChannel(name)
	a.statusbar.SetChannelType(chType)
	if memberIDs, ok := a.compose.ChannelMemberIDs(id); ok {
		a.messagepane.SetMemberCount(len(memberIDs))
	}
	// Clear any syncing indicator stranded by an in-flight tier-2
	// verify fetch: the SetSyncing(false) in the MessagesLoadedMsg arm
	// is gated on the then-active channel, so a focus change away from
	// the verifying channel would otherwise leave the glyph stuck.
	// Safe for the selection path too — all three tiers set syncing
	// explicitly right after this retarget runs.
	a.statusbar.SetSyncing(false)
}

// reduceChannelSelected handles ChannelSelectedMsg. Extracted from
// the reduceChannels dispatch switch because the arm is ~120 lines
// with three tiered cache-freshness branches.
//
// Returns (cmd, fetchFired) where fetchFired reports whether a network
// message-fetch was dispatched (i.e. a MessagesLoadedMsg will follow).
// The permalink-completion hook uses !fetchFired as its `authoritative`
// flag.
func (a *App) selectionIsDirect(m ChannelSelectedMsg) bool {
	if channelTypeIsDirect(m.Type) {
		return true
	}
	if m.Type != "" {
		return false
	}
	for _, it := range a.sidebar.Items() {
		if it.ID == m.ID {
			return channelTypeIsDirect(it.Type)
		}
	}
	return false
}

func reduceChannelSelected(a *App, m ChannelSelectedMsg) (tea.Cmd, bool) {
	if a.compose.Uploading() || a.threadCompose.Uploading() {
		return a.uploadToastCmd("Upload in progress", 2*time.Second), false
	}
	// Perf instrumentation: wall-clock the synchronous portion of the
	// channel-switch reducer. This covers everything up to and including
	// the tier decision, but NOT the subsequent View() (which triggers
	// messages.buildCache / sidebar.buildCache and is already instrumented
	// separately). The big synchronous cost inside this arm is
	// a.channels.ReadCache -> loadCachedMessages's SQLite N+1; the
	// loadCachedMessages [perf] line will attribute that.
	var perfStart time.Time
	var tier string
	if debuglog.Enabled() {
		perfStart = time.Now()
		defer func() {
			debuglog.Perf("reduceChannelSelected channel=%s name=%q tier=%s total=%s",
				m.ID, m.Name, tier, time.Since(perfStart))
		}()
	}
	a.cancelEdit()
	// Opening a DM from the DMs tab stays on that tab so the full
	// conversation list remains the second sidebar. Any other
	// destination (Home channel, finder, permalink) returns to Home.
	if a.view == ViewDMs && a.selectionIsDirect(m) {
		a.setInboxView(ViewDMs)
	} else {
		a.setInboxView(ViewChannels)
	}
	a.lastOpenedChannelID = ""
	a.lastOpenedThreadTS = ""
	// Close thread panel when switching channels.
	a.CloseThread()
	a.clearSelections()
	// Search state is per-channel: drop highlights, match list, and
	// the status-line segment on every switch.
	a.clearActiveSearch()
	// Move focus to the messages pane so the user can immediately
	// j/k through messages, react, open threads, etc. without first
	// having to Tab/h-l out of the sidebar after picking a channel.
	a.focusedPanel = PanelMessages
	// Update local finder ordering immediately so the next Ctrl+T
	// sees this channel at the top of the recents.
	now := time.Now().Unix()
	a.channelFinder.UpdateLastVisited(m.ID, now)
	a.sidebar.TouchVisit(m.ID, now)
	// Persist the visit (SQLite write + WorkspaceContext map update)
	// asynchronously via main.go's recorder closure.
	a.channels.RecordVisit(ids.ChannelID(m.ID))
	if !m.FromHistory {
		a.navHistory.Push(a.activeTeamID, m.ID)
	}
	// Tell the sidebar which channel is active so the staleness
	// filter never hides it out from under the user.
	a.sidebar.SetActiveChannelID(m.ID)
	a.messagepane.SetChannel(m.Name, topicForChannel(a, m.ID, m.Topic))
	a.messagepane.SetChannelType(m.Type)

	// Close any open mention picker before switching channels.
	// SetUsers replaces the user list but does NOT re-run the
	// picker's filter, so an open picker would render the previous
	// channel's matches until the user typed or moved. CloseMention
	// is nil-safe (no-op when already closed).
	a.compose.CloseMention()
	a.threadCompose.CloseMention()

	a.retargetActiveChannel(m.ID, m.Name, m.Type)
	// Record the applied selection on the focused window so window
	// focus changes can retarget to it (see internal/ui/windows.go).
	a.setFocusedWindowChannel(m.ID, m.Name, m.Type)

	cached := a.channels.ReadCache(ids.ChannelID(m.ID))
	syncedAt := a.channels.SyncedAt(ids.ChannelID(m.ID))
	age := time.Duration(0)
	if syncedAt > 0 {
		age = time.Since(time.Unix(syncedAt, 0))
	}
	debuglog.Cache("ChannelSelectedMsg: channel=%s name=%q cache_hit_count=%d synced_at=%d age_ms=%d",
		m.ID, m.Name, len(cached), syncedAt, age.Milliseconds())

	fetchCmd := func() tea.Cmd {
		channels := a.channels
		chID, chName := ids.ChannelID(m.ID), m.Name
		debuglog.Cache("ChannelSelectedMsg: channel=%s firing background network fetch", m.ID)
		return func() tea.Msg { return channels.Fetch(chID, chName) }
	}

	switch {
	case syncedAt > 0 && age < cacheFreshThreshold:
		// Tier 1: provably fresh (cache was just synced). Render
		// whatever we have (cached can legitimately be empty here
		// -- e.g., a channel verified empty within the last 30s).
		// Mark-as-read if non-empty. No fetch.
		a.messagepane.SetLoading(false)
		a.messagepane.SetMessages(a.applyStarredFlags(m.ID, cached))
		a.statusbar.SetSyncing(false)
		debuglog.Cache("ChannelSelectedMsg: channel=%s tier=1_fresh", m.ID)
		tier = "1_fresh"
		if len(cached) == 0 {
			return withChromeFetch(a, m.ID, nil), false
		}
		channels := a.channels
		chID := ids.ChannelID(m.ID)
		latestTS := ids.MessageTS(cached[len(cached)-1].TS)
		// MarkRead produces ChannelMarkedReadMsg, NOT MessagesLoadedMsg,
		// so no authoritative permalink completion will follow.
		return withChromeFetch(a, m.ID, func() tea.Msg { return channels.MarkRead(chID, latestTS) }), false

	case len(cached) > 0:
		// Tier 2: cache exists, verify in background. Covers
		// (a) syncedAt > 0 with age >= 30s (any age -- we render
		//     and verify rather than blanking the pane),
		// (b) syncedAt == 0 (freshness unknown; could be a prior
		//     session's cache or an un-wired reader). Always
		//     render + fire fetch + show indicator so the user
		//     knows it's being checked.
		a.messagepane.SetLoading(false)
		a.messagepane.SetMessages(a.applyStarredFlags(m.ID, cached))
		a.statusbar.SetSyncing(true)
		debuglog.Cache("ChannelSelectedMsg: channel=%s tier=2_verify", m.ID)
		tier = "2_verify"
		return withChromeFetch(a, m.ID, fetchCmd()), true

	default:
		// Tier 3: no cache at all (genuine cold-start,
		// never-visited channel). Spinner + fetch.
		a.messagepane.SetLoading(true)
		a.messagepane.SetMessages(nil)
		a.statusbar.SetSyncing(false)
		debuglog.Cache("ChannelSelectedMsg: channel=%s tier=3_spinner", m.ID)
		tier = "3_spinner"
		return withChromeFetch(a, m.ID, tea.Batch(spinnerTickCmd(), fetchCmd())), true
	}
}

// withChromeFetch batches the header bookmarks/pins fetch onto cmd
// so it runs in parallel with the messages load (or alone on tier 1).
func withChromeFetch(a *App, channelID string, cmd tea.Cmd) tea.Cmd {
	chrome := a.chromeFetchCmd(channelID)
	reopen := a.reopenClosedDMCmd(channelID)
	return tea.Batch(chrome, reopen, cmd)
}

func (a *App) reopenClosedDMCmd(channelID string) tea.Cmd {
	closed := false
	direct := false
	for _, it := range a.sidebar.Items() {
		if it.ID != channelID {
			continue
		}
		closed = it.Closed
		direct = channelTypeIsDirect(it.Type)
		break
	}
	if !closed || !direct {
		return nil
	}
	channels := a.channels
	id := ids.ChannelID(channelID)
	return func() tea.Msg { return channels.Reopen(id) }
}

func (a *App) chromeFetchCmd(channelID string) tea.Cmd {
	if channelID == "" {
		return nil
	}
	a.chromeGen++
	gen := a.chromeGen
	channels := a.channels
	chID := ids.ChannelID(channelID)
	return func() tea.Msg {
		msg := channels.FetchChrome(chID)
		cm, ok := msg.(ChannelChromeMsg)
		if !ok {
			return msg
		}
		if cm.ChannelID == "" {
			cm.ChannelID = string(chID)
		}
		cm.Gen = gen
		return cm
	}
}

// topicForChannel returns the header topic for a channel. Prefer the
// value already on ChannelSelectedMsg; otherwise take it from the
// sidebar item (seeded from cache at bootstrap). Empty means the
// message-pane header stays a single line.
func topicForChannel(a *App, id, msgTopic string) string {
	if msgTopic != "" {
		return msgTopic
	}
	for _, it := range a.sidebar.Items() {
		if it.ID == id {
			return it.Topic
		}
	}
	return ""
}
