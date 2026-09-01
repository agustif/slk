# slk

> **A blazingly fast Slack TUI.**
> Keyboard-driven, beautifully themed, and under 20MB. One static binary. No Electron required.

**This is [agustif/slk](https://github.com/agustif/slk), a fork of [gammons/slk](https://github.com/gammons/slk).** It is a daily-driver aimed at official-client (OG) Slack parity using only reverse-engineered browser APIs — no invented endpoints, form fields, sorts, or prefs.

Upstream marketing site: [getslk.sh](https://getslk.sh) · This fork’s docs: [wiki/](wiki/)

![slk screenshot](docs/assets/screenshot.png)

`slk` is a daily-driver replacement for the official Slack desktop client, built in Go with [bubbletea](https://github.com/charmbracelet/bubbletea) and [lipgloss](https://github.com/charmbracelet/lipgloss).

## Why slk?

- **Fast.** Cold start in milliseconds. Render-cached messages. SQLite-backed scrollback. Real-time over WebSocket.
- **Tiny.** ~19 MB on disk. ~60 MB RSS for a live multi-workspace session vs. 500 MB–1.5 GB for the official client. No node_modules, no Chromium, no 1Gb RAM tax.
- **Keyboard-first.** Vim-style modal editing. `j/k`, `h/l`, `i`, `Esc`.
- **Pretty.** 59 built-in themes, lipgloss-styled panels, true-pixel avatars on kitty (half-block fallback elsewhere), emoji shortcodes, day separators, and pill-style reactions.
- **Multi-workspace.** All your workspaces stay connected in parallel. `1`–`9` to instantly jump between them, with live unread badges in the rail.
- **Yours.** TOML config, custom themes, custom channel sections via glob, XDG-compliant paths.

## Highlights

- Real-time messages, edits, deletes, reactions, typing indicators
- Inline images (kitty graphics / sixel / half-block fallback) with full-screen preview; link-unfurl images use the same pipeline
- Home surfaces matching Slack: **Activity**, **Later**, **Threads**, **Direct Messages**, **Drafts**, **Unreads**, **Starred**
- Threads side panel + a followed-threads view (`subscriptions.thread`)
- Jump to date (`J` / `:date` / `:jump`), share/forward (`x` → Share or `:share`)
- Workspace search (`Ctrl+f`) with Messages / Files / People tabs
- Message actions menu (`x` / right-click): react, reply, save for later, remind, permalink, share, pin, follow, download, edit/delete, mark unread, star
- Per-channel drafts synced to Slack (`drafts.list` / create / update / delete) plus scheduled send (`Ctrl+g`)
- Slack-native sidebar sections, **writable** (`:move` / `:section` / `:rename` / `:section-delete`); or glob-based config sections
- Automatic auth from the Slack desktop app — no tokens to copy, no Slack App required
- Vim-style modal keybindings, fuzzy channel finder, workspace picker
- 59 themes + drop-in custom themes, live theme switcher
- OS desktop notifications on DMs, mentions, and configurable keywords (quiet hours + mute honored)

Full feature breakdown: **[Features](wiki/Features.md)** · every key: **[Keybindings](wiki/Keybindings.md)**

## Fork differences vs [gammons/slk](https://github.com/gammons/slk)

Compared to upstream `main`, this fork adds the OG-parity work below. Upstream already has the TUI core: real-time messaging, threads panel, in-channel + workspace search, smart paste, Slack-native **read-only** sections, themes, desktop-app auth, notifications, and DND.

Everything here was captured from the official client (HAR / browser protocol). Uncaptured surfaces are **not** invented — they are omitted. Inventory: [wiki/Gaps.md](wiki/Gaps.md).

### Home surfaces

| Surface | Slack | How |
|---|---|---|
| **Activity** (`◎ Activity`) | Activity tab | `activity.feed` / `activity.views`. Built-in All / DMs / Mentions / Threads plus **your custom views** (Unreads, Reactions, VIP, …). `f`/`F` tabs, `s` sort, `u` unread-only. Reaction cards toggle on click. |
| **Later** (`◷ Later`) | Save for later | `saved.list` / `saved.add` / `saved.update`. Tabs: In progress / Completed / Archived. `w` save, `W` / `:remind` remind-me (`reminders.add`). |
| **Threads** | Followed threads | `subscriptions.thread` (followed set), not upstream’s cache-only “threads you participated in”. `t` follow/unfollow. |
| **Direct Messages** (`✉ Direct Messages`) | DMs tab | Full 1:1 + group + app DM list with last-message preview. Home stays compact. Esc / ← Home returns. `:leave` **closes** a DM (`conversations.close`). |
| **Drafts** (`✎ Drafts`) | Drafts & sent | `drafts.list` (`is_active`, `next_ts`) + `chat.scheduledMessages.list`. Open restores compose; `D` deletes / cancels. |
| **Unreads** (`◉ Unreads`) | Home All Unreads | `conversations.history` from `last_read` (`limit=28`, `ignore_replies=true`). Header Mark as Read / Undo via `conversations.mark`. Session-local sort only (`f`/`F`). |
| **Starred** (`★ Starred`) | Starred items | `stars.list` `type=message`. Enter opens the message; `*` unstars. Channel stars stay in the Starred sidebar *section*. |

### Messaging

- **Jump to date** — `J`, `:date`, `:jump` (`YYYY-MM-DD` or `YYYY-MM-DD HH:MM`; no arg opens an overlay). History around that ts, nearest message.
- **Message actions menu** — `x` or right-click.
- **Share / forward** — menu Share or `:share`; posts the permalink so Slack unfurls it (no extra comment).
- **Pin / unpin** — `P`; header `📌 N` and `:pins` to list / jump.
- **Star** — `*` stars a **channel** (Slack Starred sidebar section). Actions menu stars a **message** (`stars.add` with timestamp). **Starred items** inbox (`★ Starred`) lists those messages from `stars.list`.
- **Yank** — `yy` copies the selected message text.
- **Workspace search** — `Ctrl+f` adds **Files** and **People** tabs (`Tab` / `Shift+Tab`), pagination on messages/files, Enter on a person opens a DM (`edge.UsersSearch`).
- **Link unfurls** — preview images render through the same image pipeline as attachments.

### Compose

- **Drafts** — switching conversations saves compose (text + pending attachments). Text persists in SQLite and syncs to Slack so the official client sees the same unsent box.
- **Scheduled send** — `Ctrl+g` / `:schedule` / `:scheduled` via `chat.scheduleMessage`.
- **Also send to channel** — in a thread, `Ctrl+Enter` broadcasts the reply (`thread_broadcast`).

### Sidebar, channels, header

- **Writable Slack sections** — `:move`, `:section <name>`, `:rename`, `:section-delete`, `:section-up` / `:section-down` (`users.channelSections.*`). Upstream sections are read-only.
- **Mute** — `m` writes Slack’s per-channel notification pref. Muted rows dim, drop unread dots, and suppress desktop notifications (including mentions).
- **Collapse** — `Enter` / `Space` on a section header, or **double-click** the header (two left-clicks within ~500ms).
- **Channel topic** under the name; **bookmarks** (clickable) and pin count on the header row.
- **Channel members** — `I` overlay; Enter opens a DM.
- **Leave** — `:leave` leaves a channel or closes a DM.
- **DM presentation** — Home can split 1:1 vs group DMs (`[sidebar] group_dms = "split"` default) or keep one Direct Messages section (`"together"`). Compact peer avatars; workspace-rail team logos.
- **Sort atoms** — `[sidebar.sort]` pipelines (`slack`, `alphabetical`, `recent`, `vip_first`, `unread_first`, `starred_first`) plus `[sidebar.vip]` extras on top of Slack VIP people.

### Status, profile, notifications

- **Custom status** from `Ctrl+S` (`:emoji:` + text, optional clear-after).
- **Profile overlay** — `p` on a message (name, title, status, local time, presence; Message opens a DM).
- DM rows show a muted status emoji (or truncated text) when the counterpart’s status is live.
- **Quiet hours** — `[notifications] quiet_hours` (`"22:00-08:00"`, overnight wrap).

### Not in this fork

Full table (capture status, why omitted): **[Remaining gaps](wiki/Gaps.md)**.

Permanent non-goals (same as upstream): huddles, Slack Connect, Workflow Builder, Canvas, Lists, Slack AI, slash commands, custom emoji management, animated reactions.

OG holes still omitted until a HAR exists — **not invented**:

- Create channel / invite members (Join / Leave already exist)
- Mentions-only notification prefs (`users.prefs.setNotifications` is captured for `muted` only)
- Starred **file** / `type=im` items (`stars.list`; message inbox ships)
- Unreads “recommended / scientifically” sort, section-filter chips, persisted sort pref
- `stars.list` past the first page (`limit=1000`)

Packaging: no GitHub Release / tag yet; AUR `slk` is upstream; in-tree `packaging/aur` is unpublished `slk-git`.

See [Tradeoffs and Non-Goals](wiki/Tradeoffs-and-Non-Goals.md).

## Quick install

These commands install **this fork** (`agustif/slk`). There is **no GitHub Release or semver tag yet** — use `@main` or Homebrew `--HEAD`. `@latest` is not a fork version until a tag exists.

**Homebrew** (macOS and Linux) — tap is [agustif/homebrew-tap](https://github.com/agustif/homebrew-tap), not `gammons/tap`. Uninstall the upstream cask first if you have it (`brew uninstall --cask slk`):

```bash
brew install --HEAD agustif/tap/slk
```

**Arch** — AUR package `slk` is upstream. This fork’s `slk-git` PKGBUILD is in-tree and **not published** to the AUR:

```bash
git clone https://github.com/agustif/slk.git
cd slk/packaging/aur
makepkg -si
```

**Go:**

```bash
go install -ldflags="-s -w" -trimpath github.com/agustif/slk/cmd/slk@main
```

**From source:**

```bash
git clone https://github.com/agustif/slk.git
cd slk
go build -ldflags="-s -w" -trimpath -o slk ./cmd/slk
mv -f slk ~/.local/bin/slk
```

Details, Wayland/X11 paste deps, and Windows: [Installation](wiki/Installation.md).

## Setup

slk reads your session directly from the **Slack desktop app** — no DevTools,
no tokens to copy. Make sure you're signed in to the desktop app, then:

```bash
slk --add-workspace
```

slk lists the workspaces you're signed in to; pick the ones you want and
you're done.

Full walkthrough: [Setup](wiki/Setup.md).

## Debugging

Set `SLK_DEBUG=1` to enable a comprehensive debug log written to
`slk-debug.log` in the current working directory. The file is
**truncated each run**, so reproduce the issue, quit slk, then copy
the file before relaunching. Log lines are categorized
(`[cache]`, `[imgfetch]`, `[imgrender]`, `[ws]`, `[general]`) so
`grep '\[cache\]' slk-debug.log` slices to one focus area.

## Documentation

In-tree wiki (this fork):

- [Installation](wiki/Installation.md) — Homebrew `--HEAD`, Go `@main`, in-tree AUR PKGBUILD, source
- [Setup](wiki/Setup.md) — desktop-app auth, adding workspaces
- [Features](wiki/Features.md) — full feature breakdown
- [Keybindings](wiki/Keybindings.md) — every key, every mode
- [Configuration](wiki/Configuration.md) — `config.toml`, sidebar sort atoms, activity, XDG paths
- [Gaps](wiki/Gaps.md) — remaining OG holes, capture status, packaging
- [Protocol](wiki/Protocol.md) — unofficial browser protocol (envelope, `_x_reason`, methods)
- [Terminal Compatibility](wiki/Terminal-Compatibility.md) — what each terminal supports, including tmux
- [Clipboard and OSC 52](wiki/Clipboard-and-OSC-52.md) — copy/paste setup notes
- [Tradeoffs and Non-Goals](wiki/Tradeoffs-and-Non-Goals.md) — roadmap, caveats, TOS notice
- [Architecture](wiki/Architecture.md) — service layout, data layer

Upstream wiki (gammons/slk, without fork features): [github.com/gammons/slk/wiki](https://github.com/gammons/slk/wiki).

## Contributing

This fork tracks official-client parity from captures. Bug fixes and captured-API features are welcome here.

- **Do not invent Slack APIs.** New network calls need a HAR / official-client capture.
- Understand the diff, make sure it builds and passes `go vet ./...` and `go test ./...`.
- Large changes: open an issue first.

Contributions aimed at upstream slk should go to [gammons/slk](https://github.com/gammons/slk).

## Disclaimer

`slk` is an independent, unofficial project. It is not affiliated with, endorsed by, or sponsored by Slack Technologies, LLC or Salesforce, Inc. "Slack" is a trademark of Slack Technologies, LLC; it is used here only to describe the service this client interoperates with.

slk talks to Slack via the same internal browser protocol the official web client uses. This is unofficial and not sanctioned by Slack — see [Tradeoffs and Non-Goals](wiki/Tradeoffs-and-Non-Goals.md#unofficial--tos-caveat) for details.

## License

[MIT](LICENSE) © Grant Ammons
