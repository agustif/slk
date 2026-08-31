package main

import (
	"context"
	"errors"

	"github.com/agustif/slk/internal/debuglog"
	"github.com/agustif/slk/internal/slack/edge"
	"github.com/agustif/slk/internal/ui/searchresults"
)

// userSearcher is edgeapi's users/search, the endpoint that replaces
// walking users.list for the ctrl+f People tab.
//
// An interface rather than *edge.Client so the mapping below is
// testable without a server.
type userSearcher interface {
	UsersSearch(ctx context.Context, query, currentChannel string, topUsers []string) ([]edge.User, error)
}

// searchPeopleRemote asks the server which users match query and
// converts the answer into ctrl+f People rows.
//
// Callers must debounce: edge.UsersSearch's contract is one request
// per typing pause (~300 ms), never per keystroke. An empty query
// makes no request (the edge client also no-ops, but queueing one
// from here would still be the shape the official client never
// produces).
//
// Deleted users are dropped: Enter on a People row opens a DM, and a
// deleted account is a dead end. Server ranking order is preserved.
func searchPeopleRemote(ctx context.Context, s userSearcher, query, currentChannel string) ([]searchresults.Item, error) {
	if query == "" {
		return nil, nil
	}
	if s == nil {
		return nil, errors.New("people search unavailable")
	}
	users, err := s.UsersSearch(ctx, query, currentChannel, nil)
	if err != nil {
		debuglog.General("workspace search: users/search for %q: %v", query, err)
		return nil, err
	}
	return peopleSearchItems(users), nil
}

// peopleSearchItems converts edge.UsersSearch results into overlay
// rows. Names come from the payload (display_name, then real_name,
// then handle) — no extra API. Presence is left empty for the UI to
// fill from its live map when cheap.
func peopleSearchItems(users []edge.User) []searchresults.Item {
	items := make([]searchresults.Item, 0, len(users))
	for _, u := range users {
		if u.ID == "" || u.Deleted {
			continue
		}
		items = append(items, searchresults.Item{
			Kind:     searchresults.KindPeople,
			UserID:   u.ID,
			UserName: peopleDisplayName(u),
			Text:     u.Name,
		})
	}
	return items
}

// peopleDisplayName matches bootstrap.userDisplayName / resolveUser:
// display name, then real name, then the handle.
func peopleDisplayName(u edge.User) string {
	if u.Profile.DisplayName != "" {
		return u.Profile.DisplayName
	}
	if u.Profile.RealName != "" {
		return u.Profile.RealName
	}
	if u.Name != "" {
		return u.Name
	}
	return u.ID
}
