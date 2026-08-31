package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ui/linkpicker"
	"github.com/gammons/slk/internal/ui/messages"
)

func (a *App) handleHeaderChromeHit(hit messages.ChromeHit) tea.Cmd {
	switch hit.Kind {
	case messages.ChromeHitBookmark:
		bms := a.messagepane.Bookmarks()
		if hit.Index < 0 || hit.Index >= len(bms) {
			return nil
		}
		url := bms[hit.Index].URL
		if url == "" {
			return nil
		}
		return func() tea.Msg { return OpenLinkMsg{URL: url} }
	case messages.ChromeHitMore:
		return a.openBookmarksPicker(a.messagepane.Bookmarks())
	case messages.ChromeHitPins:
		return a.handlePinsChip()
	}
	return nil
}

func (a *App) handlePinsChip() tea.Cmd {
	pins := a.messagepane.Pins()
	if pin, ok := mostRecentMessagePin(pins); ok {
		return a.jumpToMessageTS(a.activeChannelID, pin.TS)
	}
	if len(pins) == 0 {
		return nil
	}
	return a.openPinsPicker(pins)
}

func mostRecentMessagePin(pins []messages.Pin) (messages.Pin, bool) {
	var best messages.Pin
	found := false
	for _, p := range pins {
		if p.TS == "" {
			continue
		}
		if !found || p.Created > best.Created || (p.Created == best.Created && p.TS > best.TS) {
			best = p
			found = true
		}
	}
	return best, found
}

func (a *App) jumpToMessageTS(channelID, ts string) tea.Cmd {
	if channelID == "" || ts == "" {
		return nil
	}
	a.pendingLinkNav = &pendingLinkNav{channelID: channelID, messageTS: ts}
	return a.completePendingLinkNav(channelID, true)
}

func (a *App) jumpToPin(pin messages.Pin) tea.Cmd {
	if pin.TS != "" {
		return a.jumpToMessageTS(a.activeChannelID, pin.TS)
	}
	if pin.Permalink != "" {
		url := pin.Permalink
		return func() tea.Msg { return OpenLinkMsg{URL: url} }
	}
	return nil
}

func (a *App) openBookmarksPicker(bms []messages.Bookmark) tea.Cmd {
	items := make([]linkpicker.Item, 0, len(bms))
	for _, b := range bms {
		if b.URL == "" && b.Title == "" {
			continue
		}
		items = append(items, linkpicker.Item{URL: b.URL, Label: b.Title})
	}
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 && items[0].URL != "" {
		url := items[0].URL
		return func() tea.Msg { return OpenLinkMsg{URL: url} }
	}
	a.pickerKind = "links"
	a.linkPicker.Open("Bookmarks", items)
	a.SetMode(ModeLinkPicker)
	return nil
}

func (a *App) openPinsPicker(pins []messages.Pin) tea.Cmd {
	items := make([]linkpicker.Item, len(pins))
	for i, p := range pins {
		label := p.Text
		if label == "" {
			label = p.TS
		}
		if label == "" {
			label = p.Permalink
		}
		items[i] = linkpicker.Item{URL: p.Permalink, Label: label}
	}
	a.pickerKind = "pins"
	a.pickerPins = pins
	a.linkPicker.Open("Pinned messages", items)
	a.SetMode(ModeLinkPicker)
	return nil
}
