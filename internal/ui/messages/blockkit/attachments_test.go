package blockkit

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	imgpkg "github.com/gammons/slk/internal/image"
)

func TestRenderLegacyEmptyReturnsZero(t *testing.T) {
	r := RenderLegacy(nil, Context{}, 80)
	if r.Height != 0 {
		t.Errorf("Height = %d, want 0", r.Height)
	}
}

func TestRenderLegacyTitleAndText(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title: "Service down",
		Text:  "checkout-svc returning 5xx",
	}}, ctx, 80)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Service down") {
		t.Errorf("missing title: %q", plain)
	}
	if !strings.Contains(plain, "checkout-svc returning 5xx") {
		t.Errorf("missing text: %q", plain)
	}
}

func TestRenderLegacyHasColorStripeOnEveryRow(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Color: "danger",
		Title: "T",
		Text:  "line1\nline2\nline3",
	}}, ctx, 40)
	for i, line := range r.Lines {
		plain := ansi.Strip(line)
		if !strings.HasPrefix(plain, "█") {
			t.Errorf("line %d does not start with stripe glyph: %q", i, plain)
		}
	}
}

func TestRenderLegacyPretextRendersAboveStripe(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Pretext: "Heads up:",
		Title:   "Inside",
	}}, ctx, 40)
	if r.Height < 2 {
		t.Fatalf("Height = %d, want >= 2", r.Height)
	}
	first := ansi.Strip(r.Lines[0])
	if !strings.Contains(first, "Heads up:") {
		t.Errorf("Lines[0] = %q, want pretext", first)
	}
	if strings.HasPrefix(first, "█") {
		t.Errorf("Lines[0] = %q, pretext must NOT have stripe", first)
	}
}

func TestRenderLegacyFooterAndTimestamp(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title:  "T",
		Footer: "Datadog",
		TS:     1700000000,
	}}, ctx, 60)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Datadog") {
		t.Errorf("missing footer: %q", plain)
	}
	// 1700000000 = 2023-11-14 in UTC
	if !strings.Contains(plain, "2023") {
		t.Errorf("expected formatted timestamp '2023…' in %q", plain)
	}
}

func TestRenderLegacyFieldsGridShortPairsShareRow(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title: "T",
		Fields: []LegacyField{
			{Title: "Service", Value: "web", Short: true},
			{Title: "Region", Value: "us-east-1", Short: true},
		},
	}}, ctx, 80)
	foundShared := false
	for _, line := range r.Lines {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "Service") && strings.Contains(plain, "Region") {
			foundShared = true
			break
		}
	}
	if !foundShared {
		t.Errorf("expected Service and Region on a shared row; lines = %q",
			ansi.Strip(strings.Join(r.Lines, "\n")))
	}
}

func TestRenderLegacyFieldsGridLongFieldFullWidth(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title: "T",
		Fields: []LegacyField{
			{Title: "Notes", Value: "long form", Short: false},
			{Title: "After", Value: "more", Short: false},
		},
	}}, ctx, 80)
	for _, line := range r.Lines {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "Notes") && strings.Contains(plain, "After") {
			t.Errorf("Notes and After should not share a row: %q", plain)
		}
	}
}

func TestRenderLegacyFieldRowsHaveStripe(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title: "T",
		Fields: []LegacyField{
			{Title: "K", Value: "V", Short: false},
		},
	}}, ctx, 60)
	for i, line := range r.Lines {
		plain := ansi.Strip(line)
		if !strings.HasPrefix(plain, "█") {
			t.Errorf("line %d does not start with stripe: %q", i, plain)
		}
	}
}

func TestRenderLegacyImageURLFallbackWhenNoFetcher(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title:    "T",
		ImageURL: "https://example.com/chart.png",
	}}, ctx, 60)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "https://example.com/chart.png") {
		t.Errorf("expected ImageURL fallback link, got %q", plain)
	}
	if !strings.Contains(plain, "[image]") {
		t.Errorf("expected '[image]' marker in fallback, got %q", plain)
	}
}

func TestRenderLegacyThumbURLFallbackWhenNoImageURL(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title:    "T",
		ThumbURL: "https://example.com/thumb.png",
	}}, ctx, 60)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "https://example.com/thumb.png") {
		t.Errorf("expected ThumbURL fallback link, got %q", plain)
	}
	if !strings.Contains(plain, "[image]") {
		t.Errorf("expected '[image]' marker in fallback, got %q", plain)
	}
}

func TestRenderLegacyPrefersImageURLOverThumbURL(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title:    "T",
		ImageURL: "https://example.com/og.png",
		ThumbURL: "https://example.com/thumb.png",
	}}, ctx, 60)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "https://example.com/og.png") {
		t.Errorf("expected ImageURL in fallback, got %q", plain)
	}
	if strings.Contains(plain, "https://example.com/thumb.png") {
		t.Errorf("ThumbURL should not render when ImageURL is set: %q", plain)
	}
}

