package main

import (
	"context"
	"testing"

	"github.com/gammons/slk/internal/cache"
)

func TestHydrateMessages_EmptyCachedTextIsMiss(t *testing.T) {
	db, err := cache.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertMessage(cache.Message{
		TS: "1.0", ChannelID: "C1", WorkspaceID: "T1", UserID: "U1", Text: "",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertMessage(cache.Message{
		TS: "2.0", ChannelID: "C1", WorkspaceID: "T1", UserID: "U1", Text: "hello",
	}); err != nil {
		t.Fatal(err)
	}
	hits := hydrateMessages(context.Background(), db, nil, []hydratePair{
		{ChannelID: "C1", TS: "1.0"},
		{ChannelID: "C1", TS: "2.0"},
	})
	if _, ok := hits[hydratePairKey("C1", "1.0")]; ok {
		t.Fatal("empty cached text must not count as a hydrate hit")
	}
	got, ok := hits[hydratePairKey("C1", "2.0")]
	if !ok || got.Text != "hello" {
		t.Fatalf("populated cache row: %+v ok=%v", got, ok)
	}
}
