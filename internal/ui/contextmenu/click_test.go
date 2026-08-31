package contextmenu

import (
	"testing"
)

func TestClickRowSelectsItem(t *testing.T) {
	m := New()
	m.Open(sampleItems())

	if !m.ClickRow(80, 24, listTopOffset+3) {
		t.Fatal("ClickRow on a populated row should return true")
	}
	if m.selected != 3 {
		t.Errorf("ClickRow set selected=%d, want 3", m.selected)
	}
	if m.ClickRow(80, 24, listTopOffset-1) {
		t.Error("ClickRow above the list should return false")
	}
	if m.ClickRow(80, 24, listTopOffset+len(m.items)) {
		t.Error("ClickRow past the last row should return false")
	}
}
