package service

import (
	"strings"
	"sync"

	slk "github.com/agustif/slk/internal/slack"
)

// VIPStore is the per-workspace set of Slack VIP people/apps
// (prefs.vip_users). Bootstrapped from client.userBoot prefs and kept
// fresh by pref_change for "vip_users".
type VIPStore struct {
	mu    sync.RWMutex
	ids   map[string]bool
	ready bool
}

func NewVIPStore() *VIPStore {
	return &VIPStore{ids: map[string]bool{}}
}

func (s *VIPStore) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// Replace installs the authoritative ID set (from userBoot or
// users.prefs.get). Empty ids still marks the store ready.
func (s *VIPStore) Replace(ids []string) {
	next := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			next[id] = true
		}
	}
	s.mu.Lock()
	s.ids = next
	s.ready = true
	s.mu.Unlock()
}

func (s *VIPStore) IsVIP(id string) bool {
	if id == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready && s.ids[id]
}

func (s *VIPStore) UserIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready || len(s.ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.ids))
	for id := range s.ids {
		out = append(out, id)
	}
	return out
}

// ApplyPrefChange handles pref_change name=vip_users. value is the
// full comma-separated list. Returns whether the set changed.
func (s *VIPStore) ApplyPrefChange(name, value string) bool {
	if name != "vip_users" {
		return false
	}
	next := map[string]bool{}
	for _, id := range slk.ParseVIPUsers(value) {
		next[id] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := !s.ready || !sameMuteSet(s.ids, next)
	s.ids = next
	s.ready = true
	return changed
}