func TestRenderLegacyUnfurlImageRequestsFetcher(t *testing.T) {
	srv, fetcher := imagePipeline(t)
	url := srv.URL + "/og.png"
	const maxRows = 6
	ctx := imagePipelineCtx(fetcher, maxRows)
	r := RenderLegacy([]LegacyAttachment{{
		Title:       "Hello World",
		ImageURL:    url,
		ImageWidth:  1200,
		ImageHeight: 630,
	}}, ctx, 80)

	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "loading") {
		t.Errorf("expected reserved-height placeholder, got %q", plain)
	}
	if !strings.Contains(plain, "Hello World") {
		t.Errorf("missing title: %q", plain)
	}
	if len(r.Hits) != 1 {
		t.Fatalf("Hits = %d, want 1", len(r.Hits))
	}
	if r.Hits[0].URL != url {
		t.Errorf("Hit.URL = %q, want %q", r.Hits[0].URL, url)
	}
	gotRows := r.Hits[0].RowEnd - r.Hits[0].RowStart
	if gotRows > maxRows {
		t.Errorf("image rows = %d, want <= max_image_rows %d", gotRows, maxRows)
	}
	if gotRows < 1 {
		t.Errorf("image rows = %d, want a reserved block", gotRows)
	}
	waitFetched(t, srv, "/og.png")
}

func TestRenderLegacyThumbURLRequestsFetcherWhenNoImageURL(t *testing.T) {
	srv, fetcher := imagePipeline(t)
	url := srv.URL + "/thumb.png"
	ctx := imagePipelineCtx(fetcher, 4)
	r := RenderLegacy([]LegacyAttachment{{
		Title:    "Card",
		ThumbURL: url,
	}}, ctx, 80)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "loading") {
		t.Errorf("expected placeholder for thumb_url unfurl, got %q", plain)
	}
	if len(r.Hits) != 1 || r.Hits[0].URL != url {
		t.Errorf("Hits = %+v, want URL %q", r.Hits, url)
	}
	waitFetched(t, srv, "/thumb.png")
}

func TestRenderLegacyNestedImageBlockRequestsFetcher(t *testing.T) {
	srv, fetcher := imagePipeline(t)
	url := srv.URL + "/block.png"
	ctx := imagePipelineCtx(fetcher, 5)
	r := RenderLegacy([]LegacyAttachment{{
		Blocks: []Block{
			SectionBlock{Text: "Unfurl card"},
			ImageBlock{URL: url, Alt: "preview"},
		},
	}}, ctx, 80)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Unfurl card") {
		t.Errorf("missing nested section: %q", plain)
	}
	if !strings.Contains(plain, "loading") {
		t.Errorf("expected nested image block to hit the image pipeline, got %q", plain)
	}
	if len(r.Hits) != 1 || r.Hits[0].URL != url {
		t.Errorf("Hits = %+v, want nested image URL %q", r.Hits, url)
	}
	waitFetched(t, srv, "/block.png")
}

func TestRenderLegacyNoImageUnchanged(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Title: "Just a card",
		Text:  "no preview image",
	}}, ctx, 60)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Just a card") || !strings.Contains(plain, "no preview image") {
		t.Errorf("text card changed: %q", plain)
	}
	if strings.Contains(plain, "[image]") || strings.Contains(plain, "loading") {
		t.Errorf("unfurl without image should not reserve an image: %q", plain)
	}
	if len(r.Hits) != 0 {
		t.Errorf("Hits = %d, want 0", len(r.Hits))
	}
}

type pipelineServer struct {
	*httptest.Server
	got chan string
}

func imagePipeline(t *testing.T) (*pipelineServer, *imgpkg.Fetcher) {
	t.Helper()
	pngBytes := testPNGBytes(t)
	ps := &pipelineServer{got: make(chan string, 8)}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case ps.got <- r.URL.Path:
		default:
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	t.Cleanup(ps.Close)
	cache, err := imgpkg.NewCache(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	return ps, imgpkg.NewFetcher(cache, ps.Client())
}

func imagePipelineCtx(fetcher *imgpkg.Fetcher, maxRows int) Context {
	return Context{
		Protocol:   imgpkg.ProtoHalfBlock,
		Fetcher:    fetcher,
		CellPixels: image.Pt(8, 16),
		MaxRows:    maxRows,
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
		SendMsg:    func(any) {},
		Channel:    "C1",
		MessageTS:  "1.0",
	}
}

func waitFetched(t *testing.T, srv *pipelineServer, wantPath string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case p := <-srv.got:
			if p == wantPath {
				return
			}
		case <-deadline:
			t.Fatalf("image pipeline was not requested for %s", wantPath)
		}
	}
}

func testPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestRenderLegacyRendersNestedBlocks covers link-unfurl attachments
// (Linear/Jira/etc.) whose entire content lives in nested Block Kit
// blocks while Title/Text/Fields are empty. Those blocks must render
// inside the attachment's colored stripe, otherwise the card shows
// nothing.
func TestRenderLegacyRendersNestedBlocks(t *testing.T) {
	ctx := Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
	r := RenderLegacy([]LegacyAttachment{{
		Color: "#2d1c9c",
		Blocks: []Block{
			SectionBlock{Text: "TRU-111 Customer Facing Blacklist Monitoring"},
			ContextBlock{Elements: []ContextElement{{Text: "*State*  In Progress"}}},
		},
	}}, ctx, 60)
	if r.Height == 0 {
		t.Fatalf("Height = 0, expected nested blocks to render")
	}
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "TRU-111 Customer Facing Blacklist Monitoring") {
		t.Errorf("missing section text: %q", plain)
	}
	if !strings.Contains(plain, "In Progress") {
		t.Errorf("missing context text: %q", plain)
	}
	for i, line := range r.Lines {
		if !strings.HasPrefix(ansi.Strip(line), "█") {
			t.Errorf("line %d missing stripe prefix: %q", i, ansi.Strip(line))
		}
	}
}
