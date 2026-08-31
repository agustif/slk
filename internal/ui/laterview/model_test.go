package laterview

import (
	"strings"
	"testing"
	"time"

	"github.com/agustif/slk/internal/config"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/styles"
)

func init() {
	styles.Apply("nord", config.Theme{})
}

func sampleItems() []Item {
	return []Item{
		{SavedItem: slackclient.SavedItem{
			Key: "C1\t1.0", ItemID: "C1", ItemType: "message", TS: "1.0",
			State: "in_progress", DateCreated: time.Now().Unix() - 120,
		}, ChannelName: "general", ChannelType: "channel", AuthorName: "bob", Preview: "ship the build tonight"},
		{SavedItem: slackclient.SavedItem{
			Key: "D1\t2.0", ItemID: "D1", ItemType: "message", TS: "2.0",
			State: "in_progress", DateCreated: time.Now().Unix() - 3600,
			DateDue: time.Now().Unix() + 600,
		}, ChannelName: "alice", ChannelType: "dm", Preview: "can you review this"},
	}
}

func TestSetItemsPreservesSelectionByKey(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	m.MoveDown()
	it, _ := m.SelectedItem()
	if it.Key != "D1\t2.0" {
		t.Fatalf("precondition: selected %q", it.Key)
	}
	reranked := []Item{sampleItems()[1], sampleItems()[0]}
	m.SetItems(reranked)
	got, _ := m.SelectedItem()
	if got.Key != "D1\t2.0" {
		t.Errorf("selection should follow key, got %q", got.Key)
	}
}

func TestView_RendersLaterAndHint(t *testing.T) {
	m := New()
	m.SetBadge(2)
	m.SetItems(sampleItems())
	out := m.View(12, 70)
	for _, want := range []string{"In progress", "Completed", "Archived", "general", "alice", "enter open", "f/F", "ship the build tonight", "can you review this"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "•2") {
		t.Errorf("badge missing:\n%s", out)
	}
}

func TestClickAtSelectsCard(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	// toolbar = 3 lines; first card at y=3, second card at y=3+4=7
	if m.ClickAt(7, 2) != ClickItem {
		t.Fatal("click on second card should hit")
	}
	it, _ := m.SelectedItem()
	if it.TS != "2.0" {
		t.Errorf("selected ts = %q, want 2.0", it.TS)
	}
}

func TestCycleFilterAndClickTab(t *testing.T) {
	m := New()
	if m.Filter() != FilterInProgress {
		t.Fatalf("default filter = %q", m.Filter())
	}
	if !m.CycleFilter(1) || m.Filter() != FilterCompleted {
		t.Fatalf("f → %q, want completed", m.Filter())
	}
	if !m.CycleFilter(-1) || m.Filter() != FilterInProgress {
		t.Fatalf("F → %q, want saved", m.Filter())
	}
	m.SetItems(sampleItems())
	_ = m.View(10, 80)
	hits := m.tabHits
	if len(hits) < 2 {
		t.Fatalf("tabHits = %d, want >= 2", len(hits))
	}
	if kind := m.ClickAt(0, (hits[1].x0+hits[1].x1)/2); kind != ClickTab {
		t.Fatalf("click Completed tab = %v, want ClickTab", kind)
	}
	if m.Filter() != FilterCompleted {
		t.Errorf("filter = %q, want completed", m.Filter())
	}
}

func TestPreview_UsesMessageTextNotPlaceholder(t *testing.T) {
	m := New()
	m.SetItems([]Item{{
		SavedItem:   slackclient.SavedItem{Key: "C1\t1.0", ItemID: "C1", ItemType: "message", TS: "1.0"},
		ChannelName: "eng",
		Preview:     "actual saved body",
	}})
	out := m.View(10, 60)
	if !strings.Contains(out, "actual saved body") {
		t.Errorf("want message body, got:\n%s", out)
	}
	if strings.Contains(out, "saved for later") {
		t.Errorf("placeholder should not render when preview is set:\n%s", out)
	}
}

func TestSetPage_AppendAndLoadMore(t *testing.T) {
	m := New()
	m.SetPage(sampleItems(), "next-page", false)
	if !m.HasMore() || m.NextCursor() != "next-page" {
		t.Fatalf("cursor = %q", m.NextCursor())
	}
	m.GoToBottom()
	if !m.LoadMoreSelected() {
		t.Fatal("G should land on load more")
	}
	m.SetPage([]Item{{
		SavedItem: slackclient.SavedItem{Key: "C9\t9.0", ItemID: "C9", ItemType: "message", TS: "9.0"},
	}}, "", true)
	if m.HasMore() {
		t.Fatal("empty cursor should clear has-more")
	}
	if n := len(m.Items()); n != 3 {
		t.Fatalf("len = %d, want 3", n)
	}
	out := m.View(14, 60)
	if strings.Contains(out, "load more") {
		t.Errorf("load more should be gone:\n%s", out)
	}
}

func TestEmptyState(t *testing.T) {
	m := New()
	out := m.View(8, 40)
	if !strings.Contains(out, "nothing saved for later") {
		t.Errorf("empty view = %q", out)
	}
}
