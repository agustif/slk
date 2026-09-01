package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	slk "github.com/agustif/slk/internal/slack"
)

// SectionsClient is the subset of slk.Client SectionStore needs.
// Defined as an interface so tests can pass fakes.
type SectionsClient interface {
	GetChannelSections(ctx context.Context) ([]slk.SidebarSection, error)
	// GetStarredChannels backs the stars section membership.
	// channelSections.list returns built-in section types (stars,
	// recent_apps) with an empty channel_ids array; stars.list is the
	// authoritative source Bootstrap uses to fill them.
	GetStarredChannels(ctx context.Context) ([]string, error)
}

// SectionStore is the per-workspace authoritative cache of the user's
// Slack-side sidebar sections. Populated on bootstrap from the REST
// endpoint and kept fresh by WS event handlers (Apply* methods).
//
// All public methods are safe for concurrent use.
type SectionStore struct {
	mu               sync.RWMutex
	ready            bool
	sectionsByID     map[string]*slk.SidebarSection
	channelToSection map[string]string
	lastBootstrap    time.Time
}

// NewSectionStore returns an empty store. It reports Ready()==false until
// Bootstrap completes successfully.
func NewSectionStore() *SectionStore {
	return &SectionStore{
		sectionsByID:     map[string]*slk.SidebarSection{},
		channelToSection: map[string]string{},
	}
}

// Ready reports whether the store has successfully bootstrapped at least
// once. Callers should treat !Ready as "fall through to config-glob".
func (s *SectionStore) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// Bootstrap fetches the section list and replaces any prior state
// atomically. Returns an error without mutating state if the fetch fails.
//
// v1 limitation (Task 3 deferred): when ChannelsCount exceeds
// len(ChannelIDs), the section is partially populated. We trust what we
// have; remaining channels stay in the catch-all bucket until either a
// WS channel_sections_channels_upserted event migrates them or a
// reconnect-triggered re-bootstrap fetches fresher data.
func (s *SectionStore) Bootstrap(ctx context.Context, client SectionsClient) error {
	sections, err := client.GetChannelSections(ctx)
	if err != nil {
		return fmt.Errorf("fetching sections: %w", err)
	}

	for i := range sections {
		sec := &sections[i]
		if sec.ChannelsCount > len(sec.ChannelIDs) {
			log.Printf("section store: section %q (%s) reports %d channels but server returned %d on first page; remaining channels will fall through to default bucket",
				sec.Name, sec.ID, sec.ChannelsCount, len(sec.ChannelIDs))
		}
	}

	// Build new maps.
	byID := make(map[string]*slk.SidebarSection, len(sections))
	c2s := map[string]string{}
	for i := range sections {
		sec := &sections[i]
		byID[sec.ID] = sec
		for _, ch := range sec.ChannelIDs {
			c2s[ch] = sec.ID
		}
	}

	s.mu.Lock()
	s.sectionsByID = byID
	s.channelToSection = c2s
	s.ready = true
	s.lastBootstrap = time.Now()
	s.mu.Unlock()

	// Slack's users.channelSections.list returns the stars section with
	// an empty channel_ids array (it doesn't populate built-in section
	// types). stars.list is the authoritative source for starred
	// channels; fetch and inject here so EVERY Bootstrap — including
	// reconnect-triggered re-bootstraps via MaybeRebootstrap — leaves
	// the Starred header populated. Without this, a reconnect Bootstrap
	// atomically replaced the store state and wiped the ChannelIDs that
	// PopulateStars had filled, making includeInSidebar hide the header.
	// Best-effort: on error the stars section stays empty and
	// includeInSidebar hides it until the next bootstrap.
	//
	// PopulateStars re-locks internally; safe to call after unlock now
	// that ready==true.
	if stars, sErr := client.GetStarredChannels(ctx); sErr != nil {
		log.Printf("section store: stars.list failed: %v (starred channels hidden until next bootstrap)", sErr)
	} else if len(stars) > 0 {
		s.PopulateStars(stars)
	}
	return nil
}

// PopulateStars fills the stars section's ChannelIDs from stars.list,
// the authoritative source for starred channels. Slack's
// users.channelSections.list returns the stars section with an empty
// channel_ids array (it doesn't populate built-in section types); without
// this call the stars section stays empty and includeInSidebar hides it.
//
// Safe to call repeatedly — each call replaces the previous star list
// and remaps the channelToSection index accordingly. Channels dropped
// from the star list are restored to another section that still lists
// them (so unstar doesn't dump them into the type-default bucket).
// No-op when the workspace has no stars section (won't synthesize one).
func (s *SectionStore) PopulateStars(channelIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.populateStarsLocked(channelIDs)
}

