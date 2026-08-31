package activityview

import (
	"strings"
	"testing"

	"github.com/gammons/slk/internal/config"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/styles"
)

func init() {
	styles.Apply("nord", config.Theme{})
}

func sampleItems() []Item {
	return []Item{
		{ActivityItem: slackclient.ActivityItem{
			Key: "k1", Type: "channel", Unread: true,
			ChannelID: "C1", MessageTS: "1.0", FeedTS: "1.0",
		}, ChannelName: "general", ChannelType: "channel"},
		{ActivityItem: slackclient.ActivityItem{
			Key: "k2", Type: "dm", Unread: false,
			ChannelID: "D1", MessageTS: "2.0", FeedTS: "2.0",
		}, ChannelName: "alice", ChannelType: "dm"},
	}
}

func TestNew_DefaultsMatchConfig(t *testing.T) {
	m := New()
	q := m.Query()
	if q.Filter != config.ActivityFilterAll || q.Sort != config.ActivitySortNewest || q.UnreadOnly {
		t.Errorf("Query = %+v; want all/newest/false", q)
	}
	if m.Density() != config.ActivityDensityDetailed {
		t.Errorf("Density = %q", m.Density())
	}
}

func TestCycleFilterAndSort(t *testing.T) {
	m := New()
	if !m.CycleFilter(1) || m.Filter() != config.ActivityFilterDMs {
		t.Fatalf("f → %q, want dms", m.Filter())
	}
	if !m.CycleFilter(-1) || m.Filter() != config.ActivityFilterAll {
		t.Fatalf("F → %q, want all", m.Filter())
	}
	if !m.CycleSort() || m.Sort() != config.ActivitySortUnreadsFirst {
		t.Fatalf("s → %q", m.Sort())
	}
	if !m.ToggleUnreadOnly() || !m.UnreadOnly() {
		t.Fatal("u should turn unread-only on")
	}
}

func TestSetItemsPreservesSelectionByKey(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	m.MoveDown()
	it, _ := m.SelectedItem()
	if it.Key != "k2" {
		t.Fatalf("precondition: selected %q", it.Key)
	}
	reranked := []Item{sampleItems()[1], sampleItems()[0]}
	m.SetItems(reranked)
	got, _ := m.SelectedItem()
	if got.Key != "k2" {
		t.Errorf("selection should follow k2, got %q", got.Key)
	}
}

func TestView_RendersTabsAndHint(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	out := m.View(12, 70)
	for _, want := range []string{"All", "DMs", "Mentions", "Threads", "unread", "newest", "f/F tab"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "general") {
		t.Errorf("View missing channel name:\n%s", out)
	}
}

func TestSetViews_AddsCustomTabs(t *testing.T) {
	m := New()
	m.SetViews([]slackclient.ActivityView{
		{ID: "all", Name: "All", Type: "all", Sort: "newest"},
		{ID: "custom-unreads", Name: "Unreads", Type: "custom", Sort: "newest", Filters: slackclient.ActivityViewFilters{UnreadOnly: true}},
		{ID: "custom-vip", Name: "VIP", Type: "custom", Sort: "newest", Filters: slackclient.ActivityViewFilters{PriorityOnly: true}},
	})
	out := m.View(8, 70)
	if !strings.Contains(out, "Unreads") || !strings.Contains(out, "VIP") {
		t.Errorf("custom tabs missing:\n%s", out)
	}
	m.CycleFilter(1)
	q := m.Query()
	if !q.UnreadOnly || q.PriorityOnly {
		t.Errorf("Unreads tab query = %+v", q)
	}
	m.CycleFilter(1)
	q = m.Query()
	if !q.PriorityOnly {
		t.Errorf("VIP tab query = %+v", q)
	}
}

func TestView_ActiveFilterHighlighted(t *testing.T) {
	m := New()
	m.SetQuery(config.ActivityFilterMentions, config.ActivitySortNewest, false)
	out := m.View(8, 70)
	if !strings.Contains(out, "Mentions") {
		t.Fatalf("missing Mentions tab:\n%s", out)
	}
}

func TestClickAt_ToolbarFilter(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	_ = m.View(12, 70) // populate hitboxes
	// First tab is "All" at x=0; "DMs" follows two spaces after "All" (3).
	kind := m.ClickAt(0, 5)
	if kind != ClickControls {
		t.Fatalf("click DMs tab = %v, want ClickControls", kind)
	}
	if m.Filter() != config.ActivityFilterDMs && m.Filter() != "dms" {
		t.Errorf("Filter = %q, want dms", m.Filter())
	}
}

func TestClickAt_CardSelects(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	_ = m.View(16, 50)
	// toolbarLines=3; detailed card 1 starts at line 3+4=7
	kind := m.ClickAt(3+cardStride, 2)
	if kind != ClickItem {
		t.Fatalf("click card = %v, want ClickItem", kind)
	}
	it, ok := m.SelectedItem()
	if !ok || it.Key != "k2" {
		t.Errorf("selected %+v ok=%v, want k2", it, ok)
	}
}

func TestClickAt_BlankRowIsNone(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	_ = m.View(12, 50)
	if kind := m.ClickAt(2, 2); kind != ClickNone {
		t.Errorf("blank toolbar row click = %v, want ClickNone", kind)
	}
}

func TestCompactDensity_OneLinePerItem(t *testing.T) {
	m := New()
	m.SetDensity(config.ActivityDensityCompact)
	m.SetItems(sampleItems())
	kind := m.ClickAt(toolbarLines+1, 1)
	if kind != ClickItem {
		t.Fatalf("compact click = %v", kind)
	}
	it, _ := m.SelectedItem()
	if it.Key != "k2" {
		t.Errorf("compact index 1 = %q, want k2", it.Key)
	}
}

func TestMoveDownClamps(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	m.MoveDown()
	m.MoveDown()
	if m.SelectedIndex() != 1 {
		t.Errorf("SelectedIndex = %d, want 1", m.SelectedIndex())
	}
}

func TestEmptyState(t *testing.T) {
	m := New()
	out := m.View(10, 40)
	if !strings.Contains(out, "no activity") {
		t.Errorf("empty view:\n%s", out)
	}
	m.SetLoading(true)
	out = m.View(10, 40)
	if !strings.Contains(out, "loading activity") {
		t.Errorf("loading view:\n%s", out)
	}
}
