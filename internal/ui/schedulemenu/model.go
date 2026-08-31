// Package schedulemenu provides the duration overlay for scheduling a
// compose draft via chat.scheduleMessage.
package schedulemenu

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gammons/slk/internal/text"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/overlay"
	"github.com/gammons/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

// Action is the duration choice the user picked.
type Action int

const (
	ActionDuration Action = iota // Duration is set
	ActionTomorrowMorning
	ActionCustom
)

// Result is returned when the user commits a selection.
type Result struct {
	Action   Action
	Duration time.Duration // populated when Action == ActionDuration
}

type item struct {
	label    string
	action   Action
	duration time.Duration
}

// Model is the picker overlay.
type Model struct {
	items    []item
	filtered []int
	query    string
	selected int
	visible  bool
}

// New creates a closed schedule menu.
func New() Model {
	return Model{}
}

// Open shows the overlay with the fixed duration rows.
func (m *Model) Open() {
	m.visible = true
	m.query = ""
	m.selected = 0
	m.items = []item{
		{label: "In 20 minutes", action: ActionDuration, duration: 20 * time.Minute},
		{label: "In 1 hour", action: ActionDuration, duration: time.Hour},
		{label: "In 2 hours", action: ActionDuration, duration: 2 * time.Hour},
		{label: "In 4 hours", action: ActionDuration, duration: 4 * time.Hour},
		{label: "In 8 hours", action: ActionDuration, duration: 8 * time.Hour},
		{label: "Tomorrow morning (9:00 AM)", action: ActionTomorrowMorning},
		{label: "Custom...", action: ActionCustom},
	}
	m.filter()
}

// Close hides the overlay.
func (m *Model) Close() {
	m.visible = false
}

// IsVisible returns whether the overlay is currently showing.
func (m Model) IsVisible() bool { return m.visible }

const listTopOffset = 5

func boxWidth(termWidth int) int {
	w := termWidth / 2
	if w < 36 {
		w = 36
	}
	if w > 60 {
		w = 60
	}
	return w
}

// BoxSize returns the rendered modal box's outer dimensions.
func (m Model) BoxSize(termWidth, termHeight int) (int, int) {
	nRows := len(m.filtered)
	if nRows < 1 {
		nRows = 1
	}
	return boxWidth(termWidth), nRows + 7
}

// ClickRow moves the selection to the menu row at box-local localY.
func (m *Model) ClickRow(termWidth, termHeight, localY int) bool {
	row := localY - listTopOffset
	if row < 0 || row >= len(m.filtered) {
		return false
	}
	m.selected = row
	return true
}

// HandleKey processes a key event and returns a non-nil Result on selection.
func (m *Model) HandleKey(keyStr string) *Result {
	switch keyStr {
	case "enter":
		if len(m.filtered) == 0 {
			return nil
		}
		it := m.items[m.filtered[m.selected]]
		return &Result{Action: it.action, Duration: it.duration}
	case "esc":
		m.Close()
		return nil
	case "down", "ctrl+n", "j":
		if m.selected < len(m.filtered)-1 {
			m.selected++
		}
		return nil
	case "up", "ctrl+p", "k":
		if m.selected > 0 {
			m.selected--
		}
		return nil
	case "backspace":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.selected = 0
			m.filter()
		}
		return nil
	}
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.query += keyStr
		m.selected = 0
		m.filter()
	}
	return nil
}

func (m *Model) filter() {
	m.filtered = nil
	q := text.Fold(m.query)
	if q == "" {
		for i := range m.items {
			m.filtered = append(m.filtered, i)
		}
		return
	}
	var prefix, sub []int
	for i, it := range m.items {
		name := text.Fold(it.label)
		switch {
		case strings.HasPrefix(name, q):
			prefix = append(prefix, i)
		case strings.Contains(name, q):
			sub = append(sub, i)
		}
	}
	m.filtered = append(prefix, sub...)
}

// ViewOverlay renders the dimmed centered modal.
func (m Model) ViewOverlay(termWidth, termHeight int, background string) string {
	if !m.visible {
		return background
	}
	box := m.renderBox(termWidth)
	if box == "" {
		return background
	}
	return overlay.DimmedOverlay(termWidth, termHeight, background, box, 0.5)
}

func (m Model) renderBox(termWidth int) string {
	overlayWidth := boxWidth(termWidth)
	innerWidth := overlayWidth - 4
	bg := styles.Background

	title := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(styles.Primary).
		Render("Schedule message")

	var inputText string
	if m.query == "" {
		placeholder := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Render("Type to filter…")
		inputText = "█ " + placeholder
	} else {
		inputText = m.query + "█"
	}
	input := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "▌"}).
		BorderLeft(true).
		BorderForeground(styles.Primary).
		BorderBackground(bg).
		PaddingLeft(1).
		Background(bg).
		Foreground(styles.TextPrimary).
		Render(inputText)

	var rows []string
	for i, idx := range m.filtered {
		line := m.items[idx].label
		if lipgloss.Width(line) > innerWidth-1 {
			line = truncate.StringWithTail(line, uint(innerWidth-1), "…")
		}
		var row string
		if i == m.selected {
			indicator := lipgloss.NewStyle().Background(bg).Foreground(styles.Accent).Render("▌")
			label := lipgloss.NewStyle().
				Background(bg).
				Foreground(styles.Primary).
				Bold(true).
				Width(innerWidth - 1).
				Render(line)
			row = indicator + label
		} else {
			label := lipgloss.NewStyle().
				Background(bg).
				Foreground(styles.TextPrimary).
				Width(innerWidth - 1).
				Render(line)
			row = " " + label
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Background(bg).
			Foreground(styles.TextMuted).
			Italic(true).
			Render("No matching options"))
	}

	content := title + "\n" + input + "\n\n" + strings.Join(rows, "\n")
	content = messages.ReapplyBgAfterResets(content, messages.BgANSI()+messages.FgANSI())

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		BorderBackground(bg).
		Background(bg).
		Padding(1, 1).
		Width(overlayWidth).
		Render(content)
}

// CustomDurationView returns a centered minutes input used by
// ModeScheduleCustom. query is the digits typed so far.
func CustomDurationView(termWidth, termHeight int, background, query string) string {
	overlayWidth := termWidth / 2
	if overlayWidth < 36 {
		overlayWidth = 36
	}
	if overlayWidth > 60 {
		overlayWidth = 60
	}
	bg := styles.Background

	title := lipgloss.NewStyle().Bold(true).Background(bg).Foreground(styles.Primary).
		Render("Schedule in how many minutes?")

	cursor := query + "█"
	input := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "▌"}).
		BorderLeft(true).
		BorderForeground(styles.Primary).
		BorderBackground(bg).
		PaddingLeft(1).
		Background(bg).
		Foreground(styles.TextPrimary).
		Render(cursor)

	hint := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Italic(true).
		Render("Enter to schedule · Esc to go back")

	content := title + "\n\n" + input + "\n\n" + hint
	content = messages.ReapplyBgAfterResets(content, messages.BgANSI()+messages.FgANSI())

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		BorderBackground(bg).
		Background(bg).
		Padding(1, 1).
		Width(overlayWidth).
		Render(content)

	return overlay.DimmedOverlay(termWidth, termHeight, background, box, 0.5)
}
