// Package unreadsview is Slack's Home All Unreads list: one section per
// unread conversation (header + messages since last_read).
package unreadsview

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

const (
	toolbarLines = 3
	headerLines  = 2
	msgLines     = 1
)

const (
	SortSidebar = "sidebar"
	SortAlpha   = "alpha"
	SortNewest  = "newest"
	SortOldest  = "oldest"
)

var unreadsSorts = []string{SortSidebar, SortAlpha, SortNewest, SortOldest}

var sortLabels = map[string]string{
	SortSidebar: "sidebar",
	SortAlpha:   "alphabetical",
	SortNewest:  "newest",
	SortOldest:  "oldest",
}

// Session-local All Unreads section chips. Slack's Home chips are All /
// VIP / Starred / Channels / DMs. Only all_unreads_section_filter=all_sections
// was captured; slk does not write that pref. VIP/Starred use the same
// conversation flags the sidebar already has (prefs.vip_users / stars.list).
const (
	FilterAll      = "all"
	FilterVIP      = "vip"
	FilterStarred  = "starred"
	FilterChannels = "channels"
	FilterDMs      = "dms"
)

var unreadsFilters = []string{FilterAll, FilterVIP, FilterStarred, FilterChannels, FilterDMs}

var filterLabels = map[string]string{
	FilterAll:      "All",
	FilterVIP:      "VIP",
	FilterStarred:  "Starred",
	FilterChannels: "Channels",
	FilterDMs:      "DMs",
}

type ClickKind int

const (
	ClickNone ClickKind = iota
	ClickHeader
	ClickMessage
	ClickFilter
)

type filterHit struct {
	x0, x1 int
	filter string
}

const (
	rowHeader = iota
	rowMessage
)

type Message struct {
	TS       string
	UserID   string
	UserName string
	Text     string
}

// Block is one unread conversation: header + messages since last_read.
type Block struct {
	ChannelID   string
	ChannelName string
	ChannelType string
	LastRead    string
	LatestTS    string
	Messages    []Message
	MarkedRead  bool
	MarkedCount int
	IsStarred   bool
	IsVIP       bool
}

type selRow struct {
	kind     int
	blockIdx int
	msgIdx   int
	y0, y1   int
}

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.TextMuted)
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

// BlockFromHistory builds an All Unreads section from a captured
// conversations.history result. Display drops the last_read message
// (inclusive=true on the wire includes it); LatestTS is the newest
// timestamp in the response, used for conversations.mark.
func BlockFromHistory(channelID, lastRead string, hist slackclient.HistoryResult) Block {
	var latest string
	var msgs []Message
	for _, m := range hist.Messages {
		ts := m.Timestamp
		if ts == "" {
			continue
		}
		if latest == "" || ts > latest {
			latest = ts
		}
		if lastRead != "" && ts <= lastRead {
			continue
		}
		uid := m.User
		if uid == "" {
			uid = m.BotID
		}
		msgs = append(msgs, Message{TS: ts, UserID: uid, Text: m.Text})
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return Block{
		ChannelID: channelID,
		LastRead:  lastRead,
		LatestTS:  latest,
		Messages:  msgs,
	}
}

type Model struct {
	blocks       []Block
	sidebarOrder []string
	sort         string
	filter       string
	filterHits   []filterHit
	rows         []selRow

	channelNames map[string]string

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
	return Model{channelNames: map[string]string{}, sort: SortSidebar}
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

func (m *Model) Clear() {
	m.blocks = nil
	m.rows = nil
	m.err = ""
	m.loading = false
	m.selected = 0
	m.yOffset = 0
	m.hasSnapped = false
	m.dirty()
}

func ClampSort(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case SortAlpha, SortNewest, SortOldest:
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return SortSidebar
	}
}

func (m *Model) Sort() string { return ClampSort(m.sort) }

func (m *Model) SortLabel() string {
	if lab, ok := sortLabels[m.Sort()]; ok {
		return lab
	}
	return sortLabels[SortSidebar]
}

func (m *Model) SetSort(s string) bool {
	s = ClampSort(s)
	if m.Sort() == s {
		return false
	}
	m.sort = s
	m.applySort()
	m.rebuildRows()
	m.clampSelection()
	m.hasSnapped = false
	m.dirty()
	return true
}

func (m *Model) CycleSort(dir int) bool {
	if dir == 0 {
		return false
	}
	cur := m.Sort()
	idx := 0
	for i, s := range unreadsSorts {
		if s == cur {
			idx = i
			break
		}
	}
	n := len(unreadsSorts)
	next := (idx + dir) % n
	if next < 0 {
		next += n
	}
	return m.SetSort(unreadsSorts[next])
}

func ClampFilter(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case FilterVIP, FilterStarred, FilterChannels, FilterDMs:
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return FilterAll
	}
}

