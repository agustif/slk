// internal/ui/reducer_io.go
//
// IO / toast / asset-loading reducer for App.Update (Phase 4l).
//
// Owns the leftover arms that don't belong to any domain reducer:
//
//	tea.PasteMsg              - bracketed paste from the terminal.
//	                             Try clipboard image / file path
//	                             first, else forward to the
//	                             focused compose's textarea.
//	UploadProgressMsg         - in-flight upload progress toast.
//	UploadResultMsg           - upload finished: clear compose
//	                            attachments + Sent/Failed toast.
//	ConnectionStateMsg        - WS connection state changed:
//	                            push to status bar.
//	ToastMsg                  - generic toast (3s auto-clear).
//	editEmptyToastMsg         - "Edit must have text" toast.
//
//	imgrender.ImageReadyMsg   - lazy attachment-image fetch
//	                            landed: invalidate the affected
//	                            render caches.
//	imgrender.ImageFailedMsg  - lazy attachment-image fetch
//	                            permanently failed: clear
//	                            in-flight bookkeeping.
//	messages.AvatarReadyMsg   - lazy avatar fetch landed:
//	                            invalidate both pane caches.
//
//	statusbar.CopiedMsg               - "N chars copied"
//	statusbar.CopiedClearMsg          - 2/3s expiry tick
//	statusbar.PermalinkCopiedMsg      - "Copied permalink"
//	statusbar.PermalinkCopyFailedMsg  - "Failed to copy link"
//	statusbar.MessageCopiedMsg        - "Copied message"
//	prefixTimeoutMsg                  - cancel pending gg/yy prefix
//	statusbar.MarkedUnreadMsg         - "Marked unread"
//	statusbar.MarkUnreadFailedMsg     - "Mark unread failed: ..."
//	statusbar.EditFailedMsg           - "Edit failed: ..."
//	statusbar.EditNotOwnMsg           - "Can only edit your own..."
//	statusbar.DeleteFailedMsg         - "Delete failed: ..."
//	statusbar.DeleteNotOwnMsg         - "Can only delete your own..."
//	statusbar.SendFailedMsg           - "Send failed: ..."
//	MessageSharedMsg                  - "Shared to #channel"
//	MessageShareFailedMsg             - "Share failed: ..."
//
// Free reducer: these arms have no shared domain or invariant,
// only the common "push to status bar / clear after N seconds"
// shape. Grouping them here keeps the residual Update switch
// near-empty.
//
// Two small helpers (toastCmd, fixedToastCmd) collapse the
// repetitive `cmds = append(cmds, tea.Tick(Ns, ... CopiedClearMsg))`
// idiom that recurred ~11 times in the original switch.
package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/debuglog"
	"github.com/agustif/slk/internal/ui/imgrender"
	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/statusbar"
)

// copiedClearAfter schedules a CopiedClearMsg `d` from now. The
// status bar's CopiedClearMsg handler clears the toast slot.
func copiedClearAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return statusbar.CopiedClearMsg{}
	})
}

// emojiInvalidateDebounce is the coalesce window for EmojiImageReadyMsg
// arrivals. Chosen to absorb the typical burst of fetch completions on
// a busy channel's first render (50+ unique emoji finishing within tens
// of milliseconds of each other) into a single cache rebuild. Tuned for
// "imperceptibly delayed" rather than "instantaneous": a 100ms wait
// before image emoji appear is well within the budget where a user
// would not notice, while the savings from coalescing N rebuilds into
// 1 are dramatic.
const emojiInvalidateDebounce = 100 * time.Millisecond

// toastWithClear pushes text into the status bar's toast slot and
// schedules the clear after `d`. Used by the fixed-text and
// formatted-reason toasts below.
func toastWithClear(a *App, text string, d time.Duration) tea.Cmd {
	a.statusbar.SetToast(text)
	return copiedClearAfter(d)
}

