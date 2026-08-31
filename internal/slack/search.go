package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// SearchFileMatch is one search.files hit. Slack's File object is large;
// this is the subset the workspace-search modal needs: filename, user,
// timestamp, permalink, channel, and a download URL when present.
type SearchFileMatch struct {
	ID                 string
	Name               string
	Title              string
	User               string
	Username           string
	Created            int64
	Permalink          string
	URLPrivate         string
	URLPrivateDownload string
	ChannelID          string
	ChannelName        string
	Size               int64
}

// SearchFilesResult is the parsed search.files envelope (same shape as
// slack-go's SearchFiles: matches + total + paging).
type SearchFilesResult struct {
	Total   int
	Page    int
	Pages   int
	Matches []SearchFileMatch
}

// SearchFiles runs a workspace-wide file search via Slack's search.files
// endpoint. The query string is passed through verbatim, so Slack-side
// modifiers (from:, in:, before:, ...) work unmodified. Results are
// relevance-sorted (Slack's default). count/page follow the same rules
// as SearchMessages: non-positive count keeps Slack's default; page < 1
// is sent as 1.
func (c *Client) SearchFiles(ctx context.Context, query string, count, page int) (*SearchFilesResult, error) {
	form := searchForm(query, count, page)
	raw, err := c.postForm(ctx, "search.files", form)
	if err != nil {
		return nil, fmt.Errorf("searching files: %w", err)
	}
	res, err := parseSearchFiles(raw)
	if err != nil {
		return nil, fmt.Errorf("searching files: %w", err)
	}
	return res, nil
}

// searchForm is the request body SearchMessages (via slack-go) and
// SearchFiles (via postForm) share: query plus optional count/page.
func searchForm(query string, count, page int) url.Values {
	form := url.Values{"query": {query}}
	if count > 0 {
		form.Set("count", strconv.Itoa(count))
	}
	if page > 1 {
		form.Set("page", strconv.Itoa(page))
	}
	return form
}

type searchFilesEnvelope struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Files struct {
		Total  int `json:"total"`
		Paging struct {
			Count int `json:"count"`
			Total int `json:"total"`
			Page  int `json:"page"`
			Pages int `json:"pages"`
		} `json:"paging"`
		Matches []searchFileJSON `json:"matches"`
	} `json:"files"`
}

type searchFileJSON struct {
	ID                 string           `json:"id"`
	Created            int64            `json:"created"`
	Timestamp          int64            `json:"timestamp"`
	Name               string           `json:"name"`
	Title              string           `json:"title"`
	User               string           `json:"user"`
	Username           string           `json:"username"`
	Size               int64            `json:"size"`
	URLPrivate         string           `json:"url_private"`
	URLPrivateDownload string           `json:"url_private_download"`
	Permalink          string           `json:"permalink"`
	Channels           []string         `json:"channels"`
	Groups             []string         `json:"groups"`
	IMs                []string         `json:"ims"`
	Shares             searchFileShares `json:"shares"`
}

type searchFileShares struct {
	Public  map[string][]searchFileShare `json:"public"`
	Private map[string][]searchFileShare `json:"private"`
}

type searchFileShare struct {
	ChannelName string `json:"channel_name"`
}

// parseSearchFiles decodes a search.files response into the fields the
// Files tab needs. Unknown JSON is ignored.
func parseSearchFiles(raw []byte) (*SearchFilesResult, error) {
	var env searchFilesEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}
	if !env.OK {
		errStr := env.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("%s", errStr)
	}
	matches := make([]SearchFileMatch, 0, len(env.Files.Matches))
	for _, m := range env.Files.Matches {
		created := m.Created
		if created == 0 {
			created = m.Timestamp
		}
		chID, chName := fileChannel(m)
		matches = append(matches, SearchFileMatch{
			ID:                 m.ID,
			Name:               m.Name,
			Title:              m.Title,
			User:               m.User,
			Username:           m.Username,
			Created:            created,
			Permalink:          m.Permalink,
			URLPrivate:         m.URLPrivate,
			URLPrivateDownload: m.URLPrivateDownload,
			ChannelID:          chID,
			ChannelName:        chName,
			Size:               m.Size,
		})
	}
	page := env.Files.Paging.Page
	if page < 1 {
		page = 1
	}
	total := env.Files.Total
	if total == 0 {
		total = env.Files.Paging.Total
	}
	return &SearchFilesResult{
		Total:   total,
		Page:    page,
		Pages:   env.Files.Paging.Pages,
		Matches: matches,
	}, nil
}

// fileChannel picks the first conversation a file is shared in
// (channels, then private groups, then IMs) and a display name from
// shares when Slack included one.
func fileChannel(m searchFileJSON) (id, name string) {
	switch {
	case len(m.Channels) > 0:
		id = m.Channels[0]
	case len(m.Groups) > 0:
		id = m.Groups[0]
	case len(m.IMs) > 0:
		id = m.IMs[0]
	}
	if id != "" {
		if n := shareName(m.Shares, id); n != "" {
			return id, n
		}
		return id, ""
	}
	if id, name = firstShare(m.Shares.Public); id != "" {
		return id, name
	}
	return firstShare(m.Shares.Private)
}

func shareName(s searchFileShares, id string) string {
	if infos := s.Public[id]; len(infos) > 0 && infos[0].ChannelName != "" {
		return infos[0].ChannelName
	}
	if infos := s.Private[id]; len(infos) > 0 && infos[0].ChannelName != "" {
		return infos[0].ChannelName
	}
	return ""
}

func firstShare(m map[string][]searchFileShare) (id, name string) {
	for cid, infos := range m {
		if cid == "" {
			continue
		}
		if len(infos) > 0 {
			return cid, infos[0].ChannelName
		}
		return cid, ""
	}
	return "", ""
}

// DownloadURL prefers the dedicated download link, then url_private.
func (m SearchFileMatch) DownloadURL() string {
	if m.URLPrivateDownload != "" {
		return m.URLPrivateDownload
	}
	return m.URLPrivate
}

// DisplayName is the filename shown in the Files tab: name, else title.
func (m SearchFileMatch) DisplayName() string {
	if m.Name != "" {
		return m.Name
	}
	return m.Title
}
