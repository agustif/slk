package activityview

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func reactionItem() Item {
	return Item{
		ActivityItem: slackclient.ActivityItem{
			Key: "r1", Type: "message_reaction", Unread: false,
			ChannelID: "C1", MessageTS: "1.0", FeedTS: "1.0",
			ActorID: "U1", Reaction: "eyes",
		},
		ChannelName: "eng", ChannelType: "channel", ActorName: "Alice",
		ParentText: "hello parent",
	}
}

func TestView_ReactionRendersEyesGlyph(t *testing.T) {
	m := New()
	m.SetItems([]Item{reactionItem()})
	out := ansi.Strip(m.View(16, 70))
	if !strings.Contains(out, "👀") {
		t.Errorf("detailed reaction card missing eyes glyph:\n%s", out)
	}
	if !strings.Contains(out, "reacted") {
		t.Errorf("detailed reaction card missing 'reacted':\n%s", out)
	}
	if !strings.Contains(out, "hello parent") {
		t.Errorf("detailed reaction card missing parent quote:\n%s", out)
	}
	if strings.Contains(out, "Reacted :eyes:") {
		t.Errorf("should not print shortcode Reacted :eyes::\n%s", out)
	}
}

func TestView_CustomEmojiFallback(t *testing.T) {
	m := New()
	it := reactionItem()
	it.Reaction = "not-a-real-emoji"
	m.SetItems([]Item{it})
	out := ansi.Strip(m.View(16, 70))
	if !strings.Contains(out, ":not-a-real-emoji:") {
		t.Errorf("custom/unknown shortcode should fall back to :name::\n%s", out)
	}
}

func TestView_CompactReactionLayout(t *testing.T) {
	m := New()
	m.SetDensity(config.ActivityDensityCompact)
	m.SetItems([]Item{reactionItem()})
	out := ansi.Strip(m.View(12, 70))
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "👀") || !strings.Contains(out, "eng") {
		t.Errorf("compact reaction missing Alice/eyes/#eng:\n%s", out)
	}
	if !strings.Contains(out, "hello parent") {
		t.Errorf("compact reaction missing quote:\n%s", out)
	}
}

func TestClickAt_ReactionVsBody(t *testing.T) {
	m := New()
	m.SetItems([]Item{reactionItem()})
	out := m.View(16, 70)
	lines := strings.Split(out, "\n")
	if len(lines) <= toolbarLines {
		t.Fatalf("too few lines:\n%s", out)
	}
	header := ansi.Strip(lines[toolbarLines])
	idx := strings.Index(header, "👀")
	if idx < 0 {
		t.Fatalf("eyes glyph not in header %q", header)
	}
	col := lipgloss.Width(header[:idx])
	if kind := m.ClickAt(toolbarLines, col); kind != ClickReaction {
		t.Errorf("click on eyes = %v, want ClickReaction", kind)
	}
	if kind := m.ClickAt(toolbarLines, 2); kind != ClickItem {
		t.Errorf("click on body = %v, want ClickItem", kind)
	}
}

func TestHitTestReaction_MatchesClickAt(t *testing.T) {
	m := New()
	m.SetItems([]Item{reactionItem()})
	_ = m.View(16, 70)
	if len(m.reactionHits) == 0 {
		t.Fatal("expected a reaction hitbox after View")
	}
	h := m.reactionHits[0]
	row := toolbarLines + h.absLine
	emoji, ok := m.HitTestReaction(row, h.x0)
	if !ok || emoji != "eyes" {
		t.Errorf("HitTestReaction(%d,%d) = (%q,%v), want eyes", row, h.x0, emoji, ok)
	}
	if _, ok := m.HitTestReaction(row, 2); ok {
		t.Error("body column should not be a reaction hit")
	}
}

func TestHandleEmojiImageReady_Dirties(t *testing.T) {
	m := New()
	v := m.Version()
	m.HandleEmojiImageReady("https://example.com/x.png")
	if m.Version() == v {
		t.Error("HandleEmojiImageReady must dirty so the panel cache does not keep blank holes")
	}
	v = m.Version()
	m.SetEmojiContext(EmojiContext{Cells: 2})
	if m.Version() == v {
		t.Error("SetEmojiContext must dirty")
	}
	v = m.Version()
	m.SetEmojiCustoms(map[string]string{"partyparrot": "https://e.example/p.gif"})
	if m.Version() == v {
		t.Error("SetEmojiCustoms must dirty")
	}
}

func TestApplyReaction_TogglesHasReacted(t *testing.T) {
	m := New()
	m.SetItems([]Item{reactionItem()})
	m.ApplyReaction("C1", "1.0", "eyes", true, false)
	it, _ := m.SelectedItem()
	if !it.HasReacted || !it.ReactionsKnown {
		t.Errorf("after add: HasReacted=%v ReactionsKnown=%v", it.HasReacted, it.ReactionsKnown)
	}
	m.ApplyReaction("C1", "1.0", "eyes", true, true)
	it, _ = m.SelectedItem()
	if it.HasReacted {
		t.Error("after remove: HasReacted still true")
	}
}
