package slackclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testWriteClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return &Client{
		token:      "xoxc-test",
		apiBaseURL: srv.URL + "/api/",
		httpClient: srv.Client(),
	}
}

func TestAssignChannelToSection_InsertAndRemoveForm(t *testing.T) {
	var gotPath, gotInsert, gotRemove, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		form, _ := parseFormBody(string(body))
		gotToken = form["token"]
		gotInsert = form["insert"]
		gotRemove = form["remove"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	err := c.AssignChannelToSection(context.Background(), "C1", "L_TO", "L_FROM")
	if err != nil {
		t.Fatalf("AssignChannelToSection: %v", err)
	}
	if gotPath != "/api/users.channelSections.channels.bulkUpdate" {
		t.Errorf("path = %q, want /api/users.channelSections.channels.bulkUpdate", gotPath)
	}
	if gotToken != "xoxc-test" {
		t.Errorf("token = %q", gotToken)
	}

	var insert, remove []ChannelSectionChannelOp
	if err := json.Unmarshal([]byte(gotInsert), &insert); err != nil {
		t.Fatalf("insert JSON %q: %v", gotInsert, err)
	}
	if err := json.Unmarshal([]byte(gotRemove), &remove); err != nil {
		t.Fatalf("remove JSON %q: %v", gotRemove, err)
	}
	if len(insert) != 1 || insert[0].SectionID != "L_TO" || len(insert[0].ChannelIDs) != 1 || insert[0].ChannelIDs[0] != "C1" {
		t.Errorf("insert = %+v", insert)
	}
	if len(remove) != 1 || remove[0].SectionID != "L_FROM" || len(remove[0].ChannelIDs) != 1 || remove[0].ChannelIDs[0] != "C1" {
		t.Errorf("remove = %+v", remove)
	}
}

func TestAssignChannelToSection_OmitsRemoveWhenUnsectioned(t *testing.T) {
	var gotRemovePresent bool
	var gotInsert string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := parseFormBody(string(body))
		_, gotRemovePresent = form["remove"]
		gotInsert = form["insert"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	if err := c.AssignChannelToSection(context.Background(), "C1", "L_TO", ""); err != nil {
		t.Fatalf("AssignChannelToSection: %v", err)
	}
	if gotRemovePresent {
		t.Error("remove should be omitted when the channel is unsectioned")
	}
	if !strings.Contains(gotInsert, `"L_TO"`) || !strings.Contains(gotInsert, `"C1"`) {
		t.Errorf("insert = %q", gotInsert)
	}
}

func TestAssignChannelToSection_NoopWhenAlreadyThere(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	if err := c.AssignChannelToSection(context.Background(), "C1", "L1", "L1"); err != nil {
		t.Fatalf("same-section assign should no-op: %v", err)
	}
	if called {
		t.Error("same-section assign must not hit the network")
	}
}

func TestAssignChannelToSection_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	err := c.AssignChannelToSection(context.Background(), "C1", "L_TO", "")
	if err == nil {
		t.Fatal("want error on ok=false")
	}
	if !strings.Contains(err.Error(), "invalid_auth") {
		t.Errorf("err = %v", err)
	}
}

func TestAssignChannelToSection_RequiresIDs(t *testing.T) {
	c := &Client{token: "xoxc-test", apiBaseURL: "http://example.invalid/api/", httpClient: http.DefaultClient}
	if err := c.AssignChannelToSection(context.Background(), "", "L1", ""); err == nil {
		t.Error("empty channel id should error")
	}
	if err := c.AssignChannelToSection(context.Background(), "C1", "", ""); err == nil {
		t.Error("empty section id should error")
	}
}

func TestCreateChannelSection_FormAndID(t *testing.T) {
	var gotPath, gotName, gotEmoji string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		form, _ := parseFormBody(string(body))
		gotName = form["name"]
		gotEmoji = form["emoji"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel_section_id":"LNEW"}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	id, err := c.CreateChannelSection(context.Background(), "Archive", "")
	if err != nil {
		t.Fatalf("CreateChannelSection: %v", err)
	}
	if gotPath != "/api/users.channelSections.create" {
		t.Errorf("path = %q", gotPath)
	}
	if gotName != "Archive" {
		t.Errorf("name = %q", gotName)
	}
	if gotEmoji != "" {
		t.Errorf("emoji = %q, want omitted", gotEmoji)
	}
	if id != "LNEW" {
		t.Errorf("id = %q", id)
	}
}

func TestCreateChannelSection_NestedSectionObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel_section":{"channel_section_id":"LNEST"}}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	id, err := c.CreateChannelSection(context.Background(), "Nested", "books")
	if err != nil {
		t.Fatalf("CreateChannelSection: %v", err)
	}
	if id != "LNEST" {
		t.Errorf("id = %q", id)
	}
}

func TestCreateChannelSection_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"restricted_action"}`))
	}))
	defer srv.Close()

	c := testWriteClient(t, srv)
	_, err := c.CreateChannelSection(context.Background(), "Nope", "")
	if err == nil || !strings.Contains(err.Error(), "restricted_action") {
		t.Errorf("err = %v", err)
	}
}

func TestCreateChannelSection_EmptyName(t *testing.T) {
	c := &Client{token: "xoxc-test"}
	if _, err := c.CreateChannelSection(context.Background(), "", ""); err == nil {
		t.Error("empty name should error without a network call")
	}
}

func parseFormBody(body string) (map[string]string, error) {
	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(values))
	for k, vs := range values {
		if len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	return out, nil
}
