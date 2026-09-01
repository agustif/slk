# Features

Daily-driver unofficial Slack TUI. This **fork** ([agustif/slk](https://github.com/agustif/slk)) aims at official-client (OG) parity using only reverse-engineered browser APIs.

**Remaining OG holes and packaging gaps:** [[Gaps]]. **Wire protocol:** [[Protocol]].

## Home surfaces

Pinned sidebar rows, matching Slack's left rail (not the channel list):

1. **Activity** (`◎ Activity`) — recents / notifications (`activity.feed`)
2. **Later** (`◷ Later`) — save for later / remind me (`saved.*`)
3. **Threads** (`⚑ Threads`) — threads you **follow** (`subscriptions.thread`)
4. **Direct Messages** (`✉ Direct Messages`) — full DMs column
5. **Drafts** (`✎ Drafts`) — unsent compose + scheduled send
6. **Unreads** (`◉ Unreads`) — Home All Unreads
7. **Starred** (`★ Starred`) — starred **messages** (`stars.list` `type=message`)

The channel finder (`Ctrl+t` / `Ctrl+p`) pins the same destinations (type `activity`, `later`, `threads`, `dms`, `drafts`, `unreads`, `starred`). Double-click a **section header** (or `Enter` / `Space` on it) to collapse.

## Messaging

- Real-time messages, edits, deletes, reactions, and typing indicators over WebSocket
- Edit your own messages (`E`) — reuses the compose box with stash/restore for any in-progress draft
- Delete your own messages (`D`) — centered confirmation overlay with message preview
- Slack markdown rendering (bold, italic, strikethrough, code, blockquotes, links, mentions)
- Emoji shortcodes (`:rocket:` → 🚀)
- Jump to date (`J`, or `:date` / `:jump`) — land the current channel or DM on a calendar date. Optional `YYYY-MM-DD` or `YYYY-MM-DD HH:MM` (local time); no argument opens a small overlay. Fetches history around that timestamp and selects the nearest message. Toasts if there are no messages around that date or on a network error. Does not work from Activity / Later / Drafts / Unreads / Starred / Threads list.
- Day separators (Today, Yesterday, Monday, full date)
- Infinite scroll backfill into SQLite cache
- Search: vim-style in-channel search (`/`, `n`/`N`) over cached history; **Cmd+K** / `Ctrl+k` omniswitcher (channels, people, `search.inline` snippets, files); workspace search (`Ctrl+f`) with Messages/Files/People tabs (`Tab` / `Shift+Tab`), `from:` / `in:` / `before:` modifiers, Enter on **Load more** to fetch the next page of messages or files, and Enter on a person to open a DM
- New-message landmark (red `── new ──` line at the unread boundary)
- Mark-as-read synced to Slack on channel entry
- Mark-as-unread (`U`) — rolls the read watermark backward to the selected message; thread replies supported. Inbound `channel_marked` / `thread_marked` events from other Slack clients are reflected live.
- Pin / unpin (`P`) — toggles Slack's pin on the selected message; pinned rows show a muted 📌 marker
- Edited / threaded message indicators
- ANSI-aware wrapping and truncation (no broken color codes mid-line)
- Drag-to-copy: drag the mouse across messages to highlight them; release to copy plain text to the system clipboard via OSC 52
- Message actions menu (`x` or right-click): add reaction, reply in thread, save for later / remind me, copy permalink, share/forward, pin, follow thread (thread pane), download file, add to / remove from Starred files, open links, edit/delete own messages, mark unread, list reactions, star. Some terminals steal right-click; `x` always works.
- Open links in the selected message (`o`); download file attachments (`d`); list reactions (`L`)
- Share / forward (`x` → Share, or `:share`): pick a channel or DM and post the selected message's permalink so Slack unfurls it. Permalink only (no extra comment prompt). Works from the messages pane, the thread pane, Later, and Starred.
- Channel history: `Ctrl+h` back, `Alt+Right` forward (same visit stack as opening channels from the finder)
- **Cmd+K Search** (`Ctrl+k` / `Cmd+k`): official omniswitcher — edge `channels/search` + `users/search`, `search.inline` message snippets, `search.autocomplete.files`. “Search Slack for …” opens the `Ctrl+f` workspace search.
- Help overlay (`?`) lists the current key map

## Compose

- Multi-line input, `Shift+Enter` for newlines
- In a thread, `Ctrl+Enter` sends the reply and also posts it to the channel (`thread_broadcast`)
- Inline `@mention` autocomplete (resolves to `<@UserID>` on send)
- Special mentions: `@here`, `@channel`, `@everyone`
- Bracketed paste — paste multi-line text from the system clipboard without it being interpreted as keystrokes
- Smart paste (`Ctrl+V`) — pastes a clipboard image as an attachment, or a copied file path as an attached file, or falls through to text. Multiple attachments + caption send together via Slack's V2 file-upload API. Note: use `Ctrl+V` (not your terminal's `Ctrl+Shift+V` paste shortcut) — terminal-initiated paste only delivers text, never image bytes.
- CommonMark in compose: type `**bold**`, `~~strike~~`, `[label](url)`, `- list items`, `1. numbered`, or fenced ```code blocks``` and slk converts them on send to Slack's mrkdwn + rich_text format. Already-mrkdwn syntax (`*bold*`, `_italic_`, `~strike~`) passes through unchanged. Single-asterisk emphasis (`*x*`) is preserved as literal text since it conflicts with Slack mrkdwn bold.
- Per-channel and per-thread drafts: switching conversations saves the compose box (text + pending attachments) and restores it when you return. Text drafts persist across restarts (SQLite) and sync to Slack (`drafts.list` / `drafts.create` / `drafts.update` / `drafts.delete`) so the official client sees the same unsent compose. Browse them in the **Drafts** sidebar view (`✎ Drafts`).
- Scheduled send: `Ctrl+g` in insert mode opens a duration overlay (20m / 1h / 2h / 4h / 8h / tomorrow 9am / custom minutes). `:schedule 20m` / `:schedule 1h` (or `:schedule` with no args to pick) does the same from command mode. Confirm queues the draft with Slack `chat.scheduleMessage` and clears compose; a toast shows the local post time (e.g. `Scheduled for 3:04 PM`). `:scheduled` lists pending scheduled messages (`chat.scheduledMessages.list`); Enter cancels one (`chat.deleteScheduledMessage`). `Ctrl+Enter` is left unbound.

## Images

- Inline image attachments render automatically in the messages pane: kitty graphics protocol on capable terminals (kitty, ghostty, recent WezTerm), sixel on foot/mlterm and on any terminal that advertises sixel in its DA1 reply (xterm with sixel support, DomTerm, toyterm, …), half-block (`▀`) fallback everywhere else
- Link unfurls render their preview images inline through the same pipeline (`image_url`, `thumb_url` if no `image_url`, and nested Block Kit image blocks), capped by `max_image_rows`
- User avatars use the same kitty graphics path on capable terminals for sharper pixels; sixel and other terminals fall back to half-block
- Click any inline image in the messages pane or thread panel (or press `O` / `v` on the selected message) for a full-screen in-app preview
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
  Cards show the author, channel, and message text (hydrated from the
  local cache, then `conversations.history` / thread replies). Tabs
  match Slack: In progress (`saved`), Completed, and Archived. `f`/`F`
  cycle them; click the tab labels too. `c` marks complete, `z`
  archives, `u` restores to In progress (`saved.update` `state`).
  `Enter` on **load more…** fetches the next `saved.list` page. The
  sidebar badge is `client.counts` `saved.uncompleted_count`
  (incomplete items). `Enter` on a row opens the message through the
  same in-app permalink path Activity uses.
- `w` on a selected message (or thread reply) toggles save-for-later
  (`saved.add` / `saved.delete`). On a Later row, `w` unsaves from any
  tab. `W` opens a duration menu (same intervals as snooze) and sets a
  reminder with `reminders.add` plus `saved.update` `date_due`.
  `:remind 20m` does the same from command mode. `:reminders` lists
  pending Slack reminders; Enter marks one complete. `x` on a Later
  card opens complete / archive / restore / unsave.

## Direct Messages

- **Direct Messages view** (`✉ Direct Messages` under Threads): Slack's
  DMs tab. Home still shows a compact Direct Messages section (open /
  recently read, hiding 30-day stale 1:1 leftover rows and closed IMs
  with `is_open=false`). The dedicated view is a conversation column
  (no Activity/Later/Threads/Drafts/Unreads/Starred switcher rows) of **every** 1:1 DM, group
  DM, and app DM. Esc returns to Home. Each row shows the last-message
  preview and a relative date (Today / Yesterday / weekday / Jul 16).
  Sorted unread first, then recency (cache, then
  `conversations.history` limit=1). Group DMs use the first other
  participant's 2×1 avatar when Slack sent member IDs.
- `Enter` on a DM stays in this view so the full list remains beside
  the conversation. Opening a channel (finder, permalink) returns to
  Home. The channel finder also has a Direct Messages shortcut.
  `:leave` on a DM closes it (`conversations.close`) instead of
  leaving; it drops off Home and stays in this list. Opening a closed
  DM from this list reopens it (`conversations.open`) so it returns to
  Home. The DMs column has a **← Home** row at the top (Esc also
  returns to Home).

## Drafts

- **Drafts & sent** (`✎ Drafts` in the sidebar, under Direct Messages):
  unsent composer drafts (`drafts.list`, `is_active=true`) plus scheduled
  messages (`chat.scheduledMessages.list`). Tabs are Drafts / Scheduled
  (`f`/`F`, or click the labels). Cards show the destination channel and
  a one-line preview. `Enter` opens a draft in its channel (or thread)
  and restores the compose text; `D` deletes it (`drafts.delete` or
  `chat.deleteScheduledMessage`). `Enter` on **load more…** pages
  `drafts.list` via `next_ts`. The sidebar badge is `drafts.listActive`
  (`active_draft_ids`). The channel finder has a Drafts shortcut.

## Unreads

- **All Unreads** (`◉ Unreads` in the sidebar, under Drafts): Slack's Home
  Unreads view as captured from the official web client (2026-08-31). The
  pane lists every conversation `client.counts` reports `has_unreads` for,
  grouped as a channel header plus recent top-level messages since
  `last_read` via `conversations.history` (`limit=28`,
  `ignore_replies=true`, `inclusive=true`, `oldest=last_read`). Empty copy
  is "no unreads". The sidebar badge is the number of conversations with
  `HasUnread` from counts / sidebar read state.
- `j`/`k` move between headers and messages. `Enter` (or click) on a
  message opens it in the channel at that ts (same permalink jump path as
  Activity / Later). `Enter` or click on a header calls
  `conversations.mark` with the latest message ts; the header then shows
  "N message(s) marked read" and **undo**, which rolls the watermark back
  with `conversations.mark` via `MarkChannelUnread`.
- `f`/`F` cycle sort: sidebar, alphabetical, **recommended** (OG
  “scientifically” / `all_unreads_sort_order=priority`), newest, oldest.
  Changing sort POSTs `users.prefs.set` `name=all_unreads_sort_order`
  `_x_reason=prefs` and boot reapplies it. Recommended order is
  client-side (starred first, then mention channels, then other
  channels, IMs, MPIMs; each by `channels_priority` then name). `s`
  (or click the chips) cycles All / VIP / Starred / Channels / DMs and
  POSTs `all_unreads_section_filter` (`all_sections`, VIP=`priority`,
  others = sidebar section ids). VIP membership is still
  `prefs.vip_users`; Starred is `stars.list`. Workspace switch clears
  the list and reapplies that workspace’s Unreads prefs. The channel
  finder has an Unreads shortcut (`unreads`). Remaining holes: [[Gaps]].

## Starred items

- **Starred items** (`★ Starred` in the sidebar, under Unreads): messages
  starred via `stars.add` with a timestamp, listed from `stars.list`
  `type=message` (channel stars stay in the Starred *section*). Cards show
  author, channel, preview, and relative date. `Enter` (or click) opens the
  message in its channel (same permalink jump as Later / Unreads). `*` or
  the actions menu Unstar removes it (`stars.remove`). The sidebar badge is
  the number of starred messages (first `stars.list` page, `limit=1000`).
  File stars (`type=file`) are omitted — see [[Gaps]]. IM/MPIM
  conversation stars (`type=im|mpim|group`) land in the Starred *section*,
  not this inbox. The channel finder has a Starred shortcut (`starred`).
  `x` on a card: Open / Unstar / Share.

## Reactions

- Search-first picker overlay (`r`) with frecent emoji — also from an
  Activity card via `r` or right-click
- Quick-toggle nav across existing pills (`R`, then `h/l/Enter`)
- Pill-style display (green = yours, gray = others)
- Optimistic UI, deduped against the WebSocket echo

## Channels & Workspaces

- Three-panel layout: workspace rail, channel sidebar, message pane
- Public (`#`), private (`◆`), DM (`●`/`○` for presence), and group DMs. Home splits 1:1 DMs and group DMs into two sections (both recency-sorted); `[sidebar] group_dms = "together"` puts them in one Direct Messages section like OG Slack. Custom / Starred placements still win. 1:1 DMs show a compact 2×1 peer avatar on the same row as the name; group DMs use the first other member's face when Slack sent member IDs. The workspace rail paints each team's Slack logo (4×2, initials until the fetch lands); the connecting overlay paints the cached logo on the first frame after a previous session.
- Channel topic shown under the name in the message-pane header (omitted when empty)
- Channel header extras: bookmark titles (clickable, OSC-8) and a pin count (`📌 N`) on one row under the channel name; empty channels omit the row. Clicking `📌 N` opens the pin list (`:pins`); a single pin jumps to it. Enter on a pin jumps in-app when it has a timestamp, otherwise opens the permalink.
- **Slack-native sidebar sections** — slk reads your sections directly from Slack and reflects them live: section names, emoji, linked-list order, and channel/DM membership are kept in sync via the same WebSocket events the official client uses. `:move` assigns the active channel to an existing section (`users.channelSections.channels.bulkUpdate`); `:section <name>` creates an empty section (`users.channelSections.create`); `:rename <name>` / `:section-delete` write `users.channelSections.update` / `.delete`; `:section-up` / `:section-down` retarget each section's `next_channel_section_id`. Falls back to glob-based config sections when disabled or if the API is unavailable. Within a section, `[sidebar.sort]` atom pipelines compose (`vip_first` + `recent`, `alphabetical`, …); see [Configuration](Configuration.md#sidebar-sort-atoms).
- Star / unstar a channel with `*` — adds it to Slack's Starred sidebar **section** (hidden when empty). Star a **message** from the actions menu (`x`) — `stars.add` with timestamp; starred rows show a muted ★ marker. The **Starred items** inbox (`★ Starred`) lists those messages; `*` there unstars. Files-rail Starred **write** is in the message menu; the files inbox is omitted ([[Gaps]]).
- Collapsible sections — `Enter`/`Space` on a section header toggles it, as does double-clicking the header (two clicks within ~500ms; terminals don't report a native double-click). The default Channels section starts collapsed (`▸ Channels •3` shows aggregate unreads); pinned sections and DMs start expanded
- Live unread indicators: bold + blue dot for unread channels, muted text for read ones, aggregate dot+count on collapsed section headers
- Mute / unmute a channel (`m`) — writes Slack's per-channel notification pref (`users.prefs.setNotifications` `name=muted`). Muted conversations dim in the sidebar, drop unread dots, and suppress desktop notifications (including mentions). Sidebar-focused `m` toggles the selected row; otherwise it toggles the active channel. Reconciles live via `pref_change`.
- Mentions-only / all new posts (`:notify mentions` / `:notify all`) — writes the official client's multi-pref form (`desktop=mentions_dms` or `desktop=everything`, `_x_reason=prefs-store/setMultiChannelNotificationOverride`).
- Glob-based config sections (`[sections.*]` in `config.toml`) — used when `use_slack_sections = false` or as a fallback when Slack's API is unreachable. Channel patterns can carry an optional `":<N>"` suffix (e.g. `"eng-general:1"`) to pin order within a section; see [Configuration › Ordering channels within a section](Configuration.md#ordering-channels-within-a-section).
- Fuzzy channel finder (`Ctrl+t` / `Ctrl+p`) — auto-expands a collapsed section when you open a channel inside it; ranks 1:1 DMs above group DMs when searching by person name. Unjoined public channels can be joined from the finder (`conversations.join`). Synthetic Home rows (Activity, Later, Threads, Direct Messages, Drafts, Unreads, Starred) sit at the top of an empty query.
- Cmd+K Search (`Ctrl+k`) — same overlay titled **Search**; as you type, mixes server channel hits, people, up to 3 `search.inline` messages, and file suggestions.
- New message (`Ctrl+n`) — pick people and open a 1:1 or group DM (`conversations.open`)
- Window splits (`Ctrl+w s` / `:sp`, `Ctrl+w v` / `:vsp`) — extra message panes; `Ctrl+w h/j/k/l` moves focus, `Ctrl+w w` cycles, `Ctrl+w q` / `:q` closes, `Ctrl+w o` / `:only` keeps one
- Sidebar width `[` / `]`
- Leave the current public or private channel (`:leave`) — confirmation overlay, then the channel drops from the sidebar and slk switches to last-visited or Threads. On a DM, `:leave` closes the conversation (`conversations.close`).
- Create a public channel (`:create <name>`) or a private one (`:create private <name>`) — `conversations.create` (`validate_name=true`, `team_id`; private adds `is_private=true`; no `_x_reason`).
- Invite by email (`:invite email [email…]`) — workspace invite (`users.admin.inviteBulk`). Invite existing members (`:invite U…`) — `conversations.invite`. Remove a member (`:kick U…`) — `conversations.kick` (confirm). Make Channel Manager (`:manager U…`) — `admin.roles.addMembers` `role_id=Rl0A` (confirm).
- Add / remove a file in Files-rail Starred (`x` → **Add to Starred files** / **Remove from Starred files**) — `files.favorites.add` / `.remove` (`file_id`, `collection_id` of `type=starred`). Not the Starred **messages** inbox.
- OG Home recents — opening a channel POSTs `users.prefs.set` `name=recents` so the official client’s recents match slk.
- **Channel members** (`I`) — overlay listing members of the active channel (filter-as-you-type, `j`/`k` to move). Presence dots appear for users already in the live presence map; `[guest]` marks `is_restricted` / `is_ultra_restricted` users. `Enter` opens a DM with the selected person (same `conversations.open` path as `Ctrl+n`). The message pane header shows the member count when it is already known.
- Workspace picker (`:ws`) and direct jump (`1`–`9`)
- All workspaces stay connected in parallel for live unread badges

## Notifications

- OS-level desktop notifications via [beeep](https://github.com/gen2brain/beeep)
- Triggers on DMs, mentions, and configurable keywords
- Suppressed when you're focused on the relevant channel
- Suppressed entirely while you're in DND/snooze
- Suppressed during configured quiet hours (`quiet_hours` in config.toml; local 24h window, overnight wrap supported)
- Per-channel mentions-only / all new posts via `:notify`; mute via `m`

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

See [[Configuration]] for the full `config.toml` reference, [[Keybindings]] for the key map, and [[Gaps]] for what this fork does not ship.
