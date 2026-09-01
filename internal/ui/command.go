// internal/ui/command.go
//
// The vi-style ":" command registry (window-management design §5).
//
// executeCommand parses a command line (without the leading ':')
// and dispatches through the commands map — the designated
// extension point for future :commands. Later phases of the
// window-management plan register sp / vsp / q / only here.
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/agustif/slk/internal/ids"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/wintree"
)

// commandFunc executes a named :command. args holds the
// whitespace-separated tokens after the command name (unused by
// v1 commands, reserved for e.g. ":sp #channel").
type commandFunc func(a *App, args []string) tea.Cmd

// commands maps a command name to its handler. Names are matched
// exactly (no prefix matching); aliases get their own entries.
var commands = map[string]commandFunc{
	"ws":             cmdWorkspaceFinder,
	"sp":             cmdSplit,
	"vsp":            cmdVSplit,
	"q":              cmdCloseWindow,
	"only":           cmdOnlyWindow,
	"on":             cmdOnlyWindow,
	"leave":          cmdLeave,
	"create":         cmdCreate,
	"invite":         cmdInvite,
	"kick":           cmdKick,
	"manager":        cmdManager,
	"unmanager":      cmdUnmanager,
	"notify":         cmdNotify,
	"schedule":       cmdSchedule,
	"scheduled":      cmdScheduledList,
	"move":           cmdMove,
	"section":        cmdSection,
	"rename":         cmdSectionRename,
	"section-delete": cmdSectionDelete,
	"section-up":     cmdSectionUp,
	"section-down":   cmdSectionDown,
	"remind":         cmdRemind,
	"reminders":      cmdRemindersList,
	"pins":           cmdPinsList,
	"date":           cmdDate,
	"jump":           cmdDate,
	"share":          cmdShare,
}

// cmdSplit / cmdVSplit create a stacked / side-by-side split of the
// focused window (window-management design §5).
func cmdSplit(a *App, _ []string) tea.Cmd  { return a.splitWindow(wintree.SplitStacked) }
func cmdVSplit(a *App, _ []string) tea.Cmd { return a.splitWindow(wintree.SplitSideBySide) }

// cmdCloseWindow closes the focused window (never quits the app).
func cmdCloseWindow(a *App, _ []string) tea.Cmd { return a.closeWindow() }

// cmdOnlyWindow closes all other windows.
func cmdOnlyWindow(a *App, _ []string) tea.Cmd {
	a.onlyWindow()
	return nil
}

// cmdWorkspaceFinder opens the workspace finder overlay —
// the :command replacement for the finder's old ctrl+w binding.
func cmdWorkspaceFinder(a *App, _ []string) tea.Cmd {
	a.workspaceFinder.Open()
	a.SetMode(ModeWorkspaceFinder)
	return nil
}

func cmdLeave(a *App, _ []string) tea.Cmd {
	return a.beginLeaveChannel()
}

func cmdCreate(a *App, args []string) tea.Cmd {
	private := false
	if len(args) > 0 && strings.EqualFold(args[0], "private") {
		private = true
		args = args[1:]
	}
	name := strings.TrimSpace(strings.ToLower(strings.Join(args, "-")))
	if name == "" {
		return toastWithClear(a, "Usage: :create [private] <name>", 2*time.Second)
	}
	channels := a.channels
	return func() tea.Msg { return channels.Create(name, private) }
}

func cmdInvite(a *App, args []string) tea.Cmd {
	if len(args) == 0 {
		return toastWithClear(a, "Usage: :invite email|U… […]", 2*time.Second)
	}
	id, _, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel to invite to", 2*time.Second)
	}
	if isDirectMessage(chType) {
		return toastWithClear(a, "Invite from a channel, not a DM", 2*time.Second)
	}
	var emails, users []string
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "U") && !strings.Contains(arg, "@"):
			users = append(users, arg)
		case strings.Contains(arg, "@"):
			emails = append(emails, arg)
		default:
			return toastWithClear(a, "Usage: :invite email|U… […]", 2*time.Second)
		}
	}
	if len(emails) > 0 && len(users) > 0 {
		return toastWithClear(a, "Invite emails and user IDs separately", 2*time.Second)
	}
	channels := a.channels
	if len(users) > 0 {
		return func() tea.Msg { return channels.InviteUsers(users, ids.ChannelID(id)) }
	}
	return func() tea.Msg { return channels.InviteEmails(emails, ids.ChannelID(id)) }
}

func cmdKick(a *App, args []string) tea.Cmd {
	if len(args) != 1 || !strings.HasPrefix(args[0], "U") {
		return toastWithClear(a, "Usage: :kick U…", 2*time.Second)
	}
	id, name, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel", 2*time.Second)
	}
	if isDirectMessage(chType) {
		return toastWithClear(a, "Kick from a channel, not a DM", 2*time.Second)
	}
	userID := args[0]
	a.confirmPrompt.Open(
		"Remove "+userID+" from #"+name+"?",
		"They can be re-invited with :invite.",
		func() tea.Msg {
			return KickUserMsg{ChannelID: id, Channel: name, UserID: userID}
		},
	)
	a.SetMode(ModeConfirm)
	return nil
}

func cmdManager(a *App, args []string) tea.Cmd {
	if len(args) == 0 {
		return toastWithClear(a, "Usage: :manager U… [U…]", 2*time.Second)
	}
	var users []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "U") || strings.Contains(arg, "@") {
			return toastWithClear(a, "Usage: :manager U… [U…]", 2*time.Second)
		}
		users = append(users, arg)
	}
	id, name, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel", 2*time.Second)
	}
	if isDirectMessage(chType) {
		return toastWithClear(a, "Channel Manager is for channels, not DMs", 2*time.Second)
	}
	who := users[0]
	if len(users) > 1 {
		who = fmt.Sprintf("%d people", len(users))
	}
	a.confirmPrompt.Open(
		"Make "+who+" a Channel Manager of #"+name+"?",
		"Uses admin.roles.addMembers role_id=Rl0A.",
		func() tea.Msg {
			return AddChannelManagersMsg{ChannelID: id, Channel: name, UserIDs: users}
		},
	)
	a.SetMode(ModeConfirm)
	return nil
}

