package presencemenu

import (
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/overlay"
	"github.com/agustif/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

// StatusDuration is one "clear after" option in the set-status overlay.
type StatusDuration struct {
	Label string
	// Kind is 30m, 1h, 4h, today, or none (don't clear).
	Kind string
}

var statusDurations = []StatusDuration{
	{Label: "30 minutes", Kind: "30m"},
	{Label: "1 hour", Kind: "1h"},
	{Label: "4 hours", Kind: "4h"},
	{Label: "Today", Kind: "today"},
	{Label: "Don't clear", Kind: "none"},
}

// SetStatusModel is the Ctrl+S "Set status..." input overlay.
type SetStatusModel struct {
	visible  bool
	query    string
	duration int // index into statusDurations
}

// NewSetStatus creates a hidden set-status overlay.
func NewSetStatus() SetStatusModel {
	return SetStatusModel{}
}

// Open shows the overlay with an empty query and "Don't clear" selected.
func (m *SetStatusModel) Open() {
	m.visible = true
	m.query = ""
	m.duration = len(statusDurations) - 1
}

// Close hides the overlay.
func (m *SetStatusModel) Close() {
	m.visible = false
	m.query = ""
}

// IsVisible reports whether the overlay is showing.
func (m SetStatusModel) IsVisible() bool { return m.visible }

// Query returns the typed status line (tests).
func (m SetStatusModel) Query() string { return m.query }

// DurationKind returns the selected duration kind (tests).
func (m SetStatusModel) DurationKind() string {
	if m.duration < 0 || m.duration >= len(statusDurations) {
		return "none"
	}
	return statusDurations[m.duration].Kind
}

const setStatusListTopOffset = 5

// BoxSize returns the rendered modal's outer dimensions.
func (m SetStatusModel) BoxSize(termWidth, termHeight int) (int, int) {
	_ = termHeight
	nRows := len(statusDurations)
	if nRows < 1 {
		nRows = 1
	}
	return boxWidth(termWidth), nRows + 9
}

// ClickRow selects the duration row at box-local localY.
func (m *SetStatusModel) ClickRow(termWidth, termHeight, localY int) bool {
	_ = termWidth
	_ = termHeight
	row := localY - setStatusListTopOffset
	if row < 0 || row >= len(statusDurations) {
		return false
	}
	m.duration = row
	return true
}

// HandleKey processes a key. A non-nil Result is ActionSetStatus on enter.
func (m *SetStatusModel) HandleKey(keyStr string) *Result {
	switch keyStr {
	case "enter":
		emoji, text := ParseStatusInput(m.query)
		if emoji == "" && text == "" {
			return nil
		}
		kind := m.DurationKind()
		m.Close()
		return &Result{
			Action:           ActionSetStatus,
			StatusText:       text,
			StatusEmoji:      emoji,
			StatusExpiration: StatusExpirationUnix(kind, time.Now()),
		}
	case "esc":
		m.Close()
		return nil
	case "down", "tab":
		if m.duration < len(statusDurations)-1 {
			m.duration++
		}
		return nil
	case "up", "shift+tab":
		if m.duration > 0 {
			m.duration--
		}
		return nil
	case "backspace":
		if m.query != "" {
			m.query = trimLastGrapheme(m.query)
		}
		return nil
	}
	if isTypedStatusRune(keyStr) {
		m.query += keyStr
	}
	return nil
}

func isTypedStatusRune(keyStr string) bool {
	if keyStr == "" {
		return false
	}
	if strings.Contains(keyStr, "ctrl+") || strings.Contains(keyStr, "alt+") {
		return false
	}
	if keyStr == "tab" || keyStr == "shift+tab" || keyStr == "enter" || keyStr == "esc" {
		return false
	}
	r := []rune(keyStr)
	if len(r) != 1 {
		return false
	}
	return unicode.IsPrint(r[0])
}

func trimLastGrapheme(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

// ParseStatusInput splits an optional leading :shortcode: from status text.
func ParseStatusInput(s string) (emoji, text string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if !strings.HasPrefix(s, ":") {
		return "", s
	}
	rest := s[1:]
	i := strings.IndexByte(rest, ':')
	if i <= 0 {
		return "", s
	}
	name := rest[:i]
	for _, r := range name {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '-') {
			return "", s
		}
	}
	return ":" + name + ":", strings.TrimSpace(rest[i+1:])
}

// StatusExpirationUnix converts a duration kind to a unix expiration.
// "none" and unknown kinds return 0 (don't clear). "today" is local midnight.
func StatusExpirationUnix(kind string, now time.Time) int64 {
	switch kind {
	case "30m":
		return now.Add(30 * time.Minute).Unix()
	case "1h":
		return now.Add(time.Hour).Unix()
	case "4h":
		return now.Add(4 * time.Hour).Unix()
	case "today":
		loc := now.Location()
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
		return tomorrow.Unix()
	default:
		return 0
	}
}

// ViewOverlay renders the dimmed centered modal.
func (m SetStatusModel) ViewOverlay(termWidth, termHeight int, background string) string {
	if !m.visible {
		return background
	}
	box := m.renderBox(termWidth)
	if box == "" {
		return background
	}
	return overlay.DimmedOverlay(termWidth, termHeight, background, box, 0.5)
}

func (m SetStatusModel) renderBox(termWidth int) string {
	overlayWidth := boxWidth(termWidth)
	innerWidth := overlayWidth - 4
	bg := styles.Background

	title := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(styles.Primary).
		Render("Set status")

	var inputText string
	if m.query == "" {
		placeholder := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).
			Render(":emoji: optional text")
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
	for i, d := range statusDurations {
		line := d.Label
		if lipgloss.Width(line) > innerWidth-1 {
			line = truncate.StringWithTail(line, uint(innerWidth-1), "…")
		}
		var row string
		if i == m.duration {
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

	hint := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Italic(true).
		Render("Enter to set · Esc to cancel")

	content := title + "\n" + input + "\n\n" + strings.Join(rows, "\n") + "\n\n" + hint
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
