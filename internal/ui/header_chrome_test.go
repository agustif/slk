package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/messages"
)

func TestChannelSelected_FetchesChromeWithoutBlockingMessages(t *testing.T) {
	app := NewApp()
	fetchCalled := 0
	chromeCalled := 0
	app.setChannelFetcherForTest(func(id ids.ChannelID, name string) tea.Msg {
		fetchCalled++
		return MessagesLoadedMsg{ChannelID: string(id), Messages: nil}
	})
	app.setChannelChromeFetcherForTest(func(id ids.ChannelID) tea.Msg {
		chromeCalled++
		if fetchCalled == 0 {
			// Chrome fetch is a sibling cmd, not sequenced after
			// messages; the closure may run in either order when
			// drained. Record that it ran.
		}
		return ChannelChromeMsg{
			ChannelID: string(id),
			Bookmarks: []messages.Bookmark{{Title: "Handbook", URL: "https://example.com/hb"}},
		}
	})
	app.setChannelCacheReaderForTest(func(ids.ChannelID) []messages.MessageItem { return nil })

	_, cmd := app.Update(ChannelSelectedMsg{ID: "C1", Name: "general", Type: "channel"})
	msgs := drainBatch(cmd)
	for _, m := range msgs {
		if m != nil {
			app.Update(m)
		}
	}
	if fetchCalled != 1 {
		t.Errorf("messages fetch = %d; want 1", fetchCalled)
	}
	if chromeCalled != 1 {
		t.Errorf("chrome fetch = %d; want 1", chromeCalled)
	}
	got := app.messagepane.Bookmarks()
	if len(got) != 1 || got[0].Title != "Handbook" {
		t.Errorf("bookmarks = %+v", got)
	}
}

func TestChannelChromeMsg_RendersTitles(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.messagepane.SetChannel("general", "")
	_, _ = app.Update(ChannelChromeMsg{
		ChannelID: "C1",
		Bookmarks: []messages.Bookmark{
			{Title: "Handbook", URL: "https://example.com/handbook"},
			{Title: "Standup notes", URL: "https://docs.example.com/standup"},
		},
		Pins: []messages.Pin{{TS: "1.0", Created: 2}, {TS: "2.0", Created: 1}},
	})
	plain := ansi.Strip(app.messagepane.View(12, 80))
	if !strings.Contains(plain, "Handbook") || !strings.Contains(plain, "Standup notes") {
		t.Errorf("header missing bookmark titles:\n%s", plain)
	}
	if !strings.Contains(plain, "\U0001F4CC 2") {
		t.Errorf("header missing pin count:\n%s", plain)
	}
}

func TestChannelChromeMsg_StaleDropped(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C2"
	app.chromeGen = 3
	_, _ = app.Update(ChannelChromeMsg{
		ChannelID: "C1",
		Bookmarks: []messages.Bookmark{{Title: "Old", URL: "https://example.com"}},
		Gen:       1,
	})
	if len(app.messagepane.Bookmarks()) != 0 {
		t.Fatal("chrome for a previous channel must be dropped")
	}
	_, _ = app.Update(ChannelChromeMsg{
		ChannelID: "C2",
		Bookmarks: []messages.Bookmark{{Title: "Old", URL: "https://example.com"}},
		Gen:       1,
	})
	if len(app.messagepane.Bookmarks()) != 0 {
		t.Fatal("chrome with a stale gen must be dropped")
	}
}

func TestHeaderChrome_ClickBookmarkOpensURL(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.messagepane.SetHeaderChrome([]messages.Bookmark{{Title: "Handbook", URL: "https://example.com/handbook"}}, nil)
	cmd := app.handleHeaderChromeHit(messages.ChromeHit{Kind: messages.ChromeHitBookmark, Index: 0})
	if cmd == nil {
		t.Fatal("expected OpenLinkMsg cmd")
	}
	msg, ok := cmd().(OpenLinkMsg)
	if !ok || msg.URL != "https://example.com/handbook" {
		t.Fatalf("got %#v", msg)
	}
}

