package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/agustif/slk/internal/slackhttp"
)

// clientDMsReason is _x_reason when clicking the DMs rail tab.
// Files-rail reload uses dms-tab-populate; pass WithReason to override.
const clientDMsReason = "dms"

// ClientDM is one client.dms ims[] / mpims[] row. The capture attests
// those arrays; only id is decoded so we can list conversations.
type ClientDM struct {
	ID string `json:"id"`
}

// ClientDMs is client.dms' {ims, mpims}.
type ClientDMs struct {
	IMs   []ClientDM
	MPIMs []ClientDM
}

// ListClientDMs POSTs client.dms as captured on DMs rail click
// (2026-09-01): count=250, include_closed=true, include_channel=true,
// exclude_bots=true, priority_mode=priority (string, not a boolean),
// _x_reason=dms.
func (c *Client) ListClientDMs(ctx context.Context) (ClientDMs, error) {
	if slackhttp.ReasonFrom(ctx) == "" {
		ctx = slackhttp.WithReason(ctx, clientDMsReason)
	}
	form := url.Values{
		"count":           {"250"},
		"include_closed":  {"true"},
		"include_channel": {"true"},
		"exclude_bots":    {"true"},
		"priority_mode":   {"priority"},
	}
	raw, err := c.postForm(ctx, "client.dms", form)
	if err != nil {
		return ClientDMs{}, fmt.Errorf("client.dms: %w", err)
	}
	out, err := parseClientDMs(raw)
	if err != nil {
		return ClientDMs{}, err
	}
	return out, nil
}

func parseClientDMs(raw []byte) (ClientDMs, error) {
	var resp struct {
		OK    bool       `json:"ok"`
		Error string     `json:"error"`
		IMs   []ClientDM `json:"ims"`
		MPIMs []ClientDM `json:"mpims"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return ClientDMs{}, fmt.Errorf("client.dms: %w", err)
	}
	if !resp.OK {
		if resp.Error != "" {
			return ClientDMs{}, fmt.Errorf("client.dms: %s", resp.Error)
		}
		return ClientDMs{}, fmt.Errorf("client.dms: ok=false")
	}
	return ClientDMs{
		IMs:   dropEmptyClientDMs(resp.IMs),
		MPIMs: dropEmptyClientDMs(resp.MPIMs),
	}, nil
}

func dropEmptyClientDMs(in []ClientDM) []ClientDM {
	if len(in) == 0 {
		return in
	}
	out := make([]ClientDM, 0, len(in))
	for _, d := range in {
		if d.ID == "" {
			continue
		}
		out = append(out, d)
	}
	return out
}
