package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/slack-go/slack"
)

// UserStatus is a Slack custom status (users.profile.set / boot profile).
// Expiration is a unix-seconds timestamp; 0 means the status does not expire.
type UserStatus struct {
	Text       string
	Emoji      string
	Expiration int64
}

// Active reports whether the status should be shown at now.
// Empty text and emoji is inactive. Expiration 0 never expires;
// a non-zero expiration is inactive once now is at or past it.
func (s UserStatus) Active(now time.Time) bool {
	if s.Text == "" && s.Emoji == "" {
		return false
	}
	if s.Expiration <= 0 {
		return true
	}
	return now.Unix() < s.Expiration
}

// UserStatusFromSlack extracts custom status fields from a users.info payload.
func UserStatusFromSlack(u *slack.User) UserStatus {
	if u == nil {
		return UserStatus{}
	}
	return UserStatus{
		Text:       u.Profile.StatusText,
		Emoji:      u.Profile.StatusEmoji,
		Expiration: int64(u.Profile.StatusExpiration),
	}
}

type statusProfileWire struct {
	StatusText       string `json:"status_text"`
	StatusEmoji      string `json:"status_emoji"`
	StatusExpiration int64  `json:"status_expiration"`
}

// SetUserCustomStatus sets the authenticated user's Slack custom status
// via users.profile.set. Empty text and emoji with expiration 0 clears it.
func (c *Client) SetUserCustomStatus(ctx context.Context, st UserStatus) error {
	rawProfile, err := json.Marshal(statusProfileWire{
		StatusText:       st.Text,
		StatusEmoji:      st.Emoji,
		StatusExpiration: st.Expiration,
	})
	if err != nil {
		return fmt.Errorf("encoding status profile: %w", err)
	}
	raw, err := c.postForm(ctx, "users.profile.set", url.Values{
		"profile": {string(rawProfile)},
	})
	if err != nil {
		return fmt.Errorf("setting user status: %w", err)
	}
	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("parsing users.profile.set response: %w (body=%q)", err, truncateForLog(raw))
	}
	if !parsed.OK {
		apiErr := parsed.Error
		if apiErr == "" {
			apiErr = "ok=false with no error field"
		}
		return fmt.Errorf("users.profile.set: API error: %s", apiErr)
	}
	return nil
}

// ClearUserCustomStatus clears the authenticated user's custom status.
func (c *Client) ClearUserCustomStatus(ctx context.Context) error {
	return c.SetUserCustomStatus(ctx, UserStatus{})
}
