// internal/ui/reducer_later.go
//
// Later / Save-for-later reducer for App.Update.
package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/laterview"
)

var reduceLater reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case LaterViewActivatedMsg:
		_ = m
		a.setInboxView(ViewLater)
		a.focusedPanel = PanelMessages
		a.laterView.SetBadge(a.sidebar.LaterUnreadCount())
		if cmd := a.fetchLaterCmd(); cmd != nil {
			return cmd, true
		}
		return nil, true

	case LaterListLoadedMsg:
		if m.TeamID != a.activeTeamID || m.Gen != a.laterGen {
			return nil, true
		}
		if m.Err != nil {
			a.laterView.SetLoading(false)
			a.laterView.SetError("saved.list failed — " + m.Err.Error())
			return nil, true
		}
		a.laterView.SetPage(a.decorateLaterItems(m.Items), m.Cursor, m.Append)
		a.laterView.SetCounts(m.Counts)
		a.applyLaterCounts(m.Counts.Badge())
		if !m.Append {
			a.replaceLaterSaved(m.Items)
		} else {
			a.mergeLaterSaved(m.Items)
		}
		return nil, true

	case LaterCountsMsg:
		if m.TeamID != "" && m.TeamID != a.activeTeamID {
			return nil, true
		}
		a.applyLaterCounts(m.Count)
		return nil, true

	case laterToggledMsg:
		return a.applyLaterToggle(m), true

	case laterRemindedMsg:
		return a.applyLaterReminded(m), true

	case laterStateMsg:
		return a.applyLaterState(m), true
	}
	return nil, false
}

func (a *App) applyLaterCounts(n int) {
	a.sidebar.SetLaterUnreadCount(n)
	a.laterView.SetBadge(n)
}

func (a *App) fetchLaterCmd() tea.Cmd {
	if a.activeTeamID == "" {
		return nil
	}
	a.laterGen++
	gen := a.laterGen
	a.laterView.SetLoading(true)
	a.laterView.SetError("")
	later := a.later
	team := ids.TeamID(a.activeTeamID)
	filter := a.laterView.Filter()
	return func() tea.Msg {
		return later.List(team, gen, filter, "")
	}
}

func (a *App) fetchLaterMoreCmd() tea.Cmd {
	if a.activeTeamID == "" || !a.laterView.HasMore() {
		return nil
	}
	a.laterGen++
	gen := a.laterGen
	later := a.later
	team := ids.TeamID(a.activeTeamID)
	filter := a.laterView.Filter()
	cursor := a.laterView.NextCursor()
	return func() tea.Msg {
		return later.List(team, gen, filter, cursor)
	}
}

func (a *App) decorateLaterItems(in []slackclient.SavedItem) []laterview.Item {
	out := make([]laterview.Item, 0, len(in))
	for _, it := range in {
		if it.ItemType != "" && it.ItemType != "message" {
			continue
		}
		item := laterview.Item{SavedItem: it, Preview: it.Text}
		if name, ok := a.channelNames[it.ItemID]; ok {
			item.ChannelName = name
		}
		if _, t, ok := a.channels.Lookup(ids.ChannelID(it.ItemID)); ok {
			item.ChannelType = t
		}
		if it.UserID != "" {
			if it.UserID == a.currentUserID {
				item.AuthorName = "me"
			} else if name, ok := a.userNames[it.UserID]; ok && name != "" {
				item.AuthorName = name
			}
		}
		if item.Preview == "" && a.activityCache != nil {
			if msg, err := a.activityCache.GetMessage(it.ItemID, it.TS); err == nil && !msg.IsDeleted {
				item.Preview = msg.Text
				if item.AuthorName == "" && msg.UserID != "" {
					if msg.UserID == a.currentUserID {
						item.AuthorName = "me"
					} else if name, ok := a.userNames[msg.UserID]; ok && name != "" {
						item.AuthorName = name
					}
				}
			}
		}
		out = append(out, item)
	}
	return out
}

func (a *App) replaceLaterSaved(items []slackclient.SavedItem) {
	// Only the In progress tab is the live saved set used by `w`.
	if a.laterView.Filter() != laterview.FilterInProgress {
		return
	}
	a.laterSaved = map[string]bool{}
	for _, it := range items {
		if it.ItemType == "" || it.ItemType == "message" {
			a.laterSaved[laterKey(it.ItemID, it.TS)] = true
		}
	}
}

func laterKey(channelID, ts string) string {
	return channelID + "\t" + ts
}

func (a *App) mergeLaterSaved(items []slackclient.SavedItem) {
	if a.laterView.Filter() != laterview.FilterInProgress {
		return
	}
	if a.laterSaved == nil {
		a.laterSaved = map[string]bool{}
	}
	for _, it := range items {
		if it.ItemType == "" || it.ItemType == "message" {
			a.laterSaved[laterKey(it.ItemID, it.TS)] = true
		}
	}
}

