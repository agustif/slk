package service

import "testing"

func TestVIPStore_ReplaceAndIsVIP(t *testing.T) {
	s := NewVIPStore()
	if s.IsVIP("U1") {
		t.Fatal("IsVIP before Replace")
	}
	s.Replace([]string{"U1", " U2 ", "", "U1"})
	if !s.IsVIP("U1") || !s.IsVIP("U2") {
		t.Fatalf("missing ids: %v", s.UserIDs())
	}
	if s.IsVIP("U3") {
		t.Fatal("unknown id")
	}
}

func TestVIPStore_ApplyPrefChange(t *testing.T) {
	s := NewVIPStore()
	s.Replace([]string{"U1"})
	if !s.ApplyPrefChange("vip_users", "U2,U3") {
		t.Fatal("expected change")
	}
	if s.IsVIP("U1") || !s.IsVIP("U2") || !s.IsVIP("U3") {
		t.Fatalf("after pref_change: %v", s.UserIDs())
	}
	if s.ApplyPrefChange("muted_channels", "C1") {
		t.Fatal("other prefs must be ignored")
	}
}
