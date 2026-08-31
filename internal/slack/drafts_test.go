package slackclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseDraftsList_ExtractsActiveComposerDrafts(t *testing.T) {
	raw := []byte(`{
		"ok": true,
		"has_more": true,
		"drafts": [
			{
				"id": "Dr1",
				"last_updated_ts": "1700000000.000001",
				"is_deleted": false,
				"is_sent": false,
				"is_from_composer": true,
				"destinations": [{"channel_id": "C1", "thread_ts": "1.0"}],
				"blocks": [{"type":"rich_text","elements":[{"type":"rich_text_section","elements":[{"type":"text","text":"hello"}]}]}]
			},
			{
				"id": "Dr2",
				"last_updated_ts": "1700000001.000001",
				"is_deleted": true,
				"destinations": [{"channel_id": "C2"}],
				"blocks": []
			},
			{
				"id": "Dr3",
				"last_updated_ts": "1700000002.000001",
				"is_sent": true,
				"destinations": [{"channel_id": "C3"}],
				"blocks": []
			}
		]
	}`)
	got, next, err := parseDraftsList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if next != "1700000002.000001" {
		t.Fatalf("next_ts = %q, want last raw row (sent/deleted still advance the cursor)", next)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1 (deleted/sent skipped)", len(got))
	}
	if got[0].ID != "Dr1" || got[0].ChannelID != "C1" || got[0].ThreadTS != "1.0" || got[0].Text != "hello" {
		t.Fatalf("got %+v", got[0])
	}
}

func TestParseDraftsList_DecodesStringifiedJSON(t *testing.T) {
	dest, _ := json.Marshal([...]map[string]string{{"channel_id": "C9"}})
	blocks, _ := json.Marshal([]map[string]any{
		{"type": "rich_text", "elements": []any{
			map[string]any{"type": "rich_text_section", "elements": []any{
				map[string]any{"type": "text", "text": "parked"},
			}},
		}},
	})
	payload := map[string]any{
		"ok": true,
		"drafts": []any{
			map[string]any{
				"id":              "DrS",
				"last_updated_ts": "1.0",
				"destinations":    string(dest),
				"blocks":          string(blocks),
			},
		},
	}
	raw, _ := json.Marshal(payload)
	got, _, err := parseDraftsList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ChannelID != "C9" || got[0].Text != "parked" {
		t.Fatalf("got %+v", got)
	}
}

func TestListComposerDrafts_PaginatesOnNextTS(t *testing.T) {
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		pages = append(pages, r.PostForm.Get("next_ts"))
		if r.URL.Path != "/api/drafts.list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.PostForm.Get("is_active") != "true" {
			t.Errorf("is_active = %q", r.PostForm.Get("is_active"))
		}
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("next_ts") == "" {
			_, _ = io.WriteString(w, `{"ok":true,"has_more":true,"drafts":[{"id":"DrA","last_updated_ts":"10.1","destinations":[{"channel_id":"C1"}],"blocks":[]}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true,"has_more":false,"drafts":[{"id":"DrB","last_updated_ts":"10.2","destinations":[{"channel_id":"C2"}],"blocks":[]}]}`)
	}))
	defer srv.Close()
	c := &Client{token: "xoxc-t", cookie: "d", apiBaseURL: srv.URL + "/api/"}
	got, err := c.ListComposerDrafts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "DrA" || got[1].ID != "DrB" {
		t.Fatalf("got %+v", got)
	}
	if len(pages) != 2 || pages[0] != "" || pages[1] != "10.1" {
		t.Fatalf("pages = %v", pages)
	}
}

func TestCreateComposerDraft_PostsJSONStringFields(t *testing.T) {
	var path, blocks, dests, fromComposer, attachments string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		blocks = r.PostForm.Get("blocks")
		dests = r.PostForm.Get("destinations")
		fromComposer = r.PostForm.Get("is_from_composer")
		attachments = r.PostForm.Get("attachments")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true,"draft":{"id":"DrN","last_updated_ts":"11.1"}}`)
	}))
	defer srv.Close()
	c := &Client{token: "xoxc-t", cookie: "d", apiBaseURL: srv.URL + "/api/"}
	got, err := c.CreateComposerDraft(context.Background(), "C1", "2.0", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/drafts.create" {
		t.Errorf("path = %s", path)
	}
	if fromComposer != "true" || attachments != "[]" {
		t.Errorf("from=%q attachments=%q", fromComposer, attachments)
	}
	if !strings.Contains(blocks, `"type":"rich_text"`) || !strings.Contains(blocks, "hi") {
		t.Errorf("blocks = %s", blocks)
	}
	if !strings.Contains(dests, `"channel_id":"C1"`) || !strings.Contains(dests, `"thread_ts":"2.0"`) {
		t.Errorf("destinations = %s", dests)
	}
	if got.ID != "DrN" || got.LastUpdatedTS != "11.1" {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdateComposerDraft_SendsIDAndTS(t *testing.T) {
	var path, id, ts string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		id = r.PostForm.Get("draft_id")
		ts = r.PostForm.Get("client_last_updated_ts")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true,"id":"DrN","last_updated_ts":"12.2"}`)
	}))
	defer srv.Close()
	c := &Client{token: "xoxc-t", cookie: "d", apiBaseURL: srv.URL + "/api/"}
	got, err := c.UpdateComposerDraft(context.Background(), "DrN", "11.1", "C1", "", "later")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/drafts.update" || id != "DrN" || ts != "11.1" {
		t.Errorf("path=%s id=%s ts=%s", path, id, ts)
	}
	if got.LastUpdatedTS != "12.2" {
		t.Errorf("ts = %s", got.LastUpdatedTS)
	}
}

func TestDeleteComposerDraft_FormFields(t *testing.T) {
	var path, id, ts string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		id = r.PostForm.Get("draft_id")
		ts = r.PostForm.Get("client_last_updated_ts")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()
	c := &Client{token: "xoxc-t", cookie: "d", apiBaseURL: srv.URL + "/api/"}
	if err := c.DeleteComposerDraft(context.Background(), "DrN", "12.2"); err != nil {
		t.Fatal(err)
	}
	if path != "/api/drafts.delete" || id != "DrN" || ts != "12.2" {
		t.Errorf("path=%s id=%s ts=%s", path, id, ts)
	}
}

func TestDraftKeyFor(t *testing.T) {
	if got := DraftKeyFor("C1", ""); got != "C1" {
		t.Errorf("channel = %q", got)
	}
	if got := DraftKeyFor("C1", "1.0"); got != "C1\x001.0" {
		t.Errorf("thread = %q", got)
	}
	ch, th := SplitDraftKey("C1\x001.0")
	if ch != "C1" || th != "1.0" {
		t.Errorf("split = %s %s", ch, th)
	}
}
