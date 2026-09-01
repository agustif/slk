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
| `◉ Unreads` | Home All Unreads | `conversations.history` from `last_read` (`limit=28`, `ignore_replies=true`) |
| `★ Starred` | Starred **messages** | `stars.list` `type=message` |

Channel / IM / MPIM stars (`stars.list` `type=channel|im|mpim|group`) are the sidebar **Starred section**, not this inbox. Join (`conversations.join`) and Leave (`conversations.leave` / `conversations.close`) exist. Mute writes `users.prefs.setNotifications` `name=muted`.

## Product gaps vs official Slack

These are OG holes a daily driver still hits. They stay omitted until a capture exists. Do not HAR-mutate a live workspace to fill them unless that is an explicit ask.

| Gap | Official Slack | What this repo already knows | Why it is not built |
|---|---|---|---|
| **Create channel** | New channel UI | Public method `conversations.create`. slk wraps **join** and **leave** only. | No HAR of the official client's create form / `_x_reason` / response. No slk wrapper. |
| **Invite members** | Add people to a channel | Members overlay (`I`) is read + open DM (`conversations.open`). | No HAR of `conversations.invite` (or the current client equivalent). No slk wrapper. |
| **Mentions-only notifications** | Channel pref: all / mentions / nothing | Mute: `users.prefs.setNotifications` `name=muted`. Boot `all_notifications_prefs` includes `suppress_at_channel`. | Mentions-only **pref `name` / `value`** were not captured. Writing `muted` or guessing `suppress_at_channel` as mentions-only would invent a pref. |
| **Starred files inbox** | Starred items includes files | `stars.list` comments mention `type=file`. Inbox is `type=message` only. | File-item JSON (file id, name, permalink, …) not captured for this parse. |
| **Unreads “recommended / scientifically” sort** | Home All Unreads extra sorts | Session-local `f`/`F`: sidebar / alphabetical / newest / oldest. | Sort algorithm unknown. Not in the 2026-08-31 Unreads capture. |
| **Unreads sort / section-filter persistence** | Official client remembers sort and `all_unreads_section_filter` | Session-local `f`/`F` sort and `s`/chip filters (All / VIP / Starred / Channels / DMs). VIP/Starred use sidebar `prefs.vip_users` / `stars.list`; Channels vs DMs use conversation type. | Pref names/values other than captured `all_unreads_section_filter=all_sections` were not captured; slk does not write them. |
| **`stars.list` beyond the first page** | Starred lists can be long | One `POST stars.list` with `limit=1000`. Response has `paging.count` / `paging.total`. | Next-page form (cursor / page / offset) not captured. Hard cap 1000 message stars. |

### Not in the table on purpose

- **Join** a public channel from the finder: shipped (`conversations.join`).
- **Star / unstar a channel** (`*`): shipped (`stars.add` / `stars.remove` without timestamp).
- **Star / unstar a message** (`x` menu, or `*` in the Starred inbox): shipped (`stars.add` / `stars.remove` with timestamp).
- **Starred IMs / MPIMs** in the Starred *section*: shipped (`stars.list` `type=im|mpim|group` with a channel id).
- **Unreads section chips** (All / VIP / Starred / Channels / DMs): shipped session-local; the Slack pref is not written.

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
| **GitHub Release / semver tag** | **v0.17.0**. Homebrew formula in [agustif/homebrew-tap](https://github.com/agustif/homebrew-tap) pins that tag (`brew install agustif/tap/slk`); `--HEAD` still tracks `main`. |
| **GitHub release artifacts** | Cut with GoReleaser on tag `v*` (linux/windows static, darwin cgo). |
| **AUR** | AUR [`slk`](https://aur.archlinux.org/packages/slk) is **upstream**. This repo ships `packaging/aur` as `slk-git`; it is **not published** to the AUR. |
| **Help modal footer** | `agustif/slk · original by Grant Ammons` |
| **Nix flake** | In-tree `flake.nix` (`version = "0.17.0"`). Not a published nixpkgs/flakehub package. |

## How a gap gets filled

See [[Protocol#how-a-new-method-gets-in]]. Short version: HAR the official web client, wrap only that shape, no live mutating probes.
