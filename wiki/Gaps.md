# Remaining gaps

This fork only ships official-client (OG) Slack surfaces that were reverse-engineered from the browser protocol (HAR, form fields, `_x_reason`, response shape). A public Web API name is not enough. **No invented endpoints, prefs, sorts, or item types.**

Canonical feature list: [[Features]]. Key map: [[Keybindings]]. What we know of the wire: [[Protocol]]. Permanent non-goals and caveats: [[Tradeoffs and Non-Goals|Tradeoffs-and-Non-Goals]].

Last reviewed: 2026-09-01.

## Shipped OG Home (not gaps)

Sidebar switcher, in this order:

| Row | Slack | Capture |
|---|---|---|
| `◎ Activity` | Activity tab | `activity.feed` / `activity.views` |
| `◷ Later` | Save for later | `saved.list` / `saved.add` / `saved.update` |
| `⚑ Threads` | Followed threads | `subscriptions.thread` |
| `✉ Direct Messages` | DMs tab | cache + `conversations.history` limit=1; `:leave` → `conversations.close` |
| `✎ Drafts` | Drafts & sent | `drafts.list` + `chat.scheduledMessages.list` |
| `◉ Unreads` | Home All Unreads | `conversations.history` from `last_read` (`limit=28`, `ignore_replies=true`). Sort pref `all_unreads_sort_order` including scientifically=`priority`. |
| `★ Starred` | Starred **messages** | `stars.list` `type=message` |

Channel / IM / MPIM stars (`stars.list` `type=channel|im|mpim|group`) are the sidebar **Starred section**, not this inbox. Join, Leave, public/private create, email invite, and existing-member invite/kick exist. Mute writes `name=muted`. Mentions-only / all-new-posts write the multi-pref form. Channel views also POST OG Home recents (`users.prefs.set` `name=recents`).

## Product gaps vs official Slack

OG Home holes that needed a live form are closed (Starred files list, `stars.list` paging). Do not invent APIs. Remaining: recents `object_type` for DMs (uncaptured), plus packaging below.

### Not in the table on purpose

- **Join** a public channel from the finder: shipped (`conversations.join`).
- **Create** a public or private channel (`:create <name>` / `:create private <name>`): shipped (`conversations.create`; private adds `is_private=true`).
- **Invite by email** (`:invite email…`): shipped (`users.admin.inviteBulk`).
- **Invite existing members** (`:invite U…`): shipped (`conversations.invite` `force=true`, empty `subteams`, `_x_reason=submit-invite-channel-invite-modal`).
- **Kick** (`:kick U…`): shipped (`conversations.kick`, `_x_reason=submitKickFromChannel`).
- **Channel Manager** (`:manager U…`): shipped (`admin.roles.addMembers` `role_id=Rl0A`). Official picker still says “No matches” for Workspace Tester; slk posts the captured form for a `U…` id. Success extra JSON keys uncaptured (`parseSlackAPIAck` is `ok`).
- **OG recents**: shipped write (`users.prefs.set` `name=recents` on channel view; `object_type=CHANNEL` for `C…` ids). DM recents type not captured.
- **Mentions-only / all new posts** (`:notify mentions` / `:notify all`): shipped (`users.prefs.setNotifications` multi-pref `desktop=mentions_dms` / `desktop=everything`, `_x_reason=prefs-store/setMultiChannelNotificationOverride`). Mute (`m`) is still `name=muted`.
- **Star / unstar a channel** (`*`): shipped (`stars.add` / `stars.remove` without timestamp).
- **Star / unstar a message** (`x` menu, or `*` in the Starred inbox): shipped (`stars.add` / `stars.remove` with timestamp).
- **Add / remove a file in Files-rail Starred** (`x` → Add to Starred files / Remove from Starred files): shipped write (`files.favorites.add` / `.remove`).
- **Starred files inbox**: shipped. `files.favorites.list` `type=all` `_x_reason=starred_unified_files`; rows from `file_ids` (JS reverse). Official client skips this call when `custom_file_sections=on`; live list can still be `file_ids:[]` after add.
- **`stars.list` paging**: shipped. Follow `response_metadata.next_cursor`, else `page=N+1` when `paging.total` exceeds items (`limit=1000`).
- **Starred IMs / MPIMs** in the Starred *section*: shipped (`stars.list` `type=im|mpim|group` with a channel id).
- **Unreads section chips** (All / VIP / Starred / Channels / DMs): shipped. `s` / chips write `all_unreads_section_filter` (`all_sections`, VIP=`priority`, Starred/Channels/DMs = sidebar section ids). Boot-read applied. Custom sidebar sections are not extra chips.
- **Unreads recommended / scientifically sort** (`f`/`F` → recommended): shipped. Pref `all_unreads_sort_order=priority` written and applied from boot. Client-side `sortScientifically`: starred then not; channels-with-mentions, channels, IMs, MPIMs; `channels_priority` desc then name.

### Captured, not a product gap

Wire forms live in [[Protocol]]. They are not Home-inbox holes:

- **`client.dms`**: `count=250`, `priority_mode=priority` (string), `_x_reason=dms-tab-populate`, response `ims`/`mpims`. DMs tab already ships from boot cache + history; not wired.
- **`files.recentlyDeleted`**: `_x_reason=get-deleted-files`, `files:[]`. Files-rail support.

Third-party session catalogs (karbassi, rusq, slack-ruby, ErikKalkoken) are **hints**. Chinese Gitee/GitCode/CSDN searches found no xoxc method map. Do not wrap from those lists.

## Permanent non-goals

Same as upstream, still not on this fork’s roadmap:

- Huddles, Slack Connect, Workflow Builder, Canvas, Lists, Slack AI
- Slash commands, bot/app management, custom emoji **management** (picking/reacting shipped)
- Animated reactions (static first frame for GIF images)
- Interactive Block Kit buttons (rendered disabled; “open in Slack to interact”)

TUI `[keys]` overlays shipped (see [[Configuration]]).

## Fork / packaging gaps

Product code can be complete while the fork is not a drop-in install of a tagged release.

| Gap | Status |
|---|---|
| **GitHub Release / semver tag** | **v0.20.0** (first fork tag was v0.17.0). Homebrew formula in [agustif/homebrew-tap](https://github.com/agustif/homebrew-tap) pins the current tag (`brew install agustif/tap/slk`); `--HEAD` still tracks `main`. |
| **GitHub release artifacts** | Cut with GoReleaser on tag `v*` (linux/windows static, darwin cgo). `workflow_dispatch` can rebuild an existing tag. |
| **AUR** | AUR [`slk`](https://aur.archlinux.org/packages/slk) is **upstream**. This repo ships `packaging/aur` as `slk-git`; it is **not published** to the AUR. |
| **Nix flake** | In-tree `flake.nix` (`version = "0.20.0"`). Not a published nixpkgs/flakehub package. |
| **GitHub Wiki tab** | Disabled. Docs are `wiki/*.md` in this repo (linked from the README). |

## How a gap gets filled

See [[Protocol#how-a-new-method-gets-in]]. Short version: HAR the official web client, wrap only that shape, no live mutating probes. A public or third-party method name is not a capture.