func (s *SectionStore) populateStarsLocked(channelIDs []string) {
	if !s.ready {
		return
	}
	var starsSectionID string
	var stars *slk.SidebarSection
	for id, sec := range s.sectionsByID {
		if sec.Type == "stars" {
			starsSectionID = id
			stars = sec
			break
		}
	}
	if stars == nil {
		return
	}
	prev := stars.ChannelIDs
	newSet := make(map[string]bool, len(channelIDs))
	next := make([]string, 0, len(channelIDs))
	for _, cid := range channelIDs {
		if cid == "" || newSet[cid] {
			continue
		}
		newSet[cid] = true
		next = append(next, cid)
	}
	// Drop the old stars→channel index entries.
	for _, cid := range prev {
		if existing, ok := s.channelToSection[cid]; ok && existing == starsSectionID {
			delete(s.channelToSection, cid)
		}
	}
	stars.ChannelIDs = next
	stars.ChannelsCount = len(next)
	for _, cid := range next {
		s.channelToSection[cid] = starsSectionID
	}
	// Restore unstarred channels to another section that still lists them.
	for _, cid := range prev {
		if newSet[cid] {
			continue
		}
		if _, ok := s.channelToSection[cid]; ok {
			continue
		}
		for otherID, other := range s.sectionsByID {
			if otherID == starsSectionID {
				continue
			}
			for _, x := range other.ChannelIDs {
				if x == cid {
					s.channelToSection[cid] = otherID
					break
				}
			}
			if _, ok := s.channelToSection[cid]; ok {
				break
			}
		}
	}
}

// IDByType returns the first sidebar section id with the given type
// (stars, channels, direct_messages, …). Empty when missing.
func (s *SectionStore) IDByType(sectionType string) string {
	if s == nil || sectionType == "" {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return ""
	}
	for _, sec := range s.sectionsByID {
		if sec.Type == sectionType {
			return sec.ID
		}
	}
	return ""
}

// TypeByID returns the section type for id, or "".
func (s *SectionStore) TypeByID(id string) string {
	if s == nil || id == "" {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sec := s.sectionsByID[id]; sec != nil {
		return sec.Type
	}
	return ""
}

// StarredChannelIDs returns a copy of the stars section's channel list.
// Empty when the store isn't ready or has no stars section.
func (s *SectionStore) StarredChannelIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.starredChannelIDsLocked()
}

func (s *SectionStore) starredChannelIDsLocked() []string {
	for _, sec := range s.sectionsByID {
		if sec.Type == "stars" {
			out := make([]string, len(sec.ChannelIDs))
			copy(out, sec.ChannelIDs)
			return out
		}
	}
	return nil
}

// IsStarred reports whether channelID is currently in the stars section.
func (s *SectionStore) IsStarred(channelID string) bool {
	if channelID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.starredChannelIDsLocked() {
		if id == channelID {
			return true
		}
	}
	return false
}

// SetStarred optimistically adds or removes channelID from the stars
// section via PopulateStars. No-op when the store isn't ready, has no
// stars section, or channelID is empty.
func (s *SectionStore) SetStarred(channelID string, starred bool) {
	if channelID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return
	}
	current := s.starredChannelIDsLocked()
	next := make([]string, 0, len(current)+1)
	found := false
	for _, id := range current {
		if id == channelID {
			found = true
			if starred {
				next = append(next, id)
			}
			continue
		}
		next = append(next, id)
	}
	if starred && !found {
		next = append(next, channelID)
	}
	s.populateStarsLocked(next)
}

// Membership returns the section ID currently indexed for channelID,
// including non-renderable types. Used by section writes so a move
// can remove the channel from its real source section. Distinct from
// SectionForChannel, which hides non-renderable types so the sidebar
// can fall through to type defaults.
func (s *SectionStore) Membership(channelID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return "", false
	}
	id, ok := s.channelToSection[channelID]
	return id, ok
}

// MoveChannel relocates channelID into toSectionID in the local index.
// An empty toSectionID unsections the channel. Used for optimistic
// sidebar updates; the matching REST call and WS events confirm.
func (s *SectionStore) MoveChannel(channelID, toSectionID string) {
	if channelID == "" {
		return
	}
	if toSectionID == "" {
		from, ok := s.Membership(channelID)
		if ok {
			s.ApplyChannelsRemoved(from, []string{channelID})
		}
		return
	}
	s.ApplyChannelsAdded(toSectionID, []string{channelID})
}

