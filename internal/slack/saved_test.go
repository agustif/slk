package slackclient

import (
	"os"
	"strings"
	"testing"
)

func TestParseSavedList_CapturedShape(t *testing.T) {
	raw, err := os.ReadFile("testdata/saved-list.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := parseSavedList(raw)
	if err != nil {
		t.Fatalf("parseSavedList: %v", err)
	}
	if len(res.Items) != 3 {
		t.Fatalf("len = %d; want 3", len(res.Items))
	}
	msg := res.Items[0]
	if msg.ItemID != "C0A95SPDL3B" || msg.TS != "1787955901.383689" || msg.ItemType != "message" {
		t.Errorf("first = %+v", msg)
	}
	if msg.State != "in_progress" || msg.DateCreated != 1787956000 {
		t.Errorf("first meta = %+v", msg)
	}
	due := res.Items[1]
	if due.DateDue != 1788219500 || due.ItemID != "D0ARB1GN6Q5" {
		t.Errorf("due item = %+v", due)
	}
	file := res.Items[2]
	if file.ItemType != "file" || file.ItemID != "F012FILE" {
		t.Errorf("file = %+v", file)
	}
	if res.Counts.Uncompleted != 2 || res.Counts.UncompletedOverdue != 1 || res.Counts.Total != 6 {
		t.Errorf("counts = %+v", res.Counts)
	}
	if res.Counts.Badge() != 2 {
		t.Errorf("badge = %d; want 2", res.Counts.Badge())
	}
}

func TestParseSavedList_RejectsNotOK(t *testing.T) {
	_, err := parseSavedList([]byte(`{"ok":false,"error":"not_allowed"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
	if !strings.Contains(err.Error(), "not_allowed") {
		t.Errorf("err = %v; want not_allowed", err)
	}
}

func TestParseSavedOK_RejectsNotOK(t *testing.T) {
	if err := parseSavedOK([]byte(`{"ok":false,"error":"already_saved"}`), "saved.add"); err == nil {
		t.Fatal("want error")
	} else if !AlreadySaved(err) {
		t.Errorf("AlreadySaved(%v) = false", err)
	}
}

func TestParseSavedOK_OK(t *testing.T) {
	if err := parseSavedOK([]byte(`{"ok":true}`), "saved.add"); err != nil {
		t.Fatal(err)
	}
}

func TestParseSavedCounts(t *testing.T) {
	got := parseSavedCounts([]byte(`{"uncompleted_count":4,"uncompleted_overdue_count":1,"archived_count":0,"completed_count":2,"total_count":6}`))
	if got.Uncompleted != 4 || got.Badge() != 4 || got.UncompletedOverdue != 1 {
		t.Errorf("got %+v", got)
	}
	if parseSavedCounts(nil).Badge() != 0 {
		t.Error("empty saved object should badge 0")
	}
}

func TestBuildSavedListForm_Defaults(t *testing.T) {
	f := buildSavedListForm(SavedListOpts{})
	if f.Get("filter") != "saved" {
		t.Errorf("filter = %q; want saved", f.Get("filter"))
	}
	if f.Get("limit") != "50" {
		t.Errorf("limit = %q; want 50", f.Get("limit"))
	}
	if f.Get("include_tombstones") != "true" {
		t.Errorf("include_tombstones = %q", f.Get("include_tombstones"))
	}
}

func TestBuildSavedListForm_ClampsAndFilter(t *testing.T) {
	if got := buildSavedListForm(SavedListOpts{Limit: 0}).Get("limit"); got != "50" {
		t.Errorf("limit 0 = %q", got)
	}
	if got := buildSavedListForm(SavedListOpts{Limit: 999}).Get("limit"); got != "100" {
		t.Errorf("limit 999 = %q", got)
	}
	if got := buildSavedListForm(SavedListOpts{Filter: "completed", Cursor: "abc"}).Get("filter"); got != "completed" {
		t.Errorf("filter = %q", got)
	}
	if got := buildSavedListForm(SavedListOpts{Cursor: "nudge"}).Get("cursor"); got != "nudge" {
		t.Errorf("cursor = %q", got)
	}
}

func TestFlattenSavedItem_SkipsMessageWithoutTS(t *testing.T) {
	_, ok := flattenSavedItem(savedListItem{ItemID: "C1", ItemType: "message"})
	if ok {
		t.Fatal("message without ts should be skipped")
	}
	_, ok = flattenSavedItem(savedListItem{ItemType: "message", TS: "1.0"})
	if ok {
		t.Fatal("missing item_id should be skipped")
	}
}
