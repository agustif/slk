package slackclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParsePinsList_Fixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/pins-list.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parsePinsList(raw)
	if err != nil {
		t.Fatalf("parsePinsList: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d; want 3", len(got))
	}
	if got[0].MessageTS != "1508197641.000151" || got[0].Created != 1508881078 {
		t.Errorf("newest message pin = %+v", got[0])
	}
	if got[1].Text != "The meaning of life is 42." {
		t.Errorf("older message pin text = %q", got[1].Text)
	}
	if got[2].Type != "file" || got[2].Text != "Q3 roadmap" || got[2].MessageTS != "" {
		t.Errorf("file pin = %+v", got[2])
	}
}

func TestParsePinsList_RejectsNotOK(t *testing.T) {
	_, err := parsePinsList([]byte(`{"ok":false,"error":"not_pinnable"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
}

func TestGetPins_SendsChannel(t *testing.T) {
	var path, channel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		channel = r.PostForm.Get("channel")
		w.Header().Set("Content-Type", "application/json")
		raw, _ := os.ReadFile("testdata/pins-list.json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("xoxc-test", "d-cookie")
	pointClientAtTestServer(t, c, srv)

	got, err := c.GetPins(context.Background(), "C0123456789")
	if err != nil {
		t.Fatalf("GetPins: %v", err)
	}
	if path != "/api/pins.list" {
		t.Errorf("path = %q", path)
	}
	if channel != "C0123456789" {
		t.Errorf("channel = %q", channel)
	}
	if len(got) != 3 {
		t.Errorf("len = %d", len(got))
	}
}

func TestGetPins_RequiresChannelID(t *testing.T) {
	c := NewClient("xoxc-test", "d-cookie")
	if _, err := c.GetPins(context.Background(), ""); err == nil {
		t.Fatal("want error for empty channelID")
	}
}
