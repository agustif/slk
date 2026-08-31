package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	General       General               `toml:"general"`
	Appearance    Appearance            `toml:"appearance"`
	Animations    Animations            `toml:"animations"`
	Notifications Notifications         `toml:"notifications"`
	Cache         CacheConfig           `toml:"cache"`
	Sidebar       Sidebar               `toml:"sidebar"`
	Activity      Activity              `toml:"activity"`
	Sections      map[string]SectionDef `toml:"sections"`
	Theme         Theme                 `toml:"theme"`
	Workspaces    map[string]Workspace  `toml:"workspaces"`
}

// SectionDef defines a sidebar section with channel name patterns.
// Channels matching any pattern are placed in this section.
// Patterns support simple glob matching (* for any characters).
//
// Each entry in Channels may optionally carry a per-channel sort
// suffix of the form "<pattern>:<N>" where N is a non-negative
// integer. Channels matching a pattern with an explicit N are placed
// above un-annotated channels within the section, sorted by N
// ascending. Example: channels = ["general:1", "alerts:2", "random"].
// The ":N" syntax is only honored when use_slack_sections = false
// (or as a fallback when Slack's section endpoint is unreachable);
// in Slack-native mode, channel order is taken from Slack.
type SectionDef struct {
	Channels []string `toml:"channels"`
	Order    int      `toml:"order"` // lower = higher in sidebar
}

// parseChannelPattern splits a "<pattern>:<N>" config entry into its
// pattern and order components. If the suffix after the last ':' is
// not a non-negative integer, the whole input is returned as the
// pattern with order 0 (so e.g. accidentally-included colons in
// patterns are treated as literal characters, not orders). Slack
// channel names cannot contain ':', so well-formed configs are never
// ambiguous.
func parseChannelPattern(s string) (pattern string, order int) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return s, 0
	}
	n, err := strconv.Atoi(s[i+1:])
	if err != nil || n < 0 {
		return s, 0
	}
	return s[:i], n
}

type General struct {
	DefaultWorkspace string `toml:"default_workspace"`
	// UseSlackSections opts in/out of using the user's actual Slack
	// sidebar sections (via users.channelSections.list + WS events)
	// instead of the config-glob [sections.*] system. Pointer so we
	// can distinguish "unset" (default true) from explicit false.
	UseSlackSections *bool `toml:"use_slack_sections"`
}

type Appearance struct {
	Theme           string `toml:"theme"`
	TimestampFormat string `toml:"timestamp_format"`
	ShowAvatars     bool   `toml:"show_avatars"`
	// ImageProtocol controls how inline images are rendered.
	// One of: "auto", "kitty", "sixel", "halfblock", "off".
	ImageProtocol string `toml:"image_protocol"`
	// MaxImageRows caps the height of inline images in terminal rows.
	MaxImageRows int `toml:"max_image_rows"`
	// MaxImageCols caps the width of inline images in terminal columns.
	// If 0 or unset, defaults to 60. The image is also bounded by the
	// available message-pane width when narrower.
	MaxImageCols int `toml:"max_image_cols"`
	// MouseWheelLines controls how many lines the viewport scrolls per
	// mouse-wheel notch. Higher = faster scroll. Defaults to 3 (typical
	// terminal behavior). Clamped to >= 1 at load time.
	MouseWheelLines int `toml:"mouse_wheel_lines"`
	// EmojiImages controls whether emoji are rendered as PNG images
	// (from Slack's CDN) via the kitty graphics protocol. One of:
	// "on" (default) or "off". On non-kitty terminals this is silently
	// treated as "off"; see internal/emoji/place.go.
	EmojiImages string `toml:"emoji_images"`
	// EmojiCells is the terminal-cell footprint reserved for each
	// emoji image (cells wide x 1 row tall). 2 (default) matches the
	// East-Asian-Wide convention; 1 is an escape hatch if 2 looks too
	// large in a given font. Clamped to {1, 2} at load time.
	EmojiCells int `toml:"emoji_cells"`
	// ColoredUsernames, when true, colors each user's name deterministically
	// by hashing their user ID — mirroring how Slack clients assign stable
	// per-user colors. Default off.
	ColoredUsernames bool `toml:"colored_usernames"`
}

