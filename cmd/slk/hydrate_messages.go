package main

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gammons/slk/internal/cache"
	slackclient "github.com/gammons/slk/internal/slack"
	"github.com/gammons/slk/internal/ui"
	"github.com/gammons/slk/internal/ui/sidebar"
	"github.com/slack-go/slack"
)

const hydrateFetchWorkers = 6

type hydratePair struct {
	ChannelID string
	TS        string
}

type hydrateHit struct {
	Text   string
	UserID string
}

func hydratePairKey(channelID, ts string) string {
	return channelID + "\t" + ts
}

// hydrateMessages fills message text/user from SQLite, then fetches
// remaining rows (capped concurrency) so Later / Activity cards can
// show the actual body instead of "saved for later".
func hydrateMessages(ctx context.Context, db *cache.DB, client *slackclient.Client, pairs []hydratePair) map[string]hydrateHit {
	out := make(map[string]hydrateHit, len(pairs))
	if len(pairs) == 0 {
		return out
	}
	var misses []hydratePair
	for _, p := range pairs {
		if p.ChannelID == "" || p.TS == "" {
			continue
		}
		key := hydratePairKey(p.ChannelID, p.TS)
		if db != nil {
			if msg, err := db.GetMessage(p.ChannelID, p.TS); err == nil && !msg.IsDeleted && strings.TrimSpace(msg.Text) != "" {
				out[key] = hydrateHit{Text: msg.Text, UserID: msg.UserID}
				continue
			}
		}
		misses = append(misses, p)
	}
	if len(misses) == 0 || client == nil {
		return out
	}

	type fetched struct {
		key string
		hit hydrateHit
		ch  string
		ts  string
		raw slack.Message
		ok  bool
	}
	ch := make(chan fetched, len(misses))
	sem := make(chan struct{}, hydrateFetchWorkers)
	var wg sync.WaitGroup
	for _, p := range misses {
		wg.Add(1)
		go func(p hydratePair) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			msg, err := client.GetSingleMessage(ctx, p.ChannelID, p.TS)
			if err != nil || msg == nil {
				return
			}
			ch <- fetched{
				key: hydratePairKey(p.ChannelID, p.TS),
				hit: hydrateHit{Text: msg.Text, UserID: msg.User},
				ch:  p.ChannelID,
				ts:  p.TS,
				raw: *msg,
				ok:  true,
			}
		}(p)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	teamID := ""
	if client != nil {
		teamID = client.TeamID()
	}
	for f := range ch {
		if !f.ok {
			continue
		}
		out[f.key] = f.hit
		if db == nil {
			continue
		}
		_ = db.UpsertMessage(cache.Message{
			TS:          f.ts,
			ChannelID:   f.ch,
			WorkspaceID: teamID,
			UserID:      f.hit.UserID,
			Text:        f.hit.Text,
			ThreadTS:    f.raw.ThreadTimestamp,
			ReplyCount:  f.raw.ReplyCount,
			CreatedAt:   time.Now().Unix(),
			Subtype:     f.raw.SubType,
		})
	}
	return out
}

func hydrateSavedItems(ctx context.Context, db *cache.DB, client *slackclient.Client, items []slackclient.SavedItem) {
	var pairs []hydratePair
	for _, it := range items {
		if it.ItemType != "" && it.ItemType != "message" {
			continue
		}
		if it.Text != "" {
			continue
		}
		pairs = append(pairs, hydratePair{ChannelID: it.ItemID, TS: it.TS})
	}
	hits := hydrateMessages(ctx, db, client, pairs)
	for i, it := range items {
		if it.Text != "" {
			continue
		}
		h, ok := hits[hydratePairKey(it.ItemID, it.TS)]
		if !ok {
			continue
		}
		items[i].Text = h.Text
		if items[i].UserID == "" {
			items[i].UserID = h.UserID
		}
	}
}

func fetchDMsSnippets(ctx context.Context, db *cache.DB, client *slackclient.Client, teamID string, channelIDs []string) ui.DMsSnippetsMsg {
	snips := map[string]sidebar.DMSnippet{}
	if len(channelIDs) == 0 {
		return ui.DMsSnippetsMsg{TeamID: teamID, Snips: snips}
	}
	if db != nil {
		for id, msg := range db.LatestByChannels(channelIDs) {
			snips[id] = sidebar.DMSnippet{
				Text:     msg.Text,
				UserID:   msg.UserID,
				Activity: slackTSUnix(msg.TS),
			}
		}
	}
	var missing []string
	for _, id := range channelIDs {
		if s := snips[id]; s.Text == "" {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 || client == nil {
		return ui.DMsSnippetsMsg{TeamID: teamID, Snips: snips}
	}
	type fetched struct {
		id  string
		msg slack.Message
		ok  bool
	}
	ch := make(chan fetched, len(missing))
	sem := make(chan struct{}, hydrateFetchWorkers)
	var wg sync.WaitGroup
	for _, id := range missing {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			msg, err := client.GetLatestMessage(ctx, id)
			if err != nil || msg == nil {
				return
			}
			ch <- fetched{id: id, msg: *msg, ok: true}
		}(id)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	ws := teamID
	if client != nil && ws == "" {
		ws = client.TeamID()
	}
	for f := range ch {
		if !f.ok {
			continue
		}
		snips[f.id] = sidebar.DMSnippet{
			Text:     f.msg.Text,
			UserID:   f.msg.User,
			Activity: slackTSUnix(f.msg.Timestamp),
		}
		if db == nil {
			continue
		}
		_ = db.UpsertMessage(cache.Message{
			TS:          f.msg.Timestamp,
			ChannelID:   f.id,
			WorkspaceID: ws,
			UserID:      f.msg.User,
			Text:        f.msg.Text,
			ThreadTS:    f.msg.ThreadTimestamp,
			ReplyCount:  f.msg.ReplyCount,
			CreatedAt:   time.Now().Unix(),
			Subtype:     f.msg.SubType,
		})
	}
	return ui.DMsSnippetsMsg{TeamID: teamID, Snips: snips}
}

func slackTSUnix(ts string) int64 {
	if ts == "" {
		return 0
	}
	if i := strings.IndexByte(ts, '.'); i >= 0 {
		ts = ts[:i]
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func hydrateActivityItems(ctx context.Context, db *cache.DB, client *slackclient.Client, items []slackclient.ActivityItem) {
	var pairs []hydratePair
	for _, it := range items {
		if it.Text != "" || it.ChannelID == "" || it.MessageTS == "" {
			continue
		}
		pairs = append(pairs, hydratePair{ChannelID: it.ChannelID, TS: it.MessageTS})
	}
	hits := hydrateMessages(ctx, db, client, pairs)
	for i, it := range items {
		if it.Text != "" {
			continue
		}
		h, ok := hits[hydratePairKey(it.ChannelID, it.MessageTS)]
		if !ok {
			continue
		}
		items[i].Text = h.Text
		if items[i].ActorID == "" {
			items[i].ActorID = h.UserID
		}
	}
}
