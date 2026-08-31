package sectionpicker

import "testing"

func items3() []Item {
	return []Item{
		{ID: "L1", Label: "Engineering"},
		{ID: "L2", Label: "Channels", Detail: "current"},
		{ID: "L3", Label: "Direct Messages"},
	}
}

func TestOpenAndNavigate(t *testing.T) {
	m := New()
	if m.IsVisible() {
		t.Fatal("visible before Open")
	}
	m.Open("Move to section", items3())
	if !m.IsVisible() {
		t.Fatal("not visible after Open")
	}
	if m.Title() != "Move to section" {
		t.Errorf("title = %q", m.Title())
	}
	if _, chosen := m.HandleKey("j"); chosen {
		t.Error("j must not choose")
	}
	m.HandleKey("j")
	item, chosen := m.HandleKey("enter")
	if !chosen {
		t.Fatal("enter should choose")
	}
	if item.ID != "L3" || item.Index != 2 {
		t.Errorf("chose %+v", item)
	}
	if m.IsVisible() {
		t.Error("should close after choose")
	}
}

func TestNavigationBounds(t *testing.T) {
	m := New()
	m.Open("Move to section", items3())
	m.HandleKey("k")
	item, chosen := m.HandleKey("enter")
	if !chosen || item.ID != "L1" {
		t.Errorf("chose %+v chosen=%v", item, chosen)
	}
	m.Open("Move to section", items3())
	for i := 0; i < 10; i++ {
		m.HandleKey("j")
	}
	item, _ = m.HandleKey("enter")
	if item.ID != "L3" {
		t.Errorf("chose %q", item.ID)
	}
}

func TestEscCloses(t *testing.T) {
	m := New()
	m.Open("Move to section", items3())
	if _, chosen := m.HandleKey("esc"); chosen {
		t.Error("esc must not choose")
	}
	if m.IsVisible() {
		t.Error("esc should close")
	}
}

func TestEnterOnEmptyIsNoop(t *testing.T) {
	m := New()
	m.Open("Move to section", nil)
	if _, chosen := m.HandleKey("enter"); chosen {
		t.Error("enter on empty list must not choose")
	}
}