// SectionForChannel returns the renderable section ID a channel belongs
// to. Returns ok=false when the store isn't ready, the channel isn't
// indexed, OR the indexed section is not renderable in the sidebar
// (e.g. slack_connect, salesforce_records, agents). Hiding
// non-renderable sections at this boundary prevents the sidebar from
// trying to bucket items into headers it never created — see
// includeInSidebar for the renderability rule.
func (s *SectionStore) SectionForChannel(channelID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return "", false
	}
	id, ok := s.channelToSection[channelID]
	if !ok {
		return "", false
	}
	sec, secOK := s.sectionsByID[id]
	if !secOK || !includeInSidebar(sec) {
		// Section was deleted, or has a non-renderable type
		// (slack_connect / salesforce_records / agents / etc.). Treat
		// the channel as unclaimed so it falls into the appropriate
		// type-default bucket.
		return "", false
	}
	return id, true
}

// OrderedSections walks the linked-list (head-first) and returns the
// sections that should render in the sidebar, filtered to the
// renderable type whitelist. Cycle protection: stops if a section is revisited.
//
// Head detection: a section is the head if no other section's Next
// points at it. When multiple candidate heads exist (orphans), the
// one with the highest LastUpdate wins as a heuristic; this is a
// best-effort recovery for malformed state.
func (s *SectionStore) OrderedSections() []*slk.SidebarSection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ready {
		return nil
	}

	pointedAt := map[string]bool{}
	for _, sec := range s.sectionsByID {
		if sec.Next != "" {
			pointedAt[sec.Next] = true
		}
	}
	var head *slk.SidebarSection
	for id, sec := range s.sectionsByID {
		if !pointedAt[id] {
			if head == nil || sec.LastUpdate > head.LastUpdate {
				head = sec
			}
		}
	}
	if head == nil {
		// Cycle or empty.
		return nil
	}

	out := make([]*slk.SidebarSection, 0, len(s.sectionsByID))
	visited := map[string]bool{}
	cur := head
	for cur != nil && !visited[cur.ID] {
		visited[cur.ID] = true
		if includeInSidebar(cur) {
			out = append(out, cur)
		}
		if cur.Next == "" {
			break
		}
		cur = s.sectionsByID[cur.Next]
	}
	return out
}

// AssignableSections returns ordered sidebar sections a channel can be
// moved into: user-created (standard) plus the default Channels and
// Direct Messages buckets. Stars/apps/hidden types are omitted.
func (s *SectionStore) AssignableSections() []*slk.SidebarSection {
	all := s.OrderedSections()
	out := make([]*slk.SidebarSection, 0, len(all))
	for _, sec := range all {
		switch sec.Type {
		case "standard", "channels", "direct_messages":
			out = append(out, sec)
		}
	}
	return out
}

// includeInSidebar applies the render filter rules. Renderable types:
// standard (always, even when empty — user intent), channels (default
// catch-all), direct_messages (default DM bucket), stars (Slack's
// Starred feature — only when non-empty, mirroring recent_apps so users
// without starred channels don't see an empty header). recent_apps is
// only rendered when non-empty (slk has its own Apps logic for the
// empty case). Everything else is hidden (slack_connect,
// salesforce_records, agents, anything new).
func includeInSidebar(sec *slk.SidebarSection) bool {
	if sec.IsRedacted {
		return false
	}
	switch sec.Type {
	case "standard", "channels", "direct_messages":
		return true
	case "stars", "recent_apps":
		return len(sec.ChannelIDs) > 0
	default:
		// slack_connect, salesforce_records, agents, anything new.
		return false
	}
}

// ApplyUpsert applies a channel_section_upserted WS event (also used
// for create / rename / reorder / emoji change). Last-write-wins by
// LastUpdate: stale events are dropped.
func (s *SectionStore) ApplyUpsert(ev slk.ChannelSectionUpserted) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return
	}
	if existing, ok := s.sectionsByID[ev.ID]; ok && ev.LastUpdate < existing.LastUpdate {
		return
	}
	prev := s.sectionsByID[ev.ID]
	sec := &slk.SidebarSection{
		ID:         ev.ID,
		Name:       ev.Name,
		Type:       ev.Type,
		Emoji:      ev.Emoji,
		Next:       ev.Next,
		LastUpdate: ev.LastUpdate,
		IsRedacted: ev.IsRedacted,
	}
	if prev != nil {
		// Preserve channel membership; upsert events don't carry it.
		sec.ChannelIDs = prev.ChannelIDs
		sec.ChannelsCount = prev.ChannelsCount
		if ev.LastUpdate == 0 {
			// Local rename/create patch: keep linked-list fields the
			// WS event would have sent.
			if sec.Next == "" {
				sec.Next = prev.Next
			}
			if sec.Type == "" {
				sec.Type = prev.Type
			}
			if sec.Emoji == "" {
				sec.Emoji = prev.Emoji
			}
			if sec.Name == "" {
				sec.Name = prev.Name
			}
		}
	}
	s.sectionsByID[ev.ID] = sec
}