func (a *App) setLaterState(state string) tea.Cmd {
	it, ok := a.laterView.SelectedItem()
	if !ok || it.ItemID == "" || it.TS == "" {
		return toastWithClear(a, "No saved item selected", 2*time.Second)
	}
	later := a.later
	ch, mts := ids.ChannelID(it.ItemID), ids.MessageTS(it.TS)
	return func() tea.Msg {
		err := later.SetState(ch, mts, state)
		return laterStateMsg{ChannelID: it.ItemID, TS: it.TS, State: state, Err: err}
	}
}

type laterStateMsg struct {
	ChannelID string
	TS        string
	State     string
	Err       error
}

func (a *App) applyLaterState(m laterStateMsg) tea.Cmd {
	if m.Err != nil {
		return toastWithClear(a, "Later update failed: "+truncateReason(m.Err.Error(), 40), 3*time.Second)
	}
	label := "Moved to In progress"
	switch m.State {
	case "completed":
		label = "Marked complete"
	case "archived":
		label = "Archived"
	}
	return tea.Batch(toastWithClear(a, label, 2*time.Second), a.fetchLaterCmd())
}

func (a *App) openSelectedLaterCmd() tea.Cmd {
	it, ok := a.laterView.SelectedItem()
	if !ok || it.ItemID == "" || it.TS == "" {
		return nil
	}
	name, chType, found := a.channels.Lookup(ids.ChannelID(it.ItemID))
	if !found {
		name = it.ChannelName
		chType = it.ChannelType
		if name == "" {
			name = it.ItemID
		}
	}
	a.pendingLinkNav = &pendingLinkNav{
		channelID: it.ItemID,
		messageTS: it.TS,
	}
	id := it.ItemID
	return func() tea.Msg {
		return ChannelSelectedMsg{ID: id, Name: name, Type: chType}
	}
}

func (a *App) selectedMessageRef() (channelID, ts, preview string, ok bool) {
	if a.focusedPanel == PanelMessages && a.view == ViewLater {
		it, hit := a.laterView.SelectedItem()
		if !hit || it.ItemID == "" || it.TS == "" {
			return "", "", "", false
		}
		return it.ItemID, it.TS, it.ChannelName, true
	}
	switch a.focusedPanel {
	case PanelMessages:
		if !a.inChannelView() {
			return "", "", "", false
		}
		msg, hit := a.messagepane.SelectedMessage()
		if !hit {
			return "", "", "", false
		}
		return a.activeChannelID, msg.TS, msg.Text, true
	case PanelThread:
		reply := a.threadPanel.SelectedReply()
		if reply == nil {
			return "", "", "", false
		}
		return a.threadPanel.ChannelID(), reply.TS, reply.Text, true
	default:
		return "", "", "", false
	}
}

func (a *App) toggleSaveForLater() tea.Cmd {
	channelID, ts, _, ok := a.selectedMessageRef()
	if !ok || channelID == "" || ts == "" {
		return toastWithClear(a, "No message selected", 2*time.Second)
	}
	later := a.later
	key := laterKey(channelID, ts)
	ch, mts := ids.ChannelID(channelID), ids.MessageTS(ts)
	saved := a.laterSaved[key]
	if a.view == ViewLater && a.focusedPanel == PanelMessages {
		saved = true
	}
	if saved {
		return func() tea.Msg {
			if err := later.Remove(ch, mts); err != nil {
				return ToastMsg{Text: "Could not remove from Later: " + truncateReason(err.Error(), 40)}
			}
			return laterToggledMsg{ChannelID: channelID, TS: ts, Saved: false}
		}
	}
	return func() tea.Msg {
		if err := later.Add(ch, mts); err != nil {
			if slackclient.AlreadySaved(err) {
				return laterToggledMsg{ChannelID: channelID, TS: ts, Saved: true}
			}
			return ToastMsg{Text: "Could not save for later: " + truncateReason(err.Error(), 40)}
		}
		return laterToggledMsg{ChannelID: channelID, TS: ts, Saved: true}
	}
}

type laterToggledMsg struct {
	ChannelID string
	TS        string
	Saved     bool
}

type laterRemindTarget struct {
	ChannelID string
	TS        string
	Preview   string
}