type Animations struct {
	Enabled          bool `toml:"enabled"`
	SmoothScrolling  bool `toml:"smooth_scrolling"`
	TypingIndicators bool `toml:"typing_indicators"`
	ToastTransitions bool `toml:"toast_transitions"`
	MessageFadeIn    bool `toml:"message_fade_in"`
}

type Notifications struct {
	Enabled   bool     `toml:"enabled"`
	OnMention bool     `toml:"on_mention"`
	OnDM      bool     `toml:"on_dm"`
	OnKeyword []string `toml:"on_keyword"`
	// QuietHours is a local 24h window "HH:MM-HH:MM" during which desktop
	// notifications are suppressed. Overnight ranges (start after end,
	// e.g. "22:00-08:00") wrap midnight. Empty disables. Invalid values
	// are cleared on Load.
	QuietHours string `toml:"quiet_hours"`
	// NotifyCommand, when set, runs instead of the built-in OS notification.
	// It is executed via `sh -c` with the notification's title and body exposed
	// as $SLK_TITLE and $SLK_BODY. Example:
	//   notify_command = 'terminal-notifier -title "$SLK_TITLE" -message "$SLK_BODY"'
	NotifyCommand string `toml:"notify_command"`
	// StatusCommand, when set, runs on every unread-state change (a message
	// arrives, a channel is read) so an external surface (a status bar, tmux, a
	// multiplexer sidebar) can mirror slk's unread state. Executed via `sh -c`
	// with these variables set:
	//   $SLK_UNREAD        unread channels in the active workspace (mute-filtered)
	//   $SLK_OTHER_UNREAD  unread count across other workspaces
	//   $SLK_WORKSPACE     active workspace name
	//   $SLK_TITLE         the window-title string, e.g. "slk SW (3) +1"
	StatusCommand string `toml:"status_command"`
}

type CacheConfig struct {
	MessageRetentionDays int `toml:"message_retention_days"`
	MaxDBSizeMB          int `toml:"max_db_size_mb"`
	// MaxImageCacheMB caps the on-disk/in-memory image cache size in MB.
	MaxImageCacheMB int64 `toml:"max_image_cache_mb"`
}

// Sidebar holds preferences governing what appears in the channel
// sidebar.
type Sidebar struct {
	// HideInactiveAfterDays auto-hides channels (of any type) whose
	// last_read_ts is older than this many days. Set to 0 to disable.
	// Channels matching a custom [sections.*] glob, channels with
	// unread messages, and the currently-selected channel are never
	// hidden regardless of this setting.
	HideInactiveAfterDays int `toml:"hide_inactive_after_days"`
	Width                 int `toml:"width"`
	// Sort is the [sidebar.sort] table of composable atom pipelines.
	Sort SidebarSort `toml:"sort"`
	// VIP is the membership list for the vip_first sort atom.
	// Patterns match a channel ID, DM user ID, or display name
	// (globs and a leading @ are accepted).
	VIP []string `toml:"vip"`
	// GroupDMs is how 1:1 DMs and group DMs are grouped on the Home
	// sidebar. "split" (default) is two sections (Direct Messages,
	// then Group DMs). "together" is one Direct Messages section,
	// matching OG Slack. The dedicated DMs view always lists both.
	GroupDMs string `toml:"group_dms"`
}

const (
	GroupDMsSplit    = "split"
	GroupDMsTogether = "together"
)

// ClampGroupDMs maps [sidebar].group_dms aliases. Empty / unknown → split.
func ClampGroupDMs(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case GroupDMsTogether, "combined", "all", "og":
		return GroupDMsTogether
	default:
		return GroupDMsSplit
	}
}

