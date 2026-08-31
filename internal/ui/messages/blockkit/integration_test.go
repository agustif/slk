package blockkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/slack-go/slack"
)

// fixturePayload mirrors the shape of the JSON files in testdata/.
// Not all fields are populated for every fixture; that's fine —
// json.Unmarshal leaves missing fields zero-valued.
type fixturePayload struct {
	Blocks      slack.Blocks       `json:"blocks"`
	Attachments []slack.Attachment `json:"attachments"`
}

func loadFixture(t *testing.T, name string) fixturePayload {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var p fixturePayload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return p
}

func makeCtx() Context {
	return Context{
		RenderText: func(s string, _ map[string]string) string { return s },
		WrapText:   func(s string, _ int) string { return s },
	}
}

func TestFixture_GitHubPR(t *testing.T) {
	p := loadFixture(t, "github_pr.json")
	blocks := Parse(p.Blocks)
	for _, w := range []int{60, 100, 140} {
		r := Render(blocks, makeCtx(), w)
		plain := ansi.Strip(strings.Join(r.Lines, "\n"))
		for _, want := range []string{"Pull Request opened", "Fix retry logic", "3 files changed"} {
			if !strings.Contains(plain, want) {
				t.Errorf("width=%d missing %q in %q", w, want, plain)
			}
		}
	}
}

func TestFixture_PagerDutyAlert(t *testing.T) {
	p := loadFixture(t, "pagerduty_alert.json")
	atts := ParseAttachments(p.Attachments)
	for _, w := range []int{60, 100, 140} {
		r := RenderLegacy(atts, makeCtx(), w)
		plain := ansi.Strip(strings.Join(r.Lines, "\n"))
		for _, want := range []string{"Service down", "checkout-svc", "SEV-2", "Datadog"} {
			if !strings.Contains(plain, want) {
				t.Errorf("width=%d missing %q in %q", w, want, plain)
			}
		}
		if !strings.Contains(plain, "█") {
			t.Errorf("width=%d missing color stripe", w)
		}
	}
}

func TestFixture_DeployApproval(t *testing.T) {
	p := loadFixture(t, "deploy_approval.json")
	blocks := Parse(p.Blocks)
	for _, w := range []int{60, 100, 140} {
		r := Render(blocks, makeCtx(), w)
		plain := ansi.Strip(strings.Join(r.Lines, "\n"))
		if !strings.Contains(plain, "Deploy v2.3.1") {
			t.Errorf("width=%d missing body: %q", w, plain)
		}
		if !strings.Contains(plain, "[ Approve ]") || !strings.Contains(plain, "[ Deny ]") {
			t.Errorf("width=%d missing buttons: %q", w, plain)
		}
		if !r.Interactive {
			t.Errorf("width=%d Interactive should be true", w)
		}
	}
}

func TestFixture_OncallHandoff(t *testing.T) {
	p := loadFixture(t, "oncall_handoff.json")
	blocks := Parse(p.Blocks)
	for _, w := range []int{60, 100, 140} {
		r := Render(blocks, makeCtx(), w)
		plain := ansi.Strip(strings.Join(r.Lines, "\n"))
		for _, want := range []string{"Weekly on-call handoff", "alice", "bob", "rotates Mondays"} {
			if !strings.Contains(plain, want) {
				t.Errorf("width=%d missing %q in %q", w, want, plain)
			}
		}
	}
}

func TestFixture_SectionWithFields(t *testing.T) {
	p := loadFixture(t, "section_with_fields.json")
	blocks := Parse(p.Blocks)
	r := Render(blocks, makeCtx(), 100)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	for _, want := range []string{"Build complete", "Branch", "Commit", "Duration", "abc1234"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in %q", want, plain)
		}
	}
}

func TestFixture_HeaderDividerSection(t *testing.T) {
	p := loadFixture(t, "header_divider_section.json")
	blocks := Parse(p.Blocks)
	r := Render(blocks, makeCtx(), 80)
	if r.Height < 3 {
		t.Errorf("Height = %d, want >= 3 (header, divider, body)", r.Height)
	}
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	for _, want := range []string{"Top header", "Body text after divider"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in %q", want, plain)
		}
	}
}

// TestFixture_LinkUnfurlImageRequestsFetcher covers Slack's classic
// link-unfurl attachment (from_url + image_url, as produced after
// link_shared). Card text still renders; the og:image must go through
// the shared Fetcher pipeline rather than a text fallback.
func TestFixture_LinkUnfurlImageRequestsFetcher(t *testing.T) {
	p := loadFixture(t, "link_unfurl.json")
	atts := ParseAttachments(p.Attachments)
	if len(atts) != 1 {
		t.Fatalf("attachments = %d, want 1", len(atts))
	}
	if atts[0].ImageURL != "https://cdn.example.com/og/hello-world.png" {
		t.Fatalf("ImageURL = %q", atts[0].ImageURL)
	}
	if atts[0].ImageWidth != 1200 || atts[0].ImageHeight != 630 {
		t.Fatalf("ImageWidth/Height = %d/%d", atts[0].ImageWidth, atts[0].ImageHeight)
	}

	srv, fetcher := imagePipeline(t)
	atts[0].ImageURL = srv.URL + "/og/hello-world.png"
	const maxRows = 8
	r := RenderLegacy(atts, imagePipelineCtx(fetcher, maxRows), 80)

	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Hello World") {
		t.Errorf("missing unfurl title: %q", plain)
	}
	if !strings.Contains(plain, "greeting the planet") {
		t.Errorf("missing unfurl text: %q", plain)
	}
	if !strings.Contains(plain, "loading") {
		t.Errorf("expected image-pipeline placeholder, got %q", plain)
	}
	if strings.Contains(plain, "[image]") {
		t.Errorf("unfurl image fell back to a text link: %q", plain)
	}
	if len(r.Hits) != 1 {
		t.Fatalf("Hits = %d, want 1 (click footprint for the unfurl image)", len(r.Hits))
	}
	gotRows := r.Hits[0].RowEnd - r.Hits[0].RowStart
	if gotRows > maxRows {
		t.Errorf("unfurl image rows = %d, exceeds max_image_rows %d", gotRows, maxRows)
	}
	waitFetched(t, srv, "/og/hello-world.png")
}

func TestFixture_LinkUnfurlWithoutImageUnchanged(t *testing.T) {
	p := loadFixture(t, "link_unfurl.json")
	atts := ParseAttachments(p.Attachments)
	atts[0].ImageURL = ""
	atts[0].ThumbURL = ""
	atts[0].ImageWidth = 0
	atts[0].ImageHeight = 0
	r := RenderLegacy(atts, makeCtx(), 80)
	plain := ansi.Strip(strings.Join(r.Lines, "\n"))
	if !strings.Contains(plain, "Hello World") {
		t.Errorf("missing title: %q", plain)
	}
	if strings.Contains(plain, "[image]") || strings.Contains(plain, "loading") {
		t.Errorf("unfurl with no image should be text-only: %q", plain)
	}
	if len(r.Hits) != 0 {
		t.Errorf("Hits = %d, want 0", len(r.Hits))
	}
}
