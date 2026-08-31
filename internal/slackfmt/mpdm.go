// Package slackfmt formats Slack-internal identifiers (channel names,
// etc.) for human display.
package slackfmt

import "strings"

// FormatMPDMName converts Slack's raw multi-party DM channel name
// (e.g. "mpdm-grant--myles--ray-1") into a human-readable participant
// list ("Grant Ammons, Myles Williamson, Ray Bradbury").
//
// Slack's mpdm naming convention is:
//
//	mpdm-<handle1>--<handle2>--...--<handleN>-<index>
//
// where each handle is a user's `name` field (the @-handle, NOT the
// display name) and `<index>` is a small integer disambiguator. The
// double-dash (`--`) is the delimiter; a single dash is a legitimate
// character within a handle.
//
// lookup is called for each parsed handle and should return the
// corresponding display name. If lookup is nil or returns an empty
// string for a handle, the handle itself is used as a fallback.
//
// If name does not match the mpdm format (no "mpdm-" prefix, empty
// body, etc.), the original string is returned unchanged so callers
// can use this unconditionally.
func FormatMPDMName(name string, lookup func(handle string) string) string {
	handles := ParseMPDMHandles(name)
	if len(handles) == 0 {
		return name
	}
	displays := make([]string, 0, len(handles))
	for _, h := range handles {
		var d string
		if lookup != nil {
			d = lookup(h)
		}
		if d == "" {
			d = h
		}
		displays = append(displays, d)
	}
	return strings.Join(displays, ", ")
}

// ParseMPDMHandles extracts @handles from an mpdm-* channel name.
// Returns nil when name is not in Slack's mpdm format.
func ParseMPDMHandles(name string) []string {
	const prefix = "mpdm-"
	if !strings.HasPrefix(name, prefix) {
		return nil
	}
	body := name[len(prefix):]
	if body == "" {
		return nil
	}
	if i := strings.LastIndexByte(body, '-'); i >= 0 && i < len(body)-1 {
		if isAllDigits(body[i+1:]) {
			body = body[:i]
		}
	}
	raw := strings.Split(body, "--")
	out := make([]string, 0, len(raw))
	for _, h := range raw {
		if h != "" {
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
