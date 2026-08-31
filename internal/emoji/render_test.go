package emoji

import (
	"strings"
	"testing"
)

func TestResolveShortcodesInText(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"no shortcodes", "hello world", "hello world"},
		{"single safe emoji", "look :smile: here", "look 😄  here"},
		{"single unsafe (ZWJ) emoji stays text", "pride :rainbow_flag: month", "pride :rainbow_flag: month"},
		{"unsafe with slack hyphen form", "pride :rainbow-flag: month", "pride :rainbow-flag: month"},
		{"VS16 emoji stays as shortcode", "love :heart: you", "love :heart: you"},
		{"unknown shortcode passes through", "wat :nope: lol", "wat :nope: lol"},
		{"multiple safe", "hi :smile: and :fire:", "hi 😄  and 🔥 "},
		{"safe + unsafe mixed", "ok :smile: :rainbow_flag: done", "ok 😄  :rainbow_flag: done"},
		{"adjacent shortcodes", ":smile::fire:", "😄 🔥 "},
		{"empty", "", ""},
		{"colons only", "::", "::"},
		{"single colon", "a:b", "a:b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveShortcodesInText(c.input)
			if got != c.want {
				t.Errorf("ResolveShortcodesInText(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestRenderShortcode_EyesGlyph(t *testing.T) {
	text, flush, asImage := RenderShortcode("eyes", PlaceContext{}, 2, nil)
	if asImage {
		t.Fatal("image mode off: asImage should be false")
	}
	if flush != nil {
		t.Fatal("legacy path should not return a kitty flush")
	}
	if !strings.Contains(text, "👀") {
		t.Errorf("RenderShortcode(eyes) = %q, want eyes glyph", text)
	}
}

func TestRenderShortcode_CustomFallback(t *testing.T) {
	text, _, asImage := RenderShortcode("not-a-real-emoji", PlaceContext{}, 2, nil)
	if asImage {
		t.Fatal("unknown shortcode should not be asImage")
	}
	if text != ":not-a-real-emoji:" {
		t.Errorf("RenderShortcode(unknown) = %q, want :not-a-real-emoji:", text)
	}
}
