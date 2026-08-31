package ui

// threadDraftKey identifies a thread compose snapshot. Channel compose
// keys are bare channel IDs; thread keys join channelID and threadTS
// with a NUL so they cannot collide with a channel ID.
func threadDraftKey(channelID, threadTS string) string {
	if channelID == "" || threadTS == "" {
		return ""
	}
	return channelID + "\x00" + threadTS
}

func (a *App) persistDraft(key, text string) {
	if a.draftSaveFn == nil || a.activeTeamID == "" || key == "" {
		return
	}
	team, save := a.activeTeamID, a.draftSaveFn
	go save(team, key, text)
}

func (a *App) flushComposeDrafts() {
	a.flushComposeDraftsFor(a.activeTeamID)
}

func (a *App) flushComposeDraftsFor(teamID string) {
	a.flushComposeDraftsForMode(teamID, false)
}

// flushComposeDraftsSync writes parked compose text on the calling
// goroutine so quit can finish Slack drafts.create/update before the
// process exits. Channel switches keep the async path.
func (a *App) flushComposeDraftsSync() {
	a.flushComposeDraftsForMode(a.activeTeamID, true)
}

func (a *App) flushComposeDraftsForMode(teamID string, sync bool) {
	if a.draftSaveFn == nil || teamID == "" {
		return
	}
	if a.activeChannelID != "" {
		a.compose.BindDraftKey(a.activeChannelID)
	}
	if ts := a.threadPanel.ThreadTS(); ts != "" {
		a.threadCompose.BindDraftKey(threadDraftKey(a.threadPanel.ChannelID(), ts))
	}
	save := a.draftSaveFn
	run := func(k, text string) {
		if sync {
			save(teamID, k, text)
			return
		}
		go save(teamID, k, text)
	}
	for k, text := range a.compose.SnapshotTextDrafts() {
		run(k, text)
	}
	for k, text := range a.threadCompose.SnapshotTextDrafts() {
		run(k, text)
	}
}

func (a *App) loadComposeDrafts(workspaceID string) {
	if a.draftLoadFn == nil || workspaceID == "" {
		return
	}
	m := a.draftLoadFn(workspaceID)
	a.compose.MergeTextDrafts(m)
	a.threadCompose.MergeTextDrafts(m)
}

func (a *App) clearComposeDraft(key string) {
	a.persistDraft(key, "")
}

// swapChannelCompose parks the live channel compose under the current
// channel and restores channelID's draft (empty if none). Cancels an
// in-progress channel-pane edit first so we save editing.stashedDraft
// rather than the message-being-edited text.
func (a *App) swapChannelCompose(channelID string) {
	if a.editing.IsActive() && a.editing.Panel() == PanelMessages {
		a.cancelEdit()
	}
	old := a.compose.DraftKey()
	a.compose.BindDraftKey(a.activeChannelID)
	a.compose.SwapDraft(channelID)
	if old != "" && old != channelID {
		if snaps := a.compose.SnapshotTextDrafts(); snaps[old] == "" {
			a.clearComposeDraft(old)
		}
	}
	a.flushComposeDrafts()
}

// swapThreadCompose parks the live thread compose under the currently
// open thread (from threadPanel) and restores (channelID, threadTS).
// Empty IDs park the box (CloseThread). Cancels a thread-pane edit
// first, same stash/restore contract as swapChannelCompose.
func (a *App) swapThreadCompose(channelID, threadTS string) {
	if a.editing.IsActive() && a.editing.Panel() == PanelThread {
		a.cancelEdit()
	}
	old := a.threadCompose.DraftKey()
	a.threadCompose.BindDraftKey(threadDraftKey(a.threadPanel.ChannelID(), a.threadPanel.ThreadTS()))
	next := threadDraftKey(channelID, threadTS)
	a.threadCompose.SwapDraft(next)
	if old != "" && old != next {
		if snaps := a.threadCompose.SnapshotTextDrafts(); snaps[old] == "" {
			a.clearComposeDraft(old)
		}
	}
	a.flushComposeDrafts()
}
