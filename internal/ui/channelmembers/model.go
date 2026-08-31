// Package channelmembers is the I-key overlay that lists members of
// the active channel. Filter-as-you-type with j/k (and arrows)
// navigation; Enter returns the selected member so the App can open
// a DM via conversations.open.
package channelmembers

import (
	"sort"
	"strings"

	"github.com/gammons/slk/internal/text"
)

// Member is one row in the overlay. Presence is the live value from
// the sidebar presence map ("active", "away", "dnd") or empty when
// the user is not in that map — empty means "do not draw a dot".
// IsGuest is true when the user object carried is_restricted or
// is_ultra_restricted. IsExternal is Slack Connect / shared-channel.
type Member struct {
	ID          string
	DisplayName string
	Username    string
	Presence    string
	IsGuest     bool
	IsExternal  bool
}

// Result is returned by HandleKey when the user confirms a row.
type Result struct {
	UserID string
	Name   string
}

// Model is the members overlay state.
type Model struct {
	members  []Member
	filtered []int
	query    string
	selected int
	visible  bool
	loading  bool
	channel  string
}

// New constructs an empty overlay.
func New() Model {
	return Model{}
}

// SetChannel records the channel name used in the title
// ("#name · N members"). Does not re-filter.
func (m *Model) SetChannel(name string) {
	m.channel = name
}

// Channel returns the name last passed to SetChannel.
func (m Model) Channel() string { return m.channel }

// SetLoading toggles the loading placeholder. Cleared automatically
// by SetMembers.
func (m *Model) SetLoading(v bool) {
	m.loading = v
}

// Loading reports whether the overlay is waiting on a member fetch.
func (m Model) Loading() bool { return m.loading }

// SetMembers replaces the member list. If the overlay is visible the
// current query is re-applied and the selection is clamped. Loading
// is cleared — arriving members are the fetch result.
func (m *Model) SetMembers(members []Member) {
	m.members = members
	m.loading = false
	if m.visible {
		m.filter()
		if m.selected >= len(m.filtered) {
			m.selected = len(m.filtered) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
	}
}

// Members returns the unfiltered list last passed to SetMembers.
func (m Model) Members() []Member {
	return m.members
}

// Open shows the overlay, clearing the query and selection.
func (m *Model) Open() {
	m.visible = true
	m.query = ""
	m.selected = 0
	m.filter()
}

// Close hides the overlay.
func (m *Model) Close() {
	m.visible = false
}

// IsVisible reports whether the overlay is showing.
func (m Model) IsVisible() bool { return m.visible }

// Query returns the current filter string.
func (m Model) Query() string { return m.query }

// Selected returns the highlight index within the filtered list.
func (m Model) Selected() int { return m.selected }

// FilteredMembers returns the rows matching the current query, in
// display order.
func (m Model) FilteredMembers() []Member {
	out := make([]Member, 0, len(m.filtered))
	for _, idx := range m.filtered {
		out = append(out, m.members[idx])
	}
	return out
}

// listTopOffset is the box-local row of the first list row: top border
// (1) + top padding (1) + title (1) + input (1) + blank separator (1).
const listTopOffset = 5

const maxVisibleRows = 10

func boxWidth(termWidth int) int {
	w := termWidth / 2
	if w < 30 {
		w = 30
	}
	if w > 80 {
		w = 80
	}
	return w
}

func (m *Model) visibleWindow() (int, int) {
	maxVisible := maxVisibleRows
	total := len(m.filtered)
	if maxVisible > total {
		maxVisible = total
	}
	startIdx := 0
	if m.selected >= maxVisible {
		startIdx = m.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > total {
		endIdx = total
		startIdx = endIdx - maxVisible
		if startIdx < 0 {
			startIdx = 0
		}
	}
	return startIdx, endIdx
}

// BoxSize returns the outer dimensions of the rendered modal box.
func (m *Model) BoxSize(termWidth, termHeight int) (int, int) {
	start, end := m.visibleWindow()
	nRows := end - start
	if nRows < 1 {
		nRows = 1
	}
	return boxWidth(termWidth), nRows + 7
}

// ClickRow moves the selection to the list row at box-local localY
// and returns true when the click lands on a visible row.
func (m *Model) ClickRow(termWidth, termHeight, localY int) bool {
	row := localY - listTopOffset
	if row < 0 {
		return false
	}
	start, end := m.visibleWindow()
	if row >= end-start {
		return false
	}
	m.selected = start + row
	return true
}

// HandleKey processes a key event. Returns a Result when the user
// confirms a member; nil otherwise. j/k (and arrows / ctrl+n/p)
// move the highlight; other printable runes filter.
func (m *Model) HandleKey(keyStr string) *Result {
	switch keyStr {
	case "enter":
		if len(m.filtered) == 0 {
			return nil
		}
		if m.selected < 0 || m.selected >= len(m.filtered) {
			return nil
		}
		mem := m.members[m.filtered[m.selected]]
		return &Result{UserID: mem.ID, Name: mem.DisplayName}

	case "esc":
		m.Close()
		return nil

	case "j", "down", "ctrl+n":
		if m.selected < len(m.filtered)-1 {
			m.selected++
		}
		return nil

	case "k", "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
		}
		return nil

	case "backspace":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.selected = 0
			m.filter()
		}
		return nil
	}

	if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
		m.query += keyStr
		m.selected = 0
		m.filter()
	}
	return nil
}

func (m *Model) filter() {
	m.filtered = m.filtered[:0]
	q := text.Fold(m.query)

	if q == "" {
		idxs := make([]int, len(m.members))
		for i := range m.members {
			idxs[i] = i
		}
		sort.SliceStable(idxs, func(i, j int) bool {
			return m.lessNoQuery(idxs[i], idxs[j])
		})
		m.filtered = idxs
		return
	}

	type match struct {
		idx  int
		tier int
	}
	var matches []match
	for i, mem := range m.members {
		tier, ok := matchTier(mem, q)
		if !ok {
			continue
		}
		matches = append(matches, match{idx: i, tier: tier})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].tier != matches[j].tier {
			return matches[i].tier < matches[j].tier
		}
		return m.lessNoQuery(matches[i].idx, matches[j].idx)
	})
	for _, mm := range matches {
		m.filtered = append(m.filtered, mm.idx)
	}
}

func (m *Model) lessNoQuery(ai, bi int) bool {
	a, b := m.members[ai], m.members[bi]
	an := strings.ToLower(a.DisplayName)
	bn := strings.ToLower(b.DisplayName)
	if an != bn {
		return an < bn
	}
	return a.ID < b.ID
}

// matchTier ranks mem against an already-folded query. 0 = prefix on
// display name or username, 1 = substring, 2 = subsequence.
func matchTier(mem Member, q string) (int, bool) {
	name := text.Fold(mem.DisplayName)
	handle := text.Fold(mem.Username)
	if strings.HasPrefix(name, q) || strings.HasPrefix(handle, q) {
		return 0, true
	}
	if strings.Contains(name, q) || strings.Contains(handle, q) {
		return 1, true
	}
	if isSubsequence(name, q) || isSubsequence(handle, q) {
		return 2, true
	}
	return 0, false
}

func isSubsequence(s, q string) bool {
	qi := 0
	qrunes := []rune(q)
	if len(qrunes) == 0 {
		return true
	}
	for _, r := range s {
		if qi >= len(qrunes) {
			break
		}
		if r == qrunes[qi] {
			qi++
		}
	}
	return qi == len(qrunes)
}
