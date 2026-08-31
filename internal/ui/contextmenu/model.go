// Package contextmenu provides a small overlay list of message actions.
package contextmenu

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/truncate"

	"github.com/agustif/slk/internal/ui/messages"
	"github.com/agustif/slk/internal/ui/overlay"
	"github.com/agustif/slk/internal/ui/styles"
)

// ActionID identifies a menu row. The App maps these onto existing
// message actions (reaction picker, permalink, edit, …).
type ActionID string

const (
	ActionAddReaction   ActionID = "add_reaction"
	ActionReplyInThread ActionID = "reply_in_thread"
	ActionSaveForLater  ActionID = "save_for_later"
	ActionRemind        ActionID = "remind"
	ActionCopyPermalink ActionID = "copy_permalink"
	ActionShare         ActionID = "share"
	ActionPin           ActionID = "pin"
	ActionStar          ActionID = "star"
	ActionFollowThread  ActionID = "follow_thread"
	ActionDownloadFile  ActionID = "download_file"
	ActionOpenLinks     ActionID = "open_links"
	ActionEdit          ActionID = "edit"
	ActionDelete        ActionID = "delete"
	ActionMarkUnread    ActionID = "mark_unread"
	ActionListReactions ActionID = "list_reactions"
	ActionLaterComplete ActionID = "later_complete"
	ActionLaterArchive  ActionID = "later_archive"
	ActionLaterRestore  ActionID = "later_restore"
)

// Item is one row in the menu.
type Item struct {
	Label   string
	Action  ActionID
	Enabled bool
}

// Model is the message-actions overlay.
type Model struct {
	items     []Item
	selected  int
	visible   bool
	hasAnchor bool
	anchorX   int
	anchorY   int
}

// New creates a hidden context menu.
func New() Model {
	return Model{}
}

// Open shows the menu centered on the terminal.
func (m *Model) Open(items []Item) {
	m.open(items, false, 0, 0)
}

// OpenAt shows the menu near (anchorX, anchorY), clamped on-screen.
func (m *Model) OpenAt(items []Item, anchorX, anchorY int) {
	m.open(items, true, anchorX, anchorY)
}

func (m *Model) open(items []Item, anchored bool, anchorX, anchorY int) {
	m.items = items
	m.visible = true
	m.hasAnchor = anchored
	m.anchorX = anchorX
	m.anchorY = anchorY
	m.selected = firstEnabled(items)
}

func firstEnabled(items []Item) int {
	for i, it := range items {
		if it.Enabled {
			return i
		}
	}
	return 0
}

// Close hides the overlay.
func (m *Model) Close() {
	m.visible = false
	m.items = nil
	m.selected = 0
	m.hasAnchor = false
}

// IsVisible reports whether the overlay is showing.
func (m Model) IsVisible() bool { return m.visible }

// Selected returns the highlighted row index.
func (m Model) Selected() int { return m.selected }

// Items returns the current rows (for tests).
func (m Model) Items() []Item { return m.items }

// listTopOffset is the box-local row of the first menu row: top border
// (1) + top padding (1) + title (1) + blank separator (1).
const listTopOffset = 4

func boxWidth(termWidth int) int {
	w := 36
	if termWidth > 0 && termWidth < w {
		w = termWidth
	}
	if w < 20 {
		w = 20
	}
	return w
}

// BoxSize returns the rendered modal box's outer dimensions.
func (m Model) BoxSize(termWidth, termHeight int) (int, int) {
	box := m.renderBox(termWidth)
	if box == "" {
		return 0, 0
	}
	return lipgloss.Width(box), lipgloss.Height(box)
}

// BoxOrigin returns the top-left of the box in terminal coordinates.
// Centered when the menu was opened without a mouse anchor; otherwise
// near the click, clamped on-screen. Matches ViewOverlay placement so
// the modal click router hits the drawn box.
func (m Model) BoxOrigin(termWidth, termHeight int) (int, int) {
	w, h := m.BoxSize(termWidth, termHeight)
	if !m.hasAnchor {
		x := (termWidth - w) / 2
		y := (termHeight - h) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		return x, y
	}
	x, y := m.anchorX, m.anchorY
	if x+w > termWidth {
		x = termWidth - w
	}
	if y+h > termHeight {
		y = termHeight - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// ClickRow moves the selection to the menu row at box-local localY and
// returns true when the click lands on a visible row.
func (m *Model) ClickRow(termWidth, termHeight, localY int) bool {
	row := localY - listTopOffset
	if row < 0 || row >= len(m.items) {
		return false
	}
	m.selected = row
	return true
}

// HandleKey processes a key event. Enter on an enabled row closes the
// menu and returns that item; Esc closes without a result; j/k (and
// arrows) move the highlight, skipping disabled rows.
func (m *Model) HandleKey(keyStr string) *Item {
	switch keyStr {
	case "enter":
		if m.selected < 0 || m.selected >= len(m.items) {
			return nil
		}
		it := m.items[m.selected]
		if !it.Enabled {
			return nil
		}
		m.Close()
		return &it
	case "esc":
		m.Close()
		return nil
	case "down", "j":
		m.move(1)
		return nil
	case "up", "k":
		m.move(-1)
		return nil
	}
	return nil
}

func (m *Model) move(delta int) {
	n := len(m.items)
	if n == 0 {
		return
	}
	i := m.selected
	for {
		i += delta
		if i < 0 || i >= n {
			return
		}
		if m.items[i].Enabled {
			m.selected = i
			return
		}
	}
}

// ViewOverlay renders the dimmed menu over background.
func (m Model) ViewOverlay(termWidth, termHeight int, background string) string {
	if !m.visible {
		return background
	}
	box := m.renderBox(termWidth)
	if box == "" {
		return background
	}
	x, y := m.BoxOrigin(termWidth, termHeight)
	return overlay.DimmedOverlayAt(termWidth, termHeight, background, box, 0.5, x, y)
}

func (m Model) renderBox(termWidth int) string {
	if !m.visible {
		return ""
	}
	overlayWidth := boxWidth(termWidth)
	innerWidth := overlayWidth - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	bg := styles.Background

	title := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(styles.Primary).
		Render("Message")

	var rows []string
	for i, it := range m.items {
		line := it.Label
		if lipgloss.Width(line) > innerWidth-1 {
			line = truncate.StringWithTail(line, uint(innerWidth-1), "…")
		}
		var row string
		if i == m.selected {
			indicator := lipgloss.NewStyle().Background(bg).Foreground(styles.Accent).Render("▌")
			fg := styles.Primary
			if !it.Enabled {
				fg = styles.TextMuted
			}
			label := lipgloss.NewStyle().
				Background(bg).
				Foreground(fg).
				Bold(it.Enabled).
				Width(innerWidth - 1).
				Render(line)
			row = indicator + label
		} else {
			fg := styles.TextPrimary
			if !it.Enabled {
				fg = styles.TextMuted
			}
			label := lipgloss.NewStyle().
				Background(bg).
				Foreground(fg).
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
			Render("No actions"))
	}

	footer := lipgloss.NewStyle().
		Background(bg).
		Foreground(styles.TextMuted).
		Render("j/k  enter  esc")

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + footer
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
