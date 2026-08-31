package channelmembers

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func testMembers() []Member {
	return []Member{
		{ID: "U1", DisplayName: "Alice Chen", Username: "alice"},
		{ID: "U2", DisplayName: "Bob Singh", Username: "bob", Presence: "active"},
		{ID: "U3", DisplayName: "Carla Diaz", Username: "carla", IsGuest: true},
		{ID: "U4", DisplayName: "Dan Evans", Username: "dan", Presence: "away"},
		{ID: "U5", DisplayName: "Eva Frank", Username: "eva", IsExternal: true},
	}
}

func TestNew_NotVisibleByDefault(t *testing.T) {
	m := New()
	if m.IsVisible() {
		t.Error("expected new model to not be visible")
	}
}

func TestOpen_MakesVisibleAndResetsQuery(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.SetChannel("general")
	m.Open()
	if !m.IsVisible() {
		t.Fatal("expected Open() to make model visible")
	}
	if m.Query() != "" {
		t.Errorf("query = %q, want empty", m.Query())
	}
	if m.Selected() != 0 {
		t.Errorf("selected = %d, want 0", m.Selected())
	}
	if got := len(m.FilteredMembers()); got != 5 {
		t.Errorf("filtered = %d, want 5", got)
	}
}

func TestClose_HidesModel(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.Close()
	if m.IsVisible() {
		t.Error("expected Close() to hide model")
	}
}

func TestFilter_EmptyQuerySortsByDisplayName(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()

	got := idsOf(m.FilteredMembers())
	want := []string{"U1", "U2", "U3", "U4", "U5"} // Alice, Bob, Carla, Dan, Eva
	if !equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestFilter_PrefixBeatsSubstring(t *testing.T) {
	m := New()
	m.SetMembers([]Member{
		{ID: "U1", DisplayName: "Marcus", Username: "marcus"},
		{ID: "U2", DisplayName: "Alice Marketing", Username: "amark"},
	})
	m.Open()
	m.HandleKey("m")
	m.HandleKey("a")
	m.HandleKey("r")

	got := m.FilteredMembers()
	if len(got) < 1 {
		t.Fatal("expected matches")
	}
	if got[0].ID != "U1" {
		t.Errorf("prefix match should rank first, got %s", got[0].ID)
	}
}

func TestFilter_CaseInsensitive(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	for _, r := range "ALICE" {
		m.HandleKey(string(r))
	}
	got := m.FilteredMembers()
	if len(got) == 0 {
		t.Fatal("expected match for ALICE")
	}
	if got[0].ID != "U1" {
		t.Errorf("first match = %s, want U1", got[0].ID)
	}
}

func TestFilter_MatchesUsernameHandle(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("d")
	m.HandleKey("a")
	m.HandleKey("n")

	got := m.FilteredMembers()
	if len(got) == 0 {
		t.Fatal("expected match for handle 'dan'")
	}
	if got[0].ID != "U4" {
		t.Errorf("first match = %s, want U4", got[0].ID)
	}
}

func TestFilter_NoMatchesReturnsEmpty(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	for _, r := range "xyzqq" {
		m.HandleKey(string(r))
	}
	if got := len(m.FilteredMembers()); got != 0 {
		t.Errorf("filtered = %d, want 0", got)
	}
}

func TestFilter_BackspaceRestoresList(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("a")
	if len(m.FilteredMembers()) == 5 {
		t.Fatal("expected filter to drop some rows")
	}
	m.HandleKey("backspace")
	if got := len(m.FilteredMembers()); got != 5 {
		t.Errorf("after backspace filtered = %d, want 5", got)
	}
}

func TestJK_NavigatesHighlight(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	if m.Selected() != 0 {
		t.Fatalf("start selected = %d, want 0", m.Selected())
	}
	m.HandleKey("j")
	if m.Selected() != 1 {
		t.Errorf("after j selected = %d, want 1", m.Selected())
	}
	m.HandleKey("j")
	if m.Selected() != 2 {
		t.Errorf("after second j selected = %d, want 2", m.Selected())
	}
	m.HandleKey("k")
	if m.Selected() != 1 {
		t.Errorf("after k selected = %d, want 1", m.Selected())
	}
}

func TestJK_DoesNotTypeIntoQuery(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("j")
	m.HandleKey("k")
	if m.Query() != "" {
		t.Errorf("j/k should navigate, not filter; query = %q", m.Query())
	}
	if got := len(m.FilteredMembers()); got != 5 {
		t.Errorf("filtered = %d, want 5", got)
	}
}

func TestEnter_ReturnsSelectedMember(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("j") // Bob
	got := m.HandleKey("enter")
	if got == nil {
		t.Fatal("enter should return a Result")
	}
	if got.UserID != "U2" {
		t.Errorf("UserID = %s, want U2", got.UserID)
	}
}

func TestEnter_EmptyListIsNoop(t *testing.T) {
	m := New()
	m.SetMembers(nil)
	m.Open()
	if got := m.HandleKey("enter"); got != nil {
		t.Errorf("enter on empty list returned %+v", got)
	}
}

func TestEsc_Closes(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("esc")
	if m.IsVisible() {
		t.Error("esc should close")
	}
}

func TestSetMembers_WhileVisibleRefilters(t *testing.T) {
	m := New()
	m.SetMembers(testMembers())
	m.Open()
	m.HandleKey("a") // Alice
	m.SetMembers([]Member{
		{ID: "U9", DisplayName: "Ann", Username: "ann"},
		{ID: "U1", DisplayName: "Alice Chen", Username: "alice"},
	})
	got := idsOf(m.FilteredMembers())
	if !equal(got, []string{"U1", "U9"}) && !equal(got, []string{"U1"}) {
		// query "a" matches both Ann and Alice
		if len(got) == 0 {
			t.Fatal("refilter dropped the query matches")
		}
	}
	for _, id := range got {
		if id != "U1" && id != "U9" {
			t.Errorf("unexpected id %s after SetMembers", id)
		}
	}
}

func TestTitle_IncludesChannelAndCount(t *testing.T) {
	m := New()
	m.SetChannel("general")
	m.SetMembers(testMembers())
	m.Open()
	out := ansi.Strip(m.View(80))
	if !strings.Contains(out, "#general · 5 members") {
		t.Errorf("title missing from view:\n%s", out)
	}
}

func TestView_ShowsGuestAndPresence(t *testing.T) {
	m := New()
	m.SetChannel("eng")
	m.SetMembers(testMembers())
	m.Open()
	out := ansi.Strip(m.View(80))
	if !strings.Contains(out, "guest") {
		t.Errorf("expected guest flag in view:\n%s", out)
	}
	if !strings.Contains(out, "●") {
		t.Errorf("expected active presence dot in view:\n%s", out)
	}
	if !strings.Contains(out, "ext") {
		t.Errorf("expected ext flag in view:\n%s", out)
	}
}

func idsOf(members []Member) []string {
	out := make([]string, len(members))
	for i, mem := range members {
		out[i] = mem.ID
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
