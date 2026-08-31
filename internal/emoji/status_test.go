package emoji

import "testing"

func TestStatusGlyph_UnicodeAndUnknown(t *testing.T) {
	resetImageMode()
	t.Cleanup(resetImageMode)

	if got := StatusGlyph(":pizza:", nil, PlaceContext{}); got == "" || got == ":pizza:" {
		t.Errorf("StatusGlyph(:pizza:) = %q, want a unicode glyph", got)
	}
	if got := StatusGlyph("pizza", nil, PlaceContext{}); got == "" || got == ":pizza:" {
		t.Errorf("StatusGlyph(pizza) = %q, want a unicode glyph", got)
	}
	if got := StatusGlyph(":not_a_real_emoji_xyz:", nil, PlaceContext{}); got != ":not_a_real_emoji_xyz:" {
		t.Errorf("unknown shortcode = %q, want literal", got)
	}
	if got := StatusGlyph("", nil, PlaceContext{}); got != "" {
		t.Errorf("empty = %q, want empty", got)
	}
}
