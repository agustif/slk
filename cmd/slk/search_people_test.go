package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gammons/slk/internal/slack/edge"
	"github.com/gammons/slk/internal/ui/searchresults"
)

type fakeUserSearch struct {
	query          string
	currentChannel string
	topUsers       []string
	users          []edge.User
	err            error
	calls          int
}

func (f *fakeUserSearch) UsersSearch(_ context.Context, query, currentChannel string, topUsers []string) ([]edge.User, error) {
	f.calls++
	f.query = query
	f.currentChannel = currentChannel
	f.topUsers = topUsers
	return f.users, f.err
}

func TestSearchPeopleRemote_MapsDisplayHandleAndSkipsDeleted(t *testing.T) {
	var alice, bot, gone, noID edge.User
	alice.ID = "U1"
	alice.Name = "alice"
	alice.Profile.DisplayName = "Alice Smith"
	bot.ID = "U2"
	bot.Name = "deploybot"
	bot.IsBot = true
	bot.Profile.RealName = "Deploy Bot"
	gone.ID = "U3"
	gone.Name = "gone"
	gone.Deleted = true
	gone.Profile.DisplayName = "Gone"
	noID.Name = "nobody"

	fake := &fakeUserSearch{users: []edge.User{alice, bot, gone, noID}}
	got, err := searchPeopleRemote(context.Background(), fake, "ali", "C1")
	if err != nil {
		t.Fatalf("searchPeopleRemote: %v", err)
	}
	if fake.calls != 1 || fake.query != "ali" || fake.currentChannel != "C1" {
		t.Fatalf("call = %+v; want query=ali currentChannel=C1", fake)
	}
	if fake.topUsers != nil {
		t.Errorf("topUsers = %v; want nil (no users.list walk, no invented frecency)", fake.topUsers)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items; want 2 (deleted and empty-id dropped): %+v", len(got), got)
	}
	if got[0].Kind != searchresults.KindPeople || got[0].UserID != "U1" || got[0].UserName != "Alice Smith" || got[0].Text != "alice" {
		t.Errorf("item0 = %+v; want Alice Smith / alice", got[0])
	}
	if got[1].UserID != "U2" || got[1].UserName != "Deploy Bot" || got[1].Text != "deploybot" {
		t.Errorf("item1 = %+v; want Deploy Bot (real_name fallback) / deploybot", got[1])
	}
}

func TestSearchPeopleRemote_EmptyQueryMakesNoRequest(t *testing.T) {
	fake := &fakeUserSearch{users: []edge.User{{ID: "U1", Name: "alice"}}}
	got, err := searchPeopleRemote(context.Background(), fake, "", "C1")
	if err != nil {
		t.Fatalf("empty query: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v; want nil", got)
	}
	if fake.calls != 0 {
		t.Errorf("empty query made %d requests; want 0", fake.calls)
	}
}

func TestSearchPeopleRemote_NilSearcherErrors(t *testing.T) {
	_, err := searchPeopleRemote(context.Background(), nil, "x", "")
	if err == nil {
		t.Fatal("nil searcher: want error so the overlay spinner is not stuck on empty")
	}
}

func TestSearchPeopleRemote_ErrorPropagates(t *testing.T) {
	fake := &fakeUserSearch{err: errors.New("ratelimited")}
	got, err := searchPeopleRemote(context.Background(), fake, "x", "")
	if err == nil {
		t.Fatal("want error")
	}
	if got != nil {
		t.Errorf("got %+v; want nil on error", got)
	}
}

func TestPeopleDisplayNameFallback(t *testing.T) {
	var u edge.User
	u.ID = "U1"
	u.Name = "handle"
	if got := peopleDisplayName(u); got != "handle" {
		t.Errorf("handle-only = %q", got)
	}
	u.Profile.RealName = "Real Name"
	if got := peopleDisplayName(u); got != "Real Name" {
		t.Errorf("real name = %q", got)
	}
	u.Profile.DisplayName = "Display"
	if got := peopleDisplayName(u); got != "Display" {
		t.Errorf("display name = %q", got)
	}
}