// Activity is the Activity inbox: Slack's activity.feed surface
// (recents + notifications), with the same filter/sort knobs the
// official All / DMs / Mentions / Threads tabs send.
//
// Config values are the session defaults. The Activity view can
// cycle them live (f/F filter, s sort, u unread-only); those
// changes are session-only and do not rewrite this file.
type Activity struct {
	// Filter is the Activity tab to open on. Matches an
	// activity.views name, view_type, or id (All / DMs / Mentions /
	// Threads, plus any custom view like Unreads / Reactions / VIP).
	// Empty → "all". Unknown names are kept so a custom tab still
	// works once views load; if it never appears, the TUI falls
	// back to All.
	Filter string `toml:"filter"`
	// Sort is the feed ranking. "newest" is Slack's chrono_v1
	// (All tab); "unreads_first" is priority_reads_and_unreads_v1
	// + sort=vip_unreads_first (the other official tabs).
	// Clamped on load; empty / unknown → "newest".
	Sort string `toml:"sort"`
	// UnreadOnly, when true, sends unread_only=true so Slack
	// returns only unread items.
	UnreadOnly bool `toml:"unread_only"`
	// Density is the Activity list layout. "detailed" is 3-line
	// cards (Threads-view style); "compact" is one line per item.
	// Clamped on load; empty / unknown → "detailed".
	Density string `toml:"density"`
	// Limit is activity.feed's page size. Clamped to 1..100;
	// 0 / unset → 50.
	Limit int `toml:"limit"`
}

const (
	ActivityFilterAll       = "all"
	ActivityFilterDMs       = "dms"
	ActivityFilterMentions  = "mentions"
	ActivityFilterThreads   = "threads"
	ActivityFilterReactions = "reactions"

	ActivitySortNewest       = "newest"
	ActivitySortUnreadsFirst = "unreads_first"

	ActivityDensityCompact  = "compact"
	ActivityDensityDetailed = "detailed"

	ActivityLimitDefault = 50
	ActivityLimitMax     = 100
)

// ActivityFilters is the cycle order for the Activity view's filter
// tabs and the f/F keys. Keep in sync with Slack's All / DMs /
// Mentions / Threads tabs, plus Reactions (a captured feed type).
var ActivityFilters = []string{
	ActivityFilterAll,
	ActivityFilterDMs,
	ActivityFilterMentions,
	ActivityFilterThreads,
	ActivityFilterReactions,
}

// ActivitySorts is the cycle order for the Activity view's sort chip
// and the s key.
var ActivitySorts = []string{
	ActivitySortNewest,
	ActivitySortUnreadsFirst,
}

// ActivityFilterLabel is the tab label shown in the Activity toolbar.
func ActivityFilterLabel(filter string) string {
	switch ClampActivityFilter(filter) {
	case ActivityFilterDMs:
		return "DMs"
	case ActivityFilterMentions:
		return "Mentions"
	case ActivityFilterThreads:
		return "Threads"
	case ActivityFilterReactions:
		return "Reactions"
	default:
		return "All"
	}
}

// ActivitySortLabel is the sort-chip label shown in the Activity toolbar.
func ActivitySortLabel(sort string) string {
	if ClampActivitySort(sort) == ActivitySortUnreadsFirst {
		return "unreads first"
	}
	return "newest"
}

func ClampActivityFilter(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ActivityFilterAll
	}
	switch strings.ToLower(s) {
	case ActivityFilterAll, ActivityFilterDMs, ActivityFilterMentions, ActivityFilterThreads, ActivityFilterReactions:
		return strings.ToLower(s)
	default:
		// Custom activity.views name (Unreads, VIP, …) — keep as typed.
		return s
	}
}

func ClampActivitySort(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case ActivitySortUnreadsFirst:
		return ActivitySortUnreadsFirst
	default:
		return ActivitySortNewest
	}
}

func ClampActivityDensity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case ActivityDensityCompact:
		return ActivityDensityCompact
	default:
		return ActivityDensityDetailed
	}
}

