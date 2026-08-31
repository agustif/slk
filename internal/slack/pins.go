package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Pin is one pins.list item flattened for the channel header chip
// and jump-to-pin navigation.
type Pin struct {
	Type      string
	ChannelID string
	Created   int64
	MessageTS string
	Text      string
	Permalink string
}

// GetPins wraps pins.list for channelID. Message pins carry ts for
// in-app jump; file pins keep a permalink when Slack sent one.
func (c *Client) GetPins(ctx context.Context, channelID string) ([]Pin, error) {
	if channelID == "" {
		return nil, fmt.Errorf("pins.list: channelID is required")
	}
	raw, err := c.PostForm(ctx, "pins.list", url.Values{"channel": {channelID}})
	if err != nil {
		return nil, fmt.Errorf("pins.list: %w", err)
	}
	out, err := parsePinsList(raw)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type pinsListResponse struct {
	OK    bool          `json:"ok"`
	Error string        `json:"error"`
	Items []pinsListRow `json:"items"`
}

type pinsListRow struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Created int64  `json:"created"`
	Message struct {
		TS        string `json:"ts"`
		Text      string `json:"text"`
		Permalink string `json:"permalink"`
	} `json:"message"`
	File struct {
		Title     string `json:"title"`
		Name      string `json:"name"`
		Permalink string `json:"permalink"`
	} `json:"file"`
}

func parsePinsList(raw []byte) ([]Pin, error) {
	var res pinsListResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("pins.list: decoding: %w", err)
	}
	if !res.OK {
		errStr := res.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("pins.list: %s", errStr)
	}
	out := make([]Pin, 0, len(res.Items))
	for _, row := range res.Items {
		p := Pin{
			Type:      row.Type,
			ChannelID: row.Channel,
			Created:   row.Created,
		}
		switch row.Type {
		case "file", "file_comment":
			p.Text = row.File.Title
			if p.Text == "" {
				p.Text = row.File.Name
			}
			p.Permalink = row.File.Permalink
		default:
			p.MessageTS = row.Message.TS
			p.Text = row.Message.Text
			p.Permalink = row.Message.Permalink
		}
		if p.MessageTS == "" && p.Text == "" && p.Permalink == "" {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
