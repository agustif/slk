package sidebar

import (
	"testing"

	"github.com/gammons/slk/internal/cache"
	"github.com/gammons/slk/internal/config"
)

func filteredIDs(m Model) []string {
	out := make([]string, 0, len(m.filtered))
	for _, idx := range m.filtered {
		out = append(out, m.items[idx].ID)
	}
	return out
}

func TestSort_SlackIsVIPUnionsConfig(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "dave", Type: "dm", DMUserID: "U1"},
		{ID: "D2", Name: "alice", Type: "dm", DMUserID: "U2", IsVIP: true},
		{ID: "D3", Name: "carol", Type: "dm", DMUserID: "U3"},
	})
	m.SetSort(config.SidebarSort{DMs: []string{"vip_first", "alphabetical"}}, []string{"carol"})
	got := filteredIDs(m)
	want := []string{"D2", "D3", "D1"} // alice (Slack VIP), carol (config), dave
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSort_VIPThenAlphabetical(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "dave", Type: "dm", DMUserID: "U1"},
		{ID: "D2", Name: "alice", Type: "dm", DMUserID: "U2"},
		{ID: "D3", Name: "carol", Type: "dm", DMUserID: "U3"},
		{ID: "D4", Name: "bob", Type: "dm", DMUserID: "U4"},
	})
	m.SetSort(config.SidebarSort{DMs: []string{"vip_first", "alphabetical"}}, []string{"carol", "alice"})
	got := filteredIDs(m)
	want := []string{"D2", "D3", "D4", "D1"} // alice, carol | bob, dave
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pos %d: %s want %s (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestSort_VIPThenRecent(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastVisited: 10},
		{ID: "D2", Name: "bob", Type: "dm", LastVisited: 30},
		{ID: "D3", Name: "carol", Type: "dm", LastVisited: 20},
		{ID: "D4", Name: "dave", Type: "dm", LastVisited: 40},
	})
	m.SetSort(config.SidebarSort{DMs: []string{"vip_first", "recent"}}, []string{"alice", "bob"})
	got := filteredIDs(m)
	want := []string{"D2", "D1", "D4", "D3"} // bob(30), alice(10) | dave(40), carol(20)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSort_DefaultDMsRecent(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 10},
		{ID: "D2", Name: "bob", Type: "dm", LastActivity: 30},
		{ID: "D3", Name: "carol", Type: "dm", LastActivity: 20},
	})
	got := filteredIDs(m)
	want := []string{"D2", "D3", "D1"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSplitGroupDMs_SeparateSections(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 1},
		{ID: "G1", Name: "jamari, lion", Type: "group_dm", LastActivity: 99},
		{ID: "C1", Name: "general", Type: "channel"},
	})
	m.ToggleCollapse("Channels")
	got := orderedSections(m.items, m.filtered)
	want := []string{"Direct Messages", "Group DMs", "Channels"}
	if len(got) != len(want) {
		t.Fatalf("sections %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sections %v want %v", got, want)
		}
	}
	if m.sectionFor(m.items[0]) != defaultDMSection {
		t.Errorf("1:1 → %q", m.sectionFor(m.items[0]))
	}
	if m.sectionFor(m.items[1]) != defaultGroupDMSection {
		t.Errorf("group DM → %q", m.sectionFor(m.items[1]))
	}
}

func TestTogetherGroupDMs_OneSection(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 1},
		{ID: "G1", Name: "jamari, lion", Type: "group_dm", LastActivity: 99},
	})
	m.SetGroupDMs(config.GroupDMsTogether)
	if m.sectionFor(m.items[1]) != defaultDMSection {
		t.Errorf("together group DM → %q, want Direct Messages", m.sectionFor(m.items[1]))
	}
	got := filteredIDs(m)
	// Same section, recent: G1 (99) then D1 (1).
	want := []string{"G1", "D1"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSort_AlphabeticalIgnoresChannelOrder(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "C1", Name: "zeta", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 1},
		{ID: "C2", Name: "alpha", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 2},
		{ID: "C3", Name: "mu", Type: "channel", Section: "Eng", SectionOrder: 1},
	})
	m.SetSort(config.SidebarSort{Section: map[string][]string{"Eng": {"alphabetical"}}}, nil)
	got := filteredIDs(m)
	want := []string{"C2", "C3", "C1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSort_DefaultSlackPreservesChannelOrder(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "C1", Name: "alpha", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 0},
		{ID: "C2", Name: "beta", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 2},
		{ID: "C3", Name: "gamma", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 0},
		{ID: "C4", Name: "delta", Type: "channel", Section: "Eng", SectionOrder: 1, ChannelOrder: 1},
	})
	got := filteredIDs(m)
	want := []string{"C4", "C2", "C1", "C3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("default slack got %v want %v", got, want)
		}
	}
}

func TestSort_UnreadFirst(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "C1", Name: "alpha", Type: "channel", Section: "Eng", SectionOrder: 1},
		{ID: "C2", Name: "beta", Type: "channel", Section: "Eng", SectionOrder: 1},
		{ID: "C3", Name: "gamma", Type: "channel", Section: "Eng", SectionOrder: 1},
	})
	m.SetReadStateReader(func() map[string]cache.ReadState {
		return map[string]cache.ReadState{
			"C2": {HasUnread: true, LastReadTS: "1.0"},
			"C3": {HasUnread: true, LastReadTS: "1.0"},
		}
	})
	m.SetSort(config.SidebarSort{Section: map[string][]string{"Eng": {"unread_first", "alphabetical"}}}, nil)
	got := filteredIDs(m)
	want := []string{"C2", "C3", "C1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestTouchVisitReordersRecent(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastVisited: 10},
		{ID: "D2", Name: "bob", Type: "dm", LastVisited: 20},
	})
	m.SetSort(config.SidebarSort{DMs: []string{"recent"}}, nil)
	if got := filteredIDs(m); got[0] != "D2" {
		t.Fatalf("before touch %v", got)
	}
	m.TouchVisit("D1", 99)
	if got := filteredIDs(m); got[0] != "D1" {
		t.Fatalf("after touch %v", got)
	}
}