func ClampActivityLimit(n int) int {
	if n < 1 {
		return ActivityLimitDefault
	}
	if n > ActivityLimitMax {
		return ActivityLimitMax
	}
	return n
}

// Normalized returns a copy with every enum/limit clamped to the
// documented set. Load applies this so a partial or typo'd [activity]
// block still starts the TUI on valid knobs.
func (a Activity) Normalized() Activity {
	a.Filter = ClampActivityFilter(a.Filter)
	a.Sort = ClampActivitySort(a.Sort)
	a.Density = ClampActivityDensity(a.Density)
	a.Limit = ClampActivityLimit(a.Limit)
	return a
}

// NextActivityFilter walks ActivityFilters. dir > 0 advances, dir < 0
// goes backward, wrapping at both ends.
func NextActivityFilter(current string, dir int) string {
	return cycleChoice(ActivityFilters, ClampActivityFilter(current), dir)
}

// NextActivitySort walks ActivitySorts forward (two values, so this
// is a toggle).
func NextActivitySort(current string) string {
	return cycleChoice(ActivitySorts, ClampActivitySort(current), 1)
}

func cycleChoice(opts []string, current string, dir int) string {
	if len(opts) == 0 {
		return current
	}
	i := 0
	for j, o := range opts {
		if o == current {
			i = j
			break
		}
	}
	n := len(opts)
	i = (i + dir) % n
	if i < 0 {
		i += n
	}
	return opts[i]
}

// Workspace holds per-workspace user preferences. The TOML key for
// the surrounding map can be either a user-chosen slug (with TeamID
// set explicitly via team_id) or — for backward compatibility —
// a raw Slack team ID (with TeamID left empty; Load fills it in
// from the key).
type Workspace struct {
	TeamID string `toml:"team_id"`
	Theme  string `toml:"theme"`
	// Order controls the workspace's position in the rail and the
	// digit-key mapping (1-9). Positive values are explicit positions
	// ascending; 0 or unset means "unordered" (sorts after ordered
	// workspaces, alphabetically by slug). Ties in Order break by slug.
	Order        int `toml:"order"`
	SidebarWidth int `toml:"sidebar_width"`
	// UseSlackSections overrides [general].use_slack_sections for this
	// workspace. Nil means "fall through to global".
	UseSlackSections *bool                 `toml:"use_slack_sections"`
	Sections         map[string]SectionDef `toml:"sections"`
	// VersionTS caches the Slack build timestamp last reported by
	// client.shouldReload, sent as _x_version_ts on every workspace-API
	// request. Empty means "use the compiled-in fallback and refresh on
	// boot". Persisted so the second and later runs start with a
	// current value rather than a stale compiled-in one.
	VersionTS string `toml:"version_ts"`
}

type Theme struct {
	Primary     string `toml:"primary"`
	Accent      string `toml:"accent"`
	Warning     string `toml:"warning"`
	Error       string `toml:"error"`
	Background  string `toml:"background"`
	Surface     string `toml:"surface"`
	SurfaceDark string `toml:"surface_dark"`
	Text        string `toml:"text"`
	TextMuted   string `toml:"text_muted"`
	Border      string `toml:"border"`
}