var reduceIO reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case tea.PasteMsg:
		return reducePaste(a, m), true

	case statusbar.CopiedMsg:
		a.statusbar.ShowCopied(m.N)
		return copiedClearAfter(2 * time.Second), true

	case statusbar.CopiedClearMsg:
		_ = m
		a.statusbar.ClearCopied()
		return nil, true

	case statusbar.PermalinkCopiedMsg:
		_ = m
		return toastWithClear(a, "Copied permalink", 2*time.Second), true

	case statusbar.PermalinkCopyFailedMsg:
		_ = m
		return toastWithClear(a, "Failed to copy link", 2*time.Second), true

	case statusbar.MessageCopiedMsg:
		_ = m
		return toastWithClear(a, "Copied message", 2*time.Second), true

	case prefixTimeoutMsg:
		if a.pendingPrefix == m.prefix && a.prefixGen == m.gen {
			a.pendingPrefix = 0
		}
		return nil, true

	case statusbar.MarkedUnreadMsg:
		_ = m
		return toastWithClear(a, "Marked unread", 2*time.Second), true

	case statusbar.MarkUnreadFailedMsg:
		return toastWithClear(a, "Mark unread failed: "+truncateReason(m.Reason, 40), 3*time.Second), true

	case statusbar.ThreadSavedMsg:
		return toastWithClear(a, "Saved "+filepath.Base(m.Path), 2*time.Second), true

	case statusbar.ThreadSaveFailedMsg:
		return toastWithClear(a, "Save failed: "+truncateReason(m.Reason, 40), 3*time.Second), true

	case statusbar.EditFailedMsg:
		return toastWithClear(a, "Edit failed: "+truncateReason(m.Reason, 40), 3*time.Second), true

	case editEmptyToastMsg:
		_ = m
		return toastWithClear(a, "Edit must have text (use D to delete)", 3*time.Second), true

	case statusbar.DeleteFailedMsg:
		return toastWithClear(a, "Delete failed: "+truncateReason(m.Reason, 40), 3*time.Second), true

	case statusbar.SendFailedMsg:
		return toastWithClear(a, "Send failed: "+truncateReason(m.Reason, 40), 3*time.Second), true

	case MessageSharedMsg:
		label := shareDestLabel(m.DestName, m.DestType)
		if label == "" {
			label = "channel"
		}
		return toastWithClear(a, "Shared to "+label, 2*time.Second), true

	case MessageShareFailedMsg:
		reason := m.Reason
		if reason == "" {
			reason = "unknown error"
		}
		return toastWithClear(a, "Share failed: "+truncateReason(reason, 40), 3*time.Second), true

	case statusbar.EditNotOwnMsg:
		_ = m
		return toastWithClear(a, "Can only edit your own messages", 2*time.Second), true

	case statusbar.DeleteNotOwnMsg:
		_ = m
		return toastWithClear(a, "Can only delete your own messages", 2*time.Second), true

	case PinToggledMsg:
		if m.Err != nil {
			action := "Pin"
			if !m.Pinned {
				action = "Unpin"
			}
			return toastWithClear(a, action+" failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		for _, mm := range a.modelsForChannel(m.ChannelID) {
			mm.SetPinned(m.TS, m.Pinned)
		}
		if a.threadVisible && a.threadPanel.ChannelID() == m.ChannelID {
			a.threadPanel.SetPinned(m.TS, m.Pinned)
		}
		text := "Pinned"
		if !m.Pinned {
			text = "Unpinned"
		}
		return toastWithClear(a, text, 2*time.Second), true

	case StarToggledMsg:
		if m.Err != nil {
			action := "Star"
			if !m.Starred {
				action = "Unstar"
			}
			return toastWithClear(a, action+" failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second), true
		}
		if a.starredMessages == nil {
			a.starredMessages = map[string]bool{}
		}
		a.starredMessages[starMessageKey(m.ChannelID, m.TS)] = m.Starred
		for _, mm := range a.modelsForChannel(m.ChannelID) {
			mm.SetStarred(m.TS, m.Starred)
		}
		if a.threadVisible && a.threadPanel.ChannelID() == m.ChannelID {
			a.threadPanel.SetStarred(m.TS, m.Starred)
		}
		text := "Starred"
		if !m.Starred {
			text = "Unstarred"
			delete(a.starredMessages, starMessageKey(m.ChannelID, m.TS))
			a.starredView.Remove(m.ChannelID, m.TS)
			a.sidebar.SetStarredCount(len(a.starredView.Items()))
		} else if cmd := a.fetchStarredMessagesCmd(); cmd != nil {
			return tea.Batch(toastWithClear(a, text, 2*time.Second), cmd), true
		}
		return toastWithClear(a, text, 2*time.Second), true

	case StarredLoadedMsg:
		if m.Err != nil {
			a.starredView.SetLoading(false)
			a.starredView.SetError("stars.list failed — " + m.Err.Error())
			return nil, true
		}
		stars := make(map[string]bool, len(m.Items))
		for _, it := range m.Items {
			if it.ChannelID != "" && it.TS != "" {
				stars[starMessageKey(it.ChannelID, it.TS)] = true
			}
		}
		a.starredMessages = stars
		a.applyStarredInbox(m.Items, m.FileIDs, m.Files)
		if a.activeChannelID != "" {
			for _, mm := range a.modelsForChannel(a.activeChannelID) {
				for i, msg := range mm.Messages() {
					_ = i
					mm.SetStarred(msg.TS, stars[starMessageKey(a.activeChannelID, msg.TS)])
				}
			}
		}
		return nil, true

	case ToastMsg:
		return toastWithClear(a, m.Text, 3*time.Second), true

	case UploadProgressMsg:
		a.statusbar.SetToast(fmt.Sprintf("Uploading %d/%d…", m.Done, m.Total))
		return nil, true

	case UploadResultMsg:
		a.compose.SetUploading(false)
		a.threadCompose.SetUploading(false)
		if m.Err != nil {
			return a.uploadToastCmd(
				"Upload failed: "+truncateReason(m.Err.Error(), 40),
				3*time.Second,
			), true
		}
		a.compose.ClearAttachments()
		a.threadCompose.ClearAttachments()
		chKey, thKey := a.compose.DraftKey(), a.threadCompose.DraftKey()
		a.compose.Reset()
		a.threadCompose.Reset()
		a.clearComposeDraft(chKey)
		a.clearComposeDraft(thKey)
		return a.uploadToastCmd("Sent", 2*time.Second), true

	case ConnectionStateMsg:
		a.statusbar.SetConnectionState(statusbar.ConnectionState(m.State))
		return nil, true

	case imgrender.ImageReadyMsg:
		debuglog.ImgFetch("recv: kind=ready channel=%s ts=%s key=%s req_id=%d",
			m.Channel, m.TS, m.Key, m.ReqID)
		// Image attachment finished downloading; invalidate the
		// messages pane's render cache for the affected channel
		// so the next View() picks up the cached bytes inline.
		// Only the specific key's in-flight bit is cleared so
		// sibling images that are still mid-fetch don't trigger
		// fresh respawns. Fan out to every window: each model
		// self-gates by its own channel name (no-op for windows
		// viewing other channels).
		for _, mp := range a.allWinModels() {
			mp.HandleImageReady(m.Channel, m.TS, m.Key)
		}
		// Thread panel: v1 uses coarse cache invalidation. If any
		// reply in the open thread has a matching TS, blow the
		// thread cache so renderThreadMessage runs again with the
		// now-cached image bytes. HasReply guards against churning
		// the thread cache on every messages-pane image arrival.
		if a.threadPanel.HasReply(m.TS) {
			a.threadPanel.InvalidateCache()
		}
		return nil, true

	case EmojiImageReadyMsg:
		debuglog.ImgFetch("recv: kind=emoji-ready url=%s pending=%v", m.URL, a.emojiInvalidatePending)
		// An emoji-image fetch landed. Naively each arrival would
		// trigger a full cache rebuild across every emoji-rendering
		// surface (messages, thread, picker). On a busy channel with
		// many cold-cache emoji this cascades into seconds of UI-
		// thread saturation — looks like a freeze. Debounce: schedule
		// one tick on the first arrival; absorb every subsequent
		// arrival into the pending batch and let them collapse to a
		// single invalidation when the tick fires.
		if a.emojiInvalidatePending {
			return nil, true
		}
		a.emojiInvalidatePending = true
		return tea.Tick(emojiInvalidateDebounce, func(time.Time) tea.Msg {
			return emojiInvalidateMsg{}
		}), true

	case emojiInvalidateMsg:
		_ = m
		// Debounce window closed. One wholesale invalidation across
		// every emoji-rendering surface; arrivals accumulated during
		// the window collapse to this. The URL argument is unused —
		// the surface handlers wipe their caches wholesale regardless
		// of URL in v1.
		a.emojiInvalidatePending = false
		for _, mp := range a.allWinModels() {
			mp.HandleEmojiImageReady("")
		}
		a.threadPanel.HandleEmojiImageReady("")
		a.reactionPicker.HandleEmojiImageReady("") // no-op in v1; future caching may use it
		a.activityView.HandleEmojiImageReady("")
		a.sidebar.InvalidateCache()
		// Autocomplete dropdowns have no cache; the no-op hooks on
		// a.compose.emojiPicker / a.threadCompose.emojiPicker keep
		// the surface symmetric. Listed here for the audit trail.
		return nil, true

	case messages.AvatarReadyMsg:
		// A lazy avatar fetch landed for m.UserID. Both the
		// messages pane and the thread panel cache avatar slots
		// in their render caches, so both must invalidate. The
		// handlers no-op when the userID isn't in their current
		// view, but coarse invalidation is cheap relative to the
		// cost of a missing avatar.
		for _, mp := range a.allWinModels() {
			mp.HandleAvatarReady(m.UserID)
		}
		a.threadPanel.HandleAvatarReady(m.UserID)
		a.activityView.HandleAvatarReady(m.UserID)
		a.threadsView.HandleAvatarReady(m.UserID)
		a.sidebar.HandleAvatarReady(m.UserID)
		a.workspaceRail.HandleLogoReady(m.UserID)
		return nil, true

	case imgrender.ImageFailedMsg:
		debuglog.ImgFetch("recv: kind=failed key=%s req_id=%d", m.Key, m.ReqID)
		// Image attachment fetch hit a permanent failure (all
		// auths exhausted, or some other terminal error). Clear
		// the in-flight bit so a future cache invalidation
		// doesn't keep retrying; don't trigger a re-render --
		// the placeholder is already on screen and we have no
		// new bytes to show.
		for _, mp := range a.allWinModels() {
			mp.HandleImageFailed(m.Key)
		}
		// Mirror the in-flight bookkeeping on the thread panel so
		// a permanently-failed image isn't re-attempted from the
		// thread.
		a.threadPanel.HandleImageFailed(m.Key)
		return nil, true
	}
	return nil, false
}

