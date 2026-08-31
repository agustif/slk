package sidebar

import (
	"strconv"
	"strings"

	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/config"
	"github.com/gammons/slk/internal/text"
)

func (m *Model) SetSort(sort config.SidebarSort, vip []string) {
	m.sortCfg = sort.Normalized()
	m.vip = append([]string(nil), vip...)
	m.rebuildFilter()
	m.rebuildNavPreserveCursor()
	m.cacheValid = false
	m.dirty()
}

// SetGroupDMs sets [sidebar].group_dms: split (default) or together.
// DMSnippet is the last-message card for the Direct Messages tab.
type DMSnippet struct {
	Text     string
	UserID   string
	Activity int64
}

// ApplyDMSnippets merges last-message text and activity into the
// matching rows and re-sorts. Survives later SetItems.
func (m *Model) ApplyDMSnippets(snips map[string]DMSnippet) {
	if len(snips) == 0 {
		return
	}
	if m.snippets == nil {
		m.snippets = map[string]DMSnippet{}
	}
	for id, s := range snips {
		prev := m.snippets[id]
		if s.Text == "" {
			s.Text = prev.Text
		}
		if s.UserID == "" {
			s.UserID = prev.UserID
		}
		if s.Activity < prev.Activity {
			s.Activity = prev.Activity
		}
		m.snippets[id] = s
	}
	m.applySnippetsToItems()
	m.rebuildFilter()
	m.rebuildNavPreserveCursor()
	m.cacheValid = false
	m.dirty()
}

func (m *Model) applySnippetsToItems() {
	if len(m.snippets) == 0 {
		return
	}
	for i := range m.items {
		s, ok := m.snippets[m.items[i].ID]
		if !ok {
			continue
		}
		if s.Text != "" {
			m.items[i].Preview = s.Text
			m.items[i].PreviewUserID = s.UserID
		}
		if s.Activity > m.items[i].LastActivity {
			m.items[i].LastActivity = s.Activity
		}
	}
}

// DMChannelIDs returns 1:1, group, and app DM ids (unfiltered).
func (m *Model) DMChannelIDs() []string {
	var ids []string
	for _, it := range m.items {
		if it.Type == "dm" || it.Type == "group_dm" || it.Type == "app" {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

func (m *Model) SetGroupDMs(mode string) {
	together := config.ClampGroupDMs(mode) == config.GroupDMsTogether
	if m.groupDMsTogether == together {
		return
	}
	m.groupDMsTogether = together
	m.rebuildFilter()
	m.rebuildNavPreserveCursor()
	m.cacheValid = false
	m.dirty()
}

// TouchVisit records that the user opened channelID and re-sorts
// so a "recent" pipeline moves the row without a full SetItems.
func (m *Model) TouchVisit(channelID string, ts int64) {
	if channelID == "" {
		return
	}
	for i := range m.items {
		if m.items[i].ID == channelID {
			m.items[i].LastVisited = ts
			m.rebuildFilter()
			m.rebuildNavPreserveCursor()
			m.cacheValid = false
			m.dirty()
			return
		}
	}
}

func (m *Model) sortKeysFor(item ChannelItem) []string {
	if m.dmsView {
		return []string{config.SortAtomUnreadFirst, config.SortAtomRecent, config.SortAtomAlphabetical}
	}
	name, typ := m.sectionSortIdentity(item)
	return m.sortCfg.Pipeline(name, typ)
}

func (m *Model) sectionSortIdentity(item ChannelItem) (name, typ string) {
	key := m.sectionFor(item)
	if m.useSlackSections() {
		for _, meta := range m.sectionsProvider.OrderedSlackSections() {
			if meta.ID == key {
				return meta.Name, meta.Type
			}
		}
	}
	switch key {
	case defaultDMSection, defaultGroupDMSection:
		return key, "direct_messages"
	case defaultChannelsSection:
		return key, "channels"
	case defaultAppsSection:
		return key, "recent_apps"
	default:
		return key, "standard"
	}
}

func (m *Model) lessInSection(ia, ib int, keys []string, readState map[string]cache.ReadState) bool {
	a, b := m.items[ia], m.items[ib]
	for _, atom := range keys {
		switch c := m.cmpAtom(atom, a, b, ia, ib, readState); {
		case c < 0:
			return true
		case c > 0:
			return false
		}
	}
	return ia < ib
}

func (m *Model) cmpAtom(atom string, a, b ChannelItem, ia, ib int, readState map[string]cache.ReadState) int {
	switch atom {
	case config.SortAtomVIPFirst:
		va := a.IsVIP || config.VIPMatch(m.vip, a.ID, a.DMUserID, a.Name)
		vb := b.IsVIP || config.VIPMatch(m.vip, b.ID, b.DMUserID, b.Name)
		if va != vb {
			if va {
				return -1
			}
			return 1
		}
	case config.SortAtomUnreadFirst:
		ua := a.IsVisiblyUnread(readState[a.ID])
		ub := b.IsVisiblyUnread(readState[b.ID])
		if ua != ub {
			if ua {
				return -1
			}
			return 1
		}
	case config.SortAtomStarredFirst:
		if a.IsStarred != b.IsStarred {
			if a.IsStarred {
				return -1
			}
			return 1
		}
	case config.SortAtomAlphabetical:
		fa, fb := text.Fold(a.Name), text.Fold(b.Name)
		if fa != fb {
			return strings.Compare(fa, fb)
		}
	case config.SortAtomRecent:
		ra, rb := recency(a, readState[a.ID]), recency(b, readState[b.ID])
		if ra != rb {
			if ra > rb {
				return -1
			}
			return 1
		}
	case config.SortAtomSlack:
		oa, ob := a.ChannelOrder, b.ChannelOrder
		if (oa > 0) != (ob > 0) {
			if oa > 0 {
				return -1
			}
			return 1
		}
		if oa != ob {
			if oa < ob {
				return -1
			}
			return 1
		}
		if ia != ib {
			if ia < ib {
				return -1
			}
			return 1
		}
	}
	return 0
}

func recency(item ChannelItem, state cache.ReadState) int64 {
	r := item.LastVisited
	if item.LastActivity > r {
		r = item.LastActivity
	}
	if state.LastReadTS == "" {
		return r
	}
	f, err := strconv.ParseFloat(state.LastReadTS, 64)
	if err != nil {
		return r
	}
	if int64(f) > r {
		return int64(f)
	}
	return r
}
