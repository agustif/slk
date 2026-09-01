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

// ActivityCounts is client.counts' activity_v2 object: unread Activity
// items grouped by the same type strings activity.feed uses. Sum is
// the Activity sidebar badge. Captured 2026-08-31 against Obvious AI
// (T099JCA82HJ) via the same xoxc session slk already holds.
type ActivityCounts struct {
	ByType map[string]int
}

// Unread returns the sum of all activity_v2 type counts.
func (a ActivityCounts) Unread() int {
	n := 0
	for _, v := range a.ByType {
		n += v
	}
	return n
}

// ActivityItem is one row in Slack's Activity tab, flattened from
// activity.feed's per-type payloads so the TUI can render and jump
// without switching on nested bundle_info shapes.
//
// Captured shapes (activity.feed, 2026-08-31):
//
//	channel / dm   — bundle_info.payload.{channel,dm}_entry.latest_message
//	thread_v2      — bundle_info.payload.thread_entry.{channel_id,thread_ts,latest_ts}
//	message_reaction — message.{channel,ts} + reaction.{user,name}
//	at_user / at_channel / at_* — message.{channel,ts,author_user_id}
type ActivityItem struct {
	Key       string
	Type      string
	Unread    bool
	VIP       bool
	FeedTS    string
	ChannelID string
	MessageTS string
	ThreadTS  string
	ActorID   string
	Reaction  string
	// Text is the message body when the feed included it (latest_message
	// or message.text). Empty means hydrate from cache / history.
	Text string
}

// ActivityView is one Activity-tab row from activity.views: Slack's
// built-in All/DMs/Mentions/Threads plus any user-created custom
// views (Unreads, Reactions, VIP, …). Captured 2026-08-31.
//
// Selecting a view does not send view_id to activity.feed — the
// official client flattens filters/sort onto the feed form
// (entry_types → types, unread_only, priority_only).
type ActivityView struct {
	ID       string
	Name     string
	Type     string // all | dms | mentions | threads | custom
	Position string
	Sort     string // newest | vip_unreads_first
	Density  string // compact | detailed
	Filters  ActivityViewFilters
}

// ActivityViewFilters is the subset of a view's filters that
// activity.feed actually honors.
type ActivityViewFilters struct {
	EntryTypes   []string
	UnreadOnly   bool
	PriorityOnly bool
}

// activityFeedTypes is the All-tab `types` field from a 2026-08-31
// capture of app.slack.com's Activity inbox (activity.feed,
// _x_reason=fetchActivityFeed). Duplicates are what the official
// client sent; they are harmless.
const activityFeedTypes = "at_user,at_user_group,at_channel,at_everyone,keyword,list_record_assigned,list_user_mentioned,list_todo_notification,list_approval_request,list_approval_reviewed,unjoined_channel_mention,at_user,unjoined_channel_mention,at_channel,at_everyone,at_user_group,keyword,thread_v2,message_reaction,bot_dm_bundle,dm,prejoin_dm_welcome_party_alert,internal_channel_invite,external_channel_invite,external_dm_invite,quietly_added_to_channel,channel,saved_reminder,list_record_edited"

// ActivityFeedOpts is the flattened activity.feed request. Prefer
// FeedOptsFromView so custom tabs (unread_only / priority_only /
// entry_types) stay in lockstep with activity.views. Filter is the
// builtin fallback when no view is loaded yet.
type ActivityFeedOpts struct {
	Filter       string
	Types        []string
	Sort         string
	UnreadOnly   bool
	PriorityOnly bool
	Limit        int
}

// FeedOptsFromView flattens an activity.views row into the feed
// params the official client sends when that tab is selected.
func FeedOptsFromView(v ActivityView) ActivityFeedOpts {
	sort := "newest"
	if v.Sort == "vip_unreads_first" {
		sort = "unreads_first"
	}
	return ActivityFeedOpts{
		Filter:       v.Type,
		Types:        append([]string(nil), v.Filters.EntryTypes...),
		Sort:         sort,
		UnreadOnly:   v.Filters.UnreadOnly,
		PriorityOnly: v.Filters.PriorityOnly,
	}
}

// GetActivityViews fetches the Activity tab list (activity.views),
// including user-created custom views. Empty / failed responses
// should fall back to BuiltinActivityViews.
func (c *Client) GetActivityViews(ctx context.Context) ([]ActivityView, error) {
	raw, err := c.PostForm(ctx, "activity.views", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("activity.views: %w", err)
	}
	views, err := parseActivityViews(raw)
	if err != nil {
		return nil, err
	}
	return views, nil
}

