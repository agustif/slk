// Package avatar downloads Slack user avatars and renders them at a
// fixed 4×2 cell footprint. Storage delegates to internal/image's
// shared cache; rendering uses kitty graphics on capable terminals
// (sharper) and falls back to half-block (▀) elsewhere. Sixel is
// intentionally NOT used for avatars: re-emitting sixel byte streams
// per visible avatar per redraw would dominate the bandwidth budget.
package avatar

import (
	"context"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	imgpkg "github.com/gammons/slk/internal/image"
)

const (
	// AvatarCols is the width of the rendered avatar in terminal columns.
	AvatarCols = 4
	// AvatarRows is the height in terminal rows. Half-block uses 2 pixel
	// rows per cell row; kitty fits the source image to AvatarCols×AvatarRows
	// cells and the terminal scales pixels appropriately.
	AvatarRows = 2

	// SidebarCols/SidebarRows is the compact portrait used in the
	// channel sidebar. 2×1 cells is roughly square on a typical
	// cell (≈8×16px) and keeps 1:1 DMs on a single sidebar row.
	SidebarCols = 2
	SidebarRows = 1
)

// Size is a rendered avatar footprint in terminal cells.
type Size struct {
	Cols int
	Rows int
}

// DefaultSize is the 4×2 footprint used in the messages pane, thread
// pane, Activity, and the workspace-rail logo.
func DefaultSize() Size { return Size{Cols: AvatarCols, Rows: AvatarRows} }

// SidebarSize is the compact 2×1 footprint used for 1:1 DM portraits
// in the channel sidebar.
func SidebarSize() Size { return Size{Cols: SidebarCols, Rows: SidebarRows} }

func normalizeSize(sz Size) Size {
	if sz.Cols <= 0 {
		sz.Cols = AvatarCols
	}
	if sz.Rows <= 0 {
		sz.Rows = AvatarRows
	}
	return sz
}

func isDefaultSize(sz Size) bool {
	sz = normalizeSize(sz)
	return sz.Cols == AvatarCols && sz.Rows == AvatarRows
}

// renderMapKey is the Cache.renders / inflight key. Default-size
// entries keep the bare id so existing Get(userID) lookups and the
// half-block parity golden stay byte-identical.
func renderMapKey(id string, sz Size) string {
	sz = normalizeSize(sz)
	if isDefaultSize(sz) {
		return id
	}
	return fmt.Sprintf("%s@%dx%d", id, sz.Cols, sz.Rows)
}

func fetchKey(id string, sz Size) string {
	sz = normalizeSize(sz)
	if isDefaultSize(sz) {
		return "avatar-" + id
	}
	return fmt.Sprintf("avatar-%s-%dx%d", id, sz.Cols, sz.Rows)
}

func kittyKey(id string, sz Size) string {
	return fetchKey(id, sz)
}

// avatarPreloadWorkers caps the number of avatar Preload jobs that
// can be downloading + rendering concurrently. Bounding this matters
// because the lazy AvatarFunc path can call Preload for many distinct
// userIDs in a single frame when a scrollback first paints; without
// a bound, each unique user spawned a goroutine that could race to
// write its kitty graphics APC upload to os.Stdout, and a burst of
// hundreds of those is enough to make a kitty terminal visibly stall
// while it decodes them. 8 keeps the disk/network subsystems busy
// without saturating the kitty graphics queue.
const avatarPreloadWorkers = 8

// avatarPreloadQueueSize caps the worker pool's pending-job queue. In
// the lazy-load model this is bounded by visible-history size in
// practice (an extreme channel with thousands of unique authors in a
// scrollback can still be drained linearly). When the queue is full,
// Preload drops the job AND removes the userID from inflight so a
// subsequent retry can re-enqueue. 256 covers ordinary workloads with
// plenty of headroom.
const avatarPreloadQueueSize = 256

