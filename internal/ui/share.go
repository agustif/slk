// internal/ui/share.go
//
// OG Slack share/forward: pick a destination with the channel finder
// overlay, then chat.postMessage the selected message's permalink so
// Slack unfurls it. No extra comment prompt.
package ui

import (
	"context"
	"log"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/gammons/slk/internal/ids"
	"github.com/gammons/slk/internal/ui/channelfinder"
)

func (a *App) openSharePicker() tea.Cmd {
	if a.focusedPanel == PanelMessages && !a.inChannelView() {
		return nil
	}
	channelID, ts, _, _, _, ok := a.selectedMessageContext()
	if !ok || channelID == "" || ts == "" {
		return nil
	}
	a.shareFromChannel = channelID
	a.shareFromTS = ts
	a.channelFinder.SetShareMode(true)
	a.channelFinder.Open()
	a.SetMode(ModeShare)
	return nil
}

func (a *App) closeSharePicker() {
	a.channelFinder.Close()
	a.channelFinder.SetShareMode(false)
	a.shareFromChannel = ""
	a.shareFromTS = ""
	a.SetMode(ModeNormal)
}

func (a *App) shareMessageTo(dest *channelfinder.ChannelResult) tea.Cmd {
	if dest == nil || dest.ID == "" {
		return nil
	}
	switch dest.Type {
	case "threads", "activity", "later", "dms", "drafts":
		return nil
	}
	fromCh, fromTS := a.shareFromChannel, a.shareFromTS
	if fromCh == "" || fromTS == "" {
		return nil
	}
	messageSvc := a.messageSvc
	srcCh := ids.ChannelID(fromCh)
	srcTS := ids.MessageTS(fromTS)
	dstCh := ids.ChannelID(dest.ID)
	dstName, dstType := dest.Name, dest.Type

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		url, err := messageSvc.Permalink(ctx, srcCh, srcTS)
		if err != nil {
			log.Printf("share permalink: %v", err)
			return MessageShareFailedMsg{Reason: err.Error()}
		}
		if url == "" {
			return MessageShareFailedMsg{Reason: "no permalink"}
		}
		sent := messageSvc.Send(dstCh, url)
		if fail, ok := sent.(MessageSendFailedMsg); ok {
			reason := fail.Reason
			if reason == "" {
				reason = "send failed"
			}
			return MessageShareFailedMsg{Reason: reason}
		}
		shared := MessageSharedMsg{DestName: dstName, DestType: dstType}
		if sent == nil {
			return shared
		}
		return tea.Batch(
			func() tea.Msg { return sent },
			func() tea.Msg { return shared },
		)()
	}
}

func shareDestLabel(name, chType string) string {
	switch chType {
	case "dm", "group_dm", "app":
		return name
	default:
		if name != "" && !strings.HasPrefix(name, "#") {
			return "#" + name
		}
		return name
	}
}
