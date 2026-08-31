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

// swapChannelCompose parks the live channel compose under the current
// channel and restores channelID's draft (empty if none). Cancels an
// in-progress channel-pane edit first so we save editing.stashedDraft
// rather than the message-being-edited text.
func (a *App) swapChannelCompose(channelID string) {
	if a.editing.IsActive() && a.editing.Panel() == PanelMessages {
		a.cancelEdit()
	}
	a.compose.BindDraftKey(a.activeChannelID)
	a.compose.SwapDraft(channelID)
}

// swapThreadCompose parks the live thread compose under the currently
// open thread (from threadPanel) and restores (channelID, threadTS).
// Empty IDs park the box (CloseThread). Cancels a thread-pane edit
// first, same stash/restore contract as swapChannelCompose.
func (a *App) swapThreadCompose(channelID, threadTS string) {
	if a.editing.IsActive() && a.editing.Panel() == PanelThread {
		a.cancelEdit()
	}
	a.threadCompose.BindDraftKey(threadDraftKey(a.threadPanel.ChannelID(), a.threadPanel.ThreadTS()))
	a.threadCompose.SwapDraft(threadDraftKey(channelID, threadTS))
}
