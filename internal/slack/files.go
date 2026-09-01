package slackclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/agustif/slk/internal/slackhttp"
)

// FileInfo is the files.info peek / files.list hydrate subset slk reads.
// Captured 2026-09-01: quip canvas F0BUXHC276C keys were {ok, file,
// comments, paging} with no content. Snippet slk-har-star2.txt included
// content / content_highlight_html. is_starred stayed false after a
// Files-rail star.
type FileInfo struct {
	ID        string
	Name      string
	Title     string
	Filetype  string
	Mode      string
	IsStarred bool
	Content   string
}

// DisplayName is title, else name, else the file id.
func (f FileInfo) DisplayName() string {
	if f.Title != "" {
		return f.Title
	}
	if f.Name != "" {
		return f.Name
	}
	return f.ID
}

// IsCanvas is filetype=quip or mode=quip as returned for template
// canvas F0BUXHC276C. Not a canvases.* document body.
func (f FileInfo) IsCanvas() bool {
	return f.Filetype == "quip" || f.Mode == "quip"
}

type fileJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Filetype  string `json:"filetype"`
	Mode      string `json:"mode"`
	IsStarred bool   `json:"is_starred"`
	Content   string `json:"content"`
}

func (f fileJSON) info() FileInfo {
	return FileInfo{
		ID:        f.ID,
		Name:      f.Name,
		Title:     f.Title,
		Filetype:  f.Filetype,
		Mode:      f.Mode,
		IsStarred: f.IsStarred,
		Content:   f.Content,
	}
}

// GetFileInfo POSTs files.info as captured from file peek
// (_x_reason=file-subscription.fetchFileInfo): file, page=1, count=500,
// truncate=true, public_shared=false, skip_shares=true.
func (c *Client) GetFileInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil, fmt.Errorf("files.info: file required")
	}
	ctx = slackhttp.WithReason(ctx, "file-subscription.fetchFileInfo")
	form := url.Values{
		"file":          {fileID},
		"page":          {"1"},
		"count":         {"500"},
		"truncate":      {"true"},
		"public_shared": {"false"},
		"skip_shares":   {"true"},
	}
	raw, err := c.postForm(ctx, "files.info", form)
	if err != nil {
		return nil, fmt.Errorf("files.info: %w", err)
	}
	info, err := parseFileInfo(raw)
	if err != nil {
		return nil, fmt.Errorf("files.info: %w", err)
	}
	return info, nil
}

func parseFileInfo(raw []byte) (*FileInfo, error) {
	var env struct {
		OK    bool     `json:"ok"`
		Error string   `json:"error"`
		File  fileJSON `json:"file"`
	}
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
	if env.File.ID == "" {
		return nil, fmt.Errorf("no file")
	}
	info := env.File.info()
	return &info, nil
}

// HydrateFiles POSTs files.list as captured from the internal store fill
// (_x_reason=files-store-unknown-fetch): files=id,id,…. Not the public
// files.list channel/user filter.
func (c *Client) HydrateFiles(ctx context.Context, ids []string) ([]FileInfo, error) {
	clean := compactFileIDs(ids)
	if len(clean) == 0 {
		return nil, nil
	}
	ctx = slackhttp.WithReason(ctx, "files-store-unknown-fetch")
	form := url.Values{"files": {strings.Join(clean, ",")}}
	raw, err := c.postForm(ctx, "files.list", form)
	if err != nil {
		return nil, fmt.Errorf("files.list: %w", err)
	}
	out, err := parseHydrateFiles(raw)
	if err != nil {
		return nil, fmt.Errorf("files.list: %w", err)
	}
	return out, nil
}

func compactFileIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func parseHydrateFiles(raw []byte) ([]FileInfo, error) {
	var env struct {
		OK    bool            `json:"ok"`
		Error string          `json:"error"`
		Files json.RawMessage `json:"files"`
	}
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
	return parseFileList(env.Files), nil
}

func parseFileList(raw json.RawMessage) []FileInfo {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var arr []fileJSON
	if err := json.Unmarshal(raw, &arr); err == nil {
		out := make([]FileInfo, 0, len(arr))
		for _, f := range arr {
			if f.ID == "" {
				continue
			}
			out = append(out, f.info())
		}
		return out
	}
	var keyed map[string]fileJSON
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil
	}
	out := make([]FileInfo, 0, len(keyed))
	for id, f := range keyed {
		if f.ID == "" {
			f.ID = id
		}
		if f.ID == "" {
			continue
		}
		out = append(out, f.info())
	}
	return out
}