// reducePaste handles tea.PasteMsg. Extracted because the arm
// does three things: insert-mode gate, clipboard-image
// hit-test, and compose-textarea forward.
func reducePaste(a *App, m tea.PasteMsg) tea.Cmd {
	// Bracketed-paste from the terminal. First check the OS
	// clipboard for an image (terminals can't deliver image bytes
	// via bracketed paste -- only the text representation -- so
	// the image data is still sitting in the clipboard waiting
	// for us to read directly). Also test the bracketed text as a
	// file path. If neither matches, fall through to forwarding
	// the paste verbatim into the active compose's textarea.
	if a.mode == ModeSearch {
		// Paste into the `/` prompt: the prompt is single-line, so
		// flatten any newlines to spaces and append.
		txt := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(m.Content)
		a.searchInput += txt
		a.statusbar.SetSearch("/" + a.searchInput)
		return nil
	}
	if a.mode != ModeInsert {
		return nil
	}
	if a.clipboardAvailable {
		target := &a.compose
		if a.focusedPanel == PanelThread && a.threadVisible {
			target = &a.threadCompose
		}
		if consumed, cmd := a.tryAttachFromClipboard(target, m.Content); consumed {
			return cmd
		}
	}
	if a.focusedPanel == PanelThread && a.threadVisible {
		var cmd tea.Cmd
		a.threadCompose, cmd = a.threadCompose.Update(m)
		return cmd
	}
	var cmd tea.Cmd
	a.compose, cmd = a.compose.Update(m)
	return cmd
}
