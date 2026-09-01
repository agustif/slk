# Configuration

Config lives at `~/.config/slk/config.toml`.

## Full example

```toml
[general]
default_workspace = "work"      # the slug, not the team ID
use_slack_sections = true       # use real Slack sidebar sections (default).
                                # set false to use [sections.*] globs instead.

# Overlay DefaultKeyMap. Names are snake_case KeyMap fields
# (toggle_star, jump_to_date, help, fuzzy_finder). A string or array.
# [keys]
# toggle_star = "s"
# help = ["?", "f1"]

[appearance]
theme = "dracula"
timestamp_format = "3:04 PM"
image_protocol = "auto"   # auto | kitty | sixel | halfblock | off
max_image_rows = 20       # cap inline image height in terminal rows
colored_usernames = false # color each user's name by ID hash (Slack-style)

[animations]
enabled = true
smooth_scrolling = true
typing_indicators = true

[notifications]
enabled = true
on_mention = true
on_dm = true
on_keyword = ["deploy", "incident"]
quiet_hours = "22:00-08:00"   # 24h local; overnight wrap ok; empty = off

# notify_command (optional): run INSTEAD of the built-in OS notification for any
# message that would notify (DM / mention / keyword). Executed via `sh -c` with
# $SLK_TITLE and $SLK_BODY set, so you can route notifications through your own
# tooling (terminal-notifier, a multiplexer's notifier, mako, ...). Values are
# passed via the environment, so message text can't inject shell syntax.
# notify_command = 'terminal-notifier -title "$SLK_TITLE" -message "$SLK_BODY"'

# status_command (optional): run on every unread-state change (a message arrives
# or a channel is read) so an external surface can mirror slk's unread state.
# Because it fires on reads too, it can clear a status as well as set one.
# Runs are serialized and coalesced: states never run concurrently or out of
# order, and under a burst intermediate states may be skipped — the newest
# state always runs last, so the surface converges on the current state.
# Executed via `sh -c` with:
#   $SLK_UNREAD        unread channels in the active workspace (mute-filtered)
#   $SLK_OTHER_UNREAD  unread count across other workspaces
#   $SLK_WORKSPACE     active workspace name
#   $SLK_TITLE         the window-title string, e.g. "slk SW (3) +1"
# status_command = 'my-statusbar --slack-unread "$SLK_UNREAD"'

# Both hooks require a POSIX `sh` on $PATH and are unavailable on Windows
# (the built-in OS notification still works there). Hook failures are silent
# in the UI; run slk with SLK_DEBUG=1 and check slk-debug.log ([notify] lines)
# to diagnose a misbehaving command.

# Muted channels and DMs never notify — including on mentions and keywords —
# matching Slack. Mute is users.prefs.setNotifications name=muted (`m`).
# Mentions-only / all new posts: :notify mentions / :notify all.

[cache]
message_retention_days = 30
max_db_size_mb = 500
max_image_cache_mb = 200

# Composable within-section sort. Each value is an ordered list of
# atoms (SQL ORDER BY). First atom partitions; later atoms sort
# inside each partition. Omitted type keys fall through to default.
#
# Atoms:
#   slack          Slack / config :N / input order (the default)
#   alphabetical   display name A–Z (case-insensitive)
#   recent         last opened, then last activity, then last_read
#   vip_first      Slack VIP people (prefs.vip_users), plus [sidebar.vip] extras
#   unread_first   unread before read
#   starred_first  Slack-starred conversations first
#
# Examples:
#   dms = ["recent"]                      — newest activity first (default)
#   dms = ["vip_first", "recent"]         — two recency lists, VIPs on top
#   dms = ["vip_first", "alphabetical"]   — two A–Z lists, VIPs on top
[sidebar.sort]
default  = ["slack"]
dms      = ["recent"]
channels = ["slack"]
starred  = ["slack"]
apps     = ["recent"]

[sidebar.sort.section]
Engineering = ["unread_first", "alphabetical"]

# Extra VIP membership on top of Slack Preferences → VIP (prefs.vip_users).
# Slack's list is the source of truth; this only adds names/IDs/globs.
#
# group_dms: "split" (default) is Direct Messages then Group DMs;
# "together" is one Direct Messages section (OG Slack).
[sidebar]
group_dms = "split"
vip = ["alice", "@cto", "exec-*"]

# Activity inbox (Slack's Activity tab). These are the session defaults;
# f / F / s / u in the Activity view cycle them live without rewriting
# this file.
[activity]
filter = "all"            # All / DMs / Mentions / Threads, or a custom
                          # activity.views name (Unreads, Reactions, VIP, …)
sort = "newest"           # newest | unreads_first
unread_only = false
density = "detailed"      # detailed | compact (Slack Detailed / Dense)
limit = 50                # activity.feed page size, 1–100

# Glob-based channel sections — only consulted when use_slack_sections
# is false (globally or per-workspace), or when Slack's section API is
# unreachable. Otherwise slk reads the user's actual Slack sections.
[sections.Alerts]
channels = ["alerts", "ops", "*-alerts"]
order = 1

# Channels can carry an optional ":<N>" suffix to pin their order
# within the section. Lower numbers sort higher. Entries without a
# suffix fall after annotated ones, in the order they appear.
# This syntax is only honored when use_slack_sections = false;
# in Slack-native mode, channel order comes from Slack.
[sections.Engineering]
channels = ["eng-general:1", "eng-alerts:2", "eng-*"]
order = 2

# Per-workspace settings: keyed by a slug you choose at --add-workspace
# time. team_id ties the slug to the underlying Slack workspace.
[workspaces.work]
team_id = "T01ABCDEF"
order   = 1                     # rail position; 1-based, used by 1-9 keys
theme   = "dracula"             # overrides [appearance].theme
use_slack_sections = false      # this workspace uses [sections.*] globs;
                                # other workspaces still use Slack sections

[workspaces.work.sections.Alerts]
channels = ["alerts", "*-alerts"]
order = 1

[workspaces.work.sections.Engineering]
channels = ["eng-*", "deploys"]
order = 2

# A second workspace with no per-workspace sections — falls back to
# the global [sections.*] above.
[workspaces.side]
team_id = "T02XYZ"
order   = 2

# Inline color overrides on top of the active theme
[theme]
primary = "#4A9EFF"
accent = "#50C878"
background = "#1A1A2E"
text = "#E0E0E0"
```

