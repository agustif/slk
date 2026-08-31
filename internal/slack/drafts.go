package slackclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gammons/slk/internal/slackhttp"
	"github.com/google/uuid"
)

// ComposerDraft is one unsent Slack composer draft (drafts.list).
// Text is flattened from the rich_text blocks; Destinations[0] is the
// channel (and optional thread) the draft is addressed to.
type ComposerDraft struct {
	ID             string
	LastUpdatedTS  string
	ChannelID      string
	ThreadTS       string
	Text           string
	DateCreated    int64
	IsFromComposer bool
}

// DraftsPage is one drafts.list page. NextTS is empty when has_more is false.
type DraftsPage struct {
	Drafts []ComposerDraft
	NextTS string
}

type draftsListResponse struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error"`
	Drafts  []draftsListRow `json:"drafts"`
	HasMore bool            `json:"has_more"`
}

type draftsListRow struct {
	ID             string          `json:"id"`
	LastUpdatedTS  string          `json:"last_updated_ts"`
	IsDeleted      bool            `json:"is_deleted"`
	IsSent         bool            `json:"is_sent"`
	IsFromComposer bool            `json:"is_from_composer"`
	DateCreated    json.RawMessage `json:"date_created"`
	Blocks         json.RawMessage `json:"blocks"`
	Destinations   json.RawMessage `json:"destinations"`
}

type draftDestination struct {
	ChannelID string `json:"channel_id"`
	ThreadTS  string `json:"thread_ts"`
}

type draftsWriteResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Draft struct {
		ID            string `json:"id"`
		LastUpdatedTS string `json:"last_updated_ts"`
	} `json:"draft"`
	ID            string `json:"id"`
	LastUpdatedTS string `json:"last_updated_ts"`
}

// ListComposerDrafts pages drafts.list with is_active=true until
// has_more is false. Observed 2026-08-31 on the official web client:
// multipart form fields is_active, limit, next_ts (not cursor).
func (c *Client) ListComposerDrafts(ctx context.Context) ([]ComposerDraft, error) {
	var all []ComposerDraft
	next := ""
	for {
		page, err := c.ListComposerDraftsPage(ctx, next, 100)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Drafts...)
		if page.NextTS == "" {
			return all, nil
		}
		next = page.NextTS
	}
}

// ListComposerDraftsPage fetches one drafts.list page (is_active=true).
func (c *Client) ListComposerDraftsPage(ctx context.Context, nextTS string, limit int) (DraftsPage, error) {
	ctx = slackhttp.WithReason(ctx, "drafts-api/list")
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	form := url.Values{
		"is_active": {"true"},
		"limit":     {fmt.Sprintf("%d", limit)},
	}
	if nextTS != "" {
		form.Set("next_ts", nextTS)
	}
	raw, err := c.PostForm(ctx, "drafts.list", form)
	if err != nil {
		return DraftsPage{}, fmt.Errorf("drafts.list: %w", err)
	}
	items, next, err := parseDraftsList(raw)
	if err != nil {
		return DraftsPage{}, err
	}
	return DraftsPage{Drafts: items, NextTS: next}, nil
}

