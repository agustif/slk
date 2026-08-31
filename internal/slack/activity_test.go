package slackclient

import (
	"os"
	"strings"
	"testing"
)

func TestParseActivityFeed_CapturedShapes(t *testing.T) {
	raw, err := os.ReadFile("testdata/activity-feed.json")
	if err != nil {
		t.Fatal(err)
	}
	items, err := parseActivityFeed(raw)
	if err != nil {
		t.Fatalf("parseActivityFeed: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("len = %d; want 6 (one of each captured type)", len(items))
	}
	byType := map[string]ActivityItem{}
	for _, it := range items {
		byType[it.Type] = it
	}
	ch := byType["channel"]
	if ch.ChannelID != "C0A95SPDL3B" || ch.MessageTS != "1787955901.383689" || !ch.Unread {
		t.Errorf("channel = %+v; want C0A95SPDL3B / 1787955901.383689 unread", ch)
	}
	th := byType["thread_v2"]
	if th.ChannelID != "C099JCB0WLC" || th.ThreadTS != "1787859788.452529" || th.MessageTS != "1788181025.052949" {
		t.Errorf("thread_v2 = %+v", th)
	}
	dm := byType["dm"]
	if dm.ChannelID != "D0ARB1GN6Q5" || dm.MessageTS != "1788133014.963939" {
		t.Errorf("dm = %+v", dm)
	}
	rx := byType["message_reaction"]
	if rx.ChannelID != "D0ARB1GN6Q5" || rx.ActorID != "U099FC5CXCJ" || rx.Reaction != "eyes" {
		t.Errorf("reaction = %+v", rx)
	}
	at := byType["at_user"]
	if at.ChannelID != "C0B4EUFSRU3" || at.ActorID != "U09PAK5229E" {
		t.Errorf("at_user = %+v", at)
	}
}

func TestParseActivityFeed_RejectsNotOK(t *testing.T) {
	_, err := parseActivityFeed([]byte(`{"ok":false,"error":"ratelimited"}`))
	if err == nil {
		t.Fatal("want error on ok=false")
	}
}

func TestActivityCountsUnread(t *testing.T) {
	a := parseActivityV2([]byte(`{"channel":30,"dm":2,"at_user":1}`))
	if got := a.Unread(); got != 33 {
		t.Errorf("Unread = %d; want 33", got)
	}
	if parseActivityV2(nil).Unread() != 0 {
		t.Error("empty activity_v2 should sum to 0")
	}
}

func TestBuildActivityFeedForm_DefaultsMatchAllTab(t *testing.T) {
	f := buildActivityFeedForm(ActivityFeedOpts{})
	if f.Get("mode") != "chrono_v1" {
		t.Errorf("mode = %q; want chrono_v1 (All-tab capture)", f.Get("mode"))
	}
	if f.Get("sort") != "" {
		t.Errorf("sort = %q; All-tab capture omitted sort", f.Get("sort"))
	}
	if f.Get("types") != activityFeedTypes {
		t.Errorf("types = %q; want the captured All-tab list", f.Get("types"))
	}
	if f.Get("unread_only") != "false" {
		t.Errorf("unread_only = %q; want false", f.Get("unread_only"))
	}
	if f.Get("is_activity_inbox") != "true" {
		t.Errorf("is_activity_inbox = %q; want true", f.Get("is_activity_inbox"))
	}
	if f.Get("limit") != "50" {
		t.Errorf("limit = %q; want 50", f.Get("limit"))
	}
}

func TestBuildActivityFeedForm_MentionsUnreadsFirst(t *testing.T) {
	f := buildActivityFeedForm(ActivityFeedOpts{
		Filter:     "mentions",
		Sort:       "unreads_first",
		UnreadOnly: true,
		Limit:      20,
	})
	if f.Get("types") != activityTypesForFilter("mentions") {
		t.Errorf("types = %q", f.Get("types"))
	}
	if !strings.Contains(f.Get("types"), "at_user") || strings.Contains(f.Get("types"), "thread_v2") {
		t.Errorf("mentions types should include at_user and exclude thread_v2: %q", f.Get("types"))
	}
	if f.Get("mode") != "priority_reads_and_unreads_v1" {
		t.Errorf("mode = %q; want priority_reads_and_unreads_v1", f.Get("mode"))
	}
	if f.Get("sort") != "vip_unreads_first" {
		t.Errorf("sort = %q; want vip_unreads_first", f.Get("sort"))
	}
	if f.Get("unread_only") != "true" {
		t.Errorf("unread_only = %q; want true", f.Get("unread_only"))
	}
	if f.Get("limit") != "20" {
		t.Errorf("limit = %q; want 20", f.Get("limit"))
	}
}

func TestBuildActivityFeedForm_ClampsLimit(t *testing.T) {
	if got := buildActivityFeedForm(ActivityFeedOpts{Limit: 0}).Get("limit"); got != "50" {
		t.Errorf("limit 0 → %q; want 50", got)
	}
	if got := buildActivityFeedForm(ActivityFeedOpts{Limit: 999}).Get("limit"); got != "100" {
		t.Errorf("limit 999 → %q; want 100", got)
	}
}

func TestParseActivityViews_IncludesCustom(t *testing.T) {
	raw, err := os.ReadFile("testdata/activity-views.json")
	if err != nil {
		t.Fatal(err)
	}
	views, err := parseActivityViews(raw)
	if err != nil {
		t.Fatalf("parseActivityViews: %v", err)
	}
	if len(views) != 7 {
		t.Fatalf("len = %d; want 7 (4 builtin + Unreads/Reactions/VIP)", len(views))
	}
	byName := map[string]ActivityView{}
	for _, v := range views {
		byName[v.Name] = v
	}
	if byName["Unreads"].Type != "custom" || !byName["Unreads"].Filters.UnreadOnly {
		t.Errorf("Unreads = %+v", byName["Unreads"])
	}
	if got := byName["Reactions"].Filters.EntryTypes; len(got) != 1 || got[0] != "message_reaction" {
		t.Errorf("Reactions types = %v", got)
	}
	if !byName["VIP"].Filters.PriorityOnly {
		t.Errorf("VIP missing priority_only: %+v", byName["VIP"])
	}
	unreads := FeedOptsFromView(byName["Unreads"])
	if !unreads.UnreadOnly || unreads.PriorityOnly || unreads.Sort != "newest" {
		t.Errorf("Unreads feed opts = %+v", unreads)
	}
	f := buildActivityFeedForm(unreads)
	if f.Get("unread_only") != "true" || f.Get("priority_only") != "false" {
		t.Errorf("Unreads form unread=%s priority=%s", f.Get("unread_only"), f.Get("priority_only"))
	}
	if f.Get("types") != activityFeedTypes {
		t.Error("Unreads tab keeps the full All-tab types list")
	}
	vip := buildActivityFeedForm(FeedOptsFromView(byName["VIP"]))
	if vip.Get("priority_only") != "true" {
		t.Errorf("VIP form priority_only = %q", vip.Get("priority_only"))
	}
	rx := buildActivityFeedForm(FeedOptsFromView(byName["Reactions"]))
	if rx.Get("types") != "message_reaction" {
		t.Errorf("Reactions types = %q", rx.Get("types"))
	}
}

func TestBuildActivityFeedForm_PriorityOnly(t *testing.T) {
	f := buildActivityFeedForm(ActivityFeedOpts{PriorityOnly: true, Sort: "newest"})
	if f.Get("priority_only") != "true" {
		t.Errorf("priority_only = %q", f.Get("priority_only"))
	}
	if f.Get("mode") != "chrono_v1" {
		t.Errorf("mode = %q", f.Get("mode"))
	}
}

func TestActivityTypesForFilter(t *testing.T) {
	if activityTypesForFilter("threads") != "thread_v2" {
		t.Errorf("threads types = %q", activityTypesForFilter("threads"))
	}
	if activityTypesForFilter("reactions") != "message_reaction" {
		t.Errorf("reactions types = %q", activityTypesForFilter("reactions"))
	}
	if activityTypesForFilter("bogus") != activityFeedTypes {
		t.Error("unknown filter should fall back to All-tab types")
	}
}

func TestFlattenMarksVIP(t *testing.T) {
	raw := []byte(`{"ok":true,"items":[{"is_unread":false,"feed_ts":"1","key":"t","priority":{"vip":{}},"item":{"type":"thread_v2","bundle_info":{"payload":{"thread_entry":{"channel_id":"C1","thread_ts":"1.0","latest_ts":"2.0"}}}}}]}`)
	items, err := parseActivityFeed(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].VIP {
		t.Errorf("VIP item = %+v", items)
	}
}

func TestFlattenSkipsMissingChannel(t *testing.T) {
	raw := []byte(`{"ok":true,"items":[{"is_unread":true,"feed_ts":"1","key":"x","item":{"type":"mystery"}}]}`)
	items, err := parseActivityFeed(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items; a type with no channel must be dropped, not shown as a blank row", len(items))
	}
}
