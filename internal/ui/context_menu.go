// internal/ui/context_menu.go
//
// Builds the message-actions overlay and dispatches selected rows onto
// existing App helpers (openPickerFromMessage, copyPermalinkOfSelected,
// openSharePicker, …). No new Slack APIs here.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ui/contextmenu"
	"github.com/agustif/slk/internal/ui/messages"
)

func (a *App) openMessageContextMenu(anchored bool, anchorX, anchorY int) tea.Cmd {
	items, ok := a.contextMenuItems()
	if !ok {
		return nil
	}
	if anchored {
		a.contextMenu.OpenAt(items, anchorX, anchorY)
	} else {
		a.contextMenu.Open(items)
	}
	a.SetMode(ModeContextMenu)
	return nil
}

func (a *App) contextMenuItems() ([]contextmenu.Item, bool) {
	if a.view == ViewLater && a.focusedPanel == PanelMessages {
		return a.laterContextMenuItems()
	}
	msg, ok := a.selectedMessageItem()
	if !ok {
		return nil, false
	}
	own := a.isOwnMessage(msg)
	hasFile := false
	for _, att := range msg.Attachments {
		if att.Kind == "file" && att.DownloadURL != "" {
			hasFile = true
			break
		}
	}
	hasLinks := len(messages.ExtractLinks(msg.Text)) > 0
	hasReactions := len(msg.Reactions) > 0

	channelID := a.activeChannelID
	if a.focusedPanel == PanelThread {
		channelID = a.threadPanel.ChannelID()
	}
	laterLabel := "Save for later"
	if a.laterSaved[laterKey(channelID, msg.TS)] {
		laterLabel = "Remove from Later"
	}
	pinLabel := "Pin message"
	if msg.Pinned {
		pinLabel = "Unpin message"
	}
	starLabel := "Star message"
	if msg.Starred || a.isMessageStarred(channelID, msg.TS) {
		starLabel = "Unstar message"
	}
	followEnabled := a.focusedPanel == PanelThread && a.threadVisible
	followLabel := "Follow thread"
	if followEnabled && a.threadPanel.Following() {
		followLabel = "Unfollow thread"
	}

	return []contextmenu.Item{
		{Label: "Add reaction", Action: contextmenu.ActionAddReaction, Enabled: true},
		{Label: "Reply in thread", Action: contextmenu.ActionReplyInThread, Enabled: true},
		{Label: laterLabel, Action: contextmenu.ActionSaveForLater, Enabled: true},
		{Label: "Remind me", Action: contextmenu.ActionRemind, Enabled: true},
		{Label: "Copy permalink", Action: contextmenu.ActionCopyPermalink, Enabled: true},
		{Label: "Share", Action: contextmenu.ActionShare, Enabled: true},
		{Label: pinLabel, Action: contextmenu.ActionPin, Enabled: true},
		{Label: starLabel, Action: contextmenu.ActionStar, Enabled: true},
		{Label: followLabel, Action: contextmenu.ActionFollowThread, Enabled: followEnabled},
		{Label: "Download file", Action: contextmenu.ActionDownloadFile, Enabled: hasFile},
		{Label: "Open links", Action: contextmenu.ActionOpenLinks, Enabled: hasLinks},
		{Label: "Edit message", Action: contextmenu.ActionEdit, Enabled: own},
		{Label: "Delete message", Action: contextmenu.ActionDelete, Enabled: own},
		{Label: "Mark unread", Action: contextmenu.ActionMarkUnread, Enabled: true},
		{Label: "List reactions", Action: contextmenu.ActionListReactions, Enabled: hasReactions},
	}, true
}

func (a *App) laterContextMenuItems() ([]contextmenu.Item, bool) {
	it, ok := a.laterView.SelectedItem()
	if !ok {
		return nil, false
	}
	complete := it.State != "completed"
	archive := it.State != "archived"
	restore := it.State != "in_progress" && it.State != ""
	return []contextmenu.Item{
		{Label: "Open", Action: contextmenu.ActionReplyInThread, Enabled: true},
		{Label: "Mark complete", Action: contextmenu.ActionLaterComplete, Enabled: complete},
		{Label: "Archive", Action: contextmenu.ActionLaterArchive, Enabled: archive},
		{Label: "Move to In progress", Action: contextmenu.ActionLaterRestore, Enabled: restore},
		{Label: "Remove from Later", Action: contextmenu.ActionSaveForLater, Enabled: true},
		{Label: "Remind me", Action: contextmenu.ActionRemind, Enabled: true},
	}, true
}

func (a *App) selectedMessageItem() (messages.MessageItem, bool) {
	switch a.focusedPanel {
	case PanelMessages:
		if !a.inChannelView() {
			return messages.MessageItem{}, false
		}
		return a.messagepane.SelectedMessage()
	case PanelThread:
		reply := a.threadPanel.SelectedReply()
		if reply == nil {
			return messages.MessageItem{}, false
		}
		return *reply, true
	default:
		return messages.MessageItem{}, false
	}
}

func (a *App) dispatchContextMenuAction(action contextmenu.ActionID) tea.Cmd {
	switch action {
	case contextmenu.ActionAddReaction:
		if a.focusedPanel == PanelThread {
			return a.openPickerFromThread()
		}
		return a.openPickerFromMessage()
	case contextmenu.ActionSaveForLater:
		return a.toggleSaveForLater()
	case contextmenu.ActionRemind:
		return a.openRemindDuration()
	case contextmenu.ActionPin:
		return a.togglePinOfSelected()
	case contextmenu.ActionStar:
		return a.toggleStarOfSelected()
	case contextmenu.ActionFollowThread:
		return a.toggleFollowOfOpenThread()
	case contextmenu.ActionReplyInThread:
		if a.view == ViewLater {
			return a.openSelectedLaterCmd()
		}
		if a.focusedPanel == PanelThread {
			a.SetMode(ModeInsert)
			a.focusedPanel = PanelThread
			return a.threadCompose.Focus()
		}
		return a.openThreadForSelectedMessage()
	case contextmenu.ActionCopyPermalink:
		return a.copyPermalinkOfSelected()
	case contextmenu.ActionShare:
		return a.openSharePicker()
	case contextmenu.ActionDownloadFile:
		return a.downloadFilesOfSelected()
	case contextmenu.ActionOpenLinks:
		return a.openLinksOfSelected()
	case contextmenu.ActionEdit:
		return a.beginEditOfSelected()
	case contextmenu.ActionDelete:
		return a.beginDeleteOfSelected()
	case contextmenu.ActionMarkUnread:
		return a.markUnreadOfSelected()
	case contextmenu.ActionListReactions:
		return a.openReactionsView()
	case contextmenu.ActionLaterComplete:
		return a.setLaterState("completed")
	case contextmenu.ActionLaterArchive:
		return a.setLaterState("archived")
	case contextmenu.ActionLaterRestore:
		return a.setLaterState("in_progress")
	}
	return nil
}
