package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/agustif/slk/internal/slackhttp"
)

// CanvasFile is the attested files.info / canned-template item shape
// for a Quip canvas (filetype=quip, mode=quip). files.info has no
// document body for these.
type CanvasFile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	Filetype     string `json:"filetype"`
	Mode         string `json:"mode"`
	PrettyType   string `json:"pretty_type"`
	Permalink    string `json:"permalink"`
	QuipThreadID string `json:"quip_thread_id"`
	Editable     bool   `json:"editable"`
	IsStarred    bool   `json:"is_starred"`
}

// ListCannedCanvasTemplates POSTs canvases.getCannedTemplates as
// captured from Files rail Explore templates (2026-09-01):
// _x_reason=fetch-canned-templates. Response {ok, files:[…quip templates]}.
func (c *Client) ListCannedCanvasTemplates(ctx context.Context) ([]CanvasFile, error) {
	ctx = slackhttp.WithReason(ctx, "fetch-canned-templates")
	raw, err := c.postForm(ctx, "canvases.getCannedTemplates", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("canvases.getCannedTemplates: %w", err)
	}
	var resp struct {
		OK    bool         `json:"ok"`
		Error string       `json:"error"`
		Files []CanvasFile `json:"files"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("canvases.getCannedTemplates: %w", err)
	}
	if !resp.OK {
		if resp.Error != "" {
			return nil, fmt.Errorf("canvases.getCannedTemplates: %s", resp.Error)
		}
		return nil, fmt.Errorf("canvases.getCannedTemplates: ok=false")
	}
	return resp.Files, nil
}

// LookupQuipThreadIDs POSTs quip.lookupThreadIds: file_ids (required),
// response lookup map file_id → quip thread id (F0BU54KT3U1 → VOQ9AAYK8SN).
func (c *Client) LookupQuipThreadIDs(ctx context.Context, fileIDs []string) (map[string]string, error) {
	if len(fileIDs) == 0 {
		return nil, fmt.Errorf("quip.lookupThreadIds: file_ids required")
	}
	raw, err := c.postForm(ctx, "quip.lookupThreadIds", url.Values{
		"file_ids": {strings.Join(fileIDs, ",")},
	})
	if err != nil {
		return nil, fmt.Errorf("quip.lookupThreadIds: %w", err)
	}
	var resp struct {
		OK     bool              `json:"ok"`
		Error  string            `json:"error"`
		Lookup map[string]string `json:"lookup"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("quip.lookupThreadIds: %w", err)
	}
	if !resp.OK {
		if resp.Error != "" {
			return nil, fmt.Errorf("quip.lookupThreadIds: %s", resp.Error)
		}
		return nil, fmt.Errorf("quip.lookupThreadIds: ok=false")
	}
	if resp.Lookup == nil {
		resp.Lookup = map[string]string{}
	}
	return resp.Lookup, nil
}

// LookupQuipFileID POSTs quip.lookupFileId: quip_thread_id → {ok, file_id}.
func (c *Client) LookupQuipFileID(ctx context.Context, quipThreadID string) (string, error) {
	if quipThreadID == "" {
		return "", fmt.Errorf("quip.lookupFileId: quip_thread_id required")
	}
	raw, err := c.postForm(ctx, "quip.lookupFileId", url.Values{
		"quip_thread_id": {quipThreadID},
	})
	if err != nil {
		return "", fmt.Errorf("quip.lookupFileId: %w", err)
	}
	var resp struct {
		OK     bool   `json:"ok"`
		Error  string `json:"error"`
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("quip.lookupFileId: %w", err)
	}
	if !resp.OK {
		if resp.Error != "" {
			return "", fmt.Errorf("quip.lookupFileId: %s", resp.Error)
		}
		return "", fmt.Errorf("quip.lookupFileId: ok=false")
	}
	return resp.FileID, nil
}

// OpenFile POSTs files.open: file_id → {ok, viewers, should_subscribe_and_ping}.
func (c *Client) OpenFile(ctx context.Context, fileID string) error {
	if fileID == "" {
		return fmt.Errorf("files.open: file_id required")
	}
	raw, err := c.postForm(ctx, "files.open", url.Values{"file_id": {fileID}})
	if err != nil {
		return fmt.Errorf("files.open: %w", err)
	}
	return parseSlackAPIAck("files.open", raw)
}

// CloseFile POSTs files.close: file_id → {ok:true}.
func (c *Client) CloseFile(ctx context.Context, fileID string) error {
	if fileID == "" {
		return fmt.Errorf("files.close: file_id required")
	}
	raw, err := c.postForm(ctx, "files.close", url.Values{"file_id": {fileID}})
	if err != nil {
		return fmt.Errorf("files.close: %w", err)
	}
	return parseSlackAPIAck("files.close", raw)
}
