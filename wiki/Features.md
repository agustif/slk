# Features

## Messaging

- Real-time messages, edits, deletes, reactions, and typing indicators over WebSocket
- Edit your own messages (`E`) — reuses the compose box with stash/restore for any in-progress draft
- Delete your own messages (`D`) — centered confirmation overlay with message preview
- Slack markdown rendering (bold, italic, strikethrough, code, blockquotes, links, mentions)
- Emoji shortcodes (`:rocket:` → 🚀)
- Day separators (Today, Yesterday, Monday, full date)
- Infinite scroll backfill into SQLite cache
- Search: vim-style in-channel search (`/`, `n`/`N`) over cached history, plus server-side workspace search (`Ctrl+f`) with Messages/Files tabs (`Tab`), `from:` / `in:` / `before:` modifiers, and Enter on **Load more** to fetch the next page
- New-message landmark (red `── new ──` line at the unread boundary)
- Mark-as-read synced to Slack on channel entry
- Mark-as-unread (`U`) — rolls the read watermark backward to the selected message; thread replies supported. Inbound `channel_marked` / `thread_marked` events from other Slack clients are reflected live.
- Pin / unpin (`P`) — toggles Slack's pin on the selected message; pinned rows show a muted 📌 marker
- Edited / threaded message indicators
- ANSI-aware wrapping and truncation (no broken color codes mid-line)
- Drag-to-copy: drag the mouse across messages to highlight them; release to copy plain text to the system clipboard via OSC 52
- Message actions menu (`x` or right-click): add reaction, reply in thread, save for later / remind me, copy permalink, pin, follow thread (thread pane), download file, open links, edit/delete own messages, mark unread, list reactions. Some terminals steal right-click; `x` always works.

## Compose

- Multi-line input, `Shift+Enter` for newlines
- In a thread, `Ctrl+Enter` sends the reply and also posts it to the channel (`thread_broadcast`)
- Inline `@mention` autocomplete (resolves to `<@UserID>` on send)
- Special mentions: `@here`, `@channel`, `@everyone`
- Bracketed paste — paste multi-line text from the system clipboard without it being interpreted as keystrokes
- Smart paste (`Ctrl+V`) — pastes a clipboard image as an attachment, or a copied file path as an attached file, or falls through to text. Multiple attachments + caption send together via Slack's V2 file-upload API. Note: use `Ctrl+V` (not your terminal's `Ctrl+Shift+V` paste shortcut) — terminal-initiated paste only delivers text, never image bytes.
- CommonMark in compose: type `**bold**`, `~~strike~~`, `[label](url)`, `- list items`, `1. numbered`, or fenced ```code blocks``` and slk converts them on send to Slack's mrkdwn + rich_text format. Already-mrkdwn syntax (`*bold*`, `_italic_`, `~strike~`) passes through unchanged. Single-asterisk emphasis (`*x*`) is preserved as literal text since it conflicts with Slack mrkdwn bold.
- In-memory per-channel and per-thread drafts: switching conversations saves the compose box (text + pending attachments) and restores it when you return; not synced to Slack
- Scheduled send: `Ctrl+g` in insert mode opens a duration overlay (20m / 1h / 2h / 4h / 8h / tomorrow 9am / custom minutes). `:schedule 20m` / `:schedule 1h` (or `:schedule` with no args to pick) does the same from command mode. Confirm queues the draft with Slack `chat.scheduleMessage` and clears compose; a toast shows the local post time (e.g. `Scheduled for 3:04 PM`). v1 does not list or delete scheduled messages. `Ctrl+Enter` is left unbound.

## Images

- Inline image attachments render automatically in the messages pane: kitty graphics protocol on capable terminals (kitty, ghostty, recent WezTerm), sixel on foot/mlterm and on any terminal that advertises sixel in its DA1 reply (xterm with sixel support, DomTerm, toyterm, …), half-block (`▀`) fallback everywhere else
- Link unfurls render their preview images inline through the same pipeline (`image_url`, `thumb_url` if no `image_url`, and nested Block Kit image blocks), capped by `max_image_rows`
- User avatars use the same kitty graphics path on capable terminals for sharper pixels; sixel and other terminals fall back to half-block
- Click any inline image (or press `O` on the selected message) for a full-screen in-app preview
- `Enter` from the preview launches the OS image viewer
- Lazy-loaded: images download only as they scroll into view
- LRU cache at `~/.cache/slk/images/` (default 200 MB cap)
- Inside tmux, slk falls back to half-block to avoid pixel-protocol pass-through pitfalls
- Configurable via `[appearance] image_protocol` (`auto` / `kitty` / `sixel` / `halfblock` / `off`) and `max_image_rows`

