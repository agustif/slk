// internal/ui/keys.go
package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up                  key.Binding
	Down                key.Binding
	Left                key.Binding
	Right               key.Binding
	Enter               key.Binding
	Escape              key.Binding
	InsertMode          key.Binding
	CommandMode         key.Binding
	SearchMode          key.Binding
	SearchNext          key.Binding
	SearchPrev          key.Binding
	WorkspaceSearch     key.Binding
	Tab                 key.Binding
	ShiftTab            key.Binding
	ToggleSidebar       key.Binding
	SidebarGrow         key.Binding
	SidebarShrink       key.Binding
	ToggleThread        key.Binding
	FuzzyFinder         key.Binding
	FuzzyFinderAlt      key.Binding
	Top                 key.Binding
	Bottom              key.Binding
	PageUp              key.Binding
	PageDown            key.Binding
	HalfPageUp          key.Binding
	HalfPageDown        key.Binding
	Quit                key.Binding
	QuitConfirm         key.Binding
	CloseThreadView     key.Binding
	Reaction            key.Binding
	ReactionNav         key.Binding
	Edit                key.Binding
	Delete              key.Binding
	Yank                key.Binding
	CopyPermalink       key.Binding
	OpenPreview         key.Binding
	OpenLink            key.Binding
	DownloadFile        key.Binding
	MarkUnread          key.Binding
	NextUnread          key.Binding
	PrevUnread          key.Binding
	WorkspaceFinder     key.Binding
	LeaveChannel        key.Binding
	NewMessage          key.Binding
	ThemeSwitcher       key.Binding
	ThemeSwitcherGlobal key.Binding
	PresenceMenu        key.Binding
	// Insert-mode schedule overlay. Ctrl+Enter is intentionally unbound
	// (reserved for also-send on another branch).
	ScheduleMessage     key.Binding
	UserProfile         key.Binding
	ToggleSection       key.Binding
	ToggleStar          key.Binding
	NavBack             key.Binding
	NavForward          key.Binding
	Help                key.Binding
	ChannelMembers      key.Binding
	SaveThread          key.Binding
	ListReactions       key.Binding
	WindowPrefix        key.Binding
	WinSplit            key.Binding
	WinVSplit           key.Binding
	WinNavigate         key.Binding
	WinCycle            key.Binding
	WinClose            key.Binding
	WinOnly             key.Binding
	ActivityFilter      key.Binding
	ActivityFilterPrev  key.Binding
	ActivitySort        key.Binding
	ActivityUnreadOnly  key.Binding
	ToggleMute          key.Binding
	MoveSection         key.Binding
	CreateSection       key.Binding
	// ContextMenu opens the message-actions overlay. Right-click on a
	// message does the same when the terminal reports MouseRight;
	// some terminals steal right-click for paste/their own menu, so
	// `x` is the keyboard path. Do not bind `g` (gg prefix) or `c`
	// (permalink with Y).
	ContextMenu         key.Binding
	Pin                 key.Binding
	FollowThread        key.Binding
	BroadcastSend       key.Binding
	SaveForLater        key.Binding
	RemindMessage       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:              key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "up")),
		Down:            key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "down")),
		Left:            key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/left", "left")),
		Right:           key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/right", "right")),
		Enter:           key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open/confirm")),
		Escape:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		InsertMode:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "insert mode")),
		CommandMode:     key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command mode")),
		SearchMode:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		SearchNext:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		SearchPrev:      key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		WorkspaceSearch: key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "search workspace (messages/files)")),
		Tab:             key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		ShiftTab:        key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
		ToggleSidebar:   key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "toggle sidebar")),
		SidebarGrow:     key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "widen sidebar")),
		SidebarShrink:   key.NewBinding(key.WithKeys("["), key.WithHelp("[", "narrow sidebar")),
		ToggleThread:    key.NewBinding(key.WithKeys("ctrl+]"), key.WithHelp("ctrl+]", "toggle thread")),
		FuzzyFinder:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "switch channel")),
		FuzzyFinderAlt:  key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "switch channel")),
		Top:             key.NewBinding(key.WithKeys("g"), key.WithHelp("gg", "top")),
		Bottom:          key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		PageUp:          key.NewBinding(key.WithKeys("pgup"), key.WithHelp("PgUp", "page up")),
		PageDown:        key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("PgDn", "page down")),
		HalfPageUp:      key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "half page up")),
		HalfPageDown:    key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "half page down")),
		Quit:            key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit (confirm)")),
		QuitConfirm:     key.NewBinding(key.WithKeys("Q"), key.WithHelp("Q", "quit (confirm)")),
		CloseThreadView: key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "close thread view")),
		Reaction:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "add reaction")),
		ReactionNav:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "navigate reactions")),
		Edit:            key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "edit message")),
		Delete:          key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete message")),
		Yank:            key.NewBinding(key.WithKeys("y"), key.WithHelp("yy", "yank message")),
		CopyPermalink:   key.NewBinding(key.WithKeys("Y", "C"), key.WithHelp("Y/C", "copy permalink")),
		OpenPreview:     key.NewBinding(key.WithKeys("O", "v"), key.WithHelp("O/v", "open image preview")),
		OpenLink:        key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open link in message")),
		DownloadFile:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download file in message")),
		MarkUnread:      key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "mark unread")),
		NextUnread:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "next unread channel")),
		PrevUnread:      key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "prev unread channel")),
		// Keyless: ctrl+w is reserved as the window-command prefix
		// (window-management design §4). The keyless binding never
		// matches but keeps the help-overlay entry pointing at :ws
		// (1-9 also switch workspaces directly).
		WorkspaceFinder:     key.NewBinding(key.WithHelp(":ws", "switch workspace")),
		LeaveChannel:        key.NewBinding(key.WithHelp(":leave", "leave channel")),
		NewMessage:          key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new message")),
		ThemeSwitcher:       key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "switch theme (per workspace)")),
		ThemeSwitcherGlobal: key.NewBinding(key.WithKeys("ctrl+shift+y"), key.WithHelp("ctrl+shift+y", "set default theme")),
		PresenceMenu:        key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "set status")),
		ScheduleMessage:     key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("ctrl+g / :schedule", "schedule message")),
		UserProfile:         key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "user profile")),
		ToggleSection:       key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle section")),
		ToggleStar:          key.NewBinding(key.WithKeys("*"), key.WithHelp("*", "star/unstar channel")),
		NavBack:             key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "navigate back")),
		NavForward:          key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "navigate forward")),
		Help:                key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "show keybindings")),
		ChannelMembers:      key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "channel members")),
		SaveThread:          key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "save thread")),
		ListReactions:       key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "list reactions")),
		// Window commands (design §4). WindowPrefix is the only real
		// binding; the Win* entries are keyless help-only bindings
		// (same trick as WorkspaceFinder above) — actual dispatch of
		// the chord key happens in handleWindowChord.
		WindowPrefix:       key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "window commands")),
		WinSplit:           key.NewBinding(key.WithHelp("ctrl+w s / :sp", "split window")),
		WinVSplit:          key.NewBinding(key.WithHelp("ctrl+w v / :vsp", "vertical split window")),
		WinNavigate:        key.NewBinding(key.WithHelp("ctrl+w h/j/k/l", "focus window in direction")),
		WinCycle:           key.NewBinding(key.WithHelp("ctrl+w w", "cycle windows")),
		WinClose:           key.NewBinding(key.WithHelp("ctrl+w q / :q", "close window")),
		WinOnly:            key.NewBinding(key.WithHelp("ctrl+w o / :only", "close other windows")),
		ActivityFilter:     key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "next activity tab")),
		ActivityFilterPrev: key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "prev activity tab")),
		ActivitySort:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "cycle activity sort")),
		ActivityUnreadOnly: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "toggle activity unread-only")),
		ToggleMute:         key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mute channel")),
		MoveSection:        key.NewBinding(key.WithHelp(":move", "move channel to section")),
		CreateSection:      key.NewBinding(key.WithHelp(":section", "create sidebar section")),
		ContextMenu:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "message actions")),
		Pin:                key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pin/unpin message")),
		FollowThread:       key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "follow/unfollow thread")),
		BroadcastSend:      key.NewBinding(key.WithKeys("ctrl+enter"), key.WithHelp("ctrl+enter", "also send reply to channel")),
		SaveForLater:       key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "save for later")),
		RemindMessage:      key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "remind me about this")),
	}
}
