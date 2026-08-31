package schedulemenu

import (
	"testing"
	"time"
)

func TestModel_Pick20m(t *testing.T) {
	m := New()
	m.Open()
	if !m.IsVisible() {
		t.Fatal("expected visible")
	}
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionDuration || r.Duration != 20*time.Minute {
		t.Fatalf("expected 20m duration, got %+v", r)
	}
}

func TestModel_Pick1h(t *testing.T) {
	m := New()
	m.Open()
	m.HandleKey("j")
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionDuration || r.Duration != time.Hour {
		t.Fatalf("expected 1h duration, got %+v", r)
	}
}

func TestModel_PickTomorrowMorning(t *testing.T) {
	m := New()
	m.Open()
	for i := 0; i < 5; i++ {
		m.HandleKey("j")
	}
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionTomorrowMorning {
		t.Fatalf("expected ActionTomorrowMorning, got %+v", r)
	}
}

func TestModel_PickCustom(t *testing.T) {
	m := New()
	m.Open()
	for i := 0; i < 6; i++ {
		m.HandleKey("j")
	}
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionCustom {
		t.Fatalf("expected ActionCustom, got %+v", r)
	}
}

func TestModel_FilterByQuery(t *testing.T) {
	m := New()
	m.Open()
	m.HandleKey("2")
	m.HandleKey("0")
	r := m.HandleKey("enter")
	if r == nil || r.Action != ActionDuration || r.Duration != 20*time.Minute {
		t.Fatalf("expected filtered 20m, got %+v", r)
	}
}

func TestModel_EscapeCloses(t *testing.T) {
	m := New()
	m.Open()
	if r := m.HandleKey("esc"); r != nil {
		t.Errorf("expected nil result on esc, got %+v", r)
	}
	if m.IsVisible() {
		t.Error("expected closed after esc")
	}
}