See [[Terminal Compatibility|Terminal-Compatibility]] for which protocol your terminal supports.

## Threads

- Side panel (35% width), opened with `Enter`, toggled with `Ctrl+]`
- Follow / unfollow the open thread (`t` with the thread panel focused); the Threads view is the followed set
- Live thread reply routing, real-time updates
- Auto-closes on channel switch or narrow terminals
- **Threads view** (`⚑ Threads` at top of sidebar): scrollable list of
  threads you follow (Slack `subscriptions.thread`). Unread first, then
  newest activity. Cards show the parent author's avatar. Selecting a
  thread opens it in the side panel; the list re-ranks live as new
  replies arrive. Follow with `t` in the thread panel; unfollow
  removes the row.

## Activity

- **Activity inbox** (`◎ Activity` above Threads in the sidebar): Slack's
  Activity tab — recents and notifications, read and unread — via
  `activity.feed`. The sidebar badge is `client.counts` `activity_v2`.
- Tabs come from `activity.views`, so built-in All / DMs / Mentions /
  Threads plus **any custom view you created in Slack** (Unreads,
  Reactions, VIP, …) show up automatically. Selecting a tab flattens
  that view's `entry_types` / `unread_only` / `priority_only` onto
  `activity.feed`, matching the official client.
- Cards follow Slack's copy: "Post in #channel", "Thread in #channel",
  "Direct Message", "Mentioned you", plus reaction cards that show the
  real emoji (`Alice · reacted 👀 in #eng`) and a one-line parent quote
  from the local cache (empty, not a spinner, on a cache miss). Detailed
  cards show the actor's 4×2 avatar (same kitty / half-block pipeline as
  the message pane).
- Click the event emoji to toggle your reaction; click the rest of the
  card to open the message. `r` (and right-click, when the terminal
  reports it — Apple Terminal may swallow right-click) opens the
  existing reaction picker on the selected item.
- Sort is newest (`chrono_v1`) or unreads-first
  (`priority_reads_and_unreads_v1` + `vip_unreads_first`). The unread
  chip is Slack's Unreads button on the current tab.
- Defaults live in `[activity]` in `config.toml`. In the view: `f`/`F`
  cycle tabs, `s` cycles sort, `u` toggles unread-only (session-only;
  config is the next-launch default). Click the tabs/chips too.
- `Enter` on a row opens the message (or thread) through the same
  in-app permalink path search results use.

## Later

- **Later** (`◷ Later` in the sidebar, below Activity and above Threads):
  Slack's Save for later / Remind me list, synced via `saved.list`.
  The sidebar badge is `client.counts` `saved.uncompleted_count`
  (incomplete items). `Enter` on a row opens the message through the
  same in-app permalink path Activity uses.
- `w` on a selected message (or thread reply) toggles save-for-later
  (`saved.add` / `saved.delete`). `W` opens a duration menu (same
  intervals as snooze) and sets a reminder with `reminders.add` plus
  `saved.update` `date_due`. `:remind 20m` does the same from command
  mode.

## Reactions

- Search-first picker overlay (`r`) with frecent emoji — also from an
  Activity card via `r` or right-click
- Quick-toggle nav across existing pills (`R`, then `h/l/Enter`)
- Pill-style display (green = yours, gray = others)
- Optimistic UI, deduped against the WebSocket echo

## Channels & Workspaces

