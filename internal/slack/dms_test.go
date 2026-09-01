package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/agustif/slk/internal/slackhttp"
)

func TestListClientDMs_PostsCapturedForm(t *testing.T) {
	var path, count, includeClosed, includeChannel, excludeBots, priorityMode, reason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		path = r.URL.Path
		count = r.PostForm.Get("count")
		includeClosed = r.PostForm.Get("include_closed")
		includeChannel = r.PostForm.Get("include_channel")
		excludeBots = r.PostForm.Get("exclude_bots")
		priorityMode = r.PostForm.Get("priority_mode")
		reason = r.PostForm.Get("_x_reason")
		w.Header().Set("Content-Type", "application/json")
		raw, err := os.ReadFile("testdata/client-dms.json")
		if err != nil {
			t.Errorf("testdata: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	c := NewClient("xoxc-test", "d-cookie")
	pointClientAtTestServer(t, c, srv)

	got, err := c.ListClientDMs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/client.dms" {
		t.Errorf("path = %q", path)
	}
	if count != "250" {
		t.Errorf("count = %q, want 250", count)
	}
	if includeClosed != "true" {
		t.Errorf("include_closed = %q, want true", includeClosed)
	}
	if includeChannel != "true" {
		t.Errorf("include_channel = %q, want true", includeChannel)
	}
	if excludeBots != "true" {
		t.Errorf("exclude_bots = %q, want true", excludeBots)
	}
	if priorityMode != "priority" {
		t.Errorf("priority_mode = %q, want priority (string, not a boolean)", priorityMode)
	}
	if reason != clientDMsReason {
		t.Errorf("_x_reason = %q, want %q", reason, clientDMsReason)
	}
	if len(got.IMs) != 2 || got.IMs[0].ID != "D0TESTIM1" || got.IMs[1].ID != "D0TESTIM2" {
		t.Errorf("ims = %+v", got.IMs)
	}
	if len(got.MPIMs) != 1 || got.MPIMs[0].ID != "G0TESTMPIM1" {
		t.Errorf("mpims = %+v", got.MPIMs)
	}
}

func TestListClientDMs_ReasonOverride(t *testing.T) {
	var reason string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		reason = r.PostForm.Get("_x_reason")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"ims":[],"mpims":[]}`))
	}))
	defer srv.Close()
	c := NewClient("xoxc-test", "d-cookie")
	pointClientAtTestServer(t, c, srv)
	ctx := slackhttp.WithReason(context.Background(), "dms-tab-populate")
	if _, err := c.ListClientDMs(ctx); err != nil {
		t.Fatal(err)
	}
	if reason != "dms-tab-populate" {
		t.Errorf("_x_reason = %q, want dms-tab-populate", reason)
	}
}

func TestParseClientDMs_Fixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/client-dms.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseClientDMs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.IMs) != 2 || got.IMs[0].ID != "D0TESTIM1" {
		t.Errorf("ims = %+v", got.IMs)
	}
	if len(got.MPIMs) != 1 || got.MPIMs[0].ID != "G0TESTMPIM1" {
		t.Errorf("mpims = %+v", got.MPIMs)
	}
}

func TestParseClientDMs_RejectsNotOK(t *testing.T) {
	_, err := parseClientDMs([]byte(`{"ok":false,"error":"invalid_auth"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
}

func TestParseClientDMs_DropsEmptyIDs(t *testing.T) {
	got, err := parseClientDMs([]byte(`{"ok":true,"ims":[{"id":""},{"id":"D1"}],"mpims":[{"id":""}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.IMs) != 1 || got.IMs[0].ID != "D1" {
		t.Errorf("ims = %+v", got.IMs)
	}
	if len(got.MPIMs) != 0 {
		t.Errorf("mpims = %+v", got.MPIMs)
	}
}