func (m *Model) Filter() string { return ClampFilter(m.filter) }

func (m *Model) FilterLabel() string {
	if lab, ok := filterLabels[m.Filter()]; ok {
		return lab
	}
	return filterLabels[FilterAll]
}

func (m *Model) SetFilter(s string) bool {
	s = ClampFilter(s)
	if m.Filter() == s {
		return false
	}
	prevID, prevTS, prevHeader, had := m.selectedKey()
	m.filter = s
	m.rebuildRows()
	m.selected = 0
	if had {
		m.restoreSelection(prevID, prevTS, prevHeader)
	}
	m.clampSelection()
	m.hasSnapped = false
	m.dirty()
	return true
}

func (m *Model) CycleFilter(dir int) bool {
	if dir == 0 {
		return false
	}
	cur := m.Filter()
	idx := 0
	for i, f := range unreadsFilters {
		if f == cur {
			idx = i
			break
		}
	}
	n := len(unreadsFilters)
	next := (idx + dir) % n
	if next < 0 {
		next += n
	}
	return m.SetFilter(unreadsFilters[next])
}

func isDirectConversation(channelType, channelID string) bool {
	switch channelType {
	case "dm", "group_dm", "app":
		return true
	case "channel", "private":
		return false
	}
	if channelID == "" {
		return false
	}
	switch channelID[0] {
	case 'D', 'G':
		return true
	default:
		return false
	}
}

func (m *Model) blockVisible(b Block) bool {
	switch m.Filter() {
	case FilterVIP:
		return b.IsVIP
	case FilterStarred:
		return b.IsStarred
	case FilterChannels:
		return !isDirectConversation(b.ChannelType, b.ChannelID)
	case FilterDMs:
		return isDirectConversation(b.ChannelType, b.ChannelID)
	default:
		return true
	}
}

func (m *Model) SetSidebarOrder(ids []string) {
	m.sidebarOrder = append([]string(nil), ids...)
	m.applySort()
	m.rebuildRows()
	m.clampSelection()
	m.dirty()
}

func (m *Model) SetBlocks(blocks []Block) {
	prevID, prevTS, prevHeader, had := m.selectedKey()
	m.blocks = blocks
	m.loading = false
	m.err = ""
	m.applySort()
	m.rebuildRows()
	m.selected = 0
	if had {
		m.restoreSelection(prevID, prevTS, prevHeader)
	}
	m.clampSelection()
	m.hasSnapped = false
	m.dirty()
}

func (m *Model) Blocks() []Block { return m.blocks }

func (m *Model) Badge() int {
	n := 0
	for _, b := range m.blocks {
		if !b.MarkedRead {
			n++
		}
	}
	return n
}

