// Package datemenu provides the jump-to-date overlay: a small
// typed-date prompt used by :date / :jump / J.
package datemenu

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/overlay"
	"github.com/agustif/slk/internal/ui/styles"
)

// Result is returned on Enter with the typed date string (trimmed).
type Result struct {
	Query string
}

// Model is the jump-to-date input overlay.
type Model struct {
	query   string
	visible bool
}

// New creates a closed date overlay.
func New() Model {
	return Model{}
}

// Open shows the overlay with an empty query.
func (m *Model) Open() {
	m.visible = true
	m.query = ""
}

// Close hides the overlay.
func (m *Model) Close() {
	m.visible = false
	m.query = ""
}

// IsVisible reports whether the overlay is showing.
func (m Model) IsVisible() bool { return m.visible }

// Query returns the typed date string (tests).
func (m Model) Query() string { return m.query }

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
	_ = termHeight
	box := m.renderBox(termWidth)
	if box == "" {
		return 0, 0
	}
	return lipgloss.Width(box), lipgloss.Height(box)
}

// ClickRow is a no-op: the overlay has no list rows.
func (m *Model) ClickRow(termWidth, termHeight, localY int) bool {
	_ = termWidth
	_ = termHeight
	_ = localY
	return false
}

// HandleKey processes a key. Enter returns the typed query (overlay
// stays open so the caller can toast and retry on a bad parse). Esc
// closes.
func (m *Model) HandleKey(keyStr string) *Result {
	if !m.visible {
		return nil
	}
	switch keyStr {
	case "enter":
		return &Result{Query: strings.TrimSpace(m.query)}
	case "esc":
		m.Close()
		return nil
	case "backspace":
		if m.query != "" {
			r := []rune(m.query)
			m.query = string(r[:len(r)-1])
		}
		return nil
	case "space":
		keyStr = " "
	}
	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.query += keyStr
	}
	return nil
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
	if !m.visible {
		return ""
	}
	overlayWidth := boxWidth(termWidth)
	bg := styles.Background

	title := lipgloss.NewStyle().Bold(true).Background(bg).Foreground(styles.Primary).
		Render("Jump to date")

	var inputText string
	if m.query == "" {
		placeholder := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).
			Render("YYYY-MM-DD or YYYY-MM-DD HH:MM")
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

	hint := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Italic(true).
		Render("Enter to jump · Esc to cancel")

	content := title + "\n\n" + input + "\n\n" + hint
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
