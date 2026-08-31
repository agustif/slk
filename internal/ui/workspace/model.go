package workspace

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/agustif/slk/internal/ui/styles"
)

type WorkspaceItem struct {
	ID        string
	Name      string
	Initials  string
	IconURL   string
	HasUnread bool
}

// LogoFunc returns a rendered 4×2 team logo for teamID, triggering a
// lazy fetch via iconURL on cache miss. Empty string means "not ready
// yet; show initials".
type LogoFunc func(teamID, iconURL string) string

type Model struct {
	items    []WorkspaceItem
	selected int
	version  int64
	logoFn   LogoFunc
	// unreadReader returns the set of workspace IDs that currently
	// have at least one channel with has_unread=true. Set by App via
	// SetUnreadReader; called by RefreshUnreads.
	unreadReader func() []string
}

const (
	railWidth   = 6
	tileRows    = 2
	tileGapRows = 1
	tilePadRows = 1
	tileStride  = tileRows + tileGapRows
)

// Version returns a counter that increments any time the View() output could
// change.
func (m *Model) Version() int64 { return m.version }

func (m *Model) dirty() { m.version++ }

func New(items []WorkspaceItem, selected int) Model {
	return Model{items: items, selected: selected}
}

func (m *Model) SelectedID() string {
	if len(m.items) == 0 {
		return ""
	}
	return m.items[m.selected].ID
}

// NameByID returns the workspace's display name, or "" if no item with
// the given ID is present. Used by the App to derive the window title's
// initials via WorkspaceInitials.
func (m *Model) NameByID(id string) string {
	for _, item := range m.items {
		if item.ID == id {
			return item.Name
		}
	}
	return ""
}

// OtherUnreadCount returns the number of workspaces with unreads,
// excluding activeID. Reads through the installed unreadReader; returns
// 0 when no reader is set. Does not filter mute -- matches the rail
// dot's existing semantics so the title's "+N" and the rail dots agree.
func (m *Model) OtherUnreadCount(activeID string) int {
	if m.unreadReader == nil {
		return 0
	}
	count := 0
	for _, id := range m.unreadReader() {
		if id != activeID {
			count++
		}
	}
	return count
}

func (m *Model) SelectedIndex() int {
	return m.selected
}

func (m *Model) Select(idx int) {
	if idx >= 0 && idx < len(m.items) && m.selected != idx {
		m.selected = idx
		m.dirty()
	}
}

func (m *Model) SetItems(items []WorkspaceItem) {
	m.items = items
	if m.selected >= len(items) {
		m.selected = 0
	}
	m.dirty()
}

// SetLogoFunc wires the lazy team-logo renderer. Initials remain the
// fallback until the fetch lands.
func (m *Model) SetLogoFunc(fn LogoFunc) {
	m.logoFn = fn
	m.dirty()
}

// LogoFunc returns the current team-logo renderer (may be nil).
func (m *Model) LogoFunc() LogoFunc { return m.logoFn }

// HandleLogoReady invalidates the rail when that team's logo lands.
func (m *Model) HandleLogoReady(teamID string) {
	if teamID == "" {
		return
	}
	for _, it := range m.items {
		if it.ID == teamID {
			m.dirty()
			return
		}
	}
}

func (m *Model) SelectByID(teamID string) {
	for i, item := range m.items {
		if item.ID == teamID {
			if m.selected != i {
				m.selected = i
				m.dirty()
			}
			return
		}
	}
}

func (m *Model) SetUnread(teamID string, hasUnread bool) {
	for i := range m.items {
		if m.items[i].ID == teamID {
			if m.items[i].HasUnread != hasUnread {
				m.items[i].HasUnread = hasUnread
				m.dirty()
			}
			return
		}
	}
}

// SetUnreadReader installs the callback used by RefreshUnreads.
func (m *Model) SetUnreadReader(f func() []string) {
	m.unreadReader = f
}

