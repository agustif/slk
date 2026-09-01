// internal/ui/reducer_mouse.go
//
// Mouse-router reducer for App.Update (Phase 4m).
//
// Owns the two remaining mouse Update arms that act as multi-panel
// routers:
//
//	tea.MouseWheelMsg  - viewport scroll for the panel under the
//	                     cursor (sidebar, messages pane / threads
//	                     view, thread panel). Decoupled from j/k
//	                     selection. Triggers maybeFetchOlderHistory
//	                     on the messages pane when the viewport
//	                     hits the top.
//	tea.MouseClickMsg  - panel-router: workspace rail (switch
//	                     workspace), sidebar (channel select /
//	                     threads view), messages pane (threads
//	                     click / reaction hit-test / image hit-test
//	                     / drag begin), thread panel (reaction
//	                     hit-test / drag begin). Right-click on a
//	                     message/thread row opens the actions menu
//	                     without starting a drag; Activity cards open
//	                     the reaction picker.
//
// Free reducer (not controller-absorbed) because both arms route
// across multiple sub-models: the sidebar, messagepane, threadPanel,
// workspaceRail, threadsView, drag controller, workspaceSwitcher
// service, and reaction toggle helper. No single existing
// controller owns this cross-section.
//
// MouseMotionMsg, MouseReleaseMsg, and autoScrollTickMsg moved to
// dragSelection.Handle in Phase 4c because those three are
// drag-FSM-specific. Click+Wheel are deliberately separated -- they
// orchestrate panel routing and other concerns beyond drag.
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ui/activityview"
	"github.com/agustif/slk/internal/ui/draftsview"
	"github.com/agustif/slk/internal/ui/laterview"
	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/starredview"
	"github.com/agustif/slk/internal/ui/unreadsview"
)

// sidebarHeaderDoubleClick is the window in which a second left-click
// on the same section header collapses or expands it. Terminals do not
// report click-count, so this is two MouseClickMsg on the same header.
const sidebarHeaderDoubleClick = 500 * time.Millisecond

// sidebarHeaderClick remembers the last left-click on a sidebar
// section header so a second click can toggle collapse.
type sidebarHeaderClick struct {
	name string
	at   time.Time
}

// registerSidebarHeaderClick records a click on the named header.
// Returns true when this click completes a double-click (same name
// within sidebarHeaderDoubleClick). A completed pair is consumed so a
// third click starts a new sequence instead of immediately re-toggling.
func (a *App) registerSidebarHeaderClick(name string) bool {
	now := time.Now()
	if name == "" {
		a.sidebarHeaderClick = sidebarHeaderClick{}
		return false
	}
	prev := a.sidebarHeaderClick
	a.sidebarHeaderClick = sidebarHeaderClick{name: name, at: now}
	if prev.name == name && !prev.at.IsZero() && now.Sub(prev.at) <= sidebarHeaderDoubleClick {
		a.sidebarHeaderClick = sidebarHeaderClick{}
		return true
	}
	return false
}

var reduceMouse reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case tea.MouseWheelMsg:
		return reduceMouseWheel(a, m), true

	case tea.MouseClickMsg:
		return reduceMouseClick(a, m), true
	}
	return nil, false
}

