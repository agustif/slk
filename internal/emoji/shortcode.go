package emoji

import "io"

// RenderShortcode resolves a Slack shortcode name (no colons) to display
// text for reaction pills and Activity cards.
//
// Image path (ImageModeActive and ctx.Fetcher set): URLForShortcode +
// Place. asImage is true for both the warm kitty placement and the
// cold-cache space reservation; flush is non-nil only on the warm path.
//
// Legacy path: Sprint the skin-tone-stripped name and keep the Unicode
// glyph only when ShouldRenderUnicode says it is composition-safe;
// otherwise return the literal :name: shortcode.
func RenderShortcode(name string, ctx PlaceContext, cells int, customs map[string]string) (text string, flush func(io.Writer) error, asImage bool) {
	if cells <= 0 {
		cells = 2
	}
	if ImageModeActive() && ctx.Fetcher != nil {
		if url, ok := URLForShortcode(name, customs); ok {
			if placement, fl, ok := Place(ctx, url, cells); ok {
				return placement, fl, true
			}
		}
	}
	legacyName := StripSkinTone(name)
	resolved := Sprint(":" + legacyName + ":")
	if ShouldRenderUnicode(resolved) {
		return resolved, nil, false
	}
	return ":" + legacyName + ":", nil, false
}
