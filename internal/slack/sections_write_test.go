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

func TestUpdateDeleteReorderChannelSection(t *testing.T) {
	var paths []string
	var names, ids, nexts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		form, _ := parseFormBody(string(body))
		names = append(names, form["name"])
		ids = append(ids, form["channel_section_id"])
		nexts = append(nexts, form["next_channel_section_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := testWriteClient(t, srv)
	if err := c.UpdateChannelSection(context.Background(), "L1", "New", ""); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteChannelSection(context.Background(), "L1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := c.ReorderChannelSections(context.Background(), []string{"L2", "L1"}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	if len(paths) != 4 {
		t.Fatalf("paths = %v", paths)
	}
	if paths[0] != "/api/users.channelSections.update" || names[0] != "New" {
		t.Errorf("update path/name = %s %s", paths[0], names[0])
	}
	if paths[1] != "/api/users.channelSections.delete" || ids[1] != "L1" {
		t.Errorf("delete path/id = %s %s", paths[1], ids[1])
	}
	if paths[2] != "/api/users.channelSections.update" || ids[2] != "L2" || nexts[2] != "L1" {
		t.Errorf("reorder[0] path/id/next = %s %s %s", paths[2], ids[2], nexts[2])
	}
	if paths[3] != "/api/users.channelSections.update" || ids[3] != "L1" || nexts[3] != "" {
		t.Errorf("reorder[1] path/id/next = %s %s %q", paths[3], ids[3], nexts[3])
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
