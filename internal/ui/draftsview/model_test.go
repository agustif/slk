package draftsview

import (
	"strings"
	"testing"
	"time"

	"github.com/gammons/slk/internal/config"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/styles"
)

func init() {
	styles.Apply("nord", config.Theme{})
}

func sampleDrafts() []Item {
	return []Item{
		ItemFromDraft(slackclient.ComposerDraft{
			ID: "Dr1", ChannelID: "C1", Text: "ship the build", LastUpdatedTS: "1.0",
			DateCreated: time.Now().Unix() - 120,
		}),
		ItemFromDraft(slackclient.ComposerDraft{
			ID: "Dr2", ChannelID: "D1", ThreadTS: "9.0", Text: "thread reply draft", LastUpdatedTS: "2.0",
			DateCreated: time.Now().Unix() - 3600,
		}),
	}
}

func TestSetPagePreservesSelectionByID(t *testing.T) {
	m := New()
	items := sampleDrafts()
	items[0].ChannelName = "general"
	items[1].ChannelName = "alice"
	m.SetItems(items)
	m.MoveDown()
	it, _ := m.SelectedItem()
	if it.ID != "Dr2" {
		t.Fatalf("precondition: selected %q", it.ID)
	}
	reranked := []Item{items[1], items[0]}
	m.SetPage(reranked, "", false)
	got, _ := m.SelectedItem()
	if got.ID != "Dr2" {
		t.Errorf("selection should follow id, got %q", got.ID)
	}
}

func TestView_RendersDraftsAndHint(t *testing.T) {
	m := New()
	m.SetBadge(2)
	items := sampleDrafts()
	items[0].ChannelName = "general"
	items[1].ChannelName = "alice"
	m.SetItems(items)
	out := m.View(12, 70)
	for _, want := range []string{"Drafts", "Scheduled", "general", "alice", "enter open", "D delete", "ship the build", "thread reply draft"} {
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
	items := sampleDrafts()
	m.SetItems(items)
	if m.ClickAt(7, 2) != ClickItem {
		t.Fatal("click on second card should hit")
	}
	it, _ := m.SelectedItem()
	if it.ID != "Dr2" {
		t.Errorf("selected id = %q, want Dr2", it.ID)
	}
}

func TestCycleFilterAndClickTab(t *testing.T) {
	m := New()
	if m.Filter() != FilterDrafts {
		t.Fatalf("default filter = %q", m.Filter())
	}
	if !m.CycleFilter(1) || m.Filter() != FilterScheduled {
		t.Fatalf("f → %q, want scheduled", m.Filter())
	}
	if !m.CycleFilter(-1) || m.Filter() != FilterDrafts {
		t.Fatalf("F → %q, want drafts", m.Filter())
	}
	m.SetItems(sampleDrafts())
	_ = m.View(10, 80)
	hits := m.tabHits
	if len(hits) < 2 {
		t.Fatalf("tabHits = %d, want >= 2", len(hits))
	}
	if kind := m.ClickAt(0, (hits[1].x0+hits[1].x1)/2); kind != ClickTab {
		t.Fatalf("click Scheduled tab = %v, want ClickTab", kind)
	}
	if m.Filter() != FilterScheduled {
		t.Errorf("filter = %q, want scheduled", m.Filter())
	}
}

func TestSetPageAppendAndLoadMore(t *testing.T) {
	m := New()
	m.SetPage(sampleDrafts()[:1], "1.0", false)
	if !m.HasMore() {
		t.Fatal("next_ts should enable load more")
	}
	m.SetPage(sampleDrafts()[1:], "", true)
	if len(m.Items()) != 2 {
		t.Fatalf("append page: len=%d, want 2", len(m.Items()))
	}
	if m.HasMore() {
		t.Fatal("empty next_ts should clear has more")
	}
}
