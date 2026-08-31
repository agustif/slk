// Package activityview is the UI model for the Activity inbox: Slack's
// activity.feed recents, with filter tabs and sort/unread chips that
// map onto the official All / DMs / Mentions / Threads request shape.
//
// Presentation only. The App layer fetches via ActivityService and
// pushes items with SetItems; f/F/s/u and toolbar clicks mutate the
// query held here so the next fetch uses the new knobs.
package activityview

import (
	"io"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gammons/slk/internal/config"
	slkemoji "github.com/gammons/slk/internal/emoji"
	imgpkg "github.com/gammons/slk/internal/image"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/messages"
	"github.com/gammons/slk/internal/ui/styles"
	"github.com/muesli/reflow/truncate"
)

const (
	toolbarLines     = 3 // tabs + hint + blank
	cardContentLines = 3
	cardStride       = cardContentLines + 1
)

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.TextMuted)
}

func unreadDotStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
}

func channelNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
}

func tabActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
}

func chipOnStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
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

// ClickKind is what a mouse click hit in the Activity panel.
type ClickKind int

const (
	ClickNone ClickKind = iota
	ClickItem
	ClickControls
	ClickReaction
)

type hitKind int

const (
	hitNone hitKind = iota
	hitFilter
	hitSort
	hitUnread
	hitDensity
)

type hitbox struct {
	x0, x1 int
	kind   hitKind
	viewID string
}

// Item is one flattened activity.feed row plus names resolved by the
// App layer so the view does not talk to Slack or the cache.
type Item struct {
	slackclient.ActivityItem
	ChannelName string
	ChannelType string
	ActorName   string
	// ParentText is the cache-first parent message body (raw mrkdwn).
	// Empty when the cache missed; the card shows a muted empty quote
	// rather than a spinner.
	ParentText string
	// HasReacted is true when the current user already used the event
	// emoji on the parent. Only meaningful when ReactionsKnown.
	HasReacted bool
	// ReactionsKnown is true when GetReactions succeeded (including an
	// empty list). False means toggle must Add, never blind-Remove.
	ReactionsKnown bool
	// OwnReactions is the current user's emoji on the parent (picker ✓).
	OwnReactions []string
}

// EmojiContext bundles emoji-image rendering for Activity cards.
// Mirrors messages.EmojiContext / reactionpicker.EmojiContext.
type EmojiContext struct {
	PlaceCtx slkemoji.PlaceContext
	Cells    int
	Customs  map[string]string
}

type reactionHit struct {
	absLine int
	x0, x1  int
	idx     int
}

// Query is the flattened activity.feed request the next fetch should
// send: the selected activity.views row plus live chip overrides.
type Query struct {
	Filter       string
	Types        []string
	Sort         string
	UnreadOnly   bool
	PriorityOnly bool
}

// Model holds Activity-list state.
type Model struct {
	items        []Item
	userNames    map[string]string
	channelNames map[string]string
	selfUserID   string

	views        []slackclient.ActivityView
	viewIdx      int
	filter       string
	sort         string
	unreadOnly   bool
	priorityOnly bool
	density      string
	unreadBadge  int

	selected int
	yOffset  int
	focused  bool
	loading  bool
	err      string

	snappedSelection int
	hasSnapped       bool

	tabHits  []hitbox
	chipHits []hitbox
	version  int64

	emojiCtx     EmojiContext
	reactionHits []reactionHit
}

// New creates an empty Model seeded from config defaults.
func New() Model {
	m := Model{
		userNames:    map[string]string{},
		channelNames: map[string]string{},
		filter:       config.ActivityFilterAll,
		sort:         config.ActivitySortNewest,
		density:      config.ActivityDensityDetailed,
		views:        slackclient.BuiltinActivityViews(),
	}
	m.applyView(m.views[0], false)
	return m
}

func (m *Model) Version() int64 { return m.version }

func (m *Model) dirty() { m.version++ }

func (m *Model) SetUserNames(names map[string]string) {
	if names == nil {
		names = map[string]string{}
	}
	if stringMapsEqual(m.userNames, names) {
		return
	}
	m.userNames = names
	m.dirty()
}

