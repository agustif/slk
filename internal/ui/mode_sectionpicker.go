// internal/ui/mode_sectionpicker.go
//
// Key handler for ModeSectionPicker: the chooser opened by :move.
// Enter assigns the active channel to the highlighted section.
package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/sectionpicker"
)

func handleSectionPickerMode(a *App, msg tea.KeyMsg) tea.Cmd {
	item, chosen := a.sectionPicker.HandleKey(msg.String())
	kind := a.sectionPickerKind
	if chosen {
		a.SetMode(ModeNormal)
		a.sectionPickerKind = ""
		if kind == "scheduled" {
			return a.confirmDeleteScheduled(item)
		}
		if kind == "reminders" {
			id := item.ID
			return func() tea.Msg {
				return a.messageSvc.CompleteReminder(id)
			}
		}
		return moveActiveChannelTo(a, item.ID, item.Label)
	}
	if !a.sectionPicker.IsVisible() {
		a.SetMode(ModeNormal)
		a.sectionPickerKind = ""
	}
	return nil
}

func (a *App) confirmDeleteScheduled(item sectionpicker.Item) tea.Cmd {
	channelID, scheduledID, ok := splitScheduledKey(item.ID)
	if !ok {
		return toastWithClear(a, "Could not cancel scheduled message", 2*time.Second)
	}
	label := item.Label
	a.confirmPrompt.Open(
		"Cancel scheduled message?",
		label,
		func() tea.Msg {
			return a.messageSvc.DeleteScheduled(ids.ChannelID(channelID), scheduledID)
		},
	)
	a.SetMode(ModeConfirm)
	return nil
}

func splitScheduledKey(id string) (channelID, scheduledID string, ok bool) {
	i := strings.IndexByte(id, '\t')
	if i < 1 || i == len(id)-1 {
		return "", "", false
	}
	return id[:i], id[i+1:], true
}
