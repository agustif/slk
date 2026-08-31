package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/sectionpicker"
	"github.com/gammons/slk/internal/ui/sidebar"
)

func cmdMove(a *App, args []string) tea.Cmd {
	secs := assignableSlackSections(a)
	if a.activeChannelID == "" {
		return toastWithClear(a, "No active channel", 2*time.Second)
	}
	if len(secs) == 0 {
		return toastWithClear(a, "Slack sections not available", 2*time.Second)
	}
	if len(args) > 0 {
		want := strings.Join(args, " ")
		for _, s := range secs {
			label := sectionPickerLabel(s)
			if strings.EqualFold(label, want) {
				return moveActiveChannelTo(a, s.ID, label)
			}
		}
		return toastWithClear(a, "Section not found: "+want, 2*time.Second)
	}
	cur := currentSectionID(a, a.activeChannelID)
	items := make([]sectionpicker.Item, 0, len(secs))
	for _, s := range secs {
		label := sectionPickerLabel(s)
		it := sectionpicker.Item{ID: s.ID, Label: label}
		if s.ID == cur {
			it.Detail = "current"
		}
		items = append(items, it)
	}
	a.sectionPicker.Open("Move to section", items)
	a.SetMode(ModeSectionPicker)
	return nil
}

func cmdSection(a *App, args []string) tea.Cmd {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		return toastWithClear(a, "Usage: :section <name>", 2*time.Second)
	}
	if a.sidebar.SlackSections() == nil {
		return toastWithClear(a, "Slack sections not available", 2*time.Second)
	}
	return func() tea.Msg {
		return a.sections.Create(name)
	}
}

func moveActiveChannelTo(a *App, sectionID, sectionName string) tea.Cmd {
	ch := a.activeChannelID
	if ch == "" {
		return toastWithClear(a, "No active channel", 2*time.Second)
	}
	if sectionID == "" {
		return toastWithClear(a, "No section selected", 2*time.Second)
	}
	if currentSectionID(a, ch) == sectionID {
		if sectionName == "" {
			sectionName = "that section"
		}
		return toastWithClear(a, "Already in "+sectionName, 2*time.Second)
	}
	patchChannelSection(a, ch, sectionID)
	return func() tea.Msg {
		return a.sections.Assign(ids.ChannelID(ch), sectionID)
	}
}

func assignableSlackSections(a *App) []sidebar.SectionMeta {
	var out []sidebar.SectionMeta
	for _, s := range a.sidebar.SlackSections() {
		switch s.Type {
		case "standard", "channels", "direct_messages":
			out = append(out, s)
		}
	}
	return out
}

func sectionPickerLabel(s sidebar.SectionMeta) string {
	if s.Name != "" {
		return s.Name
	}
	switch s.Type {
	case "channels":
		return "Channels"
	case "direct_messages":
		return "Direct Messages"
	case "stars":
		return "Starred"
	default:
		return "(unnamed)"
	}
}

func currentSectionID(a *App, channelID string) string {
	for _, it := range a.sidebar.Items() {
		if it.ID == channelID {
			return it.Section
		}
	}
	return ""
}

func patchChannelSection(a *App, channelID, sectionID string) {
	items := a.sidebar.Items()
	out := make([]sidebar.ChannelItem, len(items))
	copy(out, items)
	for i := range out {
		if out[i].ID == channelID {
			out[i].Section = sectionID
		}
	}
	a.SetChannels(out)
}

var reduceSections reducerFunc = func(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case SectionMovedMsg:
		if m.Unchanged {
			name := m.Name
			if name == "" {
				name = "that section"
			}
			return toastWithClear(a, "Already in "+name, 2*time.Second), true
		}
		if m.Name != "" {
			return toastWithClear(a, "Moved to "+m.Name, 2*time.Second), true
		}
		return toastWithClear(a, "Moved", 2*time.Second), true
	case SectionMoveFailedMsg:
		patchChannelSection(a, m.ChannelID, m.FromSectionID)
		return toastWithClear(a, "Move failed: "+truncateReason(m.Err, 40), 3*time.Second), true
	case SectionCreatedMsg:
		name := m.Name
		if name == "" {
			name = m.ID
		}
		return toastWithClear(a, "Created section "+name, 2*time.Second), true
	case SectionCreateFailedMsg:
		return toastWithClear(a, "Create failed: "+truncateReason(m.Err, 40), 3*time.Second), true
	default:
		return nil, false
	}
}
