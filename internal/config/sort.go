package config

import (
	"path/filepath"
	"strings"

	"github.com/gammons/slk/internal/text"
)

// Sort atoms for [sidebar.sort]. Each atom is one comparison key.
// Pipelines compose left-to-right like SQL ORDER BY: the first atom
// partitions, later atoms sort within each partition.
//
//	["vip_first", "recent"]         — two recency lists, VIPs on top
//	["vip_first", "alphabetical"]   — two A–Z lists, VIPs on top
//	["unread_first", "alphabetical"]
//	["slack"]                       — Slack / config :N / input order
const (
	SortAtomSlack        = "slack"
	SortAtomAlphabetical = "alphabetical"
	SortAtomRecent       = "recent"
	SortAtomVIPFirst     = "vip_first"
	SortAtomUnreadFirst  = "unread_first"
	SortAtomStarredFirst = "starred_first"
)

// SidebarSort is the [sidebar.sort] table. Every field is an ordered
// list of atoms. Empty / omitted type fields fall through to Default.
type SidebarSort struct {
	Default  []string            `toml:"default"`
	DMs      []string            `toml:"dms"`
	Channels []string            `toml:"channels"`
	Starred  []string            `toml:"starred"`
	Apps     []string            `toml:"apps"`
	Section  map[string][]string `toml:"section"`
}

var sortAtomAliases = map[string]string{
	SortAtomSlack:        SortAtomSlack,
	"native":             SortAtomSlack,
	"api":                SortAtomSlack,
	SortAtomAlphabetical: SortAtomAlphabetical,
	"alpha":              SortAtomAlphabetical,
	"name":               SortAtomAlphabetical,
	"az":                 SortAtomAlphabetical,
	SortAtomRecent:       SortAtomRecent,
	"recency":            SortAtomRecent,
	"last_visited":       SortAtomRecent,
	"last-visited":       SortAtomRecent,
	SortAtomVIPFirst:     SortAtomVIPFirst,
	"vip":                SortAtomVIPFirst,
	"vip-first":          SortAtomVIPFirst,
	SortAtomUnreadFirst:  SortAtomUnreadFirst,
	"unread":             SortAtomUnreadFirst,
	"unreads_first":      SortAtomUnreadFirst,
	"unread-first":       SortAtomUnreadFirst,
	SortAtomStarredFirst: SortAtomStarredFirst,
	"starred":            SortAtomStarredFirst,
	"star":               SortAtomStarredFirst,
	"starred-first":      SortAtomStarredFirst,
}

// ClampSortPipeline lowercases, expands comma-separated elements,
// maps aliases, drops unknowns, and de-dupes left-to-right. Empty
// input becomes ["slack"] so a missing config always has a defined
// order (Slack / :N / input).
func ClampSortPipeline(atoms []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range atoms {
		for _, part := range strings.Split(raw, ",") {
			atom, ok := sortAtomAliases[strings.ToLower(strings.TrimSpace(part))]
			if !ok || seen[atom] {
				continue
			}
			seen[atom] = true
			out = append(out, atom)
		}
	}
	if len(out) == 0 {
		return []string{SortAtomSlack}
	}
	return out
}

func clampOptionalPipeline(atoms []string) []string {
	if len(atoms) == 0 {
		return nil
	}
	return ClampSortPipeline(atoms)
}

// Normalized clamps every pipeline. Empty Default becomes ["slack"].
func (s SidebarSort) Normalized() SidebarSort {
	s.Default = ClampSortPipeline(s.Default)
	s.DMs = clampOptionalPipeline(s.DMs)
	s.Channels = clampOptionalPipeline(s.Channels)
	s.Starred = clampOptionalPipeline(s.Starred)
	s.Apps = clampOptionalPipeline(s.Apps)
	if len(s.Section) > 0 {
		next := make(map[string][]string, len(s.Section))
		for name, p := range s.Section {
			next[name] = clampOptionalPipeline(p)
		}
		s.Section = next
	}
	return s
}

// Pipeline is the atom list for one sidebar section. Precedence:
//
//  1. [sidebar.sort.section] keyed by display name (case-insensitive)
//  2. type field: dms / channels / starred / apps
//  3. default — except Direct Messages / Group DMs, which default to
//     ["recent"] when `dms` is omitted.
//
// sectionType is Slack's users.channelSections type
// (direct_messages, channels, stars, recent_apps, standard) or a
// config-glob stand-in of the same names.
func (s SidebarSort) Pipeline(sectionName, sectionType string) []string {
	s = s.Normalized()
	if p := s.lookupSection(sectionName); len(p) > 0 {
		return p
	}
	switch sortTypeKey(sectionType) {
	case "dms":
		if len(s.DMs) > 0 {
			return s.DMs
		}
		return []string{SortAtomRecent}
	case "channels":
		if len(s.Channels) > 0 {
			return s.Channels
		}
	case "starred":
		if len(s.Starred) > 0 {
			return s.Starred
		}
	case "apps":
		if len(s.Apps) > 0 {
			return s.Apps
		}
	}
	return s.Default
}

func (s SidebarSort) lookupSection(name string) []string {
	if name == "" || len(s.Section) == 0 {
		return nil
	}
	if p, ok := s.Section[name]; ok && len(p) > 0 {
		return p
	}
	fold := text.Fold(name)
	for k, p := range s.Section {
		if text.Fold(k) == fold && len(p) > 0 {
			return p
		}
	}
	return nil
}

func sortTypeKey(sectionType string) string {
	switch strings.ToLower(strings.TrimSpace(sectionType)) {
	case "direct_messages", "dm", "dms", "ims", "im", "group_dms", "group-dms", "mpim", "mpims":
		return "dms"
	case "channels", "channel":
		return "channels"
	case "stars", "starred", "star":
		return "starred"
	case "recent_apps", "apps", "app":
		return "apps"
	default:
		return strings.ToLower(strings.TrimSpace(sectionType))
	}
}

// VIPMatch reports whether a sidebar row is VIP under [sidebar.vip]
// patterns. A pattern matches a channel ID, DM user ID, or display
// name (case-insensitive). @handle is accepted. * globs use the same
// filepath.Match rules as [sections.*] channel patterns.
func VIPMatch(patterns []string, channelID, dmUserID, name string) bool {
	if len(patterns) == 0 {
		return false
	}
	nameFold := text.Fold(name)
	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		p = strings.TrimPrefix(p, "@")
		if p == "" {
			continue
		}
		if p == channelID || p == dmUserID {
			return true
		}
		if match, _ := filepath.Match(p, name); match {
			return true
		}
		if match, _ := filepath.Match(p, channelID); match {
			return true
		}
		if match, _ := filepath.Match(p, dmUserID); match {
			return true
		}
		if text.Fold(p) == nameFold {
			return true
		}
		if foldPat := text.Fold(p); foldPat != p {
			if match, _ := filepath.Match(foldPat, nameFold); match {
				return true
			}
		}
	}
	return false
}
