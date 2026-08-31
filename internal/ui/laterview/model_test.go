package laterview

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

func sampleItems() []Item {
	return []Item{
		{SavedItem: slackclient.SavedItem{
			Key: "C1\t1.0", ItemID: "C1", ItemType: "message", TS: "1.0",
			State: "in_progress", DateCreated: time.Now().Unix() - 120,
		}, ChannelName: "general", ChannelType: "channel"},
		{SavedItem: slackclient.SavedItem{
			Key: "D1\t2.0", ItemID: "D1", ItemType: "message", TS: "2.0",
			State: "in_progress", DateCreated: time.Now().Unix() - 3600,
			DateDue: time.Now().Unix() + 600,
		}, ChannelName: "alice", ChannelType: "dm"},
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
	for _, want := range []string{"Later", "general", "alice", "enter open", "saved for later"} {
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
	// toolbar = 2 lines; first card at y=2, second card at y=2+4=6
	if !m.ClickAt(6) {
		t.Fatal("click on second card should hit")
	}
	it, _ := m.SelectedItem()
	if it.TS != "2.0" {
		t.Errorf("selected ts = %q, want 2.0", it.TS)
	}
}

func TestEmptyState(t *testing.T) {
	m := New()
	out := m.View(8, 40)
	if !strings.Contains(out, "nothing saved for later") {
		t.Errorf("empty view = %q", out)
	}
}
