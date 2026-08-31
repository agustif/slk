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
	"time"

	"github.com/slack-go/slack"
)

func TestUserStatusActive(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name string
		st   UserStatus
		want bool
	}{
		{name: "empty is inactive", st: UserStatus{}, want: false},
		{name: "text never expires", st: UserStatus{Text: "in a meeting"}, want: true},
		{name: "emoji never expires", st: UserStatus{Emoji: ":calendar:"}, want: true},
		{name: "future expiration is active", st: UserStatus{Text: "ooo", Expiration: now.Unix() + 60}, want: true},
		{name: "past expiration is inactive", st: UserStatus{Text: "ooo", Emoji: ":palm_tree:", Expiration: now.Unix() - 1}, want: false},
		{name: "expiration equal to now is inactive", st: UserStatus{Text: "ooo", Expiration: now.Unix()}, want: false},
		{name: "expired with empty remaining fields", st: UserStatus{Expiration: now.Unix() - 1}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.st.Active(now); got != tc.want {
				t.Errorf("Active() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUserStatusFromSlack(t *testing.T) {
	u := &slack.User{}
	u.Profile.StatusText = "focus"
	u.Profile.StatusEmoji = ":dart:"
	u.Profile.StatusExpiration = 42
	got := UserStatusFromSlack(u)
	if got.Text != "focus" || got.Emoji != ":dart:" || got.Expiration != 42 {
		t.Errorf("UserStatusFromSlack = %+v", got)
	}
	if got := UserStatusFromSlack(nil); got != (UserStatus{}) {
		t.Errorf("nil user = %+v, want zero", got)
	}
}

func TestSetUserCustomStatus_PostsProfileJSON(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	st := UserStatus{Text: "in a meeting", Emoji: ":calendar:", Expiration: 1_700_000_100}
	if err := c.SetUserCustomStatus(context.Background(), st); err != nil {
		t.Fatalf("SetUserCustomStatus: %v", err)
	}
	if !strings.HasSuffix(gotPath, "/users.profile.set") {
		t.Errorf("path = %q, want suffix /users.profile.set", gotPath)
	}
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatalf("parse form: %v", err)
	}
	if form.Get("token") != "xoxc-test" {
		t.Errorf("token = %q", form.Get("token"))
	}
	var profile statusProfileWire
	if err := json.Unmarshal([]byte(form.Get("profile")), &profile); err != nil {
		t.Fatalf("profile JSON: %v (%s)", err, form.Get("profile"))
	}
	if profile.StatusText != st.Text || profile.StatusEmoji != st.Emoji || profile.StatusExpiration != st.Expiration {
		t.Errorf("profile = %+v, want %+v", profile, st)
	}
}

func TestClearUserCustomStatus_PostsEmptyProfile(t *testing.T) {
	var gotProfile string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotProfile = r.Form.Get("profile")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if err := c.ClearUserCustomStatus(context.Background()); err != nil {
		t.Fatalf("ClearUserCustomStatus: %v", err)
	}
	var profile statusProfileWire
	if err := json.Unmarshal([]byte(gotProfile), &profile); err != nil {
		t.Fatalf("profile JSON: %v (%s)", err, gotProfile)
	}
	if profile.StatusText != "" || profile.StatusEmoji != "" || profile.StatusExpiration != 0 {
		t.Errorf("clear profile = %+v, want empty", profile)
	}
}

func TestSetUserCustomStatus_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_profile"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	err := c.SetUserCustomStatus(context.Background(), UserStatus{Text: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid_profile") {
		t.Errorf("error = %v, want invalid_profile", err)
	}
}
