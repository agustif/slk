package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestParseSearchFiles_CannedJSON(t *testing.T) {
	raw, err := os.ReadFile("testdata/search-files.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := parseSearchFiles(raw)
	if err != nil {
		t.Fatalf("parseSearchFiles: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if res.Page != 1 {
		t.Errorf("Page = %d, want 1", res.Page)
	}
	if len(res.Matches) != 3 {
		t.Fatalf("len(Matches) = %d, want 3", len(res.Matches))
	}

	a := res.Matches[0]
	if a.ID != "F012FILEAAA" || a.Name != "deploy-plan.pdf" {
		t.Errorf("match0 id/name = %q %q", a.ID, a.Name)
	}
	if a.Username != "grant" || a.User != "U0GRANT0001" {
		t.Errorf("match0 user = %q / %q", a.Username, a.User)
	}
	if a.Created != 1700000001 {
		t.Errorf("match0 created = %d", a.Created)
	}
	if a.Permalink != "https://team.slack.com/files/U0GRANT0001/F012FILEAAA/deploy-plan.pdf" {
		t.Errorf("match0 permalink = %q", a.Permalink)
	}
	if a.ChannelID != "C0GENERAL01" || a.ChannelName != "general" {
		t.Errorf("match0 channel = %q / %q", a.ChannelID, a.ChannelName)
	}
	if a.DownloadURL() != "https://files.slack.com/files-pri/T1-F012FILEAAA/download/deploy-plan.pdf" {
		t.Errorf("match0 download = %q", a.DownloadURL())
	}

	b := res.Matches[1]
	if b.Name != "runbook.md" {
		t.Errorf("match1 name = %q", b.Name)
	}
	if b.Username != "" || b.User != "U0SAM000002" {
		t.Errorf("match1 user = %q / %q (username absent on wire)", b.Username, b.User)
	}
	if b.ChannelID != "G0PRIVATE01" || b.ChannelName != "ops-secret" {
		t.Errorf("match1 channel = %q / %q", b.ChannelID, b.ChannelName)
	}
	if b.DownloadURL() != "https://files.slack.com/files-pri/T1-F012FILEBBB/runbook.md" {
		t.Errorf("match1 download fallback = %q", b.DownloadURL())
	}

	c := res.Matches[2]
	if c.ChannelID != "D0DMCHAN01" {
		t.Errorf("match2 channel (IM) = %q", c.ChannelID)
	}
	if c.Username != "ayush" {
		t.Errorf("match2 username = %q", c.Username)
	}
}

func TestParseSearchFiles_RejectsNotOK(t *testing.T) {
	_, err := parseSearchFiles([]byte(`{"ok":false,"error":"ratelimited"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
	if !strings.Contains(err.Error(), "ratelimited") {
		t.Errorf("err = %v", err)
	}
}

func TestParseSearchFiles_RejectsBadJSON(t *testing.T) {
	_, err := parseSearchFiles([]byte(`{not json`))
	if err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestSearchFiles_PostsQueryCountPage(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		raw, err := os.ReadFile("testdata/search-files.json")
		if err != nil {
			t.Errorf("read fixture: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	c := &Client{
		token:      "xoxc-test",
		apiBaseURL: srv.URL + "/",
		httpClient: srv.Client(),
	}
	res, err := c.SearchFiles(context.Background(), "from:@grant in:#general deploy", 50, 2)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if !strings.HasSuffix(gotPath, "search.files") {
		t.Errorf("path = %q, want search.files", gotPath)
	}
	if !strings.Contains(gotBody, "query=from%3A%40grant") {
		t.Errorf("query not passed verbatim: %s", gotBody)
	}
	if !strings.Contains(gotBody, "count=50") {
		t.Errorf("count missing: %s", gotBody)
	}
	if !strings.Contains(gotBody, "page=2") {
		t.Errorf("page missing: %s", gotBody)
	}
	if res.Total != 3 || len(res.Matches) != 3 {
		t.Errorf("parsed result = %+v", res)
	}
	if res.Matches[0].Name != "deploy-plan.pdf" {
		t.Errorf("first filename = %q", res.Matches[0].Name)
	}
}

func TestSearchFiles_OmitsDefaultPageAndCount(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"files":{"total":0,"matches":[]}}`))
	}))
	defer srv.Close()

	c := &Client{
		token:      "xoxc-test",
		apiBaseURL: srv.URL + "/",
		httpClient: srv.Client(),
	}
	if _, err := c.SearchFiles(context.Background(), "x", 0, 1); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gotBody, "count=") {
		t.Errorf("non-positive count should be omitted: %s", gotBody)
	}
	if strings.Contains(gotBody, "page=") {
		t.Errorf("page=1 should be omitted (Slack default): %s", gotBody)
	}
}

func TestSearchFiles_WrapsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer srv.Close()

	c := &Client{
		token:      "xoxc-test",
		apiBaseURL: srv.URL + "/",
		httpClient: srv.Client(),
	}
	_, err := c.SearchFiles(context.Background(), "x", 50, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid_auth") {
		t.Errorf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "searching files") {
		t.Errorf("error should be wrapped: %v", err)
	}
}

func TestSearchMessages_ForwardsPage(t *testing.T) {
	var gotParams slack.SearchParameters
	mock := &mockSlackAPI{
		searchMessagesFn: func(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
			gotParams = params
			return &slack.SearchMessages{Total: 1}, nil
		},
	}
	c := &Client{api: mock}
	if _, err := c.SearchMessages(context.Background(), "deploy", 50, 3); err != nil {
		t.Fatal(err)
	}
	if gotParams.Count != 50 {
		t.Errorf("count = %d, want 50", gotParams.Count)
	}
	if gotParams.Page != 3 {
		t.Errorf("page = %d, want 3", gotParams.Page)
	}
}