// BuiltinActivityViews is the All/DMs/Mentions/Threads set the
// official inbox always shows, used before activity.views lands
// and if that call fails.
func BuiltinActivityViews() []ActivityView {
	return []ActivityView{
		{ID: "all", Name: "All", Type: "all", Position: "1000000000", Sort: "newest", Density: "compact"},
		{ID: "dms", Name: "DMs", Type: "dms", Position: "2000000000", Sort: "vip_unreads_first", Filters: ActivityViewFilters{EntryTypes: []string{"dm"}}},
		{ID: "mentions", Name: "Mentions", Type: "mentions", Position: "3000000000", Sort: "vip_unreads_first", Filters: ActivityViewFilters{EntryTypes: []string{"at_user", "at_user_group", "at_channel", "at_everyone", "keyword", "list_record_assigned", "list_user_mentioned", "list_todo_notification", "unjoined_channel_mention"}}},
		{ID: "threads", Name: "Threads", Type: "threads", Position: "4000000000", Sort: "vip_unreads_first", Filters: ActivityViewFilters{EntryTypes: []string{"thread_v2"}}},
	}
}

// GetActivityFeed fetches Slack's Activity tab (activity.feed).
//
// Official All-tab capture (2026-08-31): mode=chrono_v1,
// is_activity_inbox=true, the types list in activityFeedTypes.
// DMs/Mentions/Threads tabs send mode=priority_reads_and_unreads_v1
// and sort=vip_unreads_first. Filter maps onto the captured type
// groups for those tabs plus Reactions.
func (c *Client) GetActivityFeed(ctx context.Context, opts ActivityFeedOpts) ([]ActivityItem, error) {
	if slackhttp.ReasonFrom(ctx) == "" {
		ctx = slackhttp.WithReason(ctx, "fetchActivityFeed")
	}
	raw, err := c.PostForm(ctx, "activity.feed", buildActivityFeedForm(opts))
	if err != nil {
		return nil, fmt.Errorf("activity.feed: %w", err)
	}
	items, err := parseActivityFeed(raw)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func buildActivityFeedForm(opts ActivityFeedOpts) url.Values {
	limit := opts.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	mode, sort := activityModeForSort(opts.Sort)
	form := url.Values{
		"limit":                    {strconv.Itoa(limit)},
		"types":                    {typesForFeedOpts(opts)},
		"mode":                     {mode},
		"archive_only":             {"false"},
		"unread_only":              {boolForm(opts.UnreadOnly)},
		"priority_only":            {boolForm(opts.PriorityOnly)},
		"only_salesforce_channels": {"false"},
		"exclude_automations":      {"false"},
		"automations_only":         {"false"},
		"is_activity_inbox":        {"true"},
	}
	if sort != "" {
		form.Set("sort", sort)
	}
	return form
}

func boolForm(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func typesForFeedOpts(opts ActivityFeedOpts) string {
	if len(opts.Types) > 0 {
		return strings.Join(opts.Types, ",")
	}
	return activityTypesForFilter(opts.Filter)
}

func activityTypesForFilter(filter string) string {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "dms":
		return "dm"
	case "mentions":
		return "at_user,at_user_group,at_channel,at_everyone,keyword,list_record_assigned,list_user_mentioned,list_todo_notification,unjoined_channel_mention"
	case "threads":
		return "thread_v2"
	case "reactions":
		return "message_reaction"
	default:
		return activityFeedTypes
	}
}

func activityModeForSort(sort string) (mode, sortField string) {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "unreads_first", "vip_unreads_first":
		return "priority_reads_and_unreads_v1", "vip_unreads_first"
	default:
		return "chrono_v1", ""
	}
}

type activityViewsResponse struct {
	OK    bool                `json:"ok"`
	Error string              `json:"error"`
	Views []activityViewEntry `json:"views"`
}

type activityViewEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ViewType string `json:"view_type"`
	Position string `json:"position"`
	Sort     string `json:"sort"`
	Density  string `json:"density"`
	Filters  struct {
		EntryTypes   []string `json:"entry_types"`
		UnreadOnly   bool     `json:"unread_only"`
		PriorityOnly bool     `json:"priority_only"`
	} `json:"filters"`
}

func parseActivityViews(raw []byte) ([]ActivityView, error) {
	var res activityViewsResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("activity.views: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("activity.views: %s", errStr)
	}
	out := make([]ActivityView, 0, len(res.Views))
	for _, v := range res.Views {
		if v.ID == "" && v.Name == "" {
			continue
		}
		out = append(out, ActivityView{
			ID:       v.ID,
			Name:     v.Name,
			Type:     v.ViewType,
			Position: v.Position,
			Sort:     v.Sort,
			Density:  v.Density,
			Filters: ActivityViewFilters{
				EntryTypes:   append([]string(nil), v.Filters.EntryTypes...),
				UnreadOnly:   v.Filters.UnreadOnly,
				PriorityOnly: v.Filters.PriorityOnly,
			},
		})
	}
	return out, nil
}

type activityFeedResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error"`
	Items []activityEntry `json:"items"`
}

type activityEntry struct {
	Key      string          `json:"key"`
	Unread   bool            `json:"is_unread"`
	FeedTS   string          `json:"feed_ts"`
	Archived bool            `json:"is_archived"`
	Item     json.RawMessage `json:"item"`
	Priority json.RawMessage `json:"priority"`
}

type activityInner struct {
	Type       string `json:"type"`
	BundleInfo struct {
		Payload struct {
			ChannelEntry struct {
				LatestMessage activityMsgRef `json:"latest_message"`
			} `json:"channel_entry"`
			DMEntry struct {
				LatestMessage activityMsgRef `json:"latest_message"`
			} `json:"dm_entry"`
			ThreadEntry struct {
				ChannelID string `json:"channel_id"`
				ThreadTS  string `json:"thread_ts"`
				LatestTS  string `json:"latest_ts"`
			} `json:"thread_entry"`
		} `json:"payload"`
	} `json:"bundle_info"`
	Message struct {
		TS       string `json:"ts"`
		Channel  string `json:"channel"`
		AuthorID string `json:"author_user_id"`
		User     string `json:"user"`
		Text     string `json:"text"`
	} `json:"message"`
	Reaction struct {
		User string `json:"user"`
		Name string `json:"name"`
	} `json:"reaction"`
}

type activityMsgRef struct {
	TS      string `json:"ts"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
	User    string `json:"user"`
}

func parseActivityFeed(raw []byte) ([]ActivityItem, error) {
	var res activityFeedResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("activity.feed: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("activity.feed: %s", errStr)
	}
	out := make([]ActivityItem, 0, len(res.Items))
	for _, e := range res.Items {
		if e.Archived {
			continue
		}
		item, ok := flattenActivityEntry(e)
		if !ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func flattenActivityEntry(e activityEntry) (ActivityItem, bool) {
	var inner activityInner
	if err := json.Unmarshal(e.Item, &inner); err != nil {
		return ActivityItem{}, false
	}
	item := ActivityItem{
		Key:    e.Key,
		Type:   inner.Type,
		Unread: e.Unread,
		VIP:    len(e.Priority) > 2,
		FeedTS: e.FeedTS,
	}
	switch inner.Type {
	case "channel":
		lm := inner.BundleInfo.Payload.ChannelEntry.LatestMessage
		item.ChannelID = lm.Channel
		item.MessageTS = lm.TS
		item.Text = lm.Text
		item.ActorID = lm.User
	case "dm":
		lm := inner.BundleInfo.Payload.DMEntry.LatestMessage
		item.ChannelID = lm.Channel
		item.MessageTS = lm.TS
		item.Text = lm.Text
		item.ActorID = lm.User
	case "thread_v2":
		te := inner.BundleInfo.Payload.ThreadEntry
		item.ChannelID = te.ChannelID
		item.ThreadTS = te.ThreadTS
		item.MessageTS = te.LatestTS
		if item.MessageTS == "" {
			item.MessageTS = te.ThreadTS
		}
	case "message_reaction":
		item.ChannelID = inner.Message.Channel
		item.MessageTS = inner.Message.TS
		item.ActorID = inner.Reaction.User
		item.Reaction = inner.Reaction.Name
		item.Text = inner.Message.Text
	default:
		// Mentions, keywords, invites, lists: message.{channel,ts}.
		item.ChannelID = inner.Message.Channel
		item.MessageTS = inner.Message.TS
		item.ActorID = inner.Message.AuthorID
		if item.ActorID == "" {
			item.ActorID = inner.Message.User
		}
		item.Text = inner.Message.Text
	}
	if item.ChannelID == "" {
		return ActivityItem{}, false
	}
	return item, true
}

// parseActivityV2 decodes client.counts' activity_v2 object into
// ActivityCounts. Unknown keys are kept so a new desktop type still
// contributes to the badge.
func parseActivityV2(raw json.RawMessage) ActivityCounts {
	if len(raw) == 0 {
		return ActivityCounts{}
	}
	var m map[string]int
	if err := json.Unmarshal(raw, &m); err != nil {
		return ActivityCounts{}
	}
	return ActivityCounts{ByType: m}
}
