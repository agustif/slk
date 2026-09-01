package sidebar

import (
	"strings"
	"testing"
	"time"

	"github.com/agustif/slk/internal/cache"
)

func TestDMsSynthRow_AfterThreads(t *testing.T) {
	m := New([]ChannelItem{{ID: "C1", Name: "general", Type: "channel"}})
	if !m.IsActivitySelected() {
		t.Fatal("Activity should own the top slot")
	}
	m.MoveDown()
	if !m.IsLaterSelected() {
		t.Fatal("Later")
	}
	m.MoveDown()
	if !m.IsThreadsSelected() {
		t.Fatal("Threads")
	}
	m.MoveDown()
	if !m.IsDMsSelected() {
		t.Fatalf("Direct Messages synth should sit under Threads; got header=%v id=%q",
			func() string { n, _ := m.IsSectionHeaderSelected(); return n }(), m.SelectedID())
	}
	out := m.View(12, 30)
	if !strings.Contains(out, "Direct Messages") {
		t.Errorf("View missing Direct Messages synth: %q", out)
	}
}

func TestDMsView_ShowsStaleAndHidesChannels(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	old := now.Add(-60 * 24 * time.Hour).Unix()
	m := New([]ChannelItem{
		{ID: "C1", Name: "general", Type: "channel"},
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: now.Unix()},
		{ID: "D2", Name: "old-bob", Type: "dm", LastActivity: old},
		{ID: "G1", Name: "jamari, lion", Type: "group_dm", LastActivity: now.Unix() - 10},
	})
	m.SetReadStateReader(func() map[string]cache.ReadState {
		return map[string]cache.ReadState{
			"D1": {LastReadTS: formatTSPair(now.Unix(), 0)},
			"D2": {LastReadTS: formatTSPair(old, 0)},
			"G1": {LastReadTS: formatTSPair(now.Unix()-10, 0)},
			"C1": {LastReadTS: formatTSPair(now.Unix(), 0)},
		}
	})
	m.SetNowFunc(func() time.Time { return now })
	m.SetStaleThreshold(30 * 24 * time.Hour)

	home := map[string]bool{}
	for _, id := range filteredIDs(m) {
		home[id] = true
	}
	if home["D2"] {
		t.Error("home sidebar should hide stale D2")
	}
	if !home["D1"] {
		t.Error("home sidebar should show recent D1")
	}

	m.SetDMsView(true)
	got := filteredIDs(m)
	want := map[string]bool{"D1": true, "D2": true, "G1": true}
	if len(got) != 3 {
		t.Fatalf("DMs view ids = %v, want 3 DMs", got)
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected %s in DMs view", id)
		}
		delete(want, id)
	}
	if len(want) > 0 {
		t.Errorf("missing from DMs view: %v", want)
	}
	if m.SelectedID() == "C1" {
		t.Error("DMs view should not keep a channel cursor")
	}
}

func TestDMsView_SortsByRecency(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 10},
		{ID: "D2", Name: "bob", Type: "dm", LastActivity: 30},
		{ID: "D3", Name: "carol", Type: "dm", LastActivity: 20},
	})
	m.SetDMsView(true)
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

func TestDMsView_EmptyLastReadStillShown(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	m := New([]ChannelItem{
		{ID: "D1", Name: "closed", Type: "dm"},
	})
	m.SetNowFunc(func() time.Time { return now })
	m.SetStaleThreshold(30 * 24 * time.Hour)
	m.SetReadStateReader(func() map[string]cache.ReadState {
		return map[string]cache.ReadState{"D1": {}}
	})
	if len(filteredIDs(m)) != 0 {
		t.Fatalf("home should hide 1:1 DM with empty last_read, got %v", filteredIDs(m))
	}
	m.SetDMsView(true)
	if got := filteredIDs(m); len(got) != 1 || got[0] != "D1" {
		t.Fatalf("DMs view should show closed leftover, got %v", got)
	}
}

