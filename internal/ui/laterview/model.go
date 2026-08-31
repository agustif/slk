// Package laterview is the UI model for Slack's Later / Save-for-later
// list: a vertical card list of saved.list items. Presentation only.
package laterview

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

const (
	toolbarLines     = 2 // title + blank
	cardContentLines = 3
	cardStride       = cardContentLines + 1
)

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.TextMuted)
}

func channelNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
}

func overdueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
}

var thickLeftBorder = lipgloss.Border{Left: "▌"}

func borderInvisStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(thickLeftBorder).BorderLeft(true).
		BorderForeground(styles.Background).
		BorderBackground(styles.Background)
}

func borderSelectStyle(focused bool) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(thickLeftBorder).BorderLeft(true).
		BorderForeground(styles.SelectionBorderColor(focused)).
		BorderBackground(styles.SelectionTintColor(focused)).
		Background(styles.SelectionTintColor(focused))
}

func borderFillStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(styles.Background)
}

// Item is one saved.list row plus names resolved by the App layer.
type Item struct {
	slackclient.SavedItem
	ChannelName string
	ChannelType string
}

// Model holds Later-list state.
type Model struct {
	items        []Item
	channelNames map[string]string
	badge        int

	selected int
	yOffset  int
	focused  bool
	loading  bool
	err      string

	snappedSelection int
	hasSnapped       bool
	version          int64
}

func New() Model {
	return Model{channelNames: map[string]string{}}
}

func (m *Model) Version() int64 { return m.version }

func (m *Model) dirty() { m.version++ }

func (m *Model) SetChannelNames(names map[string]string) {
	if names == nil {
		names = map[string]string{}
	}
	m.channelNames = names
	m.dirty()
}

func (m *Model) SetFocused(f bool) {
	if m.focused != f {
		m.focused = f
		m.dirty()
	}
}

func (m *Model) SetLoading(loading bool) {
	if m.loading == loading {
		return
	}
	m.loading = loading
	m.dirty()
}

func (m *Model) SetError(err string) {
	if m.err == err {
		return
	}
	m.err = err
	m.dirty()
}

func (m *Model) SetBadge(n int) {
	if n < 0 {
		n = 0
	}
	if m.badge == n {
		return
	}
	m.badge = n
	m.dirty()
}

func (m *Model) Badge() int { return m.badge }

// SetItems replaces the list. Selection follows Key when still present.
func (m *Model) SetItems(items []Item) {
	prevKey, hadSel := m.selectedKey()
	m.items = items
	m.loading = false
	m.err = ""
	newSel := 0
	if hadSel && prevKey != "" {
		for i, it := range items {
			if it.Key == prevKey {
				newSel = i
				break
			}
		}
	}
	m.selected = newSel
	m.clampSelection()
	m.hasSnapped = false
	m.dirty()
}

func (m *Model) Items() []Item { return m.items }

func (m *Model) SelectedItem() (Item, bool) {
	if len(m.items) == 0 || m.selected < 0 || m.selected >= len(m.items) {
		return Item{}, false
	}
	return m.items[m.selected], true
}

func (m *Model) selectedKey() (string, bool) {
	it, ok := m.SelectedItem()
	if !ok {
		return "", false
	}
	return it.Key, true
}

func (m *Model) MoveDown() {
	if m.selected < len(m.items)-1 {
		m.selected++
		m.dirty()
	}
}

func (m *Model) MoveUp() {
	if m.selected > 0 {
		m.selected--
		m.dirty()
	}
}

func (m *Model) GoToTop() {
	if m.selected != 0 {
		m.selected = 0
		m.dirty()
	}
}

func (m *Model) GoToBottom() {
	if n := len(m.items); n > 0 && m.selected != n-1 {
		m.selected = n - 1
		m.dirty()
	}
}

func (m *Model) ScrollUp(n int) {
	if n <= 0 {
		return
	}
	m.yOffset -= n
	if m.yOffset < 0 {
		m.yOffset = 0
	}
	m.snappedSelection = m.selected
	m.hasSnapped = true
	m.dirty()
}

func (m *Model) ScrollDown(n int) {
	if n <= 0 {
		return
	}
	m.yOffset += n
	m.snappedSelection = m.selected
	m.hasSnapped = true
	m.dirty()
}

// ClickAt selects a card. rowY is pane-local content coordinates.
func (m *Model) ClickAt(rowY int) bool {
	if rowY < toolbarLines {
		return false
	}
	bodyY := rowY - toolbarLines
	absLine := m.yOffset + bodyY
	if absLine < 0 {
		return false
	}
	idx := m.indexAtLine(absLine)
	if idx < 0 || idx >= len(m.items) {
		return false
	}
	if m.selected != idx {
		m.selected = idx
		m.dirty()
	}
	return true
}

func (m *Model) indexAtLine(absLine int) int {
	if absLine%cardStride >= cardContentLines {
		return -1
	}
	return absLine / cardStride
}

func (m *Model) clampSelection() {
	if m.selected < 0 {
		m.selected = 0
	}
	if n := len(m.items); n == 0 {
		m.selected = 0
	} else if m.selected >= n {
		m.selected = n - 1
	}
}

