package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/agustif/slk/internal/debuglog"
	"github.com/agustif/slk/internal/ui/channelfinder"
	"github.com/google/uuid"
)

const maxOmniRecent = 15

func searchOmniRemote(ctx context.Context, wctx *WorkspaceContext, query, currentChannel, sessionID string, recent []string) []channelfinder.Item {
	query = strings.TrimSpace(query)
	if wctx == nil || wctx.Client == nil || query == "" {
		return nil
	}
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	if len(recent) > maxOmniRecent {
		recent = recent[:maxOmniRecent]
	}

	out := []channelfinder.Item{{
		ID:     query,
		Name:   fmt.Sprintf("Search Slack for %q", query),
		Type:   "search",
		Joined: true,
	}}

	if wctx.Edge != nil {
		users, err := wctx.Edge.UsersSearch(ctx, query, currentChannel, nil)
		if err != nil {
			debuglog.General("omni search: users/search %q: %v", query, err)
		} else {
			for _, u := range users {
				if u.ID == "" || u.Deleted {
					continue
				}
				name := peopleDisplayName(u)
				out = append(out, channelfinder.Item{
					ID:     u.ID,
					Name:   name,
					Type:   "user",
					UserID: u.ID,
					Joined: true,
				})
			}
		}
	}

	msgs, err := wctx.Client.SearchInline(ctx, query, sessionID, recent)
	if err != nil {
		debuglog.General("omni search: search.inline %q: %v", query, err)
	} else {
		for _, it := range msgs {
			name := it.Text
			if name == "" {
				name = it.ChannelName
			}
			if name == "" {
				name = it.ChannelID
			}
			out = append(out, channelfinder.Item{
				ID:       it.ChannelID,
				Name:     name,
				Type:     "message",
				TS:       it.TS,
				ThreadTS: it.ThreadTS,
				Joined:   true,
			})
		}
	}

	files, err := wctx.Client.AutocompleteFiles(ctx, query)
	if err != nil {
		debuglog.General("omni search: autocomplete.files %q: %v", query, err)
	} else {
		for _, f := range files {
			url := f.URLPrivate
			if url == "" {
				url = f.Permalink
			}
			out = append(out, channelfinder.Item{
				ID:      f.ID,
				Name:    f.Name,
				Type:    "file",
				FileID:  f.ID,
				FileURL: url,
				Joined:  true,
			})
		}
	}
	return out
}