- Three-panel layout: workspace rail, channel sidebar, message pane
- Public (`#`), private (`◆`), DM (`●`/`○` for presence), and group DM channels. 1:1 DMs show the peer's avatar (two sidebar rows when the face is cached) next to the presence glyph.
- Channel topic shown under the name in the message-pane header (omitted when empty)
- Channel header extras: bookmark titles (clickable, OSC-8) and a pin count (`📌 N`) on one row under the channel name; empty channels omit the row. Clicking a pin jumps to the most recent pinned message
- **Slack-native sidebar sections** — slk reads your sections directly from Slack and reflects them live: section names, emoji, linked-list order, and channel/DM membership are kept in sync via the same WebSocket events the official client uses. `:move` assigns the active channel to an existing section (`users.channelSections.channels.bulkUpdate`); `:section <name>` creates an empty section (`users.channelSections.create`). Rename, delete, and reorder still happen in the official client. Falls back to glob-based config sections when disabled or if the API is unavailable.
- Star / unstar a channel with `*` — adds it to Slack's Starred sidebar section (hidden when empty). Message stars are not supported.
- Collapsible sections — `Enter`/`Space` on a section header toggles it. The default Channels section starts collapsed (`▸ Channels •3` shows aggregate unreads); pinned sections and DMs start expanded
- Live unread indicators: bold + blue dot for unread channels, muted text for read ones, aggregate dot+count on collapsed section headers
- Mute / unmute a channel (`m`) — writes Slack's per-channel notification pref. Muted conversations dim in the sidebar, drop unread dots, and suppress desktop notifications (including mentions). Sidebar-focused `m` toggles the selected row; otherwise it toggles the active channel. Reconciles live via `pref_change`.
- Glob-based config sections (`[sections.*]` in `config.toml`) — used when `use_slack_sections = false` or as a fallback when Slack's API is unreachable. Channel patterns can carry an optional `":<N>"` suffix (e.g. `"eng-general:1"`) to pin order within a section; see [Configuration › Ordering channels within a section](Configuration.md#ordering-channels-within-a-section).
- Fuzzy channel finder (`Ctrl+t` / `Ctrl+p`) — auto-expands a collapsed section when you open a channel inside it; ranks 1:1 DMs above group DMs when searching by person name
- Leave the current public or private channel (`:leave`) — confirmation overlay, then the channel drops from the sidebar and slk switches to last-visited or Threads. DMs cannot be left from slk.
- **Channel members** (`I`) — overlay listing members of the active channel (filter-as-you-type, `j`/`k` to move). Presence dots appear for users already in the live presence map; `[guest]` marks `is_restricted` / `is_ultra_restricted` users. `Enter` opens a DM with the selected person (same `conversations.open` path as `Ctrl+n`). The message pane header shows the member count when it is already known.
- Workspace picker (`:ws`) and direct jump (`1`–`9`)
- All workspaces stay connected in parallel for live unread badges

## Notifications

- OS-level desktop notifications via [beeep](https://github.com/gen2brain/beeep)
- Triggers on DMs, mentions, and configurable keywords
- Suppressed when you're focused on the relevant channel
- Suppressed entirely while you're in DND/snooze
- Suppressed during configured quiet hours (`quiet_hours` in config.toml; local 24h window, overnight wrap supported)

## Status & DND

- Set self presence (Active / Away) and DND/snooze from `Ctrl+S`
- Set a custom Slack status (`:emoji:` + text, optional 30m / 1h / 4h / today / don't-clear) or clear it from the same menu
- DM sidebar rows show a muted status emoji (or truncated text) after the name when the counterpart's status is not expired
- `p` on a selected message opens a profile overlay (name, title, status, local time, presence) with a Message action that opens their DM
- Standard snooze durations (20m / 1h / 2h / 4h / 8h / 24h / until tomorrow morning) plus custom minutes
- Live status segment in the status bar with snooze countdown
- Reflects external state changes — set from the official Slack client or via your own API scripts — in real time over the WebSocket

## Connectivity

- Browser-cookie auth (`xoxc` + `d`) — works as any user, no Slack App required
- Direct connection to Slack's internal browser WebSocket protocol
- Auto-reconnect with exponential backoff (1s → 30s)
- Three-state connection indicator in the status bar

## Customization

- 59 built-in themes (including `ANSI Dark` / `ANSI Light` that inherit your terminal palette)
- Drop-in custom themes (`~/.config/slk/themes/*.toml`)
- Live theme switcher (`Ctrl+y`)
- TOML config for appearance, animations, notifications, and channel sections
- Deterministic per-user username coloring, opt-in via `colored_usernames`

See [[Configuration]] for the full `config.toml` reference and [[Keybindings]] for the key map.