func (m *Model) selectedKey() (channelID, ts string, header, ok bool) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return "", "", false, false
	}
	r := m.rows[m.selected]
	if r.blockIdx < 0 || r.blockIdx >= len(m.blocks) {
		return "", "", false, false
	}
	b := m.blocks[r.blockIdx]
	if r.kind == rowHeader {
		return b.ChannelID, "", true, true
	}
	if r.msgIdx >= 0 && r.msgIdx < len(b.Messages) {
		return b.ChannelID, b.Messages[r.msgIdx].TS, false, true
	}
	return b.ChannelID, "", true, true
}

func (m *Model) restoreSelection(channelID, ts string, header bool) {
	for i, r := range m.rows {
		if r.blockIdx < 0 || r.blockIdx >= len(m.blocks) {
			continue
		}
		if m.blocks[r.blockIdx].ChannelID != channelID {
			continue
		}
		if header && r.kind == rowHeader {
			m.selected = i
			return
		}
		if !header && r.kind == rowMessage && r.msgIdx >= 0 && r.msgIdx < len(m.blocks[r.blockIdx].Messages) {
			if m.blocks[r.blockIdx].Messages[r.msgIdx].TS == ts {
				m.selected = i
				return
			}
		}
	}
	for i, r := range m.rows {
		if r.kind == rowHeader && r.blockIdx >= 0 && r.blockIdx < len(m.blocks) && m.blocks[r.blockIdx].ChannelID == channelID {
			m.selected = i
			return
		}
	}
}

func (m *Model) applySort() {
	if len(m.blocks) < 2 {
		return
	}
	mode := m.Sort()
	pos := make(map[string]int, len(m.sidebarOrder))
	for i, id := range m.sidebarOrder {
		if _, ok := pos[id]; !ok {
			pos[id] = i
		}
	}
	sort.SliceStable(m.blocks, func(i, j int) bool {
		a, b := m.blocks[i], m.blocks[j]
		switch mode {
		case SortAlpha:
			na, nb := strings.ToLower(a.displayName()), strings.ToLower(b.displayName())
			if na != nb {
				return na < nb
			}
			return a.ChannelID < b.ChannelID
		case SortNewest:
			ta, tb := a.LatestTS, b.LatestTS
			if ta != tb {
				return ta > tb
			}
			return a.ChannelID < b.ChannelID
		case SortOldest:
			ta, tb := a.LatestTS, b.LatestTS
			if ta != tb {
				return ta < tb
			}
			return a.ChannelID < b.ChannelID
		default:
			pi, oki := pos[a.ChannelID]
			pj, okj := pos[b.ChannelID]
			if oki && okj && pi != pj {
				return pi < pj
			}
			if oki != okj {
				return oki
			}
			na, nb := strings.ToLower(a.displayName()), strings.ToLower(b.displayName())
			if na != nb {
				return na < nb
			}
			return a.ChannelID < b.ChannelID
		}
	})
}

func (b Block) displayName() string {
	if b.ChannelName != "" {
		return b.ChannelName
	}
	return b.ChannelID
}

func (m *Model) rebuildRows() {
	m.rows = m.rows[:0]
	line := 0
	first := true
	for i, b := range m.blocks {
		if !m.blockVisible(b) {
			continue
		}
		if !first {
			line++ // blank separator
		}
		first = false
		m.rows = append(m.rows, selRow{kind: rowHeader, blockIdx: i, y0: line, y1: line + headerLines})
		line += headerLines
		if b.MarkedRead {
			continue
		}
		for j := range b.Messages {
			m.rows = append(m.rows, selRow{kind: rowMessage, blockIdx: i, msgIdx: j, y0: line, y1: line + msgLines})
			line += msgLines
		}
	}
}

func (m *Model) SelectedIsHeader() bool {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return false
	}
	return m.rows[m.selected].kind == rowHeader
}

func (m *Model) SelectedBlock() (Block, bool) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return Block{}, false
	}
	r := m.rows[m.selected]
	if r.blockIdx < 0 || r.blockIdx >= len(m.blocks) {
		return Block{}, false
	}
	return m.blocks[r.blockIdx], true
}