func Default() Config {
	return Config{
		Appearance: Appearance{
			Theme:           "nord",
			TimestampFormat: "3:04 PM",
			ImageProtocol:   "auto",
			MaxImageRows:    20,
			MaxImageCols:    60,
			MouseWheelLines: 3,
			EmojiImages:     "on",
			EmojiCells:      2,
		},
		Animations: Animations{
			Enabled:          true,
			SmoothScrolling:  true,
			TypingIndicators: true,
			ToastTransitions: true,
			MessageFadeIn:    true,
		},
		Notifications: Notifications{
			Enabled:   true,
			OnMention: true,
			OnDM:      true,
		},
		Cache: CacheConfig{
			MessageRetentionDays: 30,
			MaxDBSizeMB:          500,
			MaxImageCacheMB:      200,
		},
		Sidebar: Sidebar{
			HideInactiveAfterDays: 30,
			GroupDMs:              GroupDMsSplit,
			Sort: SidebarSort{
				Default: []string{SortAtomSlack},
				DMs:     []string{SortAtomRecent},
			},
		},
		Activity: Activity{
			Filter:  ActivityFilterAll,
			Sort:    ActivitySortNewest,
			Density: ActivityDensityDetailed,
			Limit:   ActivityLimitDefault,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	resolved, err := resolveWorkspaceKeys(cfg.Workspaces)
	if err != nil {
		return cfg, err
	}
	cfg.Workspaces = resolved

	// Clamp MouseWheelLines: 0 (unset, after a user supplied a partial
	// [appearance] block without this key) and negative values both fall
	// back to the default. >= 1 to guarantee scroll progress per notch.
	if cfg.Appearance.MouseWheelLines < 1 {
		cfg.Appearance.MouseWheelLines = 3
	}

	// Clamp EmojiCells to the documented set {1, 2}. 0 (unset after a
	// partial [appearance] block) and any other value fall back to 2.
	if cfg.Appearance.EmojiCells != 1 && cfg.Appearance.EmojiCells != 2 {
		cfg.Appearance.EmojiCells = 2
	}

	// Clamp EmojiImages to the documented set {"on", "off"}. Empty
	// (unset) and any unrecognized value fall back to "on".
	if cfg.Appearance.EmojiImages != "on" && cfg.Appearance.EmojiImages != "off" {
		cfg.Appearance.EmojiImages = "on"
	}

	cfg.Activity = cfg.Activity.Normalized()
	cfg.Sidebar.Sort = cfg.Sidebar.Sort.Normalized()
	cfg.Sidebar.GroupDMs = ClampGroupDMs(cfg.Sidebar.GroupDMs)
	cfg.Notifications.QuietHours = ClampQuietHours(cfg.Notifications.QuietHours)

	return cfg, nil
}

// ClampQuietHours returns spec if it is empty (disabled) or a valid
// "HH:MM-HH:MM" 24h window, and "" for anything else. Invalid values
// are logged rather than failing Load.
func ClampQuietHours(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !validQuietHours(s) {
		log.Printf("config: invalid quiet_hours %q; ignoring (want HH:MM-HH:MM)", s)
		return ""
	}
	return s
}

func validQuietHours(s string) bool {
	start, end, found := strings.Cut(s, "-")
	if !found || start == "" || end == "" || strings.Contains(end, "-") {
		return false
	}
	return validHHMM(start) && validHHMM(end)
}

func validHHMM(s string) bool {
	// Require zero-padded HH:MM; time.Parse("15:04") also accepts "9:00".
	if len(s) != 5 {
		return false
	}
	_, err := time.Parse("15:04", s)
	return err == nil
}

// WorkspaceByTeamID returns the configured Workspace for the given
// team ID, scanning c.Workspaces (which is keyed by either slug or
// legacy team ID). Returns false if no workspace matches.
func (c Config) WorkspaceByTeamID(teamID string) (Workspace, bool) {
	if teamID == "" {
		return Workspace{}, false
	}
	for _, ws := range c.Workspaces {
		if ws.TeamID == teamID {
			return ws, true
		}
	}
	return Workspace{}, false
}

// TeamIDForDefaultWorkspace resolves general.default_workspace to a
// team ID. The configured value can be either a slug ([workspaces.<slug>])
// or a legacy team-ID-shaped key. Returns ("", nil) if default_workspace
// is unset, and an error if it is set but does not match any
// configured workspace.
func (c Config) TeamIDForDefaultWorkspace() (string, error) {
	key := c.General.DefaultWorkspace
	if key == "" {
		return "", nil
	}
	if ws, ok := c.Workspaces[key]; ok {
		return ws.TeamID, nil
	}
	return "", fmt.Errorf("default_workspace %q not found in [workspaces.*]", key)
}

// MatchSection returns the section name for a given channel name in
// the context of the given workspace. If the workspace has its own
// non-empty Sections map, that fully replaces the global Sections;
// otherwise the global Sections apply. Returns "" if no pattern
// matches.
func (c Config) MatchSection(teamID, channelName string) string {
	section, _ := c.MatchSectionAndOrder(teamID, channelName)
	return section
}

// MatchSectionAndOrder is like MatchSection but also returns the
// per-channel sort order encoded in the matching pattern's ":N"
// suffix (see SectionDef). Returns ("", 0) when no pattern matches,
// and (sectionName, 0) when the matching pattern has no explicit
// order suffix.
func (c Config) MatchSectionAndOrder(teamID, channelName string) (string, int) {
	sections := c.Sections
	if ws, ok := c.WorkspaceByTeamID(teamID); ok && len(ws.Sections) > 0 {
		sections = ws.Sections
	}
	return matchSectionAndOrderIn(sections, channelName)
}

// SectionOrder returns the Order field for the named section,
// resolved through the same workspace-vs-global precedence as
// MatchSection. Returns 0 if the section is not defined.
func (c Config) SectionOrder(teamID, sectionName string) int {
	sections := c.Sections
	if ws, ok := c.WorkspaceByTeamID(teamID); ok && len(ws.Sections) > 0 {
		sections = ws.Sections
	}
	if def, ok := sections[sectionName]; ok {
		return def.Order
	}
	return 0
}

// matchSectionAndOrderIn walks sections in Order-ascending order and
// returns the first section name whose patterns match channelName,
// along with the per-channel order encoded in the matching pattern's
// ":N" suffix (0 if absent). Patterns are stripped of any ":N" suffix
// before being passed to filepath.Match.
func matchSectionAndOrderIn(sections map[string]SectionDef, channelName string) (string, int) {
	type entry struct {
		name     string
		order    int
		patterns []string
	}
	var entries []entry
	for name, def := range sections {
		entries = append(entries, entry{name: name, order: def.Order, patterns: def.Channels})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].order < entries[j].order
	})
	for _, e := range entries {
		for _, raw := range e.patterns {
			pattern, chOrder := parseChannelPattern(raw)
			if matched, _ := filepath.Match(pattern, channelName); matched {
				return e.name, chOrder
			}
		}
	}
	return "", 0
}