func cmdUnmanager(a *App, args []string) tea.Cmd {
	if len(args) < 1 {
		return toastWithClear(a, "Usage: :unmanager U… [U…]", 2*time.Second)
	}
	var users []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "U") || strings.Contains(arg, "@") {
			return toastWithClear(a, "Usage: :unmanager U… [U…]", 2*time.Second)
		}
		users = append(users, arg)
	}
	id, name, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel", 2*time.Second)
	}
	if isDirectMessage(chType) {
		return toastWithClear(a, "Channel Manager is for channels, not DMs", 2*time.Second)
	}
	who := users[0]
	if len(users) > 1 {
		who = fmt.Sprintf("%d people", len(users))
	}
	a.confirmPrompt.Open(
		"Remove "+who+" as Channel Manager of #"+name+"?",
		"Uses admin.roles.removeMembers role_id=Rl0A.",
		func() tea.Msg {
			return RemoveChannelManagersMsg{ChannelID: id, Channel: name, UserIDs: users}
		},
	)
	a.SetMode(ModeConfirm)
	return nil
}

func cmdNotify(a *App, args []string) tea.Cmd {
	if len(args) != 1 {
		return toastWithClear(a, "Usage: :notify all|mentions", 2*time.Second)
	}
	level, ok := parseNotifyLevel(args[0])
	if !ok {
		return toastWithClear(a, "Usage: :notify all|mentions", 2*time.Second)
	}
	id, _, _, found := a.activeChannelMeta()
	if !found {
		return toastWithClear(a, "No channel", 2*time.Second)
	}
	channels := a.channels
	return func() tea.Msg { return channels.SetNotifyLevel(ids.ChannelID(id), level) }
}

func parseNotifyLevel(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "all", "everything":
		return slackclient.NotifyEverything, true
	case "mentions", "mentions_dms":
		return slackclient.NotifyMentions, true
	}
	return "", false
}

// beginLeaveChannel opens the leave-channel confirm overlay, or the
// close-DM confirm for 1:1 / group / app DMs.
func (a *App) beginLeaveChannel() tea.Cmd {
	id, name, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel to leave", 2*time.Second)
	}
	if isDirectMessage(chType) {
		a.confirmPrompt.Open(
			"Close this conversation?",
			name+" leaves Home and stays in Direct Messages.",
			func() tea.Msg {
				return CloseChannelMsg{ID: id, Name: name}
			},
		)
		a.SetMode(ModeConfirm)
		return nil
	}

	title := "Leave #" + name + "?"
	a.confirmPrompt.Open(
		title,
		"You can rejoin from the channel finder.",
		func() tea.Msg {
			return LeaveChannelMsg{ID: id, Name: name}
		},
	)
	a.SetMode(ModeConfirm)
	return nil
}

func cmdScheduledList(a *App, _ []string) tea.Cmd {
	return func() tea.Msg {
		return a.messageSvc.ListScheduled()
	}
}

func cmdRemindersList(a *App, _ []string) tea.Cmd {
	return func() tea.Msg {
		return a.messageSvc.ListReminders()
	}
}

func cmdPinsList(a *App, _ []string) tea.Cmd {
	return a.openPinsList()
}

func cmdShare(a *App, _ []string) tea.Cmd {
	return a.openSharePicker()
}

// activeChannelMeta returns the currently viewed channel. ok is false
// in Threads/Activity or when no channel is selected.
func (a *App) activeChannelMeta() (id, name, chType string, ok bool) {
	if !a.inChannelView() || a.activeChannelID == "" {
		return "", "", "", false
	}
	id = a.activeChannelID
	for _, it := range a.sidebar.Items() {
		if it.ID == id {
			return it.ID, it.Name, it.Type, true
		}
	}
	if ch, found := a.wins.Channel(a.focusedWin); found && ch.ID == id && ch.ID != "" {
		return ch.ID, ch.Name, ch.Type, true
	}
	return "", "", "", false
}

func isDirectMessage(chType string) bool {
	switch chType {
	case "dm", "group_dm", "app":
		return true
	}
	return false
}

func cmdSchedule(a *App, args []string) tea.Cmd {
	if len(args) == 0 {
		return a.openScheduleMenu()
	}
	if msg := a.schedulePrecheck(); msg != "" {
		return toastWithClear(a, msg, 2*time.Second)
	}
	postAt, err := parseScheduleSpec(strings.Join(args, ""), time.Now())
	if err != nil {
		return toastWithClear(a, "Invalid schedule: "+strings.Join(args, " ")+" (try 20m, 1h, tomorrow)", 3*time.Second)
	}
	return a.confirmSchedule(postAt)
}

func cmdRemind(a *App, args []string) tea.Cmd {
	if len(args) == 0 {
		return a.openRemindDuration()
	}
	mins, err := parseRemindDuration(args[0])
	if err != nil {
		return toastWithClear(a, "Usage: :remind 20m", 2*time.Second)
	}
	return a.remindSelected(mins)
}

// executeCommand parses and runs one command line (without the
// leading ':'). Empty input is a no-op; unknown commands show a
// transient status-bar toast.
func executeCommand(a *App, line string) tea.Cmd {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	fn, ok := commands[fields[0]]
	if !ok {
		return toastWithClear(a, "Unknown command: "+fields[0], 2*time.Second)
	}
	return fn(a, fields[1:])
}
