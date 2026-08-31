package messages

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestHeaderExtras_RendersBookmarkTitles(t *testing.T) {
	m := New(nil, "general")
	m.SetHeaderChrome([]Bookmark{
		{Title: "Handbook", URL: "https://example.com/handbook"},
		{Title: "Standup notes", URL: "https://docs.example.com/standup"},
	}, []Pin{{TS: "1508197641.000151", Text: "meaning of life"}})

	out := m.View(12, 80)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Handbook") {
		t.Errorf("expected Handbook in header, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Standup notes") {
		t.Errorf("expected Standup notes in header, got:\n%s", plain)
	}
	if !strings.Contains(plain, "\U0001F4CC 1") {
		t.Errorf("expected pin count in header, got:\n%s", plain)
	}
	if !strings.Contains(out, "\x1b]8;;https://example.com/handbook") {
		t.Error("expected OSC-8 hyperlink on Handbook")
	}
	if m.ChromeHeight() < 2 {
		t.Errorf("ChromeHeight = %d; want extras row", m.ChromeHeight())
	}
}

func TestHeaderExtras_EmptyOmitsRow(t *testing.T) {
	m := New(nil, "general")
	out := m.View(12, 60)
	plain := ansi.Strip(out)
	if strings.Contains(plain, "\U0001F4CC") {
		t.Errorf("empty chrome should omit pins, got:\n%s", plain)
	}
	if m.ChromeHeight() != 1 {
		t.Errorf("ChromeHeight = %d; want 1 (name only)", m.ChromeHeight())
	}

	m.SetHeaderChrome(nil, nil)
	out = m.View(12, 60)
	if m.ChromeHeight() != 1 {
		t.Errorf("nil chrome ChromeHeight = %d; want 1", m.ChromeHeight())
	}
	_ = out
}

func TestHeaderExtras_RendersWithoutTopic(t *testing.T) {
	m := New(nil, "general")
	m.SetChannel("general", "")
	m.SetHeaderChrome([]Bookmark{{Title: "Docs", URL: "https://example.com/docs"}}, nil)
	plain := ansi.Strip(m.View(12, 60))
	if !strings.Contains(plain, "Docs") {
		t.Errorf("bookmarks should render with empty topic, got:\n%s", plain)
	}
	if !strings.Contains(plain, "general") {
		t.Error("channel name missing")
	}
}

func TestHeaderExtras_LineIsFullWidth(t *testing.T) {
	m := New(nil, "general")
	m.SetHeaderChrome([]Bookmark{{Title: "Docs", URL: "https://example.com/docs"}}, nil)
	const width = 60
	out := m.View(12, width)
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("want extras row, got %d lines", len(lines))
	}
	if got := lipgloss.Width(lines[1]); got != width {
		t.Errorf("extras line width = %d; want %d", got, width)
	}
}

func TestHeaderExtras_MoreWhenMany(t *testing.T) {
	m := New(nil, "general")
	bms := make([]Bookmark, 8)
	for i := range bms {
		bms[i] = Bookmark{Title: "Very Long Bookmark Title", URL: "https://example.com/" + strings.Repeat("x", i+1)}
	}
	m.SetHeaderChrome(bms, nil)
	plain := ansi.Strip(m.View(12, 48))
	if !strings.Contains(plain, "more") {
		t.Errorf("expected +K more on a narrow pane, got:\n%s", plain)
	}
}

func TestHeaderExtras_HitTestBookmark(t *testing.T) {
	m := New(nil, "general")
	m.SetHeaderChrome([]Bookmark{{Title: "Handbook", URL: "https://example.com/handbook"}}, nil)
	_ = m.View(12, 80)
	row := m.chromeExtrasRow
	if row < 1 {
		t.Fatalf("extras row = %d", row)
	}
	found := false
	for x := 0; x < 40; x++ {
		hit, ok := m.HitTestChrome(row, x)
		if ok && hit.Kind == ChromeHitBookmark && hit.Index == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a bookmark hit on the extras row")
	}
	if _, ok := m.HitTestChrome(0, 2); ok {
		t.Error("title row should not be an extras hit")
	}
}

func TestHeaderExtras_HitTestPins(t *testing.T) {
	m := New(nil, "general")
	m.SetHeaderChrome(nil, []Pin{{TS: "1.0"}, {TS: "2.0"}})
	_ = m.View(12, 80)
	row := m.chromeExtrasRow
	found := false
	for x := 0; x < 20; x++ {
		hit, ok := m.HitTestChrome(row, x)
		if ok && hit.Kind == ChromeHitPins {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a pins hit")
	}
}

func TestSetChannel_ClearsHeaderChrome(t *testing.T) {
	m := New(nil, "general")
	m.SetHeaderChrome([]Bookmark{{Title: "Docs", URL: "https://example.com"}}, []Pin{{TS: "1.0"}})
	m.SetChannel("random", "")
	if len(m.Bookmarks()) != 0 || len(m.Pins()) != 0 {
		t.Fatal("switching channels should drop previous chrome")
	}
	plain := ansi.Strip(m.View(12, 60))
	if strings.Contains(plain, "Docs") {
		t.Error("stale bookmark title still rendered")
	}
}
