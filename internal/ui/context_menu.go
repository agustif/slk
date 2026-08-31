// internal/ui/context_menu.go
//
// Builds the message-actions overlay and dispatches selected rows onto
// existing App helpers (openPickerFromMessage, copyPermalinkOfSelected,
// …). No new Slack APIs here.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/contextmenu"
	"github.com/gammons/slk/internal/ui/messages"
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

	return []contextmenu.Item{
		{Label: "Add reaction", Action: contextmenu.ActionAddReaction, Enabled: true},
		{Label: "Reply in thread", Action: contextmenu.ActionReplyInThread, Enabled: true},
		{Label: "Copy permalink", Action: contextmenu.ActionCopyPermalink, Enabled: true},
		{Label: "Download file", Action: contextmenu.ActionDownloadFile, Enabled: hasFile},
		{Label: "Open links", Action: contextmenu.ActionOpenLinks, Enabled: hasLinks},
		{Label: "Edit message", Action: contextmenu.ActionEdit, Enabled: own},
		{Label: "Delete message", Action: contextmenu.ActionDelete, Enabled: own},
		{Label: "Mark unread", Action: contextmenu.ActionMarkUnread, Enabled: true},
		{Label: "List reactions", Action: contextmenu.ActionListReactions, Enabled: hasReactions},
	}, true
}

func (a *App) selectedMessageItem() (messages.MessageItem, bool) {
	switch a.focusedPanel {
	case PanelMessages:
		if a.view != ViewChannels {
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
	case contextmenu.ActionReplyInThread:
		if a.focusedPanel == PanelThread {
			a.SetMode(ModeInsert)
			a.focusedPanel = PanelThread
			return a.threadCompose.Focus()
		}
		return a.openThreadForSelectedMessage()
	case contextmenu.ActionCopyPermalink:
		return a.copyPermalinkOfSelected()
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
	}
	return nil
}
