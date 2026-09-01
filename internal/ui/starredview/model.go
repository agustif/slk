// Package starredview is Slack's Starred items list: stars.list
// type=message rows. Channel stars stay in the sidebar Starred section.
package starredview

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

const (
	toolbarLines     = 3
	cardContentLines = 3
	cardStride       = cardContentLines + 1
)

type ClickKind int

const (
	ClickNone ClickKind = iota
	ClickItem
)

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.TextMuted)
}

func channelNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
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

// Item is one stars.list message row plus names resolved by App,
// or a files.favorites.list file id (Files-rail Starred). FileTitle
// comes from files.list / files.info when the id is a F… file (quip
// canvases included). No canvas editor.
type Item struct {
	slackclient.StarredMessage
	ChannelName string
	ChannelType string
	AuthorName  string
	FileID      string
	FileTitle   string
	Filetype    string
	FileMode    string
}

func (it Item) Key() string {
	if it.FileID != "" {
		return "file\t" + it.FileID
	}
	return it.ChannelID + "\t" + it.TS
}

type Model struct {
	items    []Item
	badge    int
	selected int
	yOffset  int
	focused  bool
	loading  bool
	err      string

	snappedSelection int
	hasSnapped       bool
	version          int64
}

func New() Model { return Model{} }

func (m *Model) Version() int64 { return m.version }

func (m *Model) dirty() { m.version++ }

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

func (m *Model) SetItems(items []Item) {
	prevKey, hadSel := m.selectedKey()
	m.items = items
	m.loading = false
	m.err = ""
	newSel := 0
	if hadSel && prevKey != "" {
		for i, it := range items {
			if it.Key() == prevKey {
				newSel = i
				break
			}
		}
	}
	m.selected = newSel
	m.clampSelection()
	m.hasSnapped = false
	m.SetBadge(len(items))
	m.dirty()
}

func (m *Model) Remove(channelID, ts string) {
	key := channelID + "\t" + ts
	out := m.items[:0]
	for _, it := range m.items {
		if it.Key() != key {
			out = append(out, it)
		}
	}
	if len(out) == len(m.items) {
		return
	}
	m.items = out
	m.clampSelection()
	m.SetBadge(len(m.items))
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
	return it.Key(), true
}

func (m *Model) clampSelection() {
	if m.selected < 0 {
		m.selected = 0
	}
	if len(m.items) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.items) {
		m.selected = len(m.items) - 1
	}
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
	n := len(m.items) - 1
	if n < 0 {
		return
	}
	if m.selected != n {
		m.selected = n
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

func (m *Model) ClickAt(rowY, colX int) ClickKind {
	_ = colX
	if rowY < toolbarLines {
		return ClickNone
	}
	bodyY := rowY - toolbarLines
	absLine := m.yOffset + bodyY
	if absLine < 0 {
		return ClickNone
	}
	if absLine%cardStride >= cardContentLines {
		return ClickNone
	}
	idx := absLine / cardStride
	if idx < 0 || idx >= len(m.items) {
		return ClickNone
	}
	if m.selected != idx {
		m.selected = idx
		m.dirty()
	}
	return ClickItem
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
		body = placeCenter(width, bodyHeight, mutedStyle().Render("loading starred…"))
	case m.err != "" && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.err))
	case len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("no starred items"))
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
	title := "Starred items"
	if m.badge > 0 {
		title += "  •" + strconv.Itoa(m.badge)
	}
	title = tabTitleStyle().Render(title)
	title = clipToWidth(title, width)
	if pad := width - lipgloss.Width(title); pad > 0 {
		title += strings.Repeat(" ", pad)
	}
	hint := mutedStyle().Render("enter open  * unstar  x menu")
	hint = clipToWidth(hint, width)
	if pad := width - lipgloss.Width(hint); pad > 0 {
		hint += strings.Repeat(" ", pad)
	}
	return title + "\n" + hint + "\n" + blankLine(width)
}

func tabTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
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
	if it.FileID != "" {
		title := it.FileTitle
		if title == "" {
			title = it.FileID
		}
		kind := "Files-rail Starred"
		if it.Filetype == "quip" || it.FileMode == "quip" {
			kind = "Canvas"
		}
		header := clipToWidth(title, contentWidth)
		preview := clipToWidth("  "+it.FileID, contentWidth)
		footer := clipToWidth("  "+kind, contentWidth)
		return []string{
			m.borderFill(header, contentWidth, selected, false),
			m.borderFill(preview, contentWidth, selected, false),
			m.borderFill(footer, contentWidth, selected, true),
		}
	}
	header := clipToWidth(m.headerText(it), contentWidth)
	preview := clipToWidth("  "+oneLine(it.Text), contentWidth)
	if oneLine(it.Text) == "" {
		preview = clipToWidth("  starred message", contentWidth)
	}
	footer := clipToWidth("  "+formatUnixRel(it.DateCreate), contentWidth)
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
	ch := it.ChannelName
	if ch == "" {
		ch = it.ChannelID
	}
	in := ch
	if it.ChannelType != "dm" && it.ChannelType != "group_dm" && ch != "" && !strings.HasPrefix(ch, "#") {
		in = "#" + ch
	}
	where := channelNameStyle().Render(in)
	if it.AuthorName != "" && in != "" {
		return it.AuthorName + "  " + mutedStyle().Render("·") + "  " + where
	}
	if it.AuthorName != "" {
		return it.AuthorName
	}
	return where
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
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
