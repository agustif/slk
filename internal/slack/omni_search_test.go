package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchInline_PostsCapturedForm(t *testing.T) {
	var path, query, count, reason, fromMe, withMe, recents string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		query = r.PostForm.Get("query")
		count = r.PostForm.Get("count")
		reason = r.PostForm.Get("_x_reason")
		fromMe = r.PostForm.Get("from_me")
		withMe = r.PostForm.Get("with_me")
		recents = r.PostForm.Get("recent_channels")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"query":"help","pagination":{"total_count":0,"page":1,"per_page":3},"items":[]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.SearchInline(context.Background(), "help", "sess", []string{"C1", "D2"})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/search.inline" {
		t.Errorf("path = %q", path)
	}
	if query != "help" || count != "3" || fromMe != "true" || withMe != "true" {
		t.Errorf("query=%q count=%q from_me=%q with_me=%q", query, count, fromMe, withMe)
	}
	if recents != "C1,D2" {
		t.Errorf("recent_channels = %q", recents)
	}
	_ = reason
	if len(got) != 0 {
		t.Errorf("items = %+v", got)
	}
}

func TestParseInlineSearch_ChannelObjectAndExtract(t *testing.T) {
	raw := []byte(`{"ok":true,"items":[{"channel":{"id":"C1","name":"general"},"ts":"1.0","extract":"hello there","user":"U1","username":"alice"}]}`)
	got, err := parseInlineSearch(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ChannelID != "C1" || got[0].TS != "1.0" || got[0].Text != "hello there" {
		t.Errorf("got %+v", got)
	}
}

func TestAutocompleteFiles_PostsCapturedForm(t *testing.T) {
	var path, query, shares string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		query = r.PostForm.Get("query")
		shares = r.PostForm.Get("include_shares")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":[{"id":"F1","name":"a.txt","title":"a","permalink":"https://x","channels":["C1"]}]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.AutocompleteFiles(context.Background(), "help")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/search.autocomplete.files" || query != "help" || shares != "true" {
		t.Errorf("path=%q query=%q shares=%q", path, query, shares)
	}
	if len(got) != 1 || got[0].ID != "F1" || got[0].ChannelID != "C1" {
		t.Errorf("got %+v", got)
	}
}

func TestSearchInline_EmptyQueryNoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not POST")
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.SearchInline(context.Background(), "  ", "", nil)
	if err != nil || got != nil {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseInlineSearch_RejectsUnknown(t *testing.T) {
	_, err := parseInlineSearch([]byte(`{"ok":false,"error":"invalid_query"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid_query") {
		t.Fatalf("err = %v", err)
	}
}
