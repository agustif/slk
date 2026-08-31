package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ChannelSectionChannelOp is one insert/remove entry on
// users.channelSections.channels.bulkUpdate. The official client
// form-encodes the JSON array as a single field:
//
//	insert=[{"channel_section_id":"L…","channel_ids":["C…"]}]
//	remove=[{"channel_section_id":"L…","channel_ids":["C…"]}]
//
// Insert-only on a channel already in another section is a no-op;
// a move must include both insert (target) and remove (source).
type ChannelSectionChannelOp struct {
	SectionID  string   `json:"channel_section_id"`
	ChannelIDs []string `json:"channel_ids"`
}

type slackAPIAck struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func parseSlackAPIAck(method string, raw []byte) error {
	var ack slackAPIAck
	if err := json.Unmarshal(raw, &ack); err != nil {
		return fmt.Errorf("%s: parsing response: %w", method, err)
	}
	if !ack.OK {
		if ack.Error == "" {
			ack.Error = "unknown_error"
		}
		return fmt.Errorf("%s: API error: %s", method, ack.Error)
	}
	return nil
}

// BulkUpdateSectionChannels POSTs users.channelSections.channels.bulkUpdate.
// remove is omitted when empty (an empty [] is unverified on the wire).
func (c *Client) BulkUpdateSectionChannels(ctx context.Context, insert, remove []ChannelSectionChannelOp) error {
	const method = "users.channelSections.channels.bulkUpdate"
	if len(insert) == 0 && len(remove) == 0 {
		return nil
	}
	form := url.Values{}
	if len(insert) > 0 {
		b, err := json.Marshal(insert)
		if err != nil {
			return fmt.Errorf("%s: %w", method, err)
		}
		form.Set("insert", string(b))
	}
	if len(remove) > 0 {
		b, err := json.Marshal(remove)
		if err != nil {
			return fmt.Errorf("%s: %w", method, err)
		}
		form.Set("remove", string(b))
	}
	raw, err := c.PostForm(ctx, method, form)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return parseSlackAPIAck(method, raw)
}

// AssignChannelToSection moves channelID into toSectionID.
// fromSectionID is the channel's current section; empty means unsectioned
// (insert only). When from equals to, this is a no-op.
func (c *Client) AssignChannelToSection(ctx context.Context, channelID, toSectionID, fromSectionID string) error {
	if channelID == "" || toSectionID == "" {
		return fmt.Errorf("assign section: channel and section id required")
	}
	if fromSectionID == toSectionID {
		return nil
	}
	insert := []ChannelSectionChannelOp{{
		SectionID:  toSectionID,
		ChannelIDs: []string{channelID},
	}}
	var remove []ChannelSectionChannelOp
	if fromSectionID != "" {
		remove = []ChannelSectionChannelOp{{
			SectionID:  fromSectionID,
			ChannelIDs: []string{channelID},
		}}
	}
	return c.BulkUpdateSectionChannels(ctx, insert, remove)
}

type createChannelSectionResponse struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	SectionID string `json:"channel_section_id"`
	Section   *struct {
		ID string `json:"channel_section_id"`
	} `json:"channel_section"`
}

// CreateChannelSection POSTs users.channelSections.create with name
// (and optional emoji). Returns the new section's id.
func (c *Client) CreateChannelSection(ctx context.Context, name, emoji string) (string, error) {
	const method = "users.channelSections.create"
	if name == "" {
		return "", fmt.Errorf("%s: name required", method)
	}
	form := url.Values{"name": {name}}
	if emoji != "" {
		form.Set("emoji", emoji)
	}
	raw, err := c.PostForm(ctx, method, form)
	if err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	var resp createChannelSectionResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("%s: parsing response: %w", method, err)
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "unknown_error"
		}
		return "", fmt.Errorf("%s: API error: %s", method, resp.Error)
	}
	id := resp.SectionID
	if id == "" && resp.Section != nil {
		id = resp.Section.ID
	}
	if id == "" {
		return "", fmt.Errorf("%s: missing channel_section_id", method)
	}
	return id, nil
}
