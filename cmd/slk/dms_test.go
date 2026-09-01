package main

import (
	"testing"

	"github.com/agustif/slk/internal/config"
	slackclient "github.com/agustif/slk/internal/slack"
	"github.com/agustif/slk/internal/ui/sidebar"
)

func TestMergeClientDMsIntoWorkspace_AddsMissing(t *testing.T) {
	wctx := &WorkspaceContext{
		Channels: []sidebar.ChannelItem{
			{ID: "D1", Name: "alice", Type: "dm"},
		},
		UserNames: map[string]string{},
	}
	dms := slackclient.ClientDMs{
		IMs:   []slackclient.ClientDM{{ID: "D1"}, {ID: "D2"}},
		MPIMs: []slackclient.ClientDM{{ID: "G1"}},
	}
	added := mergeClientDMsIntoWorkspace(wctx, nil, config.Config{}, "T1", dms)
	if len(added) != 2 {
		t.Fatalf("added = %+v, want D2 and G1", added)
	}
	if wctx.Channels[0].Name != "alice" {
		t.Errorf("existing DM clobbered: %+v", wctx.Channels[0])
	}
	byID := map[string]sidebar.ChannelItem{}
	for _, it := range wctx.Channels {
		byID[it.ID] = it
	}
	if it, ok := byID["D2"]; !ok || it.Type != "dm" || !it.Closed || it.Name != "D2" {
		t.Errorf("D2 = %+v", it)
	}
	if it, ok := byID["G1"]; !ok || it.Type != "group_dm" || it.Closed {
		t.Errorf("G1 = %+v", it)
	}
	if len(wctx.FinderItems) != 2 {
		t.Errorf("finder added %d, want 2", len(wctx.FinderItems))
	}
}

func TestMergeClientDMsIntoWorkspace_EmptyNoop(t *testing.T) {
	wctx := &WorkspaceContext{
		Channels: []sidebar.ChannelItem{{ID: "D1", Name: "alice", Type: "dm"}},
	}
	added := mergeClientDMsIntoWorkspace(wctx, nil, config.Config{}, "T1", slackclient.ClientDMs{})
	if len(added) != 0 {
		t.Errorf("added = %+v", added)
	}
	if len(wctx.Channels) != 1 {
		t.Errorf("channels = %d", len(wctx.Channels))
	}
}

func TestMergeClientDMsIntoWorkspace_NilWorkspace(t *testing.T) {
	dms := slackclient.ClientDMs{IMs: []slackclient.ClientDM{{ID: "D9"}}}
	added := mergeClientDMsIntoWorkspace(nil, nil, config.Config{}, "T1", dms)
	if len(added) != 1 || added[0].ID != "D9" || added[0].Type != "dm" {
		t.Errorf("added = %+v", added)
	}
}
