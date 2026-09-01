package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	slackclient "github.com/agustif/slk/internal/slack"
)

func TestHydrateStarredFiles_ListThenInfoFallback(t *testing.T) {
	var listFiles, infoFile string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/files.list":
			listFiles = r.PostForm.Get("files")
			_, _ = w.Write([]byte(`{"ok":true,"files":[{"id":"F0BUXHC276C","title":"Employee Onboarding","filetype":"quip","mode":"quip"}]}`))
		case "/api/files.info":
			infoFile = r.PostForm.Get("file")
			_, _ = w.Write([]byte(`{"ok":true,"file":{"id":"F0BUVQHU6NL","name":"slk-har-star.txt","title":"slk-har-star.txt","filetype":"text","mode":"snippet","content":"hi"}}`))
		default:
			t.Errorf("path = %q", r.URL.Path)
			_, _ = w.Write([]byte(`{"ok":false}`))
		}
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got := hydrateStarredFiles(context.Background(), c, []string{"F0BUXHC276C", "F0BUVQHU6NL"})
	if listFiles != "F0BUXHC276C,F0BUVQHU6NL" {
		t.Errorf("files.list files = %q", listFiles)
	}
	if infoFile != "F0BUVQHU6NL" {
		t.Errorf("files.info file = %q", infoFile)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d: %+v", len(got), got)
	}
	if got[0].Title != "Employee Onboarding" || !got[0].IsCanvas() {
		t.Errorf("canvas %+v", got[0])
	}
	if got[1].ID != "F0BUVQHU6NL" || got[1].Content != "hi" {
		t.Errorf("snippet %+v", got[1])
	}
}

func TestHydrateStarredFiles_Empty(t *testing.T) {
	if got := hydrateStarredFiles(context.Background(), nil, []string{"F1"}); got != nil {
		t.Errorf("nil client = %+v", got)
	}
	if got := hydrateStarredFiles(context.Background(), &slackclient.Client{}, nil); got != nil {
		t.Errorf("nil ids = %+v", got)
	}
}