func (m *Model) SelectedMessage() (Block, Message, bool) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return Block{}, Message{}, false
	}
	r := m.rows[m.selected]
	if r.kind != rowMessage || r.blockIdx < 0 || r.blockIdx >= len(m.blocks) {
		return Block{}, Message{}, false
	}
	b := m.blocks[r.blockIdx]
	if r.msgIdx < 0 || r.msgIdx >= len(b.Messages) {
		return Block{}, Message{}, false
	}
	return b, b.Messages[r.msgIdx], true
}

func (m *Model) MarkBlockRead(channelID string) bool {
	for i := range m.blocks {
		if m.blocks[i].ChannelID != channelID {
			continue
		}
		if m.blocks[i].MarkedRead {
			return false
		}
		m.blocks[i].MarkedRead = true
		m.blocks[i].MarkedCount = len(m.blocks[i].Messages)
		m.rebuildRows()
		m.restoreSelection(channelID, "", true)
		m.clampSelection()
		m.hasSnapped = false
		m.dirty()
		return true
	}
	return false
}

func (m *Model) UndoBlock(channelID string) bool {
	for i := range m.blocks {
		if m.blocks[i].ChannelID != channelID {
			continue
		}
		if !m.blocks[i].MarkedRead {
			return false
		}
		m.blocks[i].MarkedRead = false
		m.rebuildRows()
		m.restoreSelection(channelID, "", true)
		m.clampSelection()
		m.hasSnapped = false
		m.dirty()
		return true
	}
	return false
}

func (m *Model) maxSelect() int {
	n := len(m.rows) - 1
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
	if len(m.rows) == 0 {
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
	if rowY == 1 {
		for _, h := range m.filterHits {
			if colX >= h.x0 && colX < h.x1 {
				if m.SetFilter(h.filter) {
					return ClickFilter
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
	idx := m.indexAtLine(absLine)
	if idx < 0 {
		return ClickNone
	}
	if m.selected != idx {
		m.selected = idx
		m.dirty()
	}
	if m.rows[idx].kind == rowHeader {
		return ClickHeader
	}
	return ClickMessage
}

func (m *Model) indexAtLine(absLine int) int {
	for i, r := range m.rows {
		if absLine >= r.y0 && absLine < r.y1 {
			return i
		}
	}
	return -1
}

func (m *Model) clampSelection() {
	if m.selected < 0 {
		m.selected = 0
	}
	if len(m.rows) == 0 {
		m.selected = 0
		return
	}
	max := m.maxSelect()
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
	case m.loading && len(m.blocks) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("loading unreads…"))
	case m.err != "" && len(m.blocks) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.err))
	case len(m.rows) == 0:
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
	switch m.Filter() {
	case FilterVIP:
		return "no VIP unreads"
	case FilterStarred:
		return "no starred unreads"
	case FilterChannels:
		return "no channel unreads"
	case FilterDMs:
		return "no DM unreads"
	default:
		return "no unreads"
	}
}

func (m *Model) renderToolbar(width int) string {
	title := tabActiveStyle().Render("All Unreads")
	if n := m.unreadBadge(); n > 0 {
		title += mutedStyle().Render("  •" + strconv.Itoa(n))
	}
	title = clipToWidth(title, width)
	if pad := width - lipgloss.Width(title); pad > 0 {
		title += strings.Repeat(" ", pad)
	}
	chips, hits := m.renderFilterChips()
	m.filterHits = hits
	chips = clipToWidth(chips, width)
	if pad := width - lipgloss.Width(chips); pad > 0 {
		chips += strings.Repeat(" ", pad)
	}
	hint := mutedStyle().Render("enter open  header mark read  f/F sort · " + m.SortLabel() + "  s section")
	hint = clipToWidth(hint, width)
	if pad := width - lipgloss.Width(hint); pad > 0 {
		hint += strings.Repeat(" ", pad)
	}
	return title + "\n" + chips + "\n" + hint
}

func (m *Model) renderFilterChips() (string, []filterHit) {
	var b strings.Builder
	hits := make([]filterHit, 0, len(unreadsFilters))
	x := 0
	cur := m.Filter()
	for i, f := range unreadsFilters {
		if i > 0 {
			b.WriteString("  ")
			x += 2
		}
		label := filterLabels[f]
		if label == "" {
			label = f
		}
		styled := mutedStyle().Render(label)
		if f == cur {
			styled = tabActiveStyle().Render(label)
		}
		w := lipgloss.Width(label)
		hits = append(hits, filterHit{x0: x, x1: x + w, filter: f})
		b.WriteString(styled)
		x += w
	}
	return b.String(), hits
}

func (m *Model) unreadBadge() int {
	n := 0
	for _, b := range m.blocks {
		if !b.MarkedRead {
			n++
		}
	}
	return n
}

func (m *Model) snapToSelected(height, totalLines int) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return
	}
	r := m.rows[m.selected]
	start, end := r.y0, r.y1
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
	sel := -1
	if m.selected >= 0 && m.selected < len(m.rows) {
		sel = m.selected
	}
	first := true
	for i, b := range m.blocks {
		if !m.blockVisible(b) {
			continue
		}
		if !first {
			lines = append(lines, separator)
		}
		first = false
		headerSel := sel >= 0 && m.rows[sel].kind == rowHeader && m.rows[sel].blockIdx == i
		lines = append(lines, m.renderHeader(b, width, headerSel)...)
		if b.MarkedRead {
			continue
		}
		for j, msg := range b.Messages {
			msgSel := sel >= 0 && m.rows[sel].kind == rowMessage && m.rows[sel].blockIdx == i && m.rows[sel].msgIdx == j
			lines = append(lines, m.renderMessage(msg, width, msgSel)...)
		}
	}
	return lines
}

