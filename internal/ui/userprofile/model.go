// Package userprofile provides the `p` overlay for a message author's
// Slack profile.
package userprofile

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	emojiutil "github.com/gammons/slk/internal/emoji"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/overlay"
	"github.com/gammons/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

// Profile is the data shown in the overlay.
type Profile struct {
	UserID      string
	DisplayName string
	RealName    string
	Handle      string
	Title       string
	Status      slackclient.UserStatus
	TZ          string
	TZLabel     string
	TZOffset    int
	Presence    string
	Loading     bool
	IsSelf      bool
	AlreadyInDM bool
}

// Result is returned when the user commits an action.
type Result struct {
	Message bool // open (or switch to) a DM
}

// Model is the profile overlay.
type Model struct {
	visible  bool
	profile  Profile
	selected int // 0 = Message row when present
}

// New creates a hidden profile overlay.
func New() Model {
	return Model{}
}

// Open shows the overlay for p.
func (m *Model) Open(p Profile) {
	m.visible = true
	m.profile = p
	m.selected = 0
}

// SetProfile replaces the displayed profile (async users.info fill-in).
func (m *Model) SetProfile(p Profile) {
	if !m.visible {
		return
	}
	if p.UserID != "" && m.profile.UserID != "" && p.UserID != m.profile.UserID {
		return
	}
	m.profile = p
}

// Profile returns the current profile (tests).
func (m Model) Profile() Profile { return m.profile }

// Close hides the overlay.
func (m *Model) Close() {
	m.visible = false
	m.profile = Profile{}
	m.selected = 0
}

// IsVisible reports whether the overlay is showing.
func (m Model) IsVisible() bool { return m.visible }

// UserID is the profile currently shown.
func (m Model) UserID() string { return m.profile.UserID }

func (m Model) hasMessageAction() bool {
	return m.profile.UserID != "" && !m.profile.IsSelf
}

// BoxSize returns the rendered modal's outer dimensions.
func (m Model) BoxSize(termWidth, termHeight int) (int, int) {
	_ = termHeight
	box := m.renderBox(termWidth)
	if box == "" {
		return 0, 0
	}
	return lipgloss.Width(box), lipgloss.Height(box)
}

// ClickRow selects the Message action when localY lands on it.
func (m *Model) ClickRow(termWidth, termHeight, localY int) bool {
	if !m.hasMessageAction() {
		return false
	}
	_, h := m.BoxSize(termWidth, termHeight)
	// Last padded row inside the box is the action (above bottom padding
	// and border). Measure from the bottom: border(1)+padding(1)+action.
	actionY := h - 3
	if localY != actionY {
		return false
	}
	m.selected = 0
	return true
}

// HandleKey processes a key. Enter on Message returns Result{Message:true}.
func (m *Model) HandleKey(keyStr string) *Result {
	switch keyStr {
	case "esc", "q":
		m.Close()
		return nil
	case "enter":
		if !m.hasMessageAction() {
			return nil
		}
		m.Close()
		return &Result{Message: true}
	}
	return nil
}

// ViewOverlay composites the modal onto background.
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
	p := m.profile

	name := p.DisplayName
	if name == "" {
		name = p.RealName
	}
	if name == "" {
		name = p.UserID
	}
	title := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(styles.Primary).
		Render(truncate.StringWithTail(name, uint(innerWidth), "…"))

	var lines []string
	lines = append(lines, title)
	if p.RealName != "" && p.RealName != name {
		lines = append(lines, mutedLine(p.RealName, innerWidth))
	}
	if p.Title != "" {
		lines = append(lines, mutedLine(p.Title, innerWidth))
	}
	if p.Handle != "" {
		lines = append(lines, mutedLine("@"+p.Handle, innerWidth))
	}
	if st := formatStatus(p.Status, time.Now()); st != "" {
		lines = append(lines, mutedLine(st, innerWidth))
	}
	if loc := formatLocalTime(p.TZ, p.TZLabel, p.TZOffset, time.Now()); loc != "" {
		lines = append(lines, mutedLine(loc, innerWidth))
	}
	if pres := formatPresence(p.Presence); pres != "" {
		lines = append(lines, mutedLine(pres, innerWidth))
	}
	if p.Loading {
		lines = append(lines, mutedLine("Loading…", innerWidth))
	}

	if m.hasMessageAction() {
		label := "Message"
		if p.AlreadyInDM {
			label = "Already in this DM"
		}
		indicator := lipgloss.NewStyle().Background(bg).Foreground(styles.Accent).Render("▌")
		row := lipgloss.NewStyle().
			Background(bg).
			Foreground(styles.Primary).
			Bold(true).
			Width(innerWidth - 1).
			Render(label)
		lines = append(lines, "", indicator+row)
	}

	hint := lipgloss.NewStyle().Background(bg).Foreground(styles.TextMuted).Italic(true).
		Render("Esc to close")
	lines = append(lines, "", hint)

	content := strings.Join(lines, "\n")
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

func boxWidth(termWidth int) int {
	w := termWidth / 2
	if w < 36 {
		w = 36
	}
	if w > 56 {
		w = 56
	}
	return w
}

func mutedLine(s string, width int) string {
	if lipgloss.Width(s) > width {
		s = truncate.StringWithTail(s, uint(width), "…")
	}
	return lipgloss.NewStyle().
		Background(styles.Background).
		Foreground(styles.TextMuted).
		Render(s)
}

func formatStatus(st slackclient.UserStatus, now time.Time) string {
	if !st.Active(now) {
		return ""
	}
	glyph := emojiutil.StatusGlyph(st.Emoji, nil, emojiutil.PlaceContext{})
	switch {
	case glyph != "" && st.Text != "":
		return glyph + " " + st.Text
	case glyph != "":
		return glyph
	case st.Text != "":
		return st.Text
	default:
		return ""
	}
}

func formatLocalTime(tz, label string, offset int, now time.Time) string {
	var loc *time.Location
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	if loc == nil && offset != 0 {
		name := label
		if name == "" {
			name = "tz"
		}
		loc = time.FixedZone(name, offset)
	}
	if loc == nil {
		return ""
	}
	local := now.In(loc)
	out := fmt.Sprintf("Local time %s", local.Format("3:04 PM"))
	if label != "" {
		out += " " + label
	} else if tz != "" {
		out += " " + tz
	}
	return out
}

func formatPresence(p string) string {
	switch p {
	case "active":
		return "Active"
	case "away":
		return "Away"
	default:
		return ""
	}
}