// RefreshUnreads pulls the latest set of workspaces-with-unreads from
// the reader and updates each item's HasUnread field. Called by App
// on ReadStateChangedMsg. No-op if no reader is installed.
func (m *Model) RefreshUnreads() {
	if m.unreadReader == nil {
		return
	}
	set := make(map[string]bool, len(m.items))
	for _, id := range m.unreadReader() {
		set[id] = true
	}
	changed := false
	for i := range m.items {
		want := set[m.items[i].ID]
		if m.items[i].HasUnread != want {
			m.items[i].HasUnread = want
			changed = true
		}
	}
	if changed {
		m.dirty()
	}
}

func (m Model) View(height int) string {
	if len(m.items) == 0 {
		return ""
	}

	var tiles []string
	for i, item := range m.items {
		var style lipgloss.Style
		if i == m.selected {
			style = styles.WorkspaceActive
		} else {
			style = styles.WorkspaceInactive
		}

		gutter := " "
		if item.HasUnread && i != m.selected {
			gutter = styles.PresenceOnline.Render("●")
		}
		l0, l1 := item.tileLines(m.logoFn)
		tiles = append(tiles, style.Render(gutter+l0)+"\n"+style.Render(gutter+l1))
	}

	content := strings.Join(tiles, "\n\n")

	// Height/MaxHeight in lipgloss include padding in the total,
	// so use the full height directly. Padding(1,0) adds 1 row
	// top + 1 row bottom inside that total, matching the visual
	// offset of RoundedBorder() on adjacent panels.
	rail := lipgloss.NewStyle().
		Width(railWidth).
		Height(height).
		MaxHeight(height).
		Background(styles.RailBackground).
		Padding(tilePadRows, 0).
		Align(lipgloss.Center).
		Render(content)

	return rail
}

func (item WorkspaceItem) tileLines(logoFn LogoFunc) (line0, line1 string) {
	if logoFn != nil && item.IconURL != "" {
		if s := logoFn(item.ID, item.IconURL); s != "" {
			parts := strings.SplitN(s, "\n", 3)
			line0 = parts[0]
			if len(parts) > 1 {
				line1 = parts[1]
			}
			if line1 == "" {
				line1 = strings.Repeat(" ", lipgloss.Width(line0))
			}
			return line0, line1
		}
	}
	ini := item.Initials
	if ini == "" {
		ini = "?"
	}
	return ini, strings.Repeat(" ", lipgloss.Width(ini))
}

// ClickAt returns the workspace item rendered at rail-local row y,
// or ok=false when the click landed on a padding row, a gap between
// items, or past the last item.
//
// Row layout mirrors View(): Padding(1,0) puts blank padding at row 0.
// Each tile is two content rows (logo or initials + spacer) joined by
// one gap row, so item 0 occupies y=1,2, item 1 occupies y=4,5, etc.
func (m Model) ClickAt(y int) (WorkspaceItem, bool) {
	if y < tilePadRows || len(m.items) == 0 {
		return WorkspaceItem{}, false
	}
	rel := y - tilePadRows
	if rel < 0 {
		return WorkspaceItem{}, false
	}
	idx := rel / tileStride
	off := rel % tileStride
	if off >= tileRows {
		return WorkspaceItem{}, false // gap between tiles
	}
	if idx < 0 || idx >= len(m.items) {
		return WorkspaceItem{}, false
	}
	return m.items[idx], true
}

func (m Model) Width() int {
	return railWidth // 6 content, no border
}

func WorkspaceInitials(name string) string {
	words := strings.Fields(name)
	switch len(words) {
	case 0:
		return "?"
	case 1:
		if len(words[0]) >= 2 {
			return strings.ToUpper(words[0][:2])
		}
		return strings.ToUpper(words[0])
	default:
		return strings.ToUpper(fmt.Sprintf("%c%c", words[0][0], words[1][0]))
	}
}
