package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseBookmarksList_Fixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/bookmarks-list.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseBookmarksList(raw)
	if err != nil {
		t.Fatalf("parseBookmarksList: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d; want 3 (empty title dropped)", len(got))
	}
	if got[0].Title != "Handbook" || got[0].Link != "https://example.com/handbook" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Title != "Standup notes" {
		t.Errorf("second title = %q", got[1].Title)
	}
	if got[2].Type != "message" || got[2].Title != "Kickoff thread" {
		t.Errorf("message bookmark = %+v", got[2])
	}
}

func TestParseBookmarksList_RejectsNotOK(t *testing.T) {
	_, err := parseBookmarksList([]byte(`{"ok":false,"error":"channel_not_found"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
}

func TestParseBookmarksList_Empty(t *testing.T) {
	got, err := parseBookmarksList([]byte(`{"ok":true,"bookmarks":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bookmarks; want 0", len(got))
	}
}

func TestGetBookmarks_SendsChannelID(t *testing.T) {
	var path, channelID, reason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		channelID = r.PostForm.Get("channel_id")
		reason = r.PostForm.Get("_x_reason")
		w.Header().Set("Content-Type", "application/json")
		raw, _ := os.ReadFile("testdata/bookmarks-list.json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("xoxc-test", "d-cookie")
	pointClientAtTestServer(t, c, srv)

	got, err := c.GetBookmarks(context.Background(), "C0123456789")
	if err != nil {
		t.Fatalf("GetBookmarks: %v", err)
	}
	if path != "/api/bookmarks.list" {
		t.Errorf("path = %q", path)
	}
	if channelID != "C0123456789" {
		t.Errorf("channel_id = %q", channelID)
	}
	if reason != bookmarksListReason {
		t.Errorf("_x_reason = %q; want %q", reason, bookmarksListReason)
	}
	if len(got) != 3 || got[0].Title != "Handbook" {
		t.Errorf("got %+v", got)
	}
}

func TestGetBookmarks_RequiresChannelID(t *testing.T) {
	c := NewClient("xoxc-test", "d-cookie")
	if _, err := c.GetBookmarks(context.Background(), ""); err == nil {
		t.Fatal("want error for empty channelID")
	}
}
