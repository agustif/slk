package starredview

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
		{StarredMessage: slackclient.StarredMessage{
			ChannelID: "C1", TS: "1.0", UserID: "U1", Text: "ship it", DateCreate: time.Now().Unix() - 120,
		}, ChannelName: "general", ChannelType: "channel", AuthorName: "bob"},
		{StarredMessage: slackclient.StarredMessage{
			ChannelID: "D1", TS: "2.0", UserID: "U2", Text: "please review", DateCreate: time.Now().Unix() - 3600,
		}, ChannelName: "alice", ChannelType: "dm", AuthorName: "alice"},
	}
}

func TestSetItemsPreservesSelectionByKey(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	m.MoveDown()
	it, _ := m.SelectedItem()
	if it.Key() != "D1\t2.0" {
		t.Fatalf("precondition: selected %q", it.Key())
	}
	reranked := []Item{sampleItems()[1], sampleItems()[0]}
	m.SetItems(reranked)
	got, _ := m.SelectedItem()
	if got.Key() != "D1\t2.0" {
		t.Errorf("selection should follow key, got %q", got.Key())
	}
}

func TestView_RendersStarredHint(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	out := m.View(12, 70)
	for _, want := range []string{"Starred items", "general", "alice", "enter open", "ship it", "please review", "•2"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
}

func TestClickAtSelectsCard(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	if m.ClickAt(7, 2) != ClickItem {
		t.Fatal("click on second card should hit")
	}
	it, _ := m.SelectedItem()
	if it.TS != "2.0" {
		t.Errorf("selected ts = %q, want 2.0", it.TS)
	}
}

func TestRemoveDropsRow(t *testing.T) {
	m := New()
	m.SetItems(sampleItems())
	m.Remove("C1", "1.0")
	if len(m.Items()) != 1 || m.Items()[0].TS != "2.0" {
		t.Fatalf("items = %+v", m.Items())
	}
	if m.Badge() != 1 {
		t.Errorf("badge = %d, want 1", m.Badge())
	}
}

func TestEmptyCopy(t *testing.T) {
	m := New()
	out := m.View(8, 40)
	if !strings.Contains(out, "no starred messages") {
		t.Errorf("empty copy missing:\n%s", out)
	}
}

func TestErrorCopy(t *testing.T) {
	m := New()
	m.SetError("stars.list failed — boom")
	out := m.View(8, 40)
	if !strings.Contains(out, "stars.list failed — boom") {
		t.Errorf("error copy missing:\n%s", out)
	}
}