// reduceMouseWheel handles tea.MouseWheelMsg. Extracted to keep the
// reduceMouse dispatch switch readable.
//
// Wheel notches scroll the viewport of the panel under the cursor,
// regardless of which panel currently has keyboard focus. In the
// messages pane the selected-message cursor follows the scroll,
// clamping to the nearest visible message (see viewInternal's
// cursor-clamp step). The sidebar/threads panels keep their own
// scroll/selection coupling.
func reduceMouseWheel(a *App, m tea.MouseWheelMsg) tea.Cmd {
	if a.bootstrap.IsLoading() {
		return nil
	}
	// Drain any pending held-key scroll so the wheel scroll applies on
	// top of the up-to-date viewport/selection (see coalesceContentScroll).
	a.flushScrollCoalesce()
	var up bool
	switch m.Button {
	case tea.MouseWheelUp:
		up = true
	case tea.MouseWheelDown:
		up = false
	default:
		return nil
	}
	// When a modal overlay is open it owns the screen, so the wheel
	// must scroll the items inside the modal rather than the panel
	// under the cursor on the main tab behind it. We replay each
	// wheel notch as the modal's own up/down navigation through
	// dispatchModeKey -- the exact path the arrow keys take -- so
	// every modal scrolls correctly without a bespoke per-modal
	// wheel API, and the wheel never leaks through to the main tab.
	if a.mode.IsModalOverlay() {
		return scrollActiveModal(a, up)
	}
	// Lines moved per wheel notch -- configured via
	// [appearance].mouse_wheel_lines (default 3, matches typical
	// terminal behavior). Single-row panes (sidebar) still feel
	// fine because real-world workspace lists are short and the
	// snap-back on the next j/k restores the previously-selected
	// channel.
	wheelLinesPerNotch := a.mouseWheelLines
	if wheelLinesPerNotch < 1 {
		wheelLinesPerNotch = 1
	}
	x := m.X
	switch {
	case x < a.layout.RailWidth():
		// Workspace rail: no scroll here.
		return nil
	case a.sidebarVisible && x < a.layout.SidebarEnd():
		if up {
			a.sidebar.ScrollUp(wheelLinesPerNotch)
		} else {
			a.sidebar.ScrollDown(wheelLinesPerNotch)
		}
		return nil
	case x < a.layout.MsgEnd():
		if a.view == ViewThreads {
			if up {
				a.threadsView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.threadsView.ScrollDown(wheelLinesPerNotch)
			}
			// No openSelectedThreadCmd here: pure viewport scroll
			// does not change the highlighted thread card.
			return nil
		}
		if a.view == ViewActivity {
			if up {
				a.activityView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.activityView.ScrollDown(wheelLinesPerNotch)
			}
			return nil
		}
		if a.view == ViewLater {
			if up {
				a.laterView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.laterView.ScrollDown(wheelLinesPerNotch)
			}
			return nil
		}
		if a.view == ViewDrafts {
			if up {
				a.draftsView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.draftsView.ScrollDown(wheelLinesPerNotch)
			}
			return nil
		}
		if a.view == ViewUnreads {
			if up {
				a.unreadsView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.unreadsView.ScrollDown(wheelLinesPerNotch)
			}
			return nil
		}
		if a.view == ViewStarred {
			if up {
				a.starredView.ScrollUp(wheelLinesPerNotch)
			} else {
				a.starredView.ScrollDown(wheelLinesPerNotch)
			}
			return nil
		}
		if up {
			a.messagepane.ScrollUp(wheelLinesPerNotch)
			// Backfill older history when the viewport hits the
			// top (selection-based AtTop check moved to handleUp).
			return a.maybeFetchOlderHistory(a.messagepane.ViewportAtTop())
		}
		a.messagepane.ScrollDown(wheelLinesPerNotch)
		return nil
	case a.threadVisible && x < a.layout.ThreadEnd():
		if up {
			a.threadPanel.ScrollUp(wheelLinesPerNotch)
		} else {
			a.threadPanel.ScrollDown(wheelLinesPerNotch)
		}
		return nil
	}
	return nil
}

