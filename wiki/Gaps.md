# Remaining gaps

This fork only ships official-client (OG) Slack surfaces reverse-engineered from the browser protocol (HAR, form fields, `_x_reason`, response shape). A public Web API name is not enough. **No invented endpoints, prefs, sorts, or item types.**

What ships: [[Features]]. Keys: [[Keybindings]]. Wire: [[Protocol]]. Non-goals: [[Tradeoffs and Non-Goals|Tradeoffs-and-Non-Goals]].

Last reviewed: 2026-09-01.

OG Home that needed a live form is **closed** (Activity / Later / Threads / DMs / Drafts / Unreads / Starred messages+files, recents `CHANNEL`/`DM`/`FILE`, Channel Manager add/remove, collection CRUD+sort). Remaining work is packaging, captured-but-no-TUI surfaces, and a few protocol holes.

## Packaging

Last released tag is **v0.20.0**. The working tree has unreleased protocol work; do not bump the tag from docs.

| Gap | Status |
|---|---|
| **GitHub Release / Homebrew** | v0.20.0 on [agustif/homebrew-tap](https://github.com/agustif/homebrew-tap). `--HEAD` tracks `main`. |
| **AUR** | AUR [`slk`](https://aur.archlinux.org/packages/slk) is **upstream**. In-tree `packaging/aur` is unpublished `slk-git`. |
| **Nix flake** | In-tree `flake.nix` (`version = "0.20.0"`). Not nixpkgs/flakehub. |
| **GitHub Wiki tab** | Disabled. Docs are `wiki/*.md` in this repo. |

## Captured, no TUI

Client wrappers exist (or the form is in [[Protocol]]) but slk has no matching screen:

| Surface | Wire | Why unwrapped as product |
|---|---|---|
| Files rail (All / Canvases / Lists / custom sections) | `search.modules.files`, `files.collections.*`, `canvases.getCannedTemplates` | Home Starred files inbox already lists starred `F…` ids. Full Files browser is a separate app. |
| Custom file-section UI | create / rename / delete / sort wrapped | No Files-rail chrome. |
| `files.recentlyDeleted` | `_x_reason=get-deleted-files` | Wrapped (`ListRecentlyDeletedFiles`). No Files-rail trash TUI. |
| `files.getShares` | `file_id` | File peek shares. |
| Canvas **editor** | Quip `window.Collab` + `/canvas/collab/*` + `/canvas/-/load-data` | Not a Slack `/api` form. List/star/open/close/lookup **are** wrapped. |
| Recents `FILE` writes | live `object_type=FILE` on `F…` | slk has no file/canvas viewer, so it does not bump `F…` on visit. Channel/DM recents **do** write. |
| Unreads chips for **custom** sidebar sections | `all_unreads_section_filter` = `L…` | Chips today: All / VIP / Starred / Channels / DMs. |
| `conversations.channelPrefixes.list` | `team_id`, `_x_reason=fetch-channel-prefixes` | Name-picker only. |
| Channel Manager **picker search** | official search still “No matches” | `:manager U…` posts the captured add form. |
| `client.dms` extra item fields | only `id` decoded | Arrays attested; `user` / `is_open` / `latest` not in the capture. |
| `favorites[]` item JSON | still `[]` when `file_ids` is non-empty | Inbox uses `file_ids` (reversed) then collection `files[].id`. |
| `files.info` `is_starred` | stays `false` after Files-rail star | Starred files use favorites/collections, not this flag. |

## Protocol holes (no form — do not wrap)

Pathnames or JS names only. See [[Protocol]] for the full inventory.

- `threads.getView` (followed threads are `subscriptions.thread.getView`)
- `search.modules.messages` / `.channels` / `.people` / `.dms`, `search.save`
- `today.items.list`, `emoji.collections.*`, `conversations.suggestions`
- `im.list`, `users.priority.list`, `activity.feed.scoreEntries` (forms captured; unused)
- Recents types `USER` / `PAGE` / `RECORD_CHANNEL` (MjSP enum; never POSTed here)
- Collection create **with emoji** (JS sends `emoji` when set; live create omitted it)

## Permanent non-goals

- Huddles, Slack Connect, Workflow Builder, Lists, Slack AI
- Canvas **editing** (Quip collab). Listing / starring / lookup as files is in-scope
- Slash commands, bot/app management, custom emoji **management** (picking/reacting shipped)
- Animated reactions (GIF first frame), interactive Block Kit buttons

## How a gap gets filled

See [[Protocol#how-a-new-method-gets-in]]. HAR the official web client, wrap only that shape. A public or third-party method name is not a capture.
