package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/agustif/slk/internal/slackhttp"
	"github.com/google/uuid"
)

// InlineSearchItem is one search.inline hit from the Cmd+K omniswitcher
// (2026-09-01): items[] objects with channel / ts / text. Empty items
// were observed for queries with no prefix matches; the envelope still
// carries pagination.
type InlineSearchItem struct {
	ChannelID   string
	ChannelName string
	TS          string
	ThreadTS    string
	User        string
	Username    string
	Text        string
}

// SearchInline POSTs search.inline as captured from Cmd+K
// (_x_reason=quick-messages/prototype): query, count=3, page=1,
// extract_len=110, from_me=true, with_me=true, recent_channels,
// search_session_id, client_req_id, max_ts, empty thread_replies.
func (c *Client) SearchInline(ctx context.Context, query, sessionID string, recentChannels []string) ([]InlineSearchItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	ctx = slackhttp.WithReason(ctx, "quick-messages/prototype")
	form := url.Values{
		"search_session_id": {sessionID},
		"client_req_id":     {uuid.NewString()},
		"max_ts":            {strconv.FormatInt(time.Now().Unix(), 10)},
		"count":             {"3"},
		"page":              {"1"},
		"query":             {query},
		"thread_replies":    {""},
		"extract_len":       {"110"},
		"from_me":           {"true"},
		"with_me":           {"true"},
	}
	if len(recentChannels) > 0 {
		form.Set("recent_channels", strings.Join(recentChannels, ","))
	}
	raw, err := c.postForm(ctx, "search.inline", form)
	if err != nil {
		return nil, fmt.Errorf("search.inline: %w", err)
	}
	return parseInlineSearch(raw)
}

func parseInlineSearch(raw []byte) ([]InlineSearchItem, error) {
	var env struct {
		OK    bool              `json:"ok"`
		Error string            `json:"error"`
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != "" {
			return nil, fmt.Errorf("%s", env.Error)
		}
		return nil, fmt.Errorf("ok=false")
	}
	out := make([]InlineSearchItem, 0, len(env.Items))
	for _, rawItem := range env.Items {
		it, ok := parseInlineItem(rawItem)
		if ok {
			out = append(out, it)
		}
	}
	return out, nil
}

func parseInlineItem(raw json.RawMessage) (InlineSearchItem, bool) {
	var w struct {
		Channel    json.RawMessage `json:"channel"`
		ChannelID  string          `json:"channel_id"`
		TS         string          `json:"ts"`
		Timestamp  string          `json:"timestamp"`
		ThreadTS   string          `json:"thread_ts"`
		User       string          `json:"user"`
		Username   string          `json:"username"`
		Text       string          `json:"text"`
		Extract    string          `json:"extract"`
		IID        string          `json:"iid"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return InlineSearchItem{}, false
	}
	chID, chName := parseInlineChannel(w.Channel)
	if chID == "" {
		chID = w.ChannelID
	}
	ts := w.TS
	if ts == "" {
		ts = w.Timestamp
	}
	if ts == "" && w.IID != "" {
		if i := strings.LastIndex(w.IID, "."); i > 0 {
			ts = w.IID[i+1:]
		}
	}
	text := w.Extract
	if text == "" {
		text = w.Text
	}
	if chID == "" || ts == "" {
		return InlineSearchItem{}, false
	}
	return InlineSearchItem{
		ChannelID:   chID,
		ChannelName: chName,
		TS:          ts,
		ThreadTS:    w.ThreadTS,
		User:        w.User,
		Username:    w.Username,
		Text:        text,
	}, true
}

func parseInlineChannel(raw json.RawMessage) (id, name string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}
	if raw[0] == '"' {
		_ = json.Unmarshal(raw, &id)
		return id, ""
	}
	var ch struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &ch); err != nil {
		return "", ""
	}
	return ch.ID, ch.Name
}

// AutocompleteFile is one search.autocomplete.files hit from Cmd+K
// (_x_reason=omniswitcher:suggestions-from-searcher).
type AutocompleteFile struct {
	ID         string
	Name       string
	Title      string
	Permalink  string
	URLPrivate string
	ChannelID  string
}

// AutocompleteFiles POSTs search.autocomplete.files as captured:
// query, include_shares=true, _x_reason=omniswitcher:suggestions-from-searcher.
func (c *Client) AutocompleteFiles(ctx context.Context, query string) ([]AutocompleteFile, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	ctx = slackhttp.WithReason(ctx, "omniswitcher:suggestions-from-searcher")
	form := url.Values{
		"query":          {query},
		"include_shares": {"true"},
	}
	raw, err := c.postForm(ctx, "search.autocomplete.files", form)
	if err != nil {
		return nil, fmt.Errorf("search.autocomplete.files: %w", err)
	}
	return parseAutocompleteFiles(raw)
}

func parseAutocompleteFiles(raw []byte) ([]AutocompleteFile, error) {
	var env struct {
		OK      bool              `json:"ok"`
		Error   string            `json:"error"`
		Files   []json.RawMessage `json:"files"`
		Items   []json.RawMessage `json:"items"`
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != "" {
			return nil, fmt.Errorf("%s", env.Error)
		}
		return nil, fmt.Errorf("ok=false")
	}
	rows := env.Files
	if len(rows) == 0 {
		rows = env.Items
	}
	if len(rows) == 0 {
		rows = env.Results
	}
	out := make([]AutocompleteFile, 0, len(rows))
	for _, rawItem := range rows {
		f, ok := parseAutocompleteFile(rawItem)
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}

func parseAutocompleteFile(raw json.RawMessage) (AutocompleteFile, bool) {
	var w struct {
		ID                 string   `json:"id"`
		Name               string   `json:"name"`
		Title              string   `json:"title"`
		Permalink          string   `json:"permalink"`
		URLPrivate         string   `json:"url_private"`
		URLPrivateDownload string   `json:"url_private_download"`
		Channels           []string `json:"channels"`
		Channel            string   `json:"channel"`
	}
	if err := json.Unmarshal(raw, &w); err != nil || w.ID == "" {
		return AutocompleteFile{}, false
	}
	url := w.URLPrivateDownload
	if url == "" {
		url = w.URLPrivate
	}
	ch := w.Channel
	if ch == "" && len(w.Channels) > 0 {
		ch = w.Channels[0]
	}
	name := w.Title
	if name == "" {
		name = w.Name
	}
	if name == "" {
		name = w.ID
	}
	return AutocompleteFile{
		ID:         w.ID,
		Name:       name,
		Title:      w.Title,
		Permalink:  w.Permalink,
		URLPrivate: url,
		ChannelID:  ch,
	}, true
}
