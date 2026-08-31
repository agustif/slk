package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClampSortPipeline(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{nil, []string{SortAtomSlack}},
		{[]string{}, []string{SortAtomSlack}},
		{[]string{"vip_first", "recent"}, []string{SortAtomVIPFirst, SortAtomRecent}},
		{[]string{"vip", "alpha"}, []string{SortAtomVIPFirst, SortAtomAlphabetical}},
		{[]string{"vip_first, recent"}, []string{SortAtomVIPFirst, SortAtomRecent}},
		{[]string{"VIP_FIRST", "RECENT", "vip_first"}, []string{SortAtomVIPFirst, SortAtomRecent}},
		{[]string{"nope", "alphabetical"}, []string{SortAtomAlphabetical}},
		{[]string{"bogus"}, []string{SortAtomSlack}},
	}
	for _, tc := range cases {
		got := ClampSortPipeline(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ClampSortPipeline(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSidebarSortPipelinePrecedence(t *testing.T) {
	s := SidebarSort{
		Default:  []string{"slack"},
		DMs:      []string{"vip_first", "recent"},
		Channels: []string{"alphabetical"},
		Section: map[string][]string{
			"Engineering":     []string{"unread_first", "alphabetical"},
			"Direct Messages": []string{"vip_first", "alphabetical"},
		},
	}.Normalized()

	if got := s.Pipeline("Engineering", "standard"); !reflect.DeepEqual(got, []string{SortAtomUnreadFirst, SortAtomAlphabetical}) {
		t.Errorf("name Engineering = %v", got)
	}
	if got := s.Pipeline("Direct Messages", "direct_messages"); !reflect.DeepEqual(got, []string{SortAtomVIPFirst, SortAtomAlphabetical}) {
		t.Errorf("name beats type: got %v", got)
	}
	if got := s.Pipeline("Random DMs", "direct_messages"); !reflect.DeepEqual(got, []string{SortAtomVIPFirst, SortAtomRecent}) {
		t.Errorf("type dms = %v", got)
	}
	if got := s.Pipeline("Channels", "channels"); !reflect.DeepEqual(got, []string{SortAtomAlphabetical}) {
		t.Errorf("type channels = %v", got)
	}
	if got := s.Pipeline("Books", "standard"); !reflect.DeepEqual(got, []string{SortAtomSlack}) {
		t.Errorf("fallback default = %v", got)
	}
}

func TestLoadSidebarSortFromTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[sidebar]
vip = ["alice", "@cto", "exec-*"]

[sidebar.sort]
default = ["slack"]
dms = ["vip_first", "recent"]
channels = ["unread_first", "alphabetical"]

[sidebar.sort.section]
Engineering = ["alphabetical"]
`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Sidebar.VIP, []string{"alice", "@cto", "exec-*"}) {
		t.Errorf("VIP = %v", cfg.Sidebar.VIP)
	}
	if got := cfg.Sidebar.Sort.Pipeline("Direct Messages", "direct_messages"); !reflect.DeepEqual(got, []string{SortAtomVIPFirst, SortAtomRecent}) {
		t.Errorf("dms pipeline = %v", got)
	}
	if got := cfg.Sidebar.Sort.Pipeline("Engineering", "standard"); !reflect.DeepEqual(got, []string{SortAtomAlphabetical}) {
		t.Errorf("Engineering = %v", got)
	}
}

func TestPipeline_DMsDefaultRecent(t *testing.T) {
	s := SidebarSort{Default: []string{"slack"}}.Normalized()
	if got := s.Pipeline("Direct Messages", "direct_messages"); !reflect.DeepEqual(got, []string{SortAtomRecent}) {
		t.Errorf("empty dms pipeline = %v, want [recent]", got)
	}
	if got := s.Pipeline("Group DMs", "group_dms"); !reflect.DeepEqual(got, []string{SortAtomRecent}) {
		t.Errorf("group_dms type = %v, want [recent]", got)
	}
}

func TestClampGroupDMs(t *testing.T) {
	cases := map[string]string{
		"":         GroupDMsSplit,
		"split":    GroupDMsSplit,
		"separate": GroupDMsSplit,
		"bogus":    GroupDMsSplit,
		"together": GroupDMsTogether,
		"combined": GroupDMsTogether,
		"ALL":      GroupDMsTogether,
		"og":       GroupDMsTogether,
	}
	for in, want := range cases {
		if got := ClampGroupDMs(in); got != want {
			t.Errorf("ClampGroupDMs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVIPMatch(t *testing.T) {
	p := []string{"alice", "@bob", "U123", "exec-*"}
	cases := []struct {
		id, dm, name string
		want         bool
	}{
		{"D1", "U9", "alice", true},
		{"D1", "U9", "Alice", true},
		{"D1", "U9", "bob", true},
		{"D1", "U123", "carol", true},
		{"C1", "", "exec-leads", true},
		{"D1", "U9", "dave", false},
	}
	for _, tc := range cases {
		if got := VIPMatch(p, tc.id, tc.dm, tc.name); got != tc.want {
			t.Errorf("VIPMatch(%q,%q,%q) = %v want %v", tc.id, tc.dm, tc.name, got, tc.want)
		}
	}
}
