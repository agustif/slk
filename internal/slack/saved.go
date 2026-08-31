package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/agustif/slk/internal/slackhttp"
)

// SavedCounts is client.counts' saved object and saved.list's counts:
// Later-tab incomplete / overdue / archived / completed totals.
// Captured shape matches the official client's saved.list (filter=saved)
// and client.counts.saved (2023 Later rollout; still current in 2026 HARs).
type SavedCounts struct {
	Uncompleted        int
	UncompletedOverdue int
	Archived           int
	Completed          int
	Total              int
}

// Badge is the Later sidebar count: incomplete (in-progress) items.
func (s SavedCounts) Badge() int {
	if s.Uncompleted < 0 {
		return 0
	}
	return s.Uncompleted
}

// SavedItem is one Later / Save-for-later row from saved.list, flattened
// so the TUI can render and jump without extra message fetches.
//
// Captured fields (saved.list, xoxc, _x_reason=saved-api/savedList):
//
//	item_id   — channel / DM ID for item_type=message
//	item_type — "message" (files and other types are skipped in the TUI)
//	ts        — message timestamp
//	state     — in_progress | completed | archived (filter "saved" = in progress)
//	date_due  — unix seconds when a reminder is attached; 0 if none
type SavedItem struct {
	Key           string
	ItemID        string
	ItemType      string
	TS            string
	State         string
	DateCreated   int64
	DateDue       int64
	DateCompleted int64
	IsArchived    bool
	// Text and UserID are not on the saved.list row itself (the
	// official client hydrates them). Filled from a nested message
	// object when present, then from cache / conversations.history.
	Text   string
	UserID string
}

// SavedListOpts is the flattened saved.list request.
type SavedListOpts struct {
	Filter string // saved (in progress) | completed | archived
	Limit  int
	Cursor string
}

// SavedListResult is one saved.list page plus the Later badge counts.
type SavedListResult struct {
	Items  []SavedItem
	Counts SavedCounts
	Cursor string
}

func savedListFilter(filter string) string {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "completed", "archived":
		return strings.ToLower(strings.TrimSpace(filter))
	default:
		return "saved"
	}
}

func buildSavedListForm(opts SavedListOpts) url.Values {
	limit := opts.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	form := url.Values{
		"limit":              {strconv.Itoa(limit)},
		"filter":             {savedListFilter(opts.Filter)},
		"include_tombstones": {"true"},
	}
	if opts.Cursor != "" {
		form.Set("cursor", opts.Cursor)
	}
	return form
}

// GetSavedList fetches Slack's Later tab (saved.list). filter=saved is
// the In progress tab the official client shows first.
func (c *Client) GetSavedList(ctx context.Context, opts SavedListOpts) (SavedListResult, error) {
	ctx = slackhttp.WithReason(ctx, "saved-api/savedList")
	raw, err := c.PostForm(ctx, "saved.list", buildSavedListForm(opts))
	if err != nil {
		return SavedListResult{}, fmt.Errorf("saved.list: %w", err)
	}
	res, err := parseSavedList(raw)
	if err != nil {
		return SavedListResult{}, err
	}
	return res, nil
}

// SavedAdd saves a message into Later (saved.add). item_id is the
// channel / DM ID; ts is the message timestamp.
func (c *Client) SavedAdd(ctx context.Context, channelID, ts string) error {
	ctx = slackhttp.WithReason(ctx, "saved-api/addSavedMessage")
	form := url.Values{
		"item_id":   {channelID},
		"item_type": {"message"},
		"ts":        {ts},
	}
	raw, err := c.PostForm(ctx, "saved.add", form)
	if err != nil {
		return fmt.Errorf("saved.add: %w", err)
	}
	return parseSavedOK(raw, "saved.add")
}

// SavedDelete removes a message from Later (saved.delete).
func (c *Client) SavedDelete(ctx context.Context, channelID, ts string) error {
	ctx = slackhttp.WithReason(ctx, "saved-api/deleteSavedMessage")
	form := url.Values{
		"item_id":   {channelID},
		"item_type": {"message"},
		"ts":        {ts},
	}
	raw, err := c.PostForm(ctx, "saved.delete", form)
	if err != nil {
		return fmt.Errorf("saved.delete: %w", err)
	}
	return parseSavedOK(raw, "saved.delete")
}

