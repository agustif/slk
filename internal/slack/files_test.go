package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetFileInfo_PostsCapturedPeekForm(t *testing.T) {
	var path string
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"file":{"id":"F0BUVQHU6NL","name":"slk-har-star.txt","title":"slk-har-star.txt","filetype":"text","mode":"snippet","is_starred":false},"comments":[],"paging":{"page":1,"count":500}}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.GetFileInfo(context.Background(), "F0BUVQHU6NL")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/files.info" {
		t.Errorf("path = %q", path)
	}
	want := map[string]string{
		"file":          "F0BUVQHU6NL",
		"page":          "1",
		"count":         "500",
		"truncate":      "true",
		"public_shared": "false",
		"skip_shares":   "true",
	}
	for k, v := range want {
		if form.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, form.Get(k), v)
		}
	}
	for _, invented := range []string{"channel", "user", "ts", "count_total"} {
		if form.Get(invented) != "" {
			t.Errorf("invented field %s = %q", invented, form.Get(invented))
		}
	}
	if got.ID != "F0BUVQHU6NL" || got.Name != "slk-har-star.txt" || got.Mode != "snippet" {
		t.Errorf("got %+v", got)
	}
	if got.IsStarred {
		t.Error("is_starred stayed false in captures")
	}
}

func TestParseFileInfo_QuipCanvasHasNoContent(t *testing.T) {
	raw := []byte(`{"ok":true,"file":{"id":"F0BUXHC276C","name":"Employee Onboarding","title":"Employee Onboarding","filetype":"quip","mode":"quip","is_starred":false},"comments":[],"paging":{"count":500,"total":0,"page":1,"pages":1}}`)
	got, err := parseFileInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "F0BUXHC276C" || got.Title != "Employee Onboarding" {
		t.Errorf("got %+v", got)
	}
	if got.Filetype != "quip" || got.Mode != "quip" || !got.IsCanvas() {
		t.Errorf("canvas flags: %+v", got)
	}
	if got.Content != "" {
		t.Errorf("quip peek must not invent content, got %q", got.Content)
	}
	if got.IsStarred {
		t.Error("is_starred stays false")
	}
}

func TestParseFileInfo_SnippetExposesContent(t *testing.T) {
	raw := []byte(`{"ok":true,"file":{"id":"F0BU15RCQQN","name":"slk-har-star2.txt","title":"slk-har-star2.txt","filetype":"text","mode":"snippet","is_starred":false,"content":"hello snippet","content_highlight_html":"<pre>hello snippet</pre>"},"comments":[],"paging":{}}`)
	got, err := parseFileInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "F0BU15RCQQN" || got.Content != "hello snippet" {
		t.Errorf("got %+v", got)
	}
	if got.IsCanvas() {
		t.Error("snippet is not a canvas")
	}
}

func TestGetFileInfo_EmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not POST")
	}))
	defer srv.Close()
	c := newTestClient(srv)
	if _, err := c.GetFileInfo(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetFileInfo_RejectsNotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"file_not_found"}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	_, err := c.GetFileInfo(context.Background(), "F1")
	if err == nil || !strings.Contains(err.Error(), "file_not_found") {
		t.Errorf("err = %v", err)
	}
}

func TestHydrateFiles_PostsCapturedForm(t *testing.T) {
	var path, files string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		files = r.PostForm.Get("files")
		for _, invented := range []string{"channel", "user", "ts", "types", "count", "page"} {
			if r.PostForm.Get(invented) != "" {
				t.Errorf("invented field %s = %q", invented, r.PostForm.Get(invented))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":[{"id":"F0BUXHC276C","name":"Employee Onboarding","title":"Employee Onboarding","filetype":"quip","mode":"quip","is_starred":false},{"id":"F0BUVQHU6NL","name":"slk-har-star.txt","title":"slk-har-star.txt","filetype":"text","mode":"snippet"}]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.HydrateFiles(context.Background(), []string{" F0BUXHC276C ", "F0BUVQHU6NL", "F0BUXHC276C", ""})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/files.list" {
		t.Errorf("path = %q", path)
	}
	if files != "F0BUXHC276C,F0BUVQHU6NL" {
		t.Errorf("files = %q", files)
	}
	if len(got) != 2 || got[0].ID != "F0BUXHC276C" || got[0].Title != "Employee Onboarding" {
		t.Errorf("got %+v", got)
	}
	if !got[0].IsCanvas() || got[1].IsCanvas() {
		t.Errorf("canvas flags %+v", got)
	}
}

func TestHydrateFiles_EmptyIDsSkipPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not POST")
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.HydrateFiles(context.Background(), []string{"", "  "})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("got %+v", got)
	}
}