// Cache wraps an image.Fetcher and memoizes rendered ANSI strings per user.
//
// When the active rendering protocol is kitty, the avatar's "render"
// is a small block of unicode-placeholder cells; the actual image
// upload happens via the kitty side channel (image.KittyOutput) on
// first render of a given user, deduped by the kitty registry's
// per-(key,target) tracking. When the protocol is not kitty (sixel,
// half-block, off, ...), the avatar renders as half-block ANSI text.
type Cache struct {
	fetcher  *imgpkg.Fetcher
	kitty    *imgpkg.KittyRenderer // nil when not using kitty
	useKitty bool

	mu      sync.RWMutex
	renders map[string]string // renderMapKey(id, size) -> rendered ANSI string

	// inflight is only the currently-running fetch. Successful
	// renders live in `renders`; failures go to `failed` with a
	// cooldown so AvatarFunc's per-frame Preload can retry later
	// instead of being stuck "in flight" forever.
	inflight sync.Map // renderMapKey(id, size) -> struct{}
	failed   sync.Map // renderMapKey(id, size) -> failState

	nowFn     func() time.Time
	backoffFn func(tries int) time.Duration

	// onReady is invoked from the worker goroutine after a render is
	// stored in c.renders. Hosts use it to dispatch a bubbletea
	// invalidation message (AvatarReadyMsg). May be nil; nil-safe.
	onReady func(userID string)

	// preloadCh feeds the bounded worker pool. Preload enqueues jobs
	// here; workers drain. Closed jobs are not supported (Cache lives
	// for the program's lifetime). nil disables async pool dispatch
	// (PreloadSync still works directly), which the kitty/parity tests
	// rely on.
	preloadCh chan preloadJob
}

type preloadJob struct {
	id        string
	avatarURL string
	size      Size
}

type failState struct {
	until time.Time
	tries int
	url   string
}

const (
	failBackoffInitial = time.Second
	failBackoffMax     = time.Minute
)

func defaultFailBackoff(tries int) time.Duration {
	if tries < 1 {
		tries = 1
	}
	d := failBackoffInitial
	for i := 1; i < tries; i++ {
		if d >= failBackoffMax/2 {
			return failBackoffMax
		}
		d *= 2
	}
	if d > failBackoffMax {
		return failBackoffMax
	}
	return d
}

// NewCache creates an avatar cache backed by the shared image.Fetcher.
// kitty may be nil; in that case (or when useKitty is false) the cache
// renders avatars via half-block regardless of any kitty support
// elsewhere in the app.
func NewCache(fetcher *imgpkg.Fetcher, kitty *imgpkg.KittyRenderer, useKitty bool) *Cache {
	return newCacheForTest(fetcher, kitty, useKitty, avatarPreloadWorkers, avatarPreloadQueueSize)
}

// newCacheForTest is the underlying constructor; production code uses
// NewCache, tests use this to dial worker count and queue size to
// produce deterministic backpressure behavior.
func newCacheForTest(fetcher *imgpkg.Fetcher, kitty *imgpkg.KittyRenderer, useKitty bool, workers, queueSize int) *Cache {
	c := &Cache{
		fetcher:   fetcher,
		kitty:     kitty,
		useKitty:  useKitty && kitty != nil,
		renders:   make(map[string]string),
		preloadCh: make(chan preloadJob, queueSize),
	}
	for i := 0; i < workers; i++ {
		go c.preloadWorker()
	}
	return c
}

func (c *Cache) preloadWorker() {
	for job := range c.preloadCh {
		c.preloadInner(job.id, job.avatarURL, job.size)
	}
}

// SetOnReady registers a callback invoked once per userID after a
// successful Preload completes. Not called for fetch failures. Safe to
// call once at startup before any Preload; concurrent reassignment is
// not supported.
func (c *Cache) SetOnReady(fn func(userID string)) {
	c.onReady = fn
}

// Preload enqueues a background download+render for an avatar. Bounded
// by the worker pool (see avatarPreloadWorkers). Idempotent: repeated
// calls for the same userID short-circuit via the inflight set. If the
// worker queue is full, the job is dropped AND the inflight slot is
// released so a later retry can re-enqueue (otherwise a dropped userID
// would be stuck "in flight" with no work pending and its avatar would
// never appear). avatarURL of subsequent calls for the same userID is
// ignored — the first call wins.
func (c *Cache) Preload(userID, avatarURL string) {
	c.PreloadSized(userID, avatarURL, DefaultSize())
}

// PreloadSized is Preload for a non-default cell footprint (sidebar
// 2×1, etc.). Default-size calls share the same cache entries as
// Preload so messages-pane Get(userID) still hits.
func (c *Cache) PreloadSized(id, avatarURL string, sz Size) {
	if avatarURL == "" {
		return
	}
	sz = normalizeSize(sz)
	if c.GetSized(id, sz) != "" {
		return
	}
	key := renderMapKey(id, sz)
	if v, ok := c.failed.Load(key); ok {
		fs := v.(failState)
		if fs.url == avatarURL && c.now().Before(fs.until) {
			return
		}
		c.failed.Delete(key)
	}
	if _, loaded := c.inflight.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	if c.preloadCh == nil {
		go c.preloadInner(id, avatarURL, sz)
		return
	}
	select {
	case c.preloadCh <- preloadJob{id: id, avatarURL: avatarURL, size: sz}:
	default:
		c.inflight.Delete(key)
	}
}