func (m *Model) SetChannelNames(names map[string]string) {
	if names == nil {
		names = map[string]string{}
	}
	if stringMapsEqual(m.channelNames, names) {
		return
	}
	m.channelNames = names
	m.dirty()
}

func (m *Model) SetSelfUserID(id string) {
	if m.selfUserID != id {
		m.selfUserID = id
		m.dirty()
	}
}

// SetEmojiContext configures emoji-image rendering. Invalidates the
// panel cache so the next View() picks up the new PlaceCtx/cells.
func (m *Model) SetEmojiContext(ctx EmojiContext) {
	if ctx.Cells != 1 && ctx.Cells != 2 {
		ctx.Cells = 2
	}
	m.emojiCtx = ctx
	m.dirty()
}

// SetEmojiCustoms updates the customs map without changing PlaceCtx
// or Cells. Called from App.SetCustomEmoji.
func (m *Model) SetEmojiCustoms(customs map[string]string) {
	m.emojiCtx.Customs = customs
	m.dirty()
}

// HandleEmojiImageReady bumps Version so the panel cache does not
// keep blank cold-cache holes after a kitty emoji fetch lands.
func (m *Model) HandleEmojiImageReady(_ string) {
	m.dirty()
}

// ApplyReaction updates HasReacted / OwnReactions for cards whose
// parent is (channelID, messageTS). isSelf is the current user.
func (m *Model) ApplyReaction(channelID, messageTS, emoji string, isSelf, remove bool) {
	if channelID == "" || messageTS == "" || emoji == "" || !isSelf {
		return
	}
	changed := false
	for i := range m.items {
		it := &m.items[i]
		if it.ChannelID != channelID || it.MessageTS != messageTS {
			continue
		}
		changed = true
		it.ReactionsKnown = true
		if it.Reaction == emoji {
			it.HasReacted = !remove
		}
		if remove {
			it.OwnReactions = removeEmoji(it.OwnReactions, emoji)
		} else if !containsEmoji(it.OwnReactions, emoji) {
			it.OwnReactions = append(it.OwnReactions, emoji)
		}
	}
	if changed {
		m.dirty()
	}
}

func (m *Model) SetFocused(f bool) {
	if m.focused != f {
		m.focused = f
		m.dirty()
	}
}