## Section resolution

When `use_slack_sections = true` (the default) and Slack's section endpoint
is reachable, slk reads the user's actual sidebar sections — names, emoji,
linked-list order, and channel membership — directly from Slack and keeps
them live via WebSocket events. Any `[sections.*]` or
`[workspaces.<slug>.sections.*]` blocks in `config.toml` are ignored in this
mode (a one-line info note is emitted to the debug log on first connect so
the shadowing isn't silent). Set `use_slack_sections = false` globally, or
per-workspace, to opt into glob-based sections instead.

Per-workspace `[workspaces.<slug>.sections.*]` blocks fully replace the
global `[sections.*]` for that workspace. Workspaces that define no
sections of their own fall back to the global table.

### Ordering channels within a section

Each entry in a section's `channels` list may carry an optional `:<N>`
suffix where `N` is a non-negative integer. Channels matched by an
annotated pattern sort ahead of channels matched by un-annotated
patterns; among annotated channels, lower `N` wins; un-annotated
channels keep the order Slack returned them in.

```toml
[sections.Engineering]
channels = ["eng-general:1", "eng-alerts:2", "eng-*"]
order = 2
```

In the example above, `#eng-general` is pinned to the top of the
Engineering section, followed by `#eng-alerts`, followed by every
other `eng-*` channel in Slack-API order.

This syntax is only honored when `use_slack_sections = false` (or
when Slack's section endpoint is unreachable and slk falls back to
glob mode) **and** that section's sort pipeline still includes the
`slack` atom (the default). A pipeline of `["alphabetical"]` ignores
`:N` and Slack's linked-list order.

## Sidebar sort atoms

`[sidebar.sort]` is a set of **composable keys**, not a single enum.
They apply **within** a section (section order in the sidebar is
unchanged). Lookup for a section, first match wins:

1. `[sidebar.sort.section]` by header name (case-insensitive), including
   Slack-renamed sections
2. Type shortcut: `dms` (`direct_messages`), `channels`, `starred`
   (`stars`), `apps` (`recent_apps`)
3. `default` (implicit `["slack"]`; Direct Messages / Group DMs fall
   through to `["recent"]` when `dms` is omitted)

`[sidebar] group_dms` is independent of sort: `split` (default) is two
Home sections, `together` is one.

| Atom | What it compares |
|---|---|
| `slack` | Config `:N` then Slack / bootstrap order |
| `alphabetical` | Display name, case-insensitive |
| `recent` | Last time you opened the conversation, then last activity, then `last_read_ts` |
| `vip_first` | Slack VIP people (`prefs.vip_users`) first, plus `[sidebar.vip]` extras |
| `unread_first` | Unread (unmuted) first |
| `starred_first` | Slack-starred first |

Aliases (`vip`, `alpha`, `recency`, `unread`, `starred`, …) are
accepted and unknown atoms are dropped on load.

`vip_first` uses Slack's own VIP list (`prefs.vip_users`, the same
Preferences → VIP contacts that drive Activity's VIP tab and the
desktop "VIP unreads" section). `[sidebar.vip]` is an **optional
additive override**: extra channel IDs, DM user IDs, display names,
`@handles`, and `*` globs on top of Slack's list. It does not replace
Slack VIPs. `starred_first` is a different atom (channel stars).

