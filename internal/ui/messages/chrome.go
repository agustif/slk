package messages

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/agustif/slk/internal/ui/styles"
)

// Bookmark is a channel-header bookmark (title + URL).
type Bookmark struct {
	Title string
	URL   string
}

// Pin is a pinned item used by the header pin chip. Message pins
// carry TS for in-app jump; file pins may only have Permalink.
type Pin struct {
	TS        string
	Text      string
	Permalink string
	Created   int64
}

// ChromeHitKind classifies a click on the header extras row.
type ChromeHitKind int

const (
	ChromeHitNone ChromeHitKind = iota
	ChromeHitBookmark
	ChromeHitMore
	ChromeHitPins
)

// ChromeHit is a click on the bookmarks/pins extras row.
type ChromeHit struct {
	Kind  ChromeHitKind
	Index int // bookmark index for ChromeHitBookmark
}

type chromeHitKind int

const (
	chromeHitBookmark chromeHitKind = iota
	chromeHitMore
	chromeHitPins
)

type chromeHit struct {
	kind     chromeHitKind
	index    int
	colStart int
	colEnd   int
}

const (
	extrasSep        = " · "
	extrasPad        = 1
	maxBookmarkTitle = 24
	minBookmarkTitle = 8
)

// SetHeaderChrome replaces the bookmarks/pins shown under the channel
// name. Empty bookmarks and no pins omit the extras row.
func (m *Model) SetHeaderChrome(bookmarks []Bookmark, pins []Pin) {
	m.bookmarks = slices.Clone(bookmarks)
	m.pins = slices.Clone(pins)
	m.chromeCacheValid = false
	m.dirty()
}

// Bookmarks returns a copy of the current header bookmarks.
func (m *Model) Bookmarks() []Bookmark {
	return slices.Clone(m.bookmarks)
}

// Pins returns a copy of the current header pins.
func (m *Model) Pins() []Pin {
	return slices.Clone(m.pins)
}

// HitTestChrome reports a header-extras hit at pane-local (y, x).
// Coordinates match ClickAt (y=0 is the channel name). Hits come from
// the most recent View().
func (m *Model) HitTestChrome(y, x int) (ChromeHit, bool) {
	if m.chromeExtrasRow < 0 || y != m.chromeExtrasRow {
		return ChromeHit{}, false
	}
	for _, h := range m.chromeHits {
		if x >= h.colStart && x < h.colEnd {
			switch h.kind {
			case chromeHitBookmark:
				return ChromeHit{Kind: ChromeHitBookmark, Index: h.index}, true
			case chromeHitMore:
				return ChromeHit{Kind: ChromeHitMore}, true
			case chromeHitPins:
				return ChromeHit{Kind: ChromeHitPins}, true
			}
		}
	}
	return ChromeHit{}, false
}

func extrasLinkStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(styles.Primary).
		Underline(true).
		Background(styles.Background)
}

func extrasMutedStyle() lipgloss.Style {
	return styles.Timestamp.Background(styles.Background)
}

func renderHeaderExtras(bookmarks []Bookmark, pins []Pin, width int) (line string, hits []chromeHit, ok bool) {
	pinCount := len(pins)
	if len(bookmarks) == 0 && pinCount == 0 {
		return "", nil, false
	}
	if width < 1 {
		width = 1
	}

	items := make([]extrasItem, 0, len(bookmarks))
	for i, b := range bookmarks {
		title := strings.TrimSpace(b.Title)
		if title == "" {
			title = strings.TrimSpace(b.URL)
		}
		if title == "" {
			continue
		}
		items = append(items, extrasItem{
			title: ansi.Truncate(title, maxBookmarkTitle, "\u2026"),
			url:   b.URL,
			index: i,
		})
	}

	pad := extrasPad
	inner := width - 2*pad
	if inner < 8 {
		inner = width
		pad = 0
	}

	pinLabel := ""
	pinW := 0
	if pinCount > 0 {
		pinLabel = fmt.Sprintf("\U0001F4CC %d", pinCount)
		pinW = lipgloss.Width(pinLabel)
	}
	budget := inner
	if pinW > 0 {
		gap := 0
		if len(items) > 0 {
			gap = 2
		}
		need := pinW + gap
		if need < inner {
			budget = inner - need
		} else {
			budget = 0
		}
	}

	n, more := fitBookmarkCount(items, budget)
	if n == 0 && len(items) > 0 && budget >= minBookmarkTitle {
		avail := budget
		if len(items) > 1 {
			moreW := 1 + lipgloss.Width(fmt.Sprintf("+%d more", len(items)-1))
			if budget-moreW >= minBookmarkTitle {
				avail = budget - moreW
				more = len(items) - 1
			} else {
				more = 0
			}
		}
		items[0].title = ansi.Truncate(items[0].title, avail, "\u2026")
		if lipgloss.Width(items[0].title) <= budget {
			n = 1
		}
	}

	var b strings.Builder
	col := pad
	if pad > 0 {
		b.WriteByte(' ')
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(extrasSep)
			col += lipgloss.Width(extrasSep)
		}
		it := items[i]
		start := col
		w := lipgloss.Width(it.title)
		styled := extrasLinkStyle().Render(it.title)
		if it.url != "" {
			styled = osc8Hyperlink(it.url, styled)
		}
		b.WriteString(styled)
		hits = append(hits, chromeHit{kind: chromeHitBookmark, index: it.index, colStart: start, colEnd: start + w})
		col += w
	}
	if more > 0 {
		moreStr := fmt.Sprintf("+%d more", more)
		if n > 0 {
			b.WriteByte(' ')
			col++
		}
		start := col
		b.WriteString(extrasMutedStyle().Render(moreStr))
		w := lipgloss.Width(moreStr)
		hits = append(hits, chromeHit{kind: chromeHitMore, colStart: start, colEnd: start + w})
		col += w
	}
	if pinLabel != "" {
		if col > pad {
			b.WriteString("  ")
			col += 2
		}
		start := col
		b.WriteString(extrasMutedStyle().Render(pinLabel))
		w := lipgloss.Width(pinLabel)
		hits = append(hits, chromeHit{kind: chromeHitPins, colStart: start, colEnd: start + w})
		col += w
	}

	line = b.String()
	disp := lipgloss.Width(line)
	if disp < width {
		line += strings.Repeat(" ", width-disp)
	}
	line = lipgloss.NewStyle().Background(styles.Background).Render(line)
	line = ReapplyBgAfterResets(line, BgANSI())
	return line, hits, true
}

type extrasItem struct {
	title string
	url   string
	index int
}

func fitBookmarkCount(items []extrasItem, budget int) (n, more int) {
	if len(items) == 0 || budget < 1 {
		return 0, len(items)
	}
	sepW := lipgloss.Width(extrasSep)
	for n := len(items); n >= 1; n-- {
		more := len(items) - n
		w := 0
		for i := 0; i < n; i++ {
			if i > 0 {
				w += sepW
			}
			w += lipgloss.Width(items[i].title)
		}
		if more > 0 {
			w += 1 + lipgloss.Width(fmt.Sprintf("+%d more", more))
		}
		if w <= budget {
			return n, more
		}
	}
	return 0, len(items)
}