func (m *Model) SetDensity(d string) {
	d = config.ClampActivityDensity(d)
	if m.density == d {
		return
	}
	m.density = d
	m.hasSnapped = false
	m.dirty()
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

// SetViews replaces the tab list from activity.views. Selection
// follows the previously selected id/type/name when still present.
func (m *Model) SetViews(views []slackclient.ActivityView) {
	if len(views) == 0 {
		views = slackclient.BuiltinActivityViews()
	}
	want := m.filter
	if v, ok := m.selectedView(); ok {
		want = v.ID
	}
	m.views = views
	idx := indexView(views, want)
	if idx < 0 {
		idx = 0
	}
	m.viewIdx = idx
	m.applyView(views[idx], true)
}

func (m *Model) Views() []slackclient.ActivityView { return m.views }

func (m *Model) SetUnreadBadge(n int) {
	if n < 0 {
		n = 0
	}
	if m.unreadBadge == n {
		return
	}
	m.unreadBadge = n
	m.dirty()
}

// SetQuery seeds the selected tab and chips from config (or a
// previous session). filter matches a view id, view_type, or name.
func (m *Model) SetQuery(filter, sort string, unreadOnly bool) {
	filter = config.ClampActivityFilter(filter)
	sort = config.ClampActivitySort(sort)
	if idx := indexView(m.views, filter); idx >= 0 {
		m.viewIdx = idx
		m.applyView(m.views[idx], false)
	}
	m.filter = filter
	m.sort = sort
	m.unreadOnly = unreadOnly
	m.dirty()
}

func (m *Model) Query() Query {
	var types []string
	if v, ok := m.selectedView(); ok {
		types = append([]string(nil), v.Filters.EntryTypes...)
	}
	return Query{
		Filter:       m.filter,
		Types:        types,
		Sort:         m.sort,
		UnreadOnly:   m.unreadOnly,
		PriorityOnly: m.priorityOnly,
	}
}

func (m *Model) Filter() string { return m.filter }
func (m *Model) Sort() string   { return m.sort }
func (m *Model) UnreadOnly() bool {
	return m.unreadOnly
}
func (m *Model) Density() string { return m.density }

func (m *Model) selectedView() (slackclient.ActivityView, bool) {
	if m.viewIdx < 0 || m.viewIdx >= len(m.views) {
		return slackclient.ActivityView{}, false
	}
	return m.views[m.viewIdx], true
}

func (m *Model) applyView(v slackclient.ActivityView, resetChips bool) {
	m.filter = v.ID
	if v.Type != "" && (v.Type != "custom") {
		m.filter = v.Type
	}
	opts := slackclient.FeedOptsFromView(v)
	if resetChips {
		m.sort = opts.Sort
		m.unreadOnly = opts.UnreadOnly
		m.priorityOnly = opts.PriorityOnly
		if v.Density != "" {
			m.density = config.ClampActivityDensity(v.Density)
		}
	} else {
		m.priorityOnly = opts.PriorityOnly
	}
	m.dirty()
}

func (m *Model) selectViewAt(idx int) bool {
	if idx < 0 || idx >= len(m.views) || idx == m.viewIdx {
		return false
	}
	m.viewIdx = idx
	m.applyView(m.views[idx], true)
	return true
}

// CycleFilter walks activity.views tabs (f / F). dir > 0 advances.
func (m *Model) CycleFilter(dir int) bool {
	n := len(m.views)
	if n == 0 {
		return false
	}
	idx := m.viewIdx + dir
	idx %= n
	if idx < 0 {
		idx += n
	}
	return m.selectViewAt(idx)
}

// CycleSort toggles newest ↔ unreads_first.
func (m *Model) CycleSort() bool {
	next := config.NextActivitySort(m.sort)
	if next == m.sort {
		return false
	}
	m.sort = next
	m.dirty()
	return true
}

// ToggleUnreadOnly flips Slack's Unreads chip on the current tab.
func (m *Model) ToggleUnreadOnly() bool {
	m.unreadOnly = !m.unreadOnly
	m.dirty()
	return true
}

// CycleDensity toggles detailed ↔ compact (Slack's Detailed / Dense).
func (m *Model) CycleDensity() bool {
	if m.density == config.ActivityDensityCompact {
		m.density = config.ActivityDensityDetailed
	} else {
		m.density = config.ActivityDensityCompact
	}
	m.hasSnapped = false
	m.dirty()
	return true
}

func indexView(views []slackclient.ActivityView, key string) int {
	if key == "" {
		return -1
	}
	lower := strings.ToLower(key)
	for i, v := range views {
		if v.ID == key || strings.EqualFold(v.Type, lower) || strings.EqualFold(v.Name, key) {
			return i
		}
	}
	return -1
}

// SetItems replaces the list. Selection follows Key when the previous
// row is still present; otherwise it resets to the top.
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

func (m *Model) SelectedIndex() int { return m.selected }

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

// ClickAt selects a card or hits a toolbar control. rowY/colX are
// pane-local content coordinates (border already stripped).
// activateControls is true for left-click (tabs/chips fire); false
// for right-click so the toolbar is inert.
func (m *Model) ClickAt(rowY, colX int) ClickKind {
	return m.clickAt(rowY, colX, true)
}

func (m *Model) clickAt(rowY, colX int, activateControls bool) ClickKind {
	if rowY < 0 {
		return ClickNone
	}
	if rowY == 0 {
		if !activateControls {
			return ClickNone
		}
		return m.clickHits(m.tabHits, colX)
	}
	if rowY == 1 {
		if !activateControls {
			return ClickNone
		}
		return m.clickHits(m.chipHits, colX)
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
	if idx < 0 || idx >= len(m.items) {
		return ClickNone
	}
	if m.selected != idx {
		m.selected = idx
		m.dirty()
	}
	if _, ok := m.hitReaction(absLine, colX); ok {
		return ClickReaction
	}
	return ClickItem
}

// ClickAtCard selects a card without activating toolbar chips.
// Used by right-click → reaction picker.
func (m *Model) ClickAtCard(rowY, colX int) ClickKind {
	return m.clickAt(rowY, colX, false)
}

// HitTestReaction reports whether (rowY, colX) is on a rendered
// event-emoji. Coordinate frame matches ClickAt. Does not mutate.
func (m *Model) HitTestReaction(rowY, colX int) (emoji string, ok bool) {
	if rowY < toolbarLines {
		return "", false
	}
	absLine := m.yOffset + (rowY - toolbarLines)
	return m.hitReaction(absLine, colX)
}

func (m *Model) hitReaction(absLine, colX int) (emoji string, ok bool) {
	for _, h := range m.reactionHits {
		if h.absLine == absLine && colX >= h.x0 && colX < h.x1 {
			if h.idx >= 0 && h.idx < len(m.items) {
				return m.items[h.idx].Reaction, true
			}
			return "", false
		}
	}
	return "", false
}

func (m *Model) clickHits(hits []hitbox, colX int) ClickKind {
	for _, h := range hits {
		if colX >= h.x0 && colX < h.x1 {
			switch h.kind {
			case hitFilter:
				idx := indexView(m.views, h.viewID)
				if idx >= 0 && m.selectViewAt(idx) {
					return ClickControls
				}
			case hitSort:
				if m.CycleSort() {
					return ClickControls
				}
			case hitUnread:
				if m.ToggleUnreadOnly() {
					return ClickControls
				}
			case hitDensity:
				m.CycleDensity()
				return ClickNone
			}
			return ClickNone
		}
	}
	return ClickNone
}

func (m *Model) indexAtLine(absLine int) int {
	if m.density == config.ActivityDensityCompact {
		if absLine >= len(m.items) {
			return -1
		}
		return absLine
	}
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
	m.reactionHits = m.reactionHits[:0]

	toolbar := m.renderToolbar(width)
	bodyHeight := height - toolbarLines
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	var pendingFlushes []func(io.Writer) error
	var body string
	switch {
	case m.loading && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("loading activity…"))
	case m.err != "" && len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render(m.err))
	case len(m.items) == 0:
		body = placeCenter(width, bodyHeight, mutedStyle().Render("no activity"))
	default:
		lines := m.renderRows(width, &pendingFlushes)
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

	for _, fl := range pendingFlushes {
		_ = fl(imgpkg.KittyOutput)
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
	tabs, boxes := m.renderTabs()
	tabs = clipToWidth(tabs, width)
	if pad := width - lipgloss.Width(tabs); pad > 0 {
		tabs += strings.Repeat(" ", pad)
	}

	unreadLabel := "unread"
	sortLabel := config.ActivitySortLabel(m.sort)
	densityLabel := "detailed"
	if m.density == config.ActivityDensityCompact {
		densityLabel = "dense"
	}
	unreadStyled := mutedStyle().Render(unreadLabel)
	if m.unreadOnly {
		unreadStyled = chipOnStyle().Render(unreadLabel)
	}
	sortStyled := mutedStyle().Render(sortLabel)
	if m.sort == config.ActivitySortUnreadsFirst {
		sortStyled = chipOnStyle().Render(sortLabel)
	}
	densityStyled := mutedStyle().Render(densityLabel)
	if m.density == config.ActivityDensityCompact {
		densityStyled = chipOnStyle().Render(densityLabel)
	}
	sep := mutedStyle().Render("  ")
	chips := unreadStyled + sep + sortStyled + sep + densityStyled
	hint := mutedStyle().Render("  f/F tab  s sort  u unread  r react  enter")
	line1 := chips + hint
	line1 = clipToWidth(line1, width)
	if pad := width - lipgloss.Width(line1); pad > 0 {
		line1 += strings.Repeat(" ", pad)
	}

	x := 0
	chipHits := []hitbox{
		{x0: x, x1: x + lipgloss.Width(unreadLabel), kind: hitUnread},
	}
	x += lipgloss.Width(unreadLabel) + 2
	chipHits = append(chipHits, hitbox{x0: x, x1: x + lipgloss.Width(sortLabel), kind: hitSort})
	x += lipgloss.Width(sortLabel) + 2
	chipHits = append(chipHits, hitbox{x0: x, x1: x + lipgloss.Width(densityLabel), kind: hitDensity})
	m.tabHits = boxes
	m.chipHits = chipHits

	return tabs + "\n" + line1 + "\n" + blankLine(width)
}

func (m *Model) renderTabs() (string, []hitbox) {
	var b strings.Builder
	boxes := make([]hitbox, 0, len(m.views))
	x := 0
	sel, _ := m.selectedView()
	for i, v := range m.views {
		if i > 0 {
			b.WriteString("  ")
			x += 2
		}
		label := v.Name
		if label == "" {
			label = v.Type
		}
		if (v.Type == "all" || strings.EqualFold(v.Name, "All")) && m.unreadBadge > 0 {
			label += " •" + strconv.Itoa(m.unreadBadge)
		}
		styled := mutedStyle().Render(label)
		if v.ID == sel.ID {
			styled = tabActiveStyle().Render(label)
		}
		w := lipgloss.Width(label)
		boxes = append(boxes, hitbox{x0: x, x1: x + w, kind: hitFilter, viewID: v.ID})
		b.WriteString(styled)
		x += w
	}
	return b.String(), boxes
}

func (m *Model) snapToSelected(height, totalLines int) {
	start, end := m.selectedSpan()
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

func (m *Model) selectedSpan() (start, end int) {
	if m.density == config.ActivityDensityCompact {
		return m.selected, m.selected + 1
	}
	start = m.selected * cardStride
	return start, start + cardContentLines
}

func (m *Model) renderRows(width int, flushes *[]func(io.Writer) error) []string {
	if m.density == config.ActivityDensityCompact {
		lines := make([]string, 0, len(m.items))
		for i, it := range m.items {
			lines = append(lines, m.renderCompact(it, width, i == m.selected, i, i, flushes))
		}
		return lines
	}
	separator := blankLine(width)
	var lines []string
	for i, it := range m.items {
		if i > 0 {
			lines = append(lines, separator)
		}
		absHeader := i * cardStride
		lines = append(lines, m.renderCard(it, width, i == m.selected, i, absHeader, flushes)...)
	}
	return lines
}

func blankLine(width int) string {
	return lipgloss.NewStyle().Width(width).Render("")
}

func (m *Model) renderCompact(it Item, width int, selected bool, idx, absLine int, flushes *[]func(io.Writer) error) string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	var line string
	if it.Type == "message_reaction" && it.Reaction != "" {
		line = m.renderReactionCompact(it, contentWidth, idx, absLine, flushes)
	} else {
		line = m.headerText(it)
		if q := m.renderQuote(it, contentWidth/2, flushes); q != "" {
			line += "  " + mutedStyle().Render("·") + "  " + q
		}
		if !containsKittyPlacement(line) {
			line = clipToWidth(line, contentWidth)
		} else if slkemoji.Width(line) > contentWidth {
			line = m.headerText(it)
			line = clipToWidth(line, contentWidth)
		}
	}
	return m.borderFill(line, contentWidth, selected, false)
}

func (m *Model) renderCard(it Item, width int, selected bool, idx, absHeader int, flushes *[]func(io.Writer) error) []string {
	contentWidth := width - 1
	if contentWidth < 1 {
		contentWidth = 1
	}
	var header string
	if it.Type == "message_reaction" && it.Reaction != "" {
		header = m.renderReactionHeader(it, contentWidth, idx, absHeader, flushes)
	} else {
		header = clipToWidth(m.headerText(it), contentWidth)
	}
	preview := m.renderPreview(it, contentWidth, flushes)
	footer := clipToWidth("  "+formatRelTime(it.FeedTS), contentWidth)
	return []string{
		m.borderFill(header, contentWidth, selected, false),
		m.borderFill(preview, contentWidth, selected, false),
		m.borderFill(footer, contentWidth, selected, true),
	}
}

func (m *Model) borderFill(text string, contentWidth int, selected, muted bool) string {
	borderStyle := borderInvisStyle()
	bg := styles.Background
	if selected {
		borderStyle = borderSelectStyle(m.focused)
		bg = styles.SelectionTintColor(m.focused)
	}
	// Pad with emoji.Width so kitty placements are not measured (or
	// truncated) by lipgloss.Width / clipToWidth.
	padded := padToCells(text, contentWidth)
	fill := lipgloss.NewStyle().Background(bg)
	if muted {
		fill = fill.Foreground(styles.TextMuted)
	}
	return borderStyle.Render(fill.Render(padded))
}

func (m *Model) headerText(it Item) string {
	actor := it.ActorName
	if actor == "" {
		actor = m.resolveUser(it.ActorID)
	}
	title := m.itemTitle(it)
	var header string
	if actor != "" {
		header = actor + "  " + mutedStyle().Render("·") + "  " + title
	} else {
		header = title
	}
	if it.VIP {
		header += "  " + chipOnStyle().Render("vip")
	}
	if it.Unread {
		header += "  " + unreadDotStyle().Render("●")
	}
	return header
}

func (m *Model) previewText(it Item) string {
	actor := it.ActorName
	if actor == "" {
		actor = m.resolveUser(it.ActorID)
	}
	switch it.Type {
	case "message_reaction":
		rx := it.Reaction
		if rx == "" {
			rx = "reaction"
		}
		if actor != "" {
			return actor + " reacted :" + rx + ":"
		}
		return ":" + rx + ":"
	case "thread_v2":
		if actor != "" {
			return "reply from " + actor
		}
		return "thread reply"
	case "dm":
		if actor != "" {
			return actor
		}
		return "direct message"
	default:
		if actor != "" {
			return actor
		}
		return m.itemTitle(it)
	}
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

func (m *Model) resolveUser(uid string) string {
	if uid == "" {
		return ""
	}
	if uid == m.selfUserID {
		return "me"
	}
	if name, ok := m.userNames[uid]; ok && name != "" {
		return name
	}
	return uid
}

func (m *Model) itemTitle(it Item) string {
	ch := m.channelLabel(it)
	in := ch
	if it.ChannelType != "dm" && it.ChannelType != "group_dm" && ch != "" && !strings.HasPrefix(ch, "#") {
		in = "#" + ch
	}
	switch it.Type {
	case "dm":
		return "Direct Message"
	case "channel":
		if in != "" {
			return "Post in " + channelNameStyle().Render(in)
		}
		return "Post"
	case "thread_v2":
		if in != "" {
			return "Thread in " + channelNameStyle().Render(in)
		}
		return "Thread"
	case "message_reaction":
		rx := it.Reaction
		if rx == "" {
			rx = "reaction"
		}
		if it.ChannelType == "dm" || it.ChannelType == "group_dm" {
			return "Reacted :" + rx + ": in Direct Message"
		}
		if in != "" {
			return "Reacted :" + rx + ": in " + channelNameStyle().Render(in)
		}
		return "Reacted :" + rx + ":"
	case "at_user":
		if in != "" {
			return "Mentioned you in " + channelNameStyle().Render(in)
		}
		return "Mentioned you"
	case "at_channel":
		if in != "" {
			return "Channel mention in " + channelNameStyle().Render(in)
		}
		return "Channel mention"
	case "at_everyone":
		if in != "" {
			return "@everyone in " + channelNameStyle().Render(in)
		}
		return "@everyone"
	case "at_user_group":
		if in != "" {
			return "Group mention in " + channelNameStyle().Render(in)
		}
		return "Group mention"
	case "keyword":
		if in != "" {
			return "Keyword in " + channelNameStyle().Render(in)
		}
		return "Keyword"
	default:
		if it.Type == "" {
			return "Activity"
		}
		return it.Type
	}
}

func channelGlyph(channelType string) string {
	switch channelType {
	case "private":
		return lipgloss.NewStyle().Foreground(styles.Warning).Render("◆ ")
	case "dm", "group_dm":
		return lipgloss.NewStyle().Foreground(styles.TextMuted).Render("● ")
	default:
		return "# "
	}
}

func formatRelTime(ts string) string {
	if ts == "" {
		return ""
	}
	secStr := ts
	if dot := strings.IndexByte(ts, '.'); dot >= 0 {
		secStr = ts[:dot]
	}
	sec, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
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
	if containsKittyPlacement(s) {
		return s
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return truncate.StringWithTail(s, uint(width), "…")
}

func padToCells(s string, width int) string {
	w := slkemoji.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func containsKittyPlacement(s string) bool {
	return strings.ContainsRune(s, imgpkg.PlaceholderRune)
}

func (m *Model) renderReactionHeader(it Item, contentWidth, idx, absLine int, flushes *[]func(io.Writer) error) string {
	emojiStr, flush, asImage := slkemoji.RenderShortcode(it.Reaction, m.emojiCtx.PlaceCtx, m.emojiCells(), m.emojiCtx.Customs)
	if flush != nil && flushes != nil {
		*flushes = append(*flushes, flush)
	}
	if it.HasReacted && !asImage {
		emojiStr = lipgloss.NewStyle().Foreground(styles.ReactionPillOwn.GetForeground()).Render(strings.TrimRight(emojiStr, " "))
	} else if !asImage {
		emojiStr = strings.TrimRight(emojiStr, " ")
	}
	actor := it.ActorName
	if actor == "" {
		actor = m.resolveUser(it.ActorID)
	}
	left := ""
	if actor != "" {
		left = actor + "  " + mutedStyle().Render("·") + "  reacted  "
	} else {
		left = "reacted  "
	}
	in := m.inLabel(it)
	right := ""
	if in != "" {
		right = "  in  " + in
	}
	if it.VIP {
		right += "  " + chipOnStyle().Render("vip")
	}
	if it.Unread {
		right += "  " + unreadDotStyle().Render("●")
	}
	line, x0, x1 := fitLeftFrozenRight(left, emojiStr, right, contentWidth, asImage)
	m.reactionHits = append(m.reactionHits, reactionHit{
		absLine: absLine,
		x0:      1 + x0,
		x1:      1 + x1,
		idx:     idx,
	})
	return line
}

func (m *Model) renderReactionCompact(it Item, contentWidth, idx, absLine int, flushes *[]func(io.Writer) error) string {
	emojiStr, flush, asImage := slkemoji.RenderShortcode(it.Reaction, m.emojiCtx.PlaceCtx, m.emojiCells(), m.emojiCtx.Customs)
	if flush != nil && flushes != nil {
		*flushes = append(*flushes, flush)
	}
	if it.HasReacted && !asImage {
		emojiStr = lipgloss.NewStyle().Foreground(styles.ReactionPillOwn.GetForeground()).Render(strings.TrimRight(emojiStr, " "))
	} else if !asImage {
		emojiStr = strings.TrimRight(emojiStr, " ")
	}
	actor := it.ActorName
	if actor == "" {
		actor = m.resolveUser(it.ActorID)
	}
	left := ""
	if actor != "" {
		left = actor + "  "
	}
	in := m.inLabel(it)
	right := ""
	if in != "" {
		right = "  " + in
	}
	// Fit actor+emoji+channel first so the quote never clips the placement.
	core, x0, x1 := fitLeftFrozenRight(left, emojiStr, right, contentWidth, asImage)
	remain := contentWidth - slkemoji.Width(core)
	sep := "  " + mutedStyle().Render("·") + "  "
	sepW := slkemoji.Width(sep)
	if q := m.renderQuote(it, remain-sepW, flushes); q != "" && remain > sepW {
		core += sep + q
		if !asImage && !containsKittyPlacement(core) {
			core = clipToWidth(core, contentWidth)
		}
	}
	m.reactionHits = append(m.reactionHits, reactionHit{
		absLine: absLine,
		x0:      1 + x0,
		x1:      1 + x1,
		idx:     idx,
	})
	return core
}

func (m *Model) renderPreview(it Item, contentWidth int, flushes *[]func(io.Writer) error) string {
	quoteMax := contentWidth - 4
	if quoteMax < 0 {
		quoteMax = 0
	}
	if it.Type == "message_reaction" {
		q := m.renderQuote(it, quoteMax, flushes)
		line := "  " + mutedStyle().Render(">")
		if q != "" {
			line += " " + q
		}
		if !containsKittyPlacement(line) {
			line = clipToWidth(line, contentWidth)
		}
		return line
	}
	if q := m.renderQuote(it, quoteMax, flushes); q != "" {
		line := "  " + mutedStyle().Render(">") + " " + q
		if !containsKittyPlacement(line) {
			line = clipToWidth(line, contentWidth)
		}
		return line
	}
	return clipToWidth("  "+m.previewText(it), contentWidth)
}

func (m *Model) renderQuote(it Item, maxWidth int, flushes *[]func(io.Writer) error) string {
	raw := oneLine(it.ParentText)
	if raw == "" || maxWidth < 1 {
		return ""
	}
	opts := messages.RenderSlackMarkdownOpts{
		UserNames:    m.userNames,
		ChannelNames: m.channelNames,
		PlaceCtx:     m.emojiCtx.PlaceCtx,
		EmojiCells:   m.emojiCells(),
		Customs:      m.emojiCtx.Customs,
	}
	var qf []func(io.Writer) error
	opts.EmojiFlushes = &qf
	rendered := messages.RenderSlackMarkdownWith(raw, opts)
	if slkemoji.Width(rendered) > maxWidth {
		if len(qf) > 0 {
			opts.PlaceCtx = slkemoji.PlaceContext{}
			opts.EmojiFlushes = nil
			rendered = messages.RenderSlackMarkdownWith(raw, opts)
		}
		return clipToWidth(rendered, maxWidth)
	}
	if flushes != nil && len(qf) > 0 {
		*flushes = append(*flushes, qf...)
	}
	return rendered
}

func (m *Model) emojiCells() int {
	if m.emojiCtx.Cells <= 0 {
		return 2
	}
	return m.emojiCtx.Cells
}

func (m *Model) inLabel(it Item) string {
	ch := m.channelLabel(it)
	if it.ChannelType == "dm" || it.ChannelType == "group_dm" {
		return ch
	}
	if ch != "" && !strings.HasPrefix(ch, "#") {
		return channelNameStyle().Render("#" + ch)
	}
	return channelNameStyle().Render(ch)
}

func fitLeftFrozenRight(left, frozen, right string, width int, frozenOK bool) (line string, x0, x1 int) {
	fw := slkemoji.Width(frozen)
	if !frozenOK {
		line = left + frozen + right
		if slkemoji.Width(line) > width {
			line = clipToWidth(line, width)
		}
		x0 = slkemoji.Width(left)
		x1 = x0 + fw
		if x1 > width {
			x1 = width
		}
		if x0 > width {
			x0, x1 = 0, 0
		}
		return line, x0, x1
	}
	if fw > width {
		return frozen, 0, fw
	}
	budget := width - fw
	lw := slkemoji.Width(left)
	rw := slkemoji.Width(right)
	if lw+rw > budget {
		if lw > budget {
			left = clipToWidth(left, budget)
			lw = slkemoji.Width(left)
			right = ""
		} else {
			right = clipToWidth(right, budget-lw)
		}
	}
	return left + frozen + right, lw, lw + fw
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}

func containsEmoji(list []string, emoji string) bool {
	for _, e := range list {
		if e == emoji {
			return true
		}
	}
	return false
}

func removeEmoji(list []string, emoji string) []string {
	out := list[:0]
	for _, e := range list {
		if e != emoji {
			out = append(out, e)
		}
	}
	return out
}

func stringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if vb, ok := b[k]; !ok || vb != va {
			return false
		}
	}
	return true
}