// scrollActiveModal translates one wheel notch into the active
// modal's up/down navigation, replaying it mouseWheelLines times
// through dispatchModeKey (the same dispatch the arrow keys use).
// Returns the batched cmds the modal handlers produced (usually
// nil for plain navigation). The caller guarantees a modal mode
// is active.
func scrollActiveModal(a *App, up bool) tea.Cmd {
	code := tea.KeyDown
	if up {
		code = tea.KeyUp
	}
	steps := a.mouseWheelLines
	if steps < 1 {
		steps = 1
	}
	var cmds []tea.Cmd
	for i := 0; i < steps; i++ {
		if cmd := dispatchModeKey(a, tea.KeyPressMsg{Code: code}); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// reduceMouseClick handles tea.MouseClickMsg. Extracted from the
// reduceMouse dispatch switch because the arm is ~120 lines (four
// panel-router branches: workspace rail, sidebar, messages pane,
// thread panel; each with its own hit-test sequence).
func reduceMouseClick(a *App, m tea.MouseClickMsg) tea.Cmd {
	if a.bootstrap.IsLoading() {
		return nil
	}
	// When a modal overlay owns the screen, route the click to the
	// modal (select a row / dismiss on outside-click) instead of the
	// main-tab panels behind it. See reducer_modal_click.go. Right-click
	// on an open context menu uses the same path (row = activate,
	// outside = dismiss).
	if a.mode.IsModalOverlay() {
		if m.Button == tea.MouseLeft || (m.Button == tea.MouseRight && a.mode == ModeContextMenu) {
			return reduceModalClick(a, m)
		}
		return nil
	}
	if m.Button == tea.MouseRight {
		if a.view == ViewActivity {
			return reduceActivityRightClick(a, m)
		}
		return reduceMouseRightClick(a, m)
	}
	if m.Button != tea.MouseLeft {
		return nil
	}
	// A click sets the selection absolutely (ClickAt below), so any
	// pending held-key scroll moves would be immediately overwritten --
	// discard them rather than apply (and let any in-flight flush tick
	// no-op). Prevents a post-click selection jump when the tick fires.
	a.scrollPending = 0
	x := m.X
	statusHeight := 1
	if m.Y >= a.height-statusHeight {
		return nil // click on status bar, ignore
	}

	// Determine which panel was clicked.
	switch {
	case x < a.layout.RailWidth():
		// Workspace rail: clicking a workspace tile switches to
		// that workspace (same code path as the 1-9 keybinds and
		// the workspace finder). The rail has no border above, so
		// the panel-local y is just m.Y.
		item, ok := a.workspaceRail.ClickAt(m.Y)
		if !ok {
			return nil
		}
		if a.workspaceSwitcher == nil || item.ID == a.workspaceRail.SelectedID() {
			return nil
		}
		switcher := a.workspaceSwitcher
		teamID := item.ID
		return func() tea.Msg {
			return switcher(teamID)
		}

	case a.sidebarVisible && x < a.layout.SidebarEnd():
		a.focusedPanel = PanelSidebar
		sidebarY := m.Y - 1 // account for top border
		if sidebarY < 0 {
			return nil
		}
		if item, ok := a.sidebar.ClickAt(sidebarY); ok {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg {
				return ChannelSelectedMsg{ID: item.ID, Name: item.Name, Type: item.Type, Topic: item.Topic}
			}
		}
		if !a.sidebar.HitNavAt(sidebarY) {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return nil
		}
		// ClickAt returns ok=false for the synthetic Threads row;
		// if the click landed there (sidebar updates its own
		// selection state), activate the threads view.
		if a.sidebar.IsActivitySelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return ActivityViewActivatedMsg{} }
		}
		if a.sidebar.IsLaterSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return LaterViewActivatedMsg{} }
		}
		if a.sidebar.IsThreadsSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return ThreadsViewActivatedMsg{} }
		}
		if a.sidebar.IsDMsSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return DMsViewActivatedMsg{} }
		}
		if a.sidebar.IsDraftsSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return DraftsViewActivatedMsg{} }
		}
		if a.sidebar.IsUnreadsSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return UnreadsViewActivatedMsg{} }
		}
		if a.sidebar.IsStarredSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			return func() tea.Msg { return StarredViewActivatedMsg{} }
		}
		if a.sidebar.IsHomeSelected() {
			a.sidebarHeaderClick = sidebarHeaderClick{}
			a.setInboxView(ViewChannels)
			a.sidebar.SelectDMsRow()
			return nil
		}
		if name, ok := a.sidebar.IsSectionHeaderSelected(); ok {
			if a.registerSidebarHeaderClick(name) {
				a.sidebar.ToggleCollapseSelected()
			}
			return nil
		}
		a.sidebarHeaderClick = sidebarHeaderClick{}
		return nil

	case x < a.layout.MsgEnd():
		// Interim Phase 2 guard: per-window mouse routing lands in
		// Phase 4 (window-management design §6). With splits active,
		// PanelAt still maps the entire messages region to the single
		// live pane, so a click on a placeholder window would start a
		// drag selection in the live channel at bogus coordinates.
		// Swallow clicks (and therefore drags — they can only begin
		// here); mouse-wheel scrolling of the live pane stays enabled.
		if a.wins.Len() > 1 {
			return nil
		}
		a.focusedPanel = PanelMessages
		// In the threads-list view, the messages-pane region
		// renders threadsView, not the channel messages. Route
		// the click through threadsView.ClickAt so the cursor
		// follows the click, then open the highlighted thread
		// (mirrors the mouse-wheel branch above and the j/k/Enter
		// paths). The messagepane drag-selection / reaction /
		// image-hit-test code below operates on the (hidden)
		// channel pane and must not run here.
		if a.view == ViewThreads {
			panel, _, py, ok := a.panelAt(m.X, m.Y)
			if ok && panel == PanelMessages && py >= 0 && a.threadsView.ClickAt(py) {
				return a.openSelectedThreadCmd(false)
			}
			return nil
		}
		if a.view == ViewActivity {
			panel, px, py, ok := a.panelAt(m.X, m.Y)
			if !ok || panel != PanelMessages || py < 0 {
				return nil
			}
			switch a.activityView.ClickAt(py, px) {
			case activityview.ClickReaction:
				it, ok := a.activityView.SelectedItem()
				if !ok {
					return nil
				}
				return a.toggleActivityReaction(it)
			case activityview.ClickItem:
				return a.openSelectedActivityCmd()
			case activityview.ClickControls:
				return a.fetchActivityCmd()
			}
			return nil
		}
		if a.view == ViewLater {
			panel, px, py, ok := a.panelAt(m.X, m.Y)
			if !ok || panel != PanelMessages || py < 0 {
				return nil
			}
			switch a.laterView.ClickAt(py, px) {
			case laterview.ClickItem:
				return a.openSelectedLaterCmd()
			case laterview.ClickTab:
				return a.fetchLaterCmd()
			case laterview.ClickLoadMore:
				return a.fetchLaterMoreCmd()
			}
			return nil
		}
		if a.view == ViewDrafts {
			panel, px, py, ok := a.panelAt(m.X, m.Y)
			if !ok || panel != PanelMessages || py < 0 {
				return nil
			}
			switch a.draftsView.ClickAt(py, px) {
			case draftsview.ClickItem:
				return a.openSelectedDraftCmd()
			case draftsview.ClickTab:
				return a.fetchDraftsCmd()
			case draftsview.ClickLoadMore:
				return a.fetchDraftsMoreCmd()
			}
			return nil
		}
		if a.view == ViewUnreads {
			panel, px, py, ok := a.panelAt(m.X, m.Y)
			if !ok || panel != PanelMessages || py < 0 {
				return nil
			}
			switch a.unreadsView.ClickAt(py, px) {
			case unreadsview.ClickMessage:
				return a.openSelectedUnreadCmd()
			case unreadsview.ClickHeader:
				return a.markSelectedUnreadCmd()
			}
			return nil
		}
		if a.view == ViewStarred {
			panel, px, py, ok := a.panelAt(m.X, m.Y)
			if !ok || panel != PanelMessages || py < 0 {
				return nil
			}
			if a.starredView.ClickAt(py, px) == starredview.ClickItem {
				return a.openSelectedStarredCmd()
			}
			return nil
		}
		panel, px, py, ok := a.panelAt(m.X, m.Y)
		if !ok || panel != PanelMessages || py < 0 {
			return nil
		}
		if py < a.messagepane.ChromeHeight() {
			if hit, hitOK := a.messagepane.HitTestChrome(py, px); hitOK {
				return a.handleHeaderChromeHit(hit)
			}
			return nil
		}
		// Hit-test reactions and inline images first: a click
		// that lands inside a pill toggles the user's reaction;
		// a click inside an image footprint opens the full-screen
		// preview. Either takes precedence over the
		// drag-to-copy selection and the click-to-select-message
		// behavior on this panel. lastHits / lastReactionHits are
		// keyed in pane-local content coordinates (chrome already
		// stripped), so we subtract chromeHeight here, mirroring
		// the convention used by ClickAt / BeginSelectionAt.
		contentY := py - a.messagepane.ChromeHeight()
		if contentY >= 0 {
			if hitMsgIdx, emojiName, hit := a.messagepane.HitTestReaction(contentY, px); hit && emojiName != "" {
				msgs := a.messagepane.Messages()
				if hitMsgIdx >= 0 && hitMsgIdx < len(msgs) {
					return a.toggleReactionOnMessageItem(a.activeChannelID, msgs[hitMsgIdx], emojiName)
				}
			}
			if hitMsgIdx, attIdx, fileID, hit := a.messagepane.HitTest(contentY, px); hit && fileID != "" {
				msgs := a.messagepane.Messages()
				if hitMsgIdx >= 0 && hitMsgIdx < len(msgs) {
					ch := a.activeChannelID
					messageTS := msgs[hitMsgIdx].TS
					idx := attIdx
					return func() tea.Msg {
						return messages.OpenImagePreviewMsg{
							Channel: ch,
							TS:      messageTS,
							AttIdx:  idx,
						}
					}
				}
			}
		}
		a.drag.Begin(PanelMessages, px, py)
		a.messagepane.BeginSelectionAt(py, px)
		// Remember whether this press actually landed on a message
		// row -- MouseReleaseMsg uses this to decide whether a
		// plain click (no drag) should open the message's thread
		// (mirrors pressing Enter on the selected message).
		a.drag.SetClickedMessage(a.messagepane.ClickAt(py))
		return nil

	case a.threadVisible && x < a.layout.ThreadEnd():
		a.focusedPanel = PanelThread
		panel, px, py, ok := a.panelAt(m.X, m.Y)
		if !ok || panel != PanelThread || py < 0 {
			return nil
		}
		// Hit-test reactions first on the thread pane too.
		// HitTestReaction's rows are pane-local (already inclusive
		// of the thread chromeHeight), matching the frame returned
		// by panelAt.
		if hitReplyIdx, emojiName, hit := a.threadPanel.HitTestReaction(py, px); hit && emojiName != "" {
			replies := a.threadPanel.Replies()
			if hitReplyIdx >= 0 && hitReplyIdx < len(replies) {
				return a.toggleReactionOnMessageItem(a.threadPanel.ChannelID(), replies[hitReplyIdx], emojiName)
			}
		}
		if hitReplyIdx, attIdx, fileID, hit := a.threadPanel.HitTestImage(py, px); hit && fileID != "" {
			ch := a.threadPanel.ChannelID()
			var ts string
			if hitReplyIdx < 0 {
				ts = a.threadPanel.ParentMsg().TS
			} else {
				replies := a.threadPanel.Replies()
				if hitReplyIdx >= 0 && hitReplyIdx < len(replies) {
					ts = replies[hitReplyIdx].TS
				}
			}
			if ts != "" {
				idx := attIdx
				return func() tea.Msg {
					return messages.OpenImagePreviewMsg{Channel: ch, TS: ts, AttIdx: idx}
				}
			}
		}
		a.drag.Begin(PanelThread, px, py)
		a.threadPanel.BeginSelectionAt(py, px)
		a.threadPanel.ClickAt(py)
		return nil
	}
	return nil
}