func (m *Model) View(height, width int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	toolbar := m.renderToolbar(width)
	bodyHeight := height - toolbarLines
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	var body string
	switch {
	case m.loading && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("loading later…"))
	case m.err != "" && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.err))
	case len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("nothing saved for later"))
	default:
		lines := m.renderRows(width)
		if !m.hasSnapped || m.snappedSelection != m.selected {
			m.snapToSelected(bodyHeight, len(lines))
			m.snappedSelection = m.selected
			m.hasSnapped = true
		}
		maxOffset := len(lines) - bodyHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.yOffset > maxOffset {
			m.yOffset = maxOffset
		}
		if m.yOffset < 0 {
			m.yOffset = 0
		}
		end := m.yOffset + bodyHeight
		if end > len(lines) {
			end = len(lines)
		}
		visible := lines[m.yOffset:end]
		if pad := bodyHeight - len(visible); pad > 0 {
			filler := blankLine(width)
			out := make([]string, 0, bodyHeight)
			out = append(out, visible...)
			for i := 0; i < pad; i++ {
				out = append(out, filler)
			}
			visible = out
		}
		body = strings.Join(visible, "\n")
	}
	if bodyHeight == 0 {
		return toolbar
	}
	return toolbar + "\n" + body
}

func placeCenter(width, height int, s string) string {
	if height < 1 {
		return ""
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, s)
}

func (m *Model) renderToolbar(width int) string {
	title := "Later"
	if m.badge > 0 {
		title += "  •" + strconv.Itoa(m.badge)
	}
	hint := mutedStyle().Render("  enter open  w unsave  W remind")
	line := channelNameStyle().Render(title) + hint
	line = clipToWidth(line, width)
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return line + "\n" + blankLine(width)
}

func (m *Model) snapToSelected(height, totalLines int) {
	start := m.selected * cardStride
	end := start + cardContentLines
	if end > m.yOffset+height {
		m.yOffset = end - height
	}
	if start < m.yOffset {
		m.yOffset = start
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}
	maxOffset := totalLines - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.yOffset > maxOffset {
		m.yOffset = maxOffset
	}
}

func (m *Model) renderRows(width int) []string {
	separator := blankLine(width)
	var lines []string
	for i, it := range m.items {
		if i > 0 {
			lines = append(lines, separator)
		}
		lines = append(lines, m.renderCard(it, width, i == m.selected)...)
	}
	return lines
}

func blankLine(width int) string {
	return lipgloss.NewStyle().Width(width).Render("")
}

func (m *Model) renderCard(it Item, width int, selected bool) []string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	header := clipToWidth(m.headerText(it), contentWidth)
	preview := clipToWidth("  "+m.previewText(it), contentWidth)
	footer := clipToWidth("  "+formatUnixRel(it.DateCreated), contentWidth)
	return []string{
		m.borderFill(header, contentWidth, selected, false),
		m.borderFill(preview, contentWidth, selected, false),
		m.borderFill(footer, contentWidth, selected, true),
	}
}

func (m *Model) borderFill(text string, contentWidth int, selected, muted bool) string {
	borderStyle := borderInvisStyle()
	fill := borderFillStyle().Width(contentWidth)
	if selected {
		borderStyle = borderSelectStyle(m.focused)
		fill = lipgloss.NewStyle().
			Background(styles.SelectionTintColor(m.focused)).
			Width(contentWidth)
	}
	if muted {
		fill = fill.Foreground(styles.TextMuted)
	}
	return borderStyle.Render(fill.Render(text))
}

func (m *Model) headerText(it Item) string {
	ch := m.channelLabel(it)
	in := ch
	if it.ChannelType != "dm" && it.ChannelType != "group_dm" && ch != "" && !strings.HasPrefix(ch, "#") {
		in = "#" + ch
	}
	header := channelNameStyle().Render(in)
	if it.ItemType != "message" {
		header = it.ItemType + "  " + header
	}
	if due := dueLabel(it.DateDue); due != "" {
		header += "  " + due
	}
	return header
}

func (m *Model) previewText(it Item) string {
	if it.State == "completed" {
		return "completed"
	}
	if it.DateDue > 0 {
		return "reminder"
	}
	return "saved for later"
}

func (m *Model) channelLabel(it Item) string {
	if it.ChannelName != "" {
		return it.ChannelName
	}
	if name, ok := m.channelNames[it.ItemID]; ok && name != "" {
		return name
	}
	return it.ItemID
}

func dueLabel(due int64) string {
	if due <= 0 {
		return ""
	}
	d := time.Until(time.Unix(due, 0))
	if d < 0 {
		return overdueStyle().Render("overdue")
	}
	switch {
	case d < time.Minute:
		return "due now"
	case d < time.Hour:
		return "due in " + strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 24*time.Hour:
		return "due in " + strconv.Itoa(int(d/time.Hour)) + "h"
	default:
		return "due in " + strconv.Itoa(int(d/(24*time.Hour))) + "d"
	}
}

func formatUnixRel(sec int64) string {
	if sec <= 0 {
		return ""
	}
	d := time.Since(time.Unix(sec, 0))
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h ago"
	default:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d ago"
	}
}

func clipToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return truncate.StringWithTail(s, uint(width), "…")
}
