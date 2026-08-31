package unreadsview

import (
	"strings"
	"testing"

	"github.com/gammons/slk/internal/config"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui/styles"
	"github.com/slack-go/slack"
)

func init() {
	styles.Apply("nord", config.Theme{})
}

func sampleBlocks() []Block {
	return []Block{
		{
			ChannelID: "C2", ChannelName: "random", ChannelType: "channel",
			LastRead: "1.0", LatestTS: "1.3",
			Messages: []Message{
				{TS: "1.2", UserID: "U1", UserName: "alice", Text: "hello"},
				{TS: "1.3", UserID: "U2", UserName: "bob", Text: "world"},
			},
		},
		{
			ChannelID: "C1", ChannelName: "general", ChannelType: "channel",
			LastRead: "2.0", LatestTS: "2.1",
			Messages: []Message{
				{TS: "2.1", UserID: "U3", UserName: "carol", Text: "ping"},
			},
		},
	}
}

func TestView_RendersAllUnreadsAndEmpty(t *testing.T) {
	m := New()
	out := m.View(8, 50)
	if !strings.Contains(out, "All Unreads") {
		t.Fatalf("title missing:\n%s", out)
	}
	if !strings.Contains(out, "no unreads") {
		t.Errorf("empty copy missing:\n%s", out)
	}
	m.SetBlocks(sampleBlocks())
	out = m.View(14, 70)
	for _, want := range []string{"All Unreads", "#general", "#random", "hello", "ping", "mark as read", "2 messages", "1 message", "sidebar"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
}

func TestMoveDownSelectsHeaderThenMessage(t *testing.T) {
	m := New()
	m.SetBlocks(sampleBlocks())
	if !m.SelectedIsHeader() {
		t.Fatal("first stop should be a header")
	}
	b, ok := m.SelectedBlock()
	if !ok || b.ChannelID != "C1" {
		t.Fatalf("first block = %+v ok=%v, want C1 (empty sidebar order falls back to name)", b, ok)
	}
	m.SetSidebarOrder([]string{"C2", "C1"})
	b, _ = m.SelectedBlock()
	if b.ChannelID != "C2" {
		t.Fatalf("sidebar order first = %q, want C2", b.ChannelID)
	}
	m.MoveDown()
	if m.SelectedIsHeader() {
		t.Fatal("second stop should be a message")
	}
	_, msg, ok := m.SelectedMessage()
	if !ok || msg.TS != "1.2" {
		t.Fatalf("selected message = %+v ok=%v", msg, ok)
	}
}

func TestCycleSort(t *testing.T) {
	m := New()
	m.SetBlocks(sampleBlocks())
	if m.Sort() != SortSidebar {
		t.Fatalf("default sort = %q", m.Sort())
	}
	if !m.CycleSort(1) || m.Sort() != SortAlpha {
		t.Fatalf("f → %q, want alpha", m.Sort())
	}
	ids := func() []string {
		var out []string
		for _, b := range m.Blocks() {
			out = append(out, b.ChannelID)
		}
		return out
	}
	got := ids()
	if len(got) != 2 || got[0] != "C1" || got[1] != "C2" {
		t.Fatalf("alpha order = %v, want C1 C2", got)
	}
	if !m.CycleSort(1) || m.Sort() != SortNewest {
		t.Fatalf("f → %q, want newest", m.Sort())
	}
	got = ids()
	if got[0] != "C1" { // LatestTS 2.1 > 1.3
		t.Fatalf("newest first = %v, want C1 first", got)
	}
	if !m.CycleSort(1) || m.Sort() != SortOldest {
		t.Fatalf("f → %q, want oldest", m.Sort())
	}
	got = ids()
	if got[0] != "C2" {
		t.Fatalf("oldest first = %v, want C2 first", got)
	}
	if !m.CycleSort(1) || m.Sort() != SortSidebar {
		t.Fatalf("wrap → %q, want sidebar", m.Sort())
	}
}

func TestClickAtHeaderAndMessage(t *testing.T) {
	m := New()
	m.SetBlocks(sampleBlocks())
	_ = m.View(16, 70)
	if kind := m.ClickAt(toolbarLines, 2); kind != ClickHeader {
		t.Fatalf("click header = %v, want ClickHeader", kind)
	}
	if kind := m.ClickAt(toolbarLines+headerLines, 2); kind != ClickMessage {
		t.Fatalf("click first message = %v, want ClickMessage", kind)
	}
	_, msg, ok := m.SelectedMessage()
	if !ok || msg.Text != "hello" && msg.Text != "ping" {
		t.Fatalf("clicked message = %+v ok=%v", msg, ok)
	}
}

func TestMarkReadAndUndo(t *testing.T) {
	m := New()
	m.SetBlocks(sampleBlocks())
	id := m.Blocks()[0].ChannelID
	if !m.MarkBlockRead(id) {
		t.Fatal("MarkBlockRead failed")
	}
	out := m.View(14, 70)
	if !strings.Contains(out, "marked read") {
		t.Errorf("marked copy missing:\n%s", out)
	}
	if !strings.Contains(out, "undo") {
		t.Errorf("undo missing:\n%s", out)
	}
	b, ok := m.SelectedBlock()
	if !ok || b.ChannelID != id || !m.SelectedIsHeader() {
		t.Fatalf("selection after mark = %+v header=%v", b, m.SelectedIsHeader())
	}
	if !m.UndoBlock(id) {
		t.Fatal("UndoBlock failed")
	}
	out = m.View(14, 70)
	if strings.Contains(out, "marked read") {
		t.Errorf("undo should restore messages:\n%s", out)
	}
}

func TestBlockFromHistory_DropsLastReadAndReverses(t *testing.T) {
	hist := slackclient.HistoryResult{Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "3.0", User: "U3", Text: "newest"}},
		{Msg: slack.Msg{Timestamp: "2.0", User: "U2", Text: "mid"}},
		{Msg: slack.Msg{Timestamp: "1.0", User: "U1", Text: "already read"}},
	}}
	b := BlockFromHistory("C1", "1.0", hist)
	if b.LatestTS != "3.0" {
		t.Errorf("LatestTS = %q, want 3.0", b.LatestTS)
	}
	if len(b.Messages) != 2 {
		t.Fatalf("messages = %d, want 2 (drop last_read)", len(b.Messages))
	}
	if b.Messages[0].TS != "2.0" || b.Messages[1].TS != "3.0" {
		t.Errorf("chrono = %+v", b.Messages)
	}
}

func TestEmptyCopyAndLoading(t *testing.T) {
	m := New()
	m.SetLoading(true)
	out := m.View(6, 40)
	if !strings.Contains(out, "loading unreads") {
		t.Errorf("loading missing:\n%s", out)
	}
}
