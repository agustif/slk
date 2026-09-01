package main

import (
	"testing"

	slackclient "github.com/agustif/slk/internal/slack"
)

func TestBumpRecentsNav_ChannelAndDM(t *testing.T) {
	wctx := &WorkspaceContext{}
	ch := bumpRecentsNav(wctx, "C0BTX6N7JRK")
	if len(ch) != 1 || ch[0].ObjectType != slackclient.RecentsObjectChannel || ch[0].ID != "C0BTX6N7JRK" {
		t.Fatalf("channel = %+v", ch)
	}
	dm := bumpRecentsNav(wctx, "D0BU4SLGVE0")
	if len(dm) != 2 || dm[0].ObjectType != slackclient.RecentsObjectDM || dm[0].ID != "D0BU4SLGVE0" {
		t.Fatalf("dm-first = %+v", dm)
	}
	if dm[1].ID != "C0BTX6N7JRK" {
		t.Fatalf("previous channel not kept: %+v", dm)
	}
	file := bumpRecentsNav(wctx, "F0BUXHC276C")
	if len(file) != 3 || file[0].ObjectType != slackclient.RecentsObjectFile || file[0].ID != "F0BUXHC276C" {
		t.Fatalf("file-first = %+v", file)
	}
	again := bumpRecentsNav(wctx, "C0BTX6N7JRK")
	if len(again) != 3 || again[0].ID != "C0BTX6N7JRK" || again[1].ID != "F0BUXHC276C" || again[2].ID != "D0BU4SLGVE0" {
		t.Fatalf("move-to-front = %+v", again)
	}
}

func TestBumpRecentsNav_SkipsUnknownPrefix(t *testing.T) {
	wctx := &WorkspaceContext{}
	if got := bumpRecentsNav(wctx, "G123"); got != nil {
		t.Fatalf("G… = %+v", got)
	}
	if got := bumpRecentsNav(wctx, ""); got != nil {
		t.Fatalf("empty = %+v", got)
	}
}
