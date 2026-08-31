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
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/wintree"
)

// commandFunc executes a named :command. args holds the
// whitespace-separated tokens after the command name (unused by
// v1 commands, reserved for e.g. ":sp #channel").
type commandFunc func(a *App, args []string) tea.Cmd

// commands maps a command name to its handler. Names are matched
// exactly (no prefix matching); aliases get their own entries.
var commands = map[string]commandFunc{
	"ws":       cmdWorkspaceFinder,
	"sp":       cmdSplit,
	"vsp":      cmdVSplit,
	"q":        cmdCloseWindow,
	"only":     cmdOnlyWindow,
	"on":       cmdOnlyWindow,
	"leave":    cmdLeave,
	"schedule": cmdSchedule,
	"move":     cmdMove,
	"section":  cmdSection,
	"remind":   cmdRemind,
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

// dmLeaveToast is the status-bar copy when :leave is used on a DM.
// conversations.leave does not apply to IMs; we refuse rather than
// guess at conversations.close.
const dmLeaveToast = "Can't leave a DM; close it from Slack"

func cmdLeave(a *App, _ []string) tea.Cmd {
	return a.beginLeaveChannel()
}

// beginLeaveChannel opens the leave-channel confirm overlay, or toasts
// when the current view is not a leavable channel (DMs, Threads,
// Activity, nothing selected).
func (a *App) beginLeaveChannel() tea.Cmd {
	id, name, chType, ok := a.activeChannelMeta()
	if !ok {
		return toastWithClear(a, "No channel to leave", 2*time.Second)
	}
	if isDirectMessage(chType) {
		return toastWithClear(a, dmLeaveToast, 2*time.Second)
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

// activeChannelMeta returns the currently viewed channel. ok is false
// in Threads/Activity or when no channel is selected.
func (a *App) activeChannelMeta() (id, name, chType string, ok bool) {
	if a.view != ViewChannels || a.activeChannelID == "" {
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