// EffectiveUseSlackSections returns whether Slack-native sidebar sections
// are enabled for the given workspace. Resolution: per-workspace value
// wins when set; otherwise the global [general].use_slack_sections;
// default true.
func (c Config) EffectiveUseSlackSections(teamID string) bool {
	if ws, ok := c.WorkspaceByTeamID(teamID); ok && ws.UseSlackSections != nil {
		return *ws.UseSlackSections
	}
	if c.General.UseSlackSections != nil {
		return *c.General.UseSlackSections
	}
	return true
}

// ResolveWidth returns the sidebar width to use for the given workspace,
// falling back to the global Sidebar.Width when no per-workspace width
// is set, and to 30 when no global width is set either.
func (c Config) ResolveWidth(teamID string) int {
	if ws, ok := c.WorkspaceByTeamID(teamID); ok && ws.SidebarWidth != 0 {
		return ws.SidebarWidth
	}
	if c.Sidebar.Width != 0 {
		return c.Sidebar.Width
	}
	return 30
}

// ResolveTheme returns the theme name to use for the given workspace,
// falling back to the global Appearance.Theme when no per-workspace theme
// is set, and to "nord" when no global theme is set either.
func (c Config) ResolveTheme(teamID string) string {
	if ws, ok := c.WorkspaceByTeamID(teamID); ok && ws.Theme != "" {
		return ws.Theme
	}
	if c.Appearance.Theme != "" {
		return c.Appearance.Theme
	}
	return "nord"
}