// ApplyDelete applies a channel_section_deleted WS event.
func (s *SectionStore) ApplyDelete(sectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return
	}
	deleted := s.sectionsByID[sectionID]
	next := ""
	if deleted != nil {
		next = deleted.Next
	}
	for _, sec := range s.sectionsByID {
		if sec.Next == sectionID {
			sec.Next = next
		}
	}
	delete(s.sectionsByID, sectionID)
	for ch, sec := range s.channelToSection {
		if sec == sectionID {
			delete(s.channelToSection, ch)
		}
	}
}

// MoveSection swaps a standard section with its neighbor in display
// order. delta is -1 (up) or +1 (down). Returns the full linked-list
// of section IDs for users.channelSections.reorder.
func (s *SectionStore) MoveSection(sectionID string, delta int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return nil, fmt.Errorf("sections not ready")
	}
	ids := s.fullOrderedIDsLocked()
	idx := -1
	for i, id := range ids {
		if id == sectionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("section not found")
	}
	j := idx + delta
	if j < 0 || j >= len(ids) {
		return nil, fmt.Errorf("already at the edge")
	}
	ids[idx], ids[j] = ids[j], ids[idx]
	s.relinkLocked(ids)
	out := make([]string, len(ids))
	copy(out, ids)
	return out, nil
}

func (s *SectionStore) fullOrderedIDsLocked() []string {
	pointedAt := map[string]bool{}
	for _, sec := range s.sectionsByID {
		if sec.Next != "" {
			pointedAt[sec.Next] = true
		}
	}
	var head *slk.SidebarSection
	for id, sec := range s.sectionsByID {
		if !pointedAt[id] {
			if head == nil || sec.LastUpdate > head.LastUpdate {
				head = sec
			}
		}
	}
	if head == nil {
		return nil
	}
	out := make([]string, 0, len(s.sectionsByID))
	visited := map[string]bool{}
	cur := head
	for cur != nil && !visited[cur.ID] {
		visited[cur.ID] = true
		out = append(out, cur.ID)
		if cur.Next == "" {
			break
		}
		cur = s.sectionsByID[cur.Next]
	}
	return out
}

func (s *SectionStore) relinkLocked(ids []string) {
	for i, id := range ids {
		sec := s.sectionsByID[id]
		if sec == nil {
			continue
		}
		if i+1 < len(ids) {
			sec.Next = ids[i+1]
		} else {
			sec.Next = ""
		}
	}
}

// ApplyChannelsAdded applies a channel_sections_channels_upserted WS event.
// A channel can only belong to one section, so adding to section X
// implicitly removes it from any prior section in our index.
func (s *SectionStore) ApplyChannelsAdded(sectionID string, channelIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return
	}
	sec, ok := s.sectionsByID[sectionID]
	if !ok {
		// Section we don't know about yet; skip — bootstrap or upsert
		// will reconcile.
		return
	}
	added := map[string]bool{}
	for _, ch := range sec.ChannelIDs {
		added[ch] = true
	}
	for _, ch := range channelIDs {
		if !added[ch] {
			sec.ChannelIDs = append(sec.ChannelIDs, ch)
			added[ch] = true
		}
		// Remove from any other section's ChannelIDs.
		if prevSec, prev := s.channelToSection[ch]; prev && prevSec != sectionID {
			if old, ok := s.sectionsByID[prevSec]; ok {
				filtered := old.ChannelIDs[:0]
				for _, x := range old.ChannelIDs {
					if x != ch {
						filtered = append(filtered, x)
					}
				}
				old.ChannelIDs = filtered
			}
		}
		s.channelToSection[ch] = sectionID
	}
}

// ApplyChannelsRemoved applies a channel_sections_channels_removed WS event.
func (s *SectionStore) ApplyChannelsRemoved(sectionID string, channelIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return
	}
	sec, ok := s.sectionsByID[sectionID]
	if !ok {
		return
	}
	dropped := map[string]bool{}
	for _, ch := range channelIDs {
		dropped[ch] = true
		if cur, ok := s.channelToSection[ch]; ok && cur == sectionID {
			delete(s.channelToSection, ch)
		}
	}
	filtered := sec.ChannelIDs[:0]
	for _, ch := range sec.ChannelIDs {
		if !dropped[ch] {
			filtered = append(filtered, ch)
		}
	}
	sec.ChannelIDs = filtered
}

// MaybeRebootstrap re-runs Bootstrap when the previous successful one was
// more than 30 seconds ago. Cheap insurance against missed events during
// disconnects without thundering during a flapping connection.
func (s *SectionStore) MaybeRebootstrap(ctx context.Context, client SectionsClient) error {
	s.mu.RLock()
	last := s.lastBootstrap
	s.mu.RUnlock()
	if !last.IsZero() && time.Since(last) < 30*time.Second {
		return nil
	}
	return s.Bootstrap(ctx, client)
}