// SavedUpdateDue sets (or refreshes) a Later reminder on a saved
// message via saved.update date_due. dueUnix is unix seconds.
func (c *Client) SavedUpdateDue(ctx context.Context, channelID, ts string, dueUnix int64) error {
	ctx = slackhttp.WithReason(ctx, "saved-api/updateSavedMessage")
	form := url.Values{
		"item_id":   {channelID},
		"item_type": {"message"},
		"ts":        {ts},
		"date_due":  {strconv.FormatInt(dueUnix, 10)},
	}
	raw, err := c.PostForm(ctx, "saved.update", form)
	if err != nil {
		return fmt.Errorf("saved.update: %w", err)
	}
	return parseSavedOK(raw, "saved.update")
}

// SavedUpdateState moves a Later item between In progress / Completed /
// Archived via saved.update state. Slack's saved.list rows use the
// same field (in_progress | completed | archived).
func (c *Client) SavedUpdateState(ctx context.Context, channelID, ts, state string) error {
	state = savedItemState(state)
	ctx = slackhttp.WithReason(ctx, "saved-api/updateSavedMessage")
	form := url.Values{
		"item_id":   {channelID},
		"item_type": {"message"},
		"ts":        {ts},
		"state":     {state},
	}
	raw, err := c.PostForm(ctx, "saved.update", form)
	if err != nil {
		return fmt.Errorf("saved.update: %w", err)
	}
	return parseSavedOK(raw, "saved.update")
}

func savedItemState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "completed", "archived":
		return strings.ToLower(strings.TrimSpace(state))
	default:
		return "in_progress"
	}
}

// AddReminder creates a Slack reminder (reminders.add) for the
// authenticated user. timeSpec is a unix timestamp, a seconds offset
// (if within 24h), or a natural-language phrase Slack accepts
// ("in 20 minutes"). channelID is optional; permalink/ts belong in text.
func (c *Client) AddReminder(ctx context.Context, text, timeSpec, channelID string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("reminders.add: text is required")
	}
	if strings.TrimSpace(timeSpec) == "" {
		return fmt.Errorf("reminders.add: time is required")
	}
	form := url.Values{
		"text": {text},
		"time": {timeSpec},
	}
	if channelID != "" {
		form.Set("channel", channelID)
	}
	raw, err := c.PostForm(ctx, "reminders.add", form)
	if err != nil {
		return fmt.Errorf("reminders.add: %w", err)
	}
	return parseSavedOK(raw, "reminders.add")
}

// Reminder is one reminders.list row.
type Reminder struct {
	ID         string
	Text       string
	Time       int64
	CompleteTS int64
	Recurring  bool
}

type remindersListResponse struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	Reminders []struct {
		ID         string `json:"id"`
		Text       string `json:"text"`
		Time       int64  `json:"time"`
		CompleteTS int64  `json:"complete_ts"`
		Recurring  bool   `json:"recurring"`
	} `json:"reminders"`
}

// ListReminders returns the user's reminders (reminders.list).
func (c *Client) ListReminders(ctx context.Context) ([]Reminder, error) {
	raw, err := c.PostForm(ctx, "reminders.list", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("reminders.list: %w", err)
	}
	var res remindersListResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("reminders.list: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("reminders.list: %s", errStr)
	}
	out := make([]Reminder, 0, len(res.Reminders))
	for _, e := range res.Reminders {
		if e.ID == "" {
			continue
		}
		out = append(out, Reminder{
			ID:         e.ID,
			Text:       e.Text,
			Time:       e.Time,
			CompleteTS: e.CompleteTS,
			Recurring:  e.Recurring,
		})
	}
	return out, nil
}

// CompleteReminder marks a reminder done (reminders.complete).
func (c *Client) CompleteReminder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("reminders.complete: id required")
	}
	raw, err := c.PostForm(ctx, "reminders.complete", url.Values{"reminder": {id}})
	if err != nil {
		return fmt.Errorf("reminders.complete: %w", err)
	}
	return parseSavedOK(raw, "reminders.complete")
}