func (c *Cache) now() time.Time {
	if c.nowFn != nil {
		return c.nowFn()
	}
	return time.Now()
}

func (c *Cache) backoff(tries int) time.Duration {
	if c.backoffFn != nil {
		return c.backoffFn(tries)
	}
	return defaultFailBackoff(tries)
}

func (c *Cache) noteFailure(key, url string) {
	tries := 1
	if v, ok := c.failed.Load(key); ok {
		tries = v.(failState).tries + 1
	}
	c.failed.Store(key, failState{
		until: c.now().Add(c.backoff(tries)),
		tries: tries,
		url:   url,
	})
}

// PreloadSync downloads and renders synchronously. Unlike Preload, this
// does NOT participate in the inflight dedup set — it's the worker
// entry point and tests' deterministic path. Callers that want dedup
// should use Preload.
func (c *Cache) PreloadSync(userID, avatarURL string) {
	c.preloadInner(userID, avatarURL, DefaultSize())
}

// PreloadSyncSized is PreloadSync for a non-default cell footprint.
func (c *Cache) PreloadSyncSized(id, avatarURL string, sz Size) {
	c.preloadInner(id, avatarURL, sz)
}

func (c *Cache) preloadInner(id, avatarURL string, sz Size) {
	if avatarURL == "" {
		return
	}
	sz = normalizeSize(sz)
	// Source size differs by protocol:
	//   - half-block: (Cols, Rows*2) gives the renderer exactly the
	//     pixel grid it samples. Default 4×2 matches the original
	//     pre-kitty pipeline byte-for-byte (parity test relies on this).
	//   - kitty: skip the fetcher's downscale (Target = zero point) so
	//     the renderer's own pixel-target resize starts from the highest
	//     available source resolution. With a 32×32 source (Slack's
	//     image_32) and kitty's internal target of ~32×32, this is
	//     effectively identity scaling — sharp pixels.
	target := image.Pt(sz.Cols, sz.Rows*2)
	if c.useKitty {
		target = image.Point{}
	}
	res, err := c.fetcher.Fetch(context.Background(), imgpkg.FetchRequest{
		Key:    fetchKey(id, sz),
		URL:    avatarURL,
		Target: target,
	})
	mapKey := renderMapKey(id, sz)
	if err != nil {
		c.noteFailure(mapKey, avatarURL)
		c.inflight.Delete(mapKey)
		return
	}
	rendered := c.renderAvatar(id, res.Img, sz)
	c.mu.Lock()
	c.renders[mapKey] = rendered
	c.mu.Unlock()
	c.failed.Delete(mapKey)
	c.inflight.Delete(mapKey)
	if c.onReady != nil {
		c.onReady(id)
	}
}

// Get returns the rendered default-size (4×2) avatar, or empty if not cached.
func (c *Cache) Get(userID string) string {
	return c.GetSized(userID, DefaultSize())
}

// GetSized returns a previously rendered footprint, or empty on miss.
func (c *Cache) GetSized(id string, sz Size) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.renders[renderMapKey(id, sz)]
}

// renderAvatar produces the avatar's rendered string for the active
// protocol. Kitty path: SetSource + RenderKey, immediately drain the
// upload escape to the kitty side channel (the registry's fresh-flag
// guarantees this fires only once per key), and return the
// placeholder cells. Half-block path: encode and return.
func (c *Cache) renderAvatar(id string, img image.Image, sz Size) string {
	sz = normalizeSize(sz)
	target := image.Pt(sz.Cols, sz.Rows)
	if c.useKitty {
		key := kittyKey(id, sz)
		c.kitty.SetSource(key, img)
		out := c.kitty.RenderKey(key, target)
		if out.OnFlush != nil {
			_ = out.OnFlush(imgpkg.KittyOutput)
		}
		return joinLines(out.Lines)
	}
	out := imgpkg.HalfBlockRenderer{}.Render(img, target)
	return joinLines(out.Lines)
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