// CountActiveDrafts is drafts.listActive — the Drafts sidebar badge.
func (c *Client) CountActiveDrafts(ctx context.Context) (int, error) {
	ctx = slackhttp.WithReason(ctx, "drafts-api/listActive")
	raw, err := c.PostForm(ctx, "drafts.listActive", url.Values{"limit": {"1000"}})
	if err != nil {
		return 0, fmt.Errorf("drafts.listActive: %w", err)
	}
	var res struct {
		OK             bool     `json:"ok"`
		Error          string   `json:"error"`
		ActiveDraftIDs []string `json:"active_draft_ids"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, fmt.Errorf("drafts.listActive: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return 0, fmt.Errorf("drafts.listActive: %s", errStr)
	}
	n := 0
	for _, id := range res.ActiveDraftIDs {
		if id != "" {
			n++
		}
	}
	return n, nil
}

func parseDraftsList(raw []byte) ([]ComposerDraft, string, error) {
	var res draftsListResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, "", fmt.Errorf("drafts.list: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, "", fmt.Errorf("drafts.list: %s", errStr)
	}
	out := make([]ComposerDraft, 0, len(res.Drafts))
	lastTS := ""
	for _, row := range res.Drafts {
		if row.LastUpdatedTS != "" {
			lastTS = row.LastUpdatedTS
		}
		if row.IsDeleted || row.IsSent || row.ID == "" {
			continue
		}
		d := ComposerDraft{
			ID:             row.ID,
			LastUpdatedTS:  row.LastUpdatedTS,
			IsFromComposer: row.IsFromComposer,
			DateCreated:    parseUnixField(row.DateCreated),
			Text:           richTextPlain(row.Blocks),
		}
		var dests []draftDestination
		if err := decodeJSONField(row.Destinations, &dests); err == nil && len(dests) > 0 {
			d.ChannelID = dests[0].ChannelID
			d.ThreadTS = dests[0].ThreadTS
		}
		if d.ChannelID == "" {
			continue
		}
		out = append(out, d)
	}
	next := ""
	if res.HasMore && lastTS != "" {
		next = lastTS
	}
	return out, next, nil
}

// CreateComposerDraft writes drafts.create. Official client sends
// multipart: blocks, destinations, file_ids, attachments,
// client_msg_id, is_from_composer.
func (c *Client) CreateComposerDraft(ctx context.Context, channelID, threadTS, text string) (ComposerDraft, error) {
	if channelID == "" {
		return ComposerDraft{}, fmt.Errorf("drafts.create: empty channel ID")
	}
	ctx = slackhttp.WithReason(ctx, "drafts-api/create")
	form := draftWriteForm(channelID, threadTS, text)
	form.Set("client_msg_id", uuid.NewString())
	form.Set("is_from_composer", "true")
	raw, err := c.PostForm(ctx, "drafts.create", form)
	if err != nil {
		return ComposerDraft{}, fmt.Errorf("drafts.create: %w", err)
	}
	return parseDraftWrite(raw, "drafts.create")
}

// UpdateComposerDraft writes drafts.update. Observed fields: draft_id,
// client_last_updated_ts, blocks, destinations, file_ids, attachments,
// is_from_composer.
func (c *Client) UpdateComposerDraft(ctx context.Context, draftID, lastUpdatedTS, channelID, threadTS, text string) (ComposerDraft, error) {
	if draftID == "" || lastUpdatedTS == "" {
		return ComposerDraft{}, fmt.Errorf("drafts.update: missing draft id or ts")
	}
	ctx = slackhttp.WithReason(ctx, "drafts-api/update")
	form := draftWriteForm(channelID, threadTS, text)
	form.Set("draft_id", draftID)
	form.Set("client_last_updated_ts", lastUpdatedTS)
	form.Set("is_from_composer", "true")
	raw, err := c.PostForm(ctx, "drafts.update", form)
	if err != nil {
		return ComposerDraft{}, fmt.Errorf("drafts.update: %w", err)
	}
	return parseDraftWrite(raw, "drafts.update")
}

// DeleteComposerDraft writes drafts.delete as form-encoded draft_id +
// client_last_updated_ts (matches the official client's delete path).
func (c *Client) DeleteComposerDraft(ctx context.Context, draftID, lastUpdatedTS string) error {
	if draftID == "" {
		return fmt.Errorf("drafts.delete: empty draft id")
	}
	ctx = slackhttp.WithReason(ctx, "drafts-api/delete")
	form := url.Values{"draft_id": {draftID}}
	if lastUpdatedTS != "" {
		form.Set("client_last_updated_ts", lastUpdatedTS)
	}
	raw, err := c.PostForm(ctx, "drafts.delete", form)
	if err != nil {
		return fmt.Errorf("drafts.delete: %w", err)
	}
	var res struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return fmt.Errorf("drafts.delete: decoding: %w", err)
	}
	if !res.OK {
		if res.Error == "draft_not_found" || res.Error == "not_found" {
			return nil
		}
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return fmt.Errorf("drafts.delete: %s", errStr)
	}
	return nil
}

func draftWriteForm(channelID, threadTS, text string) url.Values {
	dest := map[string]any{"channel_id": channelID}
	if threadTS != "" {
		dest["thread_ts"] = threadTS
	}
	destJSON, _ := json.Marshal([]map[string]any{dest})
	blocksJSON, _ := json.Marshal(richTextBlocks(text))
	return url.Values{
		"blocks":       {string(blocksJSON)},
		"destinations": {string(destJSON)},
		"file_ids":     {"[]"},
		"attachments":  {"[]"},
	}
}

func richTextBlocks(text string) []map[string]any {
	return []map[string]any{
		{
			"type": "rich_text",
			"elements": []map[string]any{
				{
					"type": "rich_text_section",
					"elements": []map[string]any{
						{"type": "text", "text": text},
					},
				},
			},
		},
	}
}

func parseDraftWrite(raw []byte, method string) (ComposerDraft, error) {
	var res draftsWriteResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return ComposerDraft{}, fmt.Errorf("%s: decoding: %w", method, err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return ComposerDraft{}, fmt.Errorf("%s: %s", method, errStr)
	}
	id := res.Draft.ID
	if id == "" {
		id = res.ID
	}
	ts := res.Draft.LastUpdatedTS
	if ts == "" {
		ts = res.LastUpdatedTS
	}
	if id == "" {
		return ComposerDraft{}, fmt.Errorf("%s: missing draft id", method)
	}
	return ComposerDraft{ID: id, LastUpdatedTS: ts}, nil
}

func parseUnixField(raw json.RawMessage) int64 {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return int64(f)
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		n, _ = strconv.ParseInt(s, 10, 64)
		return n
	}
	return 0
}

func decodeJSONField(raw json.RawMessage, dest any) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return json.Unmarshal([]byte(s), dest)
	}
	return json.Unmarshal(raw, dest)
}

func richTextPlain(raw json.RawMessage) string {
	var v any
	if err := decodeJSONField(raw, &v); err != nil {
		return ""
	}
	var b strings.Builder
	collectPlainText(v, &b)
	return strings.TrimSpace(b.String())
}

func collectPlainText(v any, b *strings.Builder) {
	switch t := v.(type) {
	case map[string]any:
		if typ, _ := t["type"].(string); typ == "text" {
			if s, ok := t["text"].(string); ok && s != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(s)
			}
			return
		}
		if els, ok := t["elements"]; ok {
			collectPlainText(els, b)
		}
	case []any:
		for _, el := range t {
			collectPlainText(el, b)
		}
	}
}

// DraftKeyFor builds the compose-box key used by slk (channel, or
// channel + NUL + thread_ts).
func DraftKeyFor(channelID, threadTS string) string {
	if channelID == "" {
		return ""
	}
	if threadTS == "" {
		return channelID
	}
	return channelID + "\x00" + threadTS
}

// SplitDraftKey reverses DraftKeyFor.
func SplitDraftKey(key string) (channelID, threadTS string) {
	channelID, threadTS, _ = strings.Cut(key, "\x00")
	return channelID, threadTS
}
