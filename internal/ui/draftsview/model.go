// Package draftsview is Slack's Drafts & sent list: unsent composer
// drafts plus scheduled messages.
package draftsview

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
	toolbarLines     = 3
	cardContentLines = 3
	cardStride       = cardContentLines + 1
)

const (
	FilterDrafts    = "drafts"
	FilterScheduled = "scheduled"
)

var draftFilters = []string{FilterDrafts, FilterScheduled}

var draftFilterLabels = map[string]string{
	FilterDrafts:    "Drafts",
	FilterScheduled: "Scheduled",
}

type ClickKind int

const (
	ClickNone ClickKind = iota
	ClickItem
	ClickTab
	ClickLoadMore
)

type tabHit struct {
	x0, x1 int
	filter string
}

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.TextMuted)
}

func channelNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
}

func tabActiveStyle() lipgloss.Style {
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

const (
	KindDraft     = "draft"
	KindScheduled = "scheduled"
)

// Item is one Drafts or Scheduled row.
type Item struct {
	Kind          string
	ID            string
	ChannelID     string
	ThreadTS      string
	Text          string
	LastUpdatedTS string
	DateCreated   int64
	PostAt        int64
	ChannelName   string
	ChannelType   string
}

func ItemFromDraft(d slackclient.ComposerDraft) Item {
	return Item{
		Kind:          KindDraft,
		ID:            d.ID,
		ChannelID:     d.ChannelID,
		ThreadTS:      d.ThreadTS,
		Text:          d.Text,
		LastUpdatedTS: d.LastUpdatedTS,
		DateCreated:   d.DateCreated,
	}
}

func ItemFromScheduled(s slackclient.ScheduledMessage) Item {
	return Item{
		Kind:      KindScheduled,
		ID:        s.ID,
		ChannelID: s.Channel,
		Text:      s.Text,
		PostAt:    s.PostAt,
	}
}

type Model struct {
	items        []Item
	channelNames map[string]string
	badge        int
	scheduledN   int
	filter       string
	tabHits      []tabHit
	nextTS       string

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
	return Model{channelNames: map[string]string{}, filter: FilterDrafts}
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

func (m *Model) SetScheduledCount(n int) {
	if n < 0 {
		n = 0
	}
	if m.scheduledN == n {
		return
	}
	m.scheduledN = n
	m.dirty()
}

func ClampFilter(filter string) string {
	if strings.ToLower(strings.TrimSpace(filter)) == FilterScheduled {
		return FilterScheduled
	}
	return FilterDrafts
}

func (m *Model) Filter() string { return ClampFilter(m.filter) }

func (m *Model) SetFilter(filter string) bool {
	filter = ClampFilter(filter)
	if m.Filter() == filter {
		return false
	}
	m.filter = filter
	m.items = nil
	m.nextTS = ""
	m.selected = 0
	m.hasSnapped = false
	m.err = ""
	m.dirty()
	return true
}

func (m *Model) CycleFilter(dir int) bool {
	if dir == 0 {
		return false
	}
	cur := m.Filter()
	idx := 0
	for i, f := range draftFilters {
		if f == cur {
			idx = i
			break
		}
	}
	n := len(draftFilters)
	next := (idx + dir) % n
	if next < 0 {
		next += n
	}
	return m.SetFilter(draftFilters[next])
}

func (m *Model) SetItems(items []Item) {
	prevID, hadSel := m.selectedID()
	m.items = items
	m.loading = false
	m.err = ""
	newSel := 0
	if hadSel && prevID != "" {
		for i, it := range items {
			if it.ID == prevID {
				newSel = i
				break
			}
		}
	}
	m.selected = newSel
	m.nextTS = ""
	m.clampSelection()
	m.hasSnapped = false
	m.dirty()
}

func (m *Model) SetPage(items []Item, nextTS string, appendPage bool) {
	if appendPage {
		seen := make(map[string]bool, len(m.items))
		for _, it := range m.items {
			seen[it.ID] = true
		}
		for _, it := range items {
			if it.ID != "" && seen[it.ID] {
				continue
			}
			m.items = append(m.items, it)
		}
		m.loading = false
		m.err = ""
		m.nextTS = nextTS
		m.clampSelection()
		m.dirty()
		return
	}
	m.SetItems(items)
	m.nextTS = nextTS
	m.dirty()
}

func (m *Model) NextTS() string { return m.nextTS }

func (m *Model) HasMore() bool { return m.nextTS != "" }

func (m *Model) LoadMoreSelected() bool {
	return m.HasMore() && m.selected == len(m.items)
}

func (m *Model) Items() []Item { return m.items }

func (m *Model) SelectedItem() (Item, bool) {
	if len(m.items) == 0 || m.selected < 0 || m.selected >= len(m.items) {
		return Item{}, false
	}
	return m.items[m.selected], true
}

func (m *Model) selectedID() (string, bool) {
	it, ok := m.SelectedItem()
	if !ok {
		return "", false
	}
	return it.ID, true
}

func (m *Model) maxSelect() int {
	n := len(m.items) - 1
	if m.HasMore() {
		n = len(m.items)
	}
	if n < 0 {
		return 0
	}
	return n
}

func (m *Model) MoveDown() {
	if m.selected < m.maxSelect() {
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
	n := m.maxSelect()
	if len(m.items) == 0 && !m.HasMore() {
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
	if rowY < 0 {
		return ClickNone
	}
	if rowY == 0 {
		for _, h := range m.tabHits {
			if colX >= h.x0 && colX < h.x1 {
				if m.SetFilter(h.filter) {
					return ClickTab
				}
				return ClickNone
			}
		}
		return ClickNone
	}
	if rowY < toolbarLines {
		return ClickNone
	}
	bodyY := rowY - toolbarLines
	absLine := m.yOffset + bodyY
	if absLine < 0 {
		return ClickNone
	}
	idx := m.indexAtLine(absLine)
	if idx < 0 {
		return ClickNone
	}
	if m.HasMore() && idx == len(m.items) {
		if m.selected != idx {
			m.selected = idx
			m.dirty()
		}
		return ClickLoadMore
	}
	if idx >= len(m.items) {
		return ClickNone
	}
	if m.selected != idx {
		m.selected = idx
		m.dirty()
	}
	return ClickItem
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
	max := m.maxSelect()
	if len(m.items) == 0 && !m.HasMore() {
		m.selected = 0
		return
	}
	if m.selected > max {
		m.selected = max
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
		body = placeCenter(width, bodyHeight, mutedStyle().Render("loading drafts…"))
	case m.err != "" && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.err))
	case len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.emptyCopy()))
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

func (m *Model) emptyCopy() string {
	if m.Filter() == FilterScheduled {
		return "nothing scheduled"
	}
	return "no drafts"
}

func (m *Model) renderToolbar(width int) string {
	tabs, hits := m.renderTabs()
	m.tabHits = hits
	tabs = clipToWidth(tabs, width)
	if pad := width - lipgloss.Width(tabs); pad > 0 {
		tabs += strings.Repeat(" ", pad)
	}
	hint := mutedStyle().Render("enter open  D delete  f/F tabs")
	hint = clipToWidth(hint, width)
	if pad := width - lipgloss.Width(hint); pad > 0 {
		hint += strings.Repeat(" ", pad)
	}
	return tabs + "\n" + hint + "\n" + blankLine(width)
}

func (m *Model) renderTabs() (string, []tabHit) {
	var b strings.Builder
	hits := make([]tabHit, 0, len(draftFilters))
	x := 0
	cur := m.Filter()
	for i, f := range draftFilters {
		if i > 0 {
			b.WriteString("  ")
			x += 2
		}
		label := draftFilterLabels[f]
		if f == FilterDrafts && m.badge > 0 {
			label += " •" + strconv.Itoa(m.badge)
		}
		if f == FilterScheduled && m.scheduledN > 0 {
			label += " •" + strconv.Itoa(m.scheduledN)
		}
		styled := mutedStyle().Render(label)
		if f == cur {
			styled = tabActiveStyle().Render(label)
		}
		w := lipgloss.Width(label)
		hits = append(hits, tabHit{x0: x, x1: x + w, filter: f})
		b.WriteString(styled)
		x += w
	}
	return b.String(), hits
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
	if m.HasMore() {
		if len(m.items) > 0 {
			lines = append(lines, separator)
		}
		lines = append(lines, m.renderLoadMore(width, m.LoadMoreSelected())...)
	}
	return lines
}

func (m *Model) renderLoadMore(width int, selected bool) []string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	label := clipToWidth("  load more…", contentWidth)
	blank := clipToWidth("", contentWidth)
	return []string{
		m.borderFill(label, contentWidth, selected, false),
		m.borderFill(blank, contentWidth, selected, true),
		m.borderFill(blank, contentWidth, selected, true),
	}
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
	preview := clipToWidth("  "+oneLine(it.Text), contentWidth)
	if oneLine(it.Text) == "" {
		if it.Kind == KindScheduled {
			preview = clipToWidth("  scheduled", contentWidth)
		} else {
			preview = clipToWidth("  empty draft", contentWidth)
		}
	}
	when := formatUnixRel(it.DateCreated)
	if it.Kind == KindScheduled {
		when = formatUnixWhen(it.PostAt)
	}
	footer := clipToWidth("  "+when, contentWidth)
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
	if it.ChannelType != "dm" && it.ChannelType != "group_dm" && it.ChannelType != "app" && ch != "" && !strings.HasPrefix(ch, "#") {
		in = "#" + ch
	}
	where := channelNameStyle().Render(in)
	if it.ThreadTS != "" && in != "" {
		return "Thread  " + mutedStyle().Render("·") + "  " + where
	}
	if it.Kind == KindScheduled && in != "" {
		return "Scheduled  " + mutedStyle().Render("·") + "  " + where
	}
	if in != "" {
		return where
	}
	if it.Kind == KindScheduled {
		return "Scheduled"
	}
	return "Draft"
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func (m *Model) channelLabel(it Item) string {
	if it.ChannelName != "" {
		return it.ChannelName
	}
	if name, ok := m.channelNames[it.ChannelID]; ok && name != "" {
		return name
	}
	return it.ChannelID
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
		return strconv.Itoa(int(d/(time.Hour))) + "h ago"
	default:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d ago"
	}
}

func formatUnixWhen(sec int64) string {
	if sec <= 0 {
		return ""
	}
	t := time.Unix(sec, 0)
	d := time.Until(t)
	if d <= 0 {
		return formatUnixRel(sec)
	}
	switch {
	case d < time.Minute:
		return "in a moment"
	case d < time.Hour:
		return "in " + strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 24*time.Hour:
		return "in " + strconv.Itoa(int(d/time.Hour)) + "h"
	default:
		return "in " + strconv.Itoa(int(d/(24*time.Hour))) + "d"
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
