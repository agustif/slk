package channelmembers

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/overlay"
	"github.com/agustif/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

// View renders just the overlay box.
func (m Model) View(termWidth int) string {
	return m.renderBox(termWidth)
}

// ViewOverlay renders the overlay as a centered modal with a dark backdrop.
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
	innerWidth := overlayWidth - 4
	bg := styles.Background

	titleText := fmt.Sprintf("#%s · %d members", m.channel, len(m.members))
	title := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(styles.Primary).
		Render(titleText)

	var inputText string
	if m.query == "" {
		placeholder := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Render("Type to filter...")
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

	total := len(m.filtered)
	startIdx, endIdx := m.visibleWindow()
	maxVisible := endIdx - startIdx

	showScrollbar := total > maxVisible
	contentWidth := innerWidth - 1
	if showScrollbar {
		contentWidth--
	}

	var thumbStart, thumbEnd int
	if showScrollbar {
		thumbHeight := maxVisible * maxVisible / total
		if thumbHeight < 1 {
			thumbHeight = 1
		}
		denom := total - maxVisible
		if denom < 1 {
			denom = 1
		}
		thumbStart = startIdx * (maxVisible - thumbHeight) / denom
		if thumbStart < 0 {
			thumbStart = 0
		}
		if thumbStart > maxVisible-thumbHeight {
			thumbStart = maxVisible - thumbHeight
		}
		thumbEnd = thumbStart + thumbHeight
	}
	thumbStyle := lipgloss.NewStyle().Background(bg).Foreground(styles.Primary)
	trackStyle := lipgloss.NewStyle().Background(bg).Foreground(styles.Border)

	var resultRows []string
	switch {
	case m.loading && total == 0 && m.query == "":
		resultRows = append(resultRows, lipgloss.NewStyle().
			Background(bg).
			Foreground(styles.TextMuted).
			Italic(true).
			Render("Loading members..."))
	case total == 0 && m.query != "":
		resultRows = append(resultRows, lipgloss.NewStyle().
			Background(bg).
			Foreground(styles.TextMuted).
			Italic(true).
			Render("No matching members"))
	case total == 0:
		resultRows = append(resultRows, lipgloss.NewStyle().
			Background(bg).
			Foreground(styles.TextMuted).
			Italic(true).
			Render("No members"))
	default:
		for i := startIdx; i < endIdx; i++ {
			mem := m.members[m.filtered[i]]
			row := m.renderRow(mem, contentWidth, i == m.selected, bg)
			if showScrollbar {
				rel := i - startIdx
				if rel >= thumbStart && rel < thumbEnd {
					row += thumbStyle.Render("█")
				} else {
					row += trackStyle.Render("│")
				}
			}
			resultRows = append(resultRows, row)
		}
	}

	content := title + "\n" + input + "\n\n" + strings.Join(resultRows, "\n")
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

func (m Model) renderRow(mem Member, width int, selected bool, bg color.Color) string {
	dot := presenceDot(mem.Presence, bg)

	nameStyle := lipgloss.NewStyle().Background(bg).Foreground(styles.TextPrimary)
	if selected {
		nameStyle = lipgloss.NewStyle().Background(bg).Foreground(styles.Primary).Bold(true)
	}
	name := mem.DisplayName
	if name == "" {
		name = mem.ID
	}
	line := dot + nameStyle.Render(name)

	if mem.Username != "" && mem.Username != mem.DisplayName {
		line += lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Render(" @" + mem.Username)
	}

	var tags []string
	if mem.IsGuest {
		tags = append(tags, "guest")
	}
	if mem.IsExternal {
		tags = append(tags, "ext")
	}
	if len(tags) > 0 {
		line += lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).
			Render(" [" + strings.Join(tags, ", ") + "]")
	}

	if lipgloss.Width(line) > width {
		line = truncate.StringWithTail(line, uint(width), "…")
	}
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}

	if selected {
		indicator := lipgloss.NewStyle().Background(bg).Foreground(styles.Accent).Render("▌")
		return indicator + line
	}
	return " " + line
}

func presenceDot(presence string, bg color.Color) string {
	switch presence {
	case "active":
		return lipgloss.NewStyle().Background(bg).Foreground(styles.Accent).Render("● ")
	case "away", "dnd":
		return lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Render("○ ")
	default:
		// Not in the presence map: keep the column aligned, no claim.
		return lipgloss.NewStyle().Background(bg).Render("  ")
	}
}