### Limitations of Slack-native sections

`:move` / `:section` write membership and create empty sections.
`:rename` / `:section-delete` use `users.channelSections.update` /
`users.channelSections.delete`. `:section-up` / `:section-down` rewrite
each section's `next_channel_section_id` (the same linked-list field
`users.channelSections.list` and `channel_section_upserted` already
carry). The `stars` section type (Slack's "Starred" feature) is rendered
when non-empty, with the header `Starred`. Sections of type `slack_connect`,
`salesforce_records`, and `agents` are hidden. Sections with more than 10 channels may be returned
only partially by Slack's API on initial load; the missing channels
temporarily fall into the catch-all bucket and migrate into their correct
section as WebSocket events fire or the workspace reconnects. A debug-log
warning identifies which sections were truncated.

## Workspace order

The `order` field controls workspace position in the rail and the mapping
for the `1`–`9` digit keys. Positive values sort ascending (lowest first);
workspaces without an `order` (or with `order = 0`) sort after explicitly
ordered ones, alphabetically by slug. Tokens on disk that have no
`[workspaces.<slug>]` block at all sort last, alphabetically by team ID.
The order is stable across runs. Previously the rail order depended on
which workspace's WebSocket connected first; it is now deterministic
regardless of network timing, even without an explicit `order` set.

Legacy configs that key the block by raw team ID
(`[workspaces.T01ABCDEF]`) keep working unchanged.

## Activity inbox

`[activity]` sets the defaults for the Activity sidebar row (Slack's
Activity tab). Unknown values clamp on load.

| Key | Values | Default | Maps to |
|---|---|---|---|
| `filter` | view name / type / id (`all`, `Unreads`, `VIP`, …) | `all` | selected `activity.views` tab |
| `sort` | `newest`, `unreads_first` | `newest` | `chrono_v1` vs `priority_reads_and_unreads_v1` + `vip_unreads_first` |
| `unread_only` | bool | `false` | `unread_only` |
| `density` | `detailed`, `compact` | `detailed` | 3-line cards vs one line |
| `limit` | 1–100 | `50` | `activity.feed` page size |

In the Activity view, `f`/`F`, `s`, and `u` cycle filter / sort /
unread-only for the rest of the session without rewriting
`config.toml`. Click the tabs and chips in the toolbar for the same
controls.

There is **no** `[unreads]` block. Unreads `f`/`F` sort is session-local
and is not persisted (pref name not captured). See [[Gaps]].

## Terminal-palette themes (`ANSI Dark`, `ANSI Light`)

Two built-in themes use ANSI 16 color codes exclusively rather than
fixed RGB values. They inherit the user's terminal color palette, so
changing your terminal colorscheme (light/dark, solarized,
accessibility palettes, etc.) immediately changes slk's UI colors to
match.

```toml
[appearance]
theme = "ANSI Dark"   # or "ANSI Light"
```

Pick the variant whose background matches your terminal's background.

**Trade-off:** selection-row highlights and compose-input tints are
still computed as RGB approximations, so the tint regions of those
elements use truecolor rather than your palette. The rest of the UI
honors the palette.

## Custom themes

Drop `.toml` files into `~/.config/slk/themes/`:

```toml
name = "My Theme"

[colors]
primary      = "#BD93F9"
accent       = "#50FA7B"
warning      = "#FFB86C"
error        = "#FF5555"
background   = "#282A36"
surface      = "#343746"
surface_dark = "#21222C"
text         = "#F8F8F2"
text_muted   = "#6272A4"
border       = "#44475A"

# Optional sidebar/rail overrides — lets you have a darker sidebar with a
# lighter message pane (Slack's default look). Fall back to
# background/text/text_muted/surface_dark when omitted.
sidebar_background = "#19171D"
sidebar_text       = "#D1D2D3"
sidebar_text_muted = "#9A9B9E"
rail_background    = "#19171D"
```

Every built-in theme now sets a channels-panel (sidebar) background that is
perceptibly distinct from the message pane. When writing a custom theme,
set `sidebar_background` to a clearly darker (or, on near-black themes, a
slightly lighter) shade than `background` for the same effect.

Switch themes live with `Ctrl+y`.

## Data paths (XDG)

| Path | Contents |
|---|---|
| `~/.config/slk/` | Configuration, custom themes |
| `~/.local/share/slk/` | SQLite cache, tokens |
| `~/.cache/slk/` | Avatars, image cache |

`[keys]` overlays DefaultKeyMap (snake_case field names). Prefs this fork
does not write (mentions-only, Unreads extra sorts / section-filter
persistence): [[Gaps]].