const (
	searchModulesFilesReasonBrowser    = "fetch-current-browser"
	searchModulesFilesReasonUserCanvas = "user-created-canvas-query"
	searchModulesFilesContext          = "desktop_files_browser"
	searchModulesFilesSortLastEngaged  = "last_engaged"
	searchModulesFilesDefaultCount     = 50
	searchModulesFilesUserCanvasCount  = 1
)

// SearchModulesFiles POSTs search.modules.files as captured from the
// Files rail (_x_reason=fetch-current-browser): module=files, query
// type filters, page=1, count=50, sort=last_engaged,
// search_context=desktop_files_browser. Highlight/extract flag names
// were not in the HAR notes; they are omitted. Empty workspace
// returned items:[].
//
// Workspace search Files tab is search.files (SearchFiles), not this
// method.
func (c *Client) SearchModulesFiles(ctx context.Context, query string, count int) ([]FileInfo, error) {
	ctx = slackhttp.WithReason(ctx, searchModulesFilesReasonBrowser)
	return c.searchModulesFiles(ctx, query, count)
}

// SearchUserCreatedCanvases POSTs the captured canvas probe:
// query=type:quip creator:U…, count=1,
// _x_reason=user-created-canvas-query, same Files-rail module/sort/
// search_context. Listing canvases uses SearchModulesFiles with
// query=type:quip (count=50). This probe is existence, not a CRDT read.
func (c *Client) SearchUserCreatedCanvases(ctx context.Context) ([]FileInfo, error) {
	uid := strings.TrimSpace(c.UserID())
	if uid == "" {
		return nil, fmt.Errorf("search.modules.files: user id required")
	}
	ctx = slackhttp.WithReason(ctx, searchModulesFilesReasonUserCanvas)
	return c.searchModulesFiles(ctx, "type:quip creator:"+uid, searchModulesFilesUserCanvasCount)
}

func (c *Client) searchModulesFiles(ctx context.Context, query string, count int) ([]FileInfo, error) {
	if count <= 0 {
		count = searchModulesFilesDefaultCount
	}
	form := url.Values{
		"module":         {"files"},
		"query":          {query},
		"page":           {"1"},
		"count":          {strconv.Itoa(count)},
		"sort":           {searchModulesFilesSortLastEngaged},
		"search_context": {searchModulesFilesContext},
	}
	raw, err := c.postForm(ctx, "search.modules.files", form)
	if err != nil {
		return nil, fmt.Errorf("search.modules.files: %w", err)
	}
	out, err := parseSearchModulesFiles(raw)
	if err != nil {
		return nil, fmt.Errorf("search.modules.files: %w", err)
	}
	return out, nil
}

func parseSearchModulesFiles(raw []byte) ([]FileInfo, error) {
	var env struct {
		OK    bool              `json:"ok"`
		Error string            `json:"error"`
		Items []json.RawMessage `json:"items"`
	}
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
	out := make([]FileInfo, 0, len(env.Items))
	for _, rawItem := range env.Items {
		f, ok := parseModulesFileItem(rawItem)
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}

// ListRecentlyDeletedFiles POSTs files.recentlyDeleted as captured
// from the Files rail (_x_reason=get-deleted-files). Live response
// was {ok:true, files:[]}.
func (c *Client) ListRecentlyDeletedFiles(ctx context.Context) ([]FileInfo, error) {
	ctx = slackhttp.WithReason(ctx, "get-deleted-files")
	raw, err := c.postForm(ctx, "files.recentlyDeleted", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("files.recentlyDeleted: %w", err)
	}
	var env struct {
		OK    bool            `json:"ok"`
		Error string          `json:"error"`
		Files json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("files.recentlyDeleted: %w", err)
	}
	if !env.OK {
		errStr := env.Error
		if errStr == "" {
			errStr = "ok=false"
		}
		return nil, fmt.Errorf("files.recentlyDeleted: %s", errStr)
	}
	return parseFileList(env.Files), nil
}

func parseModulesFileItem(raw json.RawMessage) (FileInfo, bool) {
	var wrap struct {
		File json.RawMessage `json:"file"`
		fileJSON
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return FileInfo{}, false
	}
	src := wrap.fileJSON
	if len(wrap.File) > 0 && string(wrap.File) != "null" {
		var nested fileJSON
		if err := json.Unmarshal(wrap.File, &nested); err == nil && nested.ID != "" {
			src = nested
		}
	}
	if src.ID == "" {
		return FileInfo{}, false
	}
	return src.info(), true
}