func (a *App) applyLaterToggle(m laterToggledMsg) tea.Cmd {
	if a.laterSaved == nil {
		a.laterSaved = map[string]bool{}
	}
	key := laterKey(m.ChannelID, m.TS)
	if m.Saved {
		a.laterSaved[key] = true
		a.applyLaterCounts(a.sidebar.LaterUnreadCount() + 1)
		toast := toastWithClear(a, "Saved for later", 2*time.Second)
		if a.view == ViewLater {
			return tea.Batch(toast, a.fetchLaterCmd())
		}
		return toast
	}
	delete(a.laterSaved, key)
	n := a.sidebar.LaterUnreadCount() - 1
	if n < 0 {
		n = 0
	}
	a.applyLaterCounts(n)
	if a.view == ViewLater {
		items := a.laterView.Items()
		kept := items[:0]
		for _, it := range items {
			if it.ItemID == m.ChannelID && it.TS == m.TS {
				continue
			}
			kept = append(kept, it)
		}
		a.laterView.SetItems(kept)
	}
	return toastWithClear(a, "Removed from Later", 2*time.Second)
}

func (a *App) openRemindDuration() tea.Cmd {
	channelID, ts, preview, ok := a.selectedMessageRef()
	if !ok || channelID == "" || ts == "" {
		return toastWithClear(a, "No message selected", 2*time.Second)
	}
	a.remindTarget = laterRemindTarget{ChannelID: channelID, TS: ts, Preview: preview}
	a.presenceMenu.OpenDurations("Remind me")
	a.SetMode(ModeRemindDuration)
	return nil
}

func (a *App) remindSelected(minutes int) tea.Cmd {
	t := a.remindTarget
	if t.ChannelID == "" || t.TS == "" {
		channelID, ts, preview, ok := a.selectedMessageRef()
		if !ok {
			return toastWithClear(a, "No message selected", 2*time.Second)
		}
		t = laterRemindTarget{ChannelID: channelID, TS: ts, Preview: preview}
	}
	if minutes < 1 {
		minutes = 1
	}
	due := time.Now().Add(time.Duration(minutes) * time.Minute).Unix()
	text := remindText(t)
	later := a.later
	permalink := a.messageSvc
	ch, mts := ids.ChannelID(t.ChannelID), ids.MessageTS(t.TS)
	return func() tea.Msg {
		if permalink != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if url, err := permalink.Permalink(ctx, ch, mts); err == nil && url != "" {
				text = "Saved message: " + url
			}
			cancel()
		}
		if err := later.Remind(ch, mts, text, due); err != nil {
			return ToastMsg{Text: "Could not set reminder: " + truncateReason(err.Error(), 40)}
		}
		return laterRemindedMsg{ChannelID: t.ChannelID, TS: t.TS, Minutes: minutes}
	}
}

type laterRemindedMsg struct {
	ChannelID string
	TS        string
	Minutes   int
}

func (a *App) applyLaterReminded(m laterRemindedMsg) tea.Cmd {
	if a.laterSaved == nil {
		a.laterSaved = map[string]bool{}
	}
	a.laterSaved[laterKey(m.ChannelID, m.TS)] = true
	label := formatRemindMinutes(m.Minutes)
	if a.view == ViewLater {
		cmd := a.fetchLaterCmd()
		return tea.Batch(toastWithClear(a, "Reminder set for "+label, 2*time.Second), cmd)
	}
	return toastWithClear(a, "Reminder set for "+label, 2*time.Second)
}

func remindText(t laterRemindTarget) string {
	preview := strings.TrimSpace(strings.ReplaceAll(t.Preview, "\n", " "))
	if len([]rune(preview)) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	if preview == "" {
		preview = "message"
	}
	return "Saved message in " + t.ChannelID + " (" + t.TS + "): " + preview
}

func formatRemindMinutes(minutes int) string {
	switch {
	case minutes < 60:
		return strconv.Itoa(minutes) + "m"
	case minutes%60 == 0 && minutes < 24*60:
		return strconv.Itoa(minutes/60) + "h"
	case minutes%(24*60) == 0:
		return strconv.Itoa(minutes/(24*60)) + "d"
	default:
		return strconv.Itoa(minutes) + "m"
	}
}

// parseRemindDuration parses "20m", "1h", "2d", or a bare minute count.
func parseRemindDuration(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if s == "tomorrow" {
		return minutesUntilTomorrowMorning(), nil
	}
	mult := 1
	unit := s[len(s)-1]
	num := s
	switch unit {
	case 'm':
		num = s[:len(s)-1]
		mult = 1
	case 'h':
		num = s[:len(s)-1]
		mult = 60
	case 'd':
		num = s[:len(s)-1]
		mult = 24 * 60
	default:
		mult = 1
	}
	n, err := strconv.Atoi(num)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return n * mult, nil
}

func minutesUntilTomorrowMorning() int {
	now := time.Now()
	loc := now.Location()
	target := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1)
	d := target.Sub(now)
	mins := int(d.Minutes())
	if mins < 1 {
		mins = 1
	}
	return mins
}