func reduceActivityRightClick(a *App, m tea.MouseClickMsg) tea.Cmd {
	if a.mode.IsModalOverlay() || a.view != ViewActivity {
		return nil
	}
	panel, px, py, ok := a.panelAt(m.X, m.Y)
	if !ok || panel != PanelMessages || py < 0 {
		return nil
	}
	kind := a.activityView.ClickAtCard(py, px)
	if kind == activityview.ClickNone {
		return nil
	}
	return a.openPickerFromActivity()
}

// reduceMouseRightClick opens the message-actions overlay on a message
// in the messages pane or thread pane. Left-click behavior (pills,
// images, drag) is unchanged. Some terminals never report MouseRight;
// the `x` keybinding is the keyboard path. Activity right-click is
// handled first (reaction picker).
func reduceMouseRightClick(a *App, m tea.MouseClickMsg) tea.Cmd {
	a.scrollPending = 0
	if m.Y >= a.height-1 {
		return nil
	}
	x := m.X
	switch {
	case x < a.layout.RailWidth():
		return nil
	case a.sidebarVisible && x < a.layout.SidebarEnd():
		return nil
	case x < a.layout.MsgEnd():
		if a.wins.Len() > 1 {
			return nil
		}
		if !a.inChannelView() {
			return nil
		}
		a.focusedPanel = PanelMessages
		panel, _, py, ok := a.panelAt(m.X, m.Y)
		if !ok || panel != PanelMessages || py < 0 {
			return nil
		}
		if !a.messagepane.ClickAt(py) {
			return nil
		}
		return a.openMessageContextMenu(true, m.X, m.Y)
	case a.threadVisible && x < a.layout.ThreadEnd():
		a.focusedPanel = PanelThread
		panel, _, py, ok := a.panelAt(m.X, m.Y)
		if !ok || panel != PanelThread || py < 0 {
			return nil
		}
		if !a.threadPanel.ClickAt(py) {
			return nil
		}
		return a.openMessageContextMenu(true, m.X, m.Y)
	}
	return nil
}