func TestParseHydrateFiles_KeyedMap(t *testing.T) {
	raw := []byte(`{"ok":true,"files":{"F1":{"id":"F1","title":"A","filetype":"quip","mode":"quip"}}}`)
	got, err := parseHydrateFiles(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "F1" || got[0].Title != "A" {
		t.Errorf("got %+v", got)
	}
}

func TestSearchModulesFiles_PostsFilesRailForm(t *testing.T) {
	var path string
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"items":[]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.SearchModulesFiles(context.Background(), "type:quip", 0)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/search.modules.files" {
		t.Errorf("path = %q", path)
	}
	want := map[string]string{
		"module":         "files",
		"query":          "type:quip",
		"page":           "1",
		"count":          "50",
		"sort":           "last_engaged",
		"search_context": "desktop_files_browser",
	}
	for k, v := range want {
		if form.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, form.Get(k), v)
		}
	}
	for _, invented := range []string{
		"highlight", "extract_len", "extra_message_data", "cursor",
		"browse_context", "sort_dir", "channel",
	} {
		if _, ok := form[invented]; ok {
			t.Errorf("unnamed/invented field %s present: %q", invented, form.Get(invented))
		}
	}
	if len(got) != 0 {
		t.Errorf("empty workspace items = %+v", got)
	}
}

func TestSearchUserCreatedCanvases_PostsProbeForm(t *testing.T) {
	var query, count, module, sort, ctx string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		query = r.PostForm.Get("query")
		count = r.PostForm.Get("count")
		module = r.PostForm.Get("module")
		sort = r.PostForm.Get("sort")
		ctx = r.PostForm.Get("search_context")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"items":[{"file":{"id":"F0BUXHC276C","title":"Employee Onboarding","filetype":"quip","mode":"quip"}}]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	c.userID = "U0BU12NKA8J"
	got, err := c.SearchUserCreatedCanvases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if query != "type:quip creator:U0BU12NKA8J" {
		t.Errorf("query = %q", query)
	}
	if count != "1" || module != "files" || sort != "last_engaged" || ctx != "desktop_files_browser" {
		t.Errorf("count=%q module=%q sort=%q search_context=%q", count, module, sort, ctx)
	}
	if len(got) != 1 || got[0].ID != "F0BUXHC276C" || got[0].Title != "Employee Onboarding" {
		t.Errorf("got %+v", got)
	}
}

func TestSearchUserCreatedCanvases_RequiresUserID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not POST")
	}))
	defer srv.Close()
	c := newTestClient(srv)
	if _, err := c.SearchUserCreatedCanvases(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseSearchModulesFiles_NestedAndFlatItems(t *testing.T) {
	raw := []byte(`{"ok":true,"items":[{"file":{"id":"F1","title":"Canvas A","filetype":"quip","mode":"quip"}},{"id":"F2","name":"notes.txt","title":"notes","filetype":"text","mode":"snippet","content":"hi"}]}`)
	got, err := parseSearchModulesFiles(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "F1" || got[0].Title != "Canvas A" || !got[0].IsCanvas() {
		t.Errorf("item0 %+v", got[0])
	}
	if got[1].ID != "F2" || got[1].Content != "hi" {
		t.Errorf("item1 %+v", got[1])
	}
}

func TestParseSearchModulesFiles_RejectsNotOK(t *testing.T) {
	_, err := parseSearchModulesFiles([]byte(`{"ok":false,"error":"ratelimited"}`))
	if err == nil || !strings.Contains(err.Error(), "ratelimited") {
		t.Errorf("err = %v", err)
	}
}

func TestFileInfoDisplayName(t *testing.T) {
	if (FileInfo{Title: "T", Name: "n", ID: "F1"}).DisplayName() != "T" {
		t.Fatal("title first")
	}
	if (FileInfo{Name: "n", ID: "F1"}).DisplayName() != "n" {
		t.Fatal("name next")
	}
	if (FileInfo{ID: "F1"}).DisplayName() != "F1" {
		t.Fatal("id last")
	}
}

func TestListRecentlyDeletedFiles_PostsCapturedForm(t *testing.T) {
	var path, reason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		reason = r.PostForm.Get("_x_reason")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":[]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.ListRecentlyDeletedFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/files.recentlyDeleted" {
		t.Errorf("path = %q", path)
	}
	if reason != "" && reason != "get-deleted-files" {
		t.Errorf("_x_reason = %q", reason)
	}
	if got == nil {
		got = []FileInfo{}
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}