func TestDMsView_HomeRowIsFirst(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 20},
	})
	m.SetDMsView(true)
	if !m.IsHomeSelected() {
		t.Fatal("DMs view should start on Home")
	}
	m.MoveDown()
	if m.SelectedID() != "D1" {
		t.Fatalf("after Home, first DM = %q", m.SelectedID())
	}
}

func TestDMsView_HidesSynthRows(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 20},
		{ID: "C1", Name: "general", Type: "channel"},
	})
	m.SetDMsView(true)
	out := m.View(12, 40)
	if strings.Contains(out, "◎ Activity") || strings.Contains(out, "✉ Direct Messages") || strings.Contains(out, "✎ Drafts") || strings.Contains(out, "★ Starred") {
		t.Errorf("DMs view should hide inbox synth rows:\n%s", out)
	}
	if !strings.Contains(out, "Home") {
		t.Errorf("DMs view should show a Home row:\n%s", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("DMs view missing conversation:\n%s", out)
	}
}

func TestHomeHidesClosedIMs(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "open", Type: "dm", Closed: false, LastActivity: 20},
		{ID: "D2", Name: "closed", Type: "dm", Closed: true, LastActivity: 30},
	})
	got := map[string]bool{}
	for _, id := range filteredIDs(m) {
		got[id] = true
	}
	if !got["D1"] {
		t.Error("open IM should show on Home")
	}
	if got["D2"] {
		t.Error("closed IM should hide on Home")
	}
	m.SetDMsView(true)
	got = map[string]bool{}
	for _, id := range filteredIDs(m) {
		got[id] = true
	}
	if !got["D1"] || !got["D2"] {
		t.Errorf("DMs view should list both, got %v", got)
	}
}

func TestDMsView_RendersPreviewAndDate(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: now.Unix(), Preview: "ship it tomorrow"},
	})
	m.SetNowFunc(func() time.Time { return now })
	m.SetDMsView(true)
	out := m.View(20, 40)
	if !strings.Contains(out, "alice") {
		t.Fatalf("missing name:\n%s", out)
	}
	if !strings.Contains(out, "ship it tomorrow") {
		t.Errorf("missing preview:\n%s", out)
	}
	if !strings.Contains(out, "Today") {
		t.Errorf("missing Today:\n%s", out)
	}
}

func TestApplyDMSnippets_ReSortsAndSurvivesSetItems(t *testing.T) {
	m := New([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm", LastActivity: 10},
		{ID: "D2", Name: "bob", Type: "dm", LastActivity: 20},
	})
	m.SetDMsView(true)
	m.ApplyDMSnippets(map[string]DMSnippet{
		"D1": {Text: "newer from alice", Activity: 50},
	})
	got := filteredIDs(m)
	if len(got) < 1 || got[0] != "D1" {
		t.Fatalf("after snippet recency D1 should lead, got %v", got)
	}
	m.SetItems([]ChannelItem{
		{ID: "D1", Name: "alice", Type: "dm"},
		{ID: "D2", Name: "bob", Type: "dm", LastActivity: 20},
	})
	out := m.View(20, 40)
	if !strings.Contains(out, "newer from alice") {
		t.Errorf("preview should survive SetItems:\n%s", out)
	}
}

func TestFormatActivityDate(t *testing.T) {
	now := time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC)
	if got := formatActivityDate(now.Unix(), now); got != "Today" {
		t.Errorf("today = %q", got)
	}
	if got := formatActivityDate(now.Add(-25*time.Hour).Unix(), now); got != "Yesterday" {
		t.Errorf("yesterday = %q", got)
	}
	if got := formatActivityDate(now.Add(-3*24*time.Hour).Unix(), now); got != "Fri" {
		t.Errorf("weekday = %q", got)
	}
	if got := formatActivityDate(time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC).Unix(), now); got != "Jul 16" {
		t.Errorf("same year = %q", got)
	}
}
