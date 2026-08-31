package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gammons/slk/internal/slackhttp"
)

// Bookmark is one channel header bookmark from bookmarks.list.
type Bookmark struct {
	ID    string
	Title string
	Link  string
	Type  string
	Emoji string
}

const bookmarksListReason = "bookmarks-store/conditional-fetching"

// GetBookmarks wraps bookmarks.list for channelID. Empty-title
// entries are dropped; the official client still returns them.
func (c *Client) GetBookmarks(ctx context.Context, channelID string) ([]Bookmark, error) {
	if channelID == "" {
		return nil, fmt.Errorf("bookmarks.list: channelID is required")
	}
	if slackhttp.ReasonFrom(ctx) == "" {
		ctx = slackhttp.WithReason(ctx, bookmarksListReason)
	}
	raw, err := c.PostForm(ctx, "bookmarks.list", url.Values{"channel_id": {channelID}})
	if err != nil {
		return nil, fmt.Errorf("bookmarks.list: %w", err)
	}
	out, err := parseBookmarksList(raw)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type bookmarksListResponse struct {
	OK        bool              `json:"ok"`
	Error     string            `json:"error"`
	Bookmarks []bookmarkListRow `json:"bookmarks"`
}

type bookmarkListRow struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

func parseBookmarksList(raw []byte) ([]Bookmark, error) {
	var res bookmarksListResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("bookmarks.list: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("bookmarks.list: %s", errStr)
	}
	out := make([]Bookmark, 0, len(res.Bookmarks))
	for _, row := range res.Bookmarks {
		if row.Title == "" {
			continue
		}
		out = append(out, Bookmark{
			ID:    row.ID,
			Title: row.Title,
			Link:  row.Link,
			Type:  row.Type,
			Emoji: row.Emoji,
		})
	}
	return out, nil
}
