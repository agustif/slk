package cache

import "testing"

func TestDrafts_UpsertListDelete(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertWorkspace(Workspace{ID: "T1", Name: "t"}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertDraft(Draft{WorkspaceID: "T1", Key: "C1", Text: "hello", SlackID: "Dr1", LastUpdatedTS: "1.0"}); err != nil {
		t.Fatal(err)
	}
	got, err := db.ListDrafts("T1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "hello" || got[0].SlackID != "Dr1" {
		t.Fatalf("got %+v", got)
	}
	if err := db.UpsertDraft(Draft{WorkspaceID: "T1", Key: "C1", Text: "later"}); err != nil {
		t.Fatal(err)
	}
	one, err := db.GetDraft("T1", "C1")
	if err != nil {
		t.Fatal(err)
	}
	if one.Text != "later" || one.SlackID != "Dr1" {
		t.Fatalf("merge slack id: %+v", one)
	}
	if err := db.DeleteDraft("T1", "C1"); err != nil {
		t.Fatal(err)
	}
	got, err = db.ListDrafts("T1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("after delete %+v", got)
	}
}
