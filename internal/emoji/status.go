package emoji

import "strings"

// StatusGlyph resolves a Slack status emoji shortcode (":name:" or "name")
// to a display string. Image-mode uses Place when a URL is cheap to resolve
// and the placement is already warm; otherwise it falls back to Unicode
// (Sprint) or the literal shortcode.
func StatusGlyph(shortcode string, customs map[string]string, ctx PlaceContext) string {
	name := strings.Trim(shortcode, ":")
	if name == "" {
		return ""
	}
	token := ":" + name + ":"
	if ImageModeActive() && ctx.Fetcher != nil {
		if url, ok := URLForShortcode(name, customs); ok {
			placed, _, ok := Place(ctx, url, 1)
			if ok && placed != "" && placed != " " {
				return placed
			}
		}
	}
	rendered := Sprint(token)
	if rendered != token {
		return strings.TrimSpace(rendered)
	}
	return token
}