func TestHeaderChrome_ClickPinsOpensPickerWhenMultiple(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "1.0", Text: "older"},
		{TS: "9.0", Text: "pinned"},
	})
	app.messagepane.SelectByIndex(0)
	app.messagepane.SetHeaderChrome(nil, []messages.Pin{
		{TS: "9.0", Created: 20, Text: "pinned"},
		{TS: "1.0", Created: 10, Text: "older"},
	})
	cmd := app.handleHeaderChromeHit(messages.ChromeHit{Kind: messages.ChromeHitPins})
	if cmd != nil {
		t.Fatalf("picker should open in-mode, cmd=%#v", cmd())
	}
	if app.mode != ModeLinkPicker {
		t.Fatalf("mode = %v; want ModeLinkPicker", app.mode)
	}
	if app.pickerKind != "pins" {
		t.Errorf("pickerKind = %q", app.pickerKind)
	}
}

func TestHeaderChrome_ClickPinsJumpsWhenSingle(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.messagepane.SetMessages([]messages.MessageItem{
		{TS: "9.0", Text: "pinned"},
	})
	app.messagepane.SelectByIndex(0)
	app.messagepane.SetHeaderChrome(nil, []messages.Pin{
		{TS: "9.0", Created: 20, Text: "pinned"},
	})
	_ = app.handleHeaderChromeHit(messages.ChromeHit{Kind: messages.ChromeHitPins})
	got, ok := app.messagepane.SelectedMessage()
	if !ok || got.TS != "9.0" {
		t.Fatalf("selected = %+v ok=%v; want ts 9.0", got, ok)
	}
}

func TestHeaderChrome_SingleFilePinOpensPermalink(t *testing.T) {
	app := NewApp()
	app.activeChannelID = "C1"
	app.messagepane.SetHeaderChrome(nil, []messages.Pin{
		{Text: "roadmap.pdf", Permalink: "https://example.com/files/roadmap.pdf"},
	})
	cmd := app.handlePinsChip()
	if cmd == nil {
		t.Fatal("expected OpenLinkMsg cmd")
	}
	msg, ok := cmd().(OpenLinkMsg)
	if !ok || msg.URL != "https://example.com/files/roadmap.pdf" {
		t.Fatalf("got %#v", msg)
	}
}

func TestMostRecentMessagePin(t *testing.T) {
	pin, ok := mostRecentMessagePin([]messages.Pin{
		{TS: "1.0", Created: 5},
		{Text: "file only"},
		{TS: "2.0", Created: 9},
		{TS: "3.0", Created: 9},
	})
	if !ok || pin.TS != "3.0" {
		t.Errorf("got %+v ok=%v; want ts 3.0 (same created, higher ts)", pin, ok)
	}
}

func TestChannelSelected_ChromeFetchIsSeparateCmd(t *testing.T) {
	app := NewApp()
	app.setChannelCacheReaderForTest(func(ids.ChannelID) []messages.MessageItem {
		return []messages.MessageItem{{TS: "1.0", Text: "cached"}}
	})
	app.setChannelSyncedAtReaderForTest(func(ids.ChannelID) int64 { return 1 }) // stale → tier 2
	app.setChannelFetcherForTest(func(id ids.ChannelID, name string) tea.Msg {
		return MessagesLoadedMsg{ChannelID: string(id), Messages: []messages.MessageItem{{TS: "1.0", Text: "net"}}}
	})
	chromeStarted := false
	app.setChannelChromeFetcherForTest(func(id ids.ChannelID) tea.Msg {
		chromeStarted = true
		return ChannelChromeMsg{ChannelID: string(id)}
	})
	_, cmd := app.Update(ChannelSelectedMsg{ID: "C1", Name: "general", Type: "channel"})
	if app.messagepane.Messages()[0].Text != "cached" {
		t.Fatal("messages should render from cache before chrome returns")
	}
	if chromeStarted {
		t.Fatal("chrome fetch must not run synchronously inside Update")
	}
	_ = drainBatch(cmd)
	if !chromeStarted {
		t.Fatal("chrome fetch cmd was not dispatched")
	}
}