func (m *Model) renderHeader(b Block, width int, selected bool) []string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	name := clipToWidth(m.channelLabel(b), contentWidth)
	n := len(b.Messages)
	if b.MarkedRead {
		n = b.MarkedCount
	}
	count := messageCountLabel(n, b.MarkedRead)
	action := "mark as read"
	if b.MarkedRead {
		action = "undo"
	}
	meta := padLeftRight(count, action, contentWidth)
	return []string{
		m.borderFill(name, contentWidth, selected, false),
		m.borderFill(meta, contentWidth, selected, true),
	}
}

func (m *Model) renderMessage(msg Message, width int, selected bool) []string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	who := msg.UserName
	if who == "" {
		who = msg.UserID
	}
	text := "  " + who
	if preview := oneLine(msg.Text); preview != "" {
		text += "  " + preview
	}
	text = clipToWidth(text, contentWidth)
	return []string{m.borderFill(text, contentWidth, selected, false)}
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

func (m *Model) channelLabel(b Block) string {
	ch := b.ChannelName
	if ch == "" {
		if name, ok := m.channelNames[b.ChannelID]; ok && name != "" {
			ch = name
		} else {
			ch = b.ChannelID
		}
	}
	if b.ChannelType != "dm" && b.ChannelType != "group_dm" && b.ChannelType != "app" && ch != "" && !strings.HasPrefix(ch, "#") {
		return "#" + ch
	}
	return ch
}

func messageCountLabel(n int, marked bool) string {
	noun := "messages"
	if n == 1 {
		noun = "message"
	}
	s := strconv.Itoa(n) + " " + noun
	if marked {
		return s + " marked read"
	}
	return s
}

func padLeftRight(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if rw == 0 {
		return clipToWidth(left, width)
	}
	if lw+1+rw >= width {
		return clipToWidth(left+" "+right, width)
	}
	return left + strings.Repeat(" ", width-lw-rw) + right
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func blankLine(width int) string {
	return lipgloss.NewStyle().Width(width).Render("")
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