// DeleteReminder removes a reminder (reminders.delete).
func (c *Client) DeleteReminder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("reminders.delete: id required")
	}
	raw, err := c.PostForm(ctx, "reminders.delete", url.Values{"reminder": {id}})
	if err != nil {
		return fmt.Errorf("reminders.delete: %w", err)
	}
	return parseSavedOK(raw, "reminders.delete")
}

type savedListResponse struct {
	OK       bool            `json:"ok"`
	Error    string          `json:"error"`
	Items    []savedListItem `json:"saved_items"`
	Counts   savedCountsJSON `json:"counts"`
	Metadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

type savedListItem struct {
	ItemID        string `json:"item_id"`
	ItemType      string `json:"item_type"`
	TS            string `json:"ts"`
	State         string `json:"state"`
	DateCreated   int64  `json:"date_created"`
	DateDue       int64  `json:"date_due"`
	DateCompleted int64  `json:"date_completed"`
	IsArchived    bool   `json:"is_archived"`
	Message       *struct {
		Text string `json:"text"`
		User string `json:"user"`
		TS   string `json:"ts"`
	} `json:"message"`
}

type savedCountsJSON struct {
	Uncompleted        int `json:"uncompleted_count"`
	UncompletedOverdue int `json:"uncompleted_overdue_count"`
	Archived           int `json:"archived_count"`
	Completed          int `json:"completed_count"`
	Total              int `json:"total_count"`
}

type savedOKResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func parseSavedList(raw []byte) (SavedListResult, error) {
	var res savedListResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return SavedListResult{}, fmt.Errorf("saved.list: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return SavedListResult{}, fmt.Errorf("saved.list: %s", errStr)
	}
	out := SavedListResult{
		Items:  make([]SavedItem, 0, len(res.Items)),
		Counts: flattenSavedCounts(res.Counts),
		Cursor: res.Metadata.NextCursor,
	}
	for _, e := range res.Items {
		item, ok := flattenSavedItem(e)
		if !ok {
			continue
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func flattenSavedItem(e savedListItem) (SavedItem, bool) {
	if e.ItemID == "" {
		return SavedItem{}, false
	}
	typ := e.ItemType
	if typ == "" {
		typ = "message"
	}
	ts := e.TS
	if typ == "message" && ts == "" {
		return SavedItem{}, false
	}
	state := e.State
	if state == "" {
		state = "in_progress"
	}
	text, userID := "", ""
	if e.Message != nil {
		text = e.Message.Text
		userID = e.Message.User
		if ts == "" {
			ts = e.Message.TS
		}
	}
	return SavedItem{
		Key:           savedItemKey(e.ItemID, ts),
		ItemID:        e.ItemID,
		ItemType:      typ,
		TS:            ts,
		State:         state,
		DateCreated:   e.DateCreated,
		DateDue:       e.DateDue,
		DateCompleted: e.DateCompleted,
		IsArchived:    e.IsArchived,
		Text:          text,
		UserID:        userID,
	}, true
}

func savedItemKey(channelID, ts string) string {
	return channelID + "\t" + ts
}

func flattenSavedCounts(c savedCountsJSON) SavedCounts {
	return SavedCounts{
		Uncompleted:        c.Uncompleted,
		UncompletedOverdue: c.UncompletedOverdue,
		Archived:           c.Archived,
		Completed:          c.Completed,
		Total:              c.Total,
	}
}

func parseSavedOK(raw []byte, method string) error {
	var res savedOKResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return fmt.Errorf("%s: decoding: %w", method, err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return fmt.Errorf("%s: %s", method, errStr)
	}
	return nil
}

// parseSavedCounts decodes client.counts' saved object into SavedCounts.
func parseSavedCounts(raw json.RawMessage) SavedCounts {
	if len(raw) == 0 {
		return SavedCounts{}
	}
	var c savedCountsJSON
	if err := json.Unmarshal(raw, &c); err != nil {
		return SavedCounts{}
	}
	return flattenSavedCounts(c)
}

// AlreadySaved reports whether err is Slack saying the message is
// already in Later (idempotent add).
func AlreadySaved(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "already") || strings.Contains(s, "duplicate")
}
