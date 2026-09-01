package ui

import (
	"testing"
)

func TestToSnake(t *testing.T) {
	cases := map[string]string{
		"ToggleStar": "toggle_star",
		"JumpToDate": "jump_to_date",
		"Help":       "help",
		"PageUp":     "page_up",
	}
	for in, want := range cases {
		if got := toSnake(in); got != want {
			t.Errorf("toSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyOverrides_RebindsAndReportsUnknown(t *testing.T) {
	km := DefaultKeyMap()
	unknown := km.ApplyOverrides(map[string][]string{
		"toggle_star": {"s"},
		"not_a_key":   {"x"},
	})
	if km.ToggleStar.Help().Key != "s" {
		t.Errorf("ToggleStar label = %q, want s", km.ToggleStar.Help().Key)
	}
	if len(unknown) != 1 || unknown[0] != "not_a_key" {
		t.Errorf("unknown = %v, want [not_a_key]", unknown)
	}
}
