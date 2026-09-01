package main

import (
	"context"

	"github.com/agustif/slk/internal/debuglog"
	slackclient "github.com/agustif/slk/internal/slack"
)

// hydrateStarredFiles fills files.favorites.list ids via files.list
// hydrate, then files.info peek for misses. Titles for F… quip files
// come from those responses; canvas bodies are not fetched.
func hydrateStarredFiles(ctx context.Context, client *slackclient.Client, ids []string) []slackclient.FileInfo {
	if client == nil || len(ids) == 0 {
		return nil
	}
	byID := make(map[string]slackclient.FileInfo, len(ids))
	files, err := client.HydrateFiles(ctx, ids)
	if err != nil {
		debuglog.Cache("files.list hydrate: %v", err)
	} else {
		for _, f := range files {
			if f.ID != "" {
				byID[f.ID] = f
			}
		}
	}
	out := make([]slackclient.FileInfo, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if f, ok := byID[id]; ok {
			out = append(out, f)
			continue
		}
		info, err := client.GetFileInfo(ctx, id)
		if err != nil {
			debuglog.Cache("files.info %s: %v", id, err)
			out = append(out, slackclient.FileInfo{ID: id})
			continue
		}
		out = append(out, *info)
	}
	return out
}
