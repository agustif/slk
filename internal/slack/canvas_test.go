package slackclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCannedCanvasTemplates_PostsCapturedForm(t *testing.T) {
	var path, reason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		reason = r.PostForm.Get("_x_reason")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":[{"id":"F080JDE025R","name":"To-do_list","title":"To-do list","filetype":"quip","mode":"quip"}]}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.ListCannedCanvasTemplates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/canvases.getCannedTemplates" {
		t.Errorf("path = %q", path)
	}
	if reason != "" && reason != "fetch-canned-templates" {
		t.Errorf("_x_reason = %q", reason)
	}
	if len(got) != 1 || got[0].ID != "F080JDE025R" || got[0].Filetype != "quip" {
		t.Errorf("got = %+v", got)
	}
}

func TestLookupQuipThreadIDs_PostsFileIDs(t *testing.T) {
	var path, ids string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		ids = r.PostForm.Get("file_ids")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"lookup":{"F0BU54KT3U1":"VOQ9AAYK8SN"}}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.LookupQuipThreadIDs(context.Background(), []string{"F0BU54KT3U1"})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/quip.lookupThreadIds" || ids != "F0BU54KT3U1" {
		t.Errorf("path=%q file_ids=%q", path, ids)
	}
	if got["F0BU54KT3U1"] != "VOQ9AAYK8SN" {
		t.Errorf("lookup = %v", got)
	}
}

func TestLookupQuipFileID_PostsThreadID(t *testing.T) {
	var path, tid string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		tid = r.PostForm.Get("quip_thread_id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"file_id":"F0BU54KT3U1"}`))
	}))
	defer srv.Close()
	c := newTestClient(srv)
	got, err := c.LookupQuipFileID(context.Background(), "VOQ9AAYK8SN")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/quip.lookupFileId" || tid != "VOQ9AAYK8SN" || got != "F0BU54KT3U1" {
		t.Errorf("path=%q tid=%q got=%q", path, tid, got)
	}
}

func TestOpenCloseFile_PostsFileID(t *testing.T) {
	var openPath, closePath, openID, closeID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/files.open":
			openPath = r.URL.Path
			openID = r.PostForm.Get("file_id")
			_, _ = w.Write([]byte(`{"ok":true,"viewers":[],"should_subscribe_and_ping":false}`))
		case "/api/files.close":
			closePath = r.URL.Path
			closeID = r.PostForm.Get("file_id")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
			_, _ = io.WriteString(w, `{"ok":false}`)
		}
	}))
	defer srv.Close()
	c := newTestClient(srv)
	if err := c.OpenFile(context.Background(), "F1"); err != nil {
		t.Fatal(err)
	}
	if err := c.CloseFile(context.Background(), "F1"); err != nil {
		t.Fatal(err)
	}
	if openPath != "/api/files.open" || openID != "F1" || closePath != "/api/files.close" || closeID != "F1" {
		t.Errorf("open %q %q close %q %q", openPath, openID, closePath, closeID)
	}
}
