# slk Implementation Status

Last updated: 2026-09-01 (agustif/slk fork; last released tag v0.20.0; working tree has unreleased protocol work).

This file is a **snapshot**, not the feature spec. Live docs:

- What ships: [wiki/Features.md](../wiki/Features.md)
- Keys: [wiki/Keybindings.md](../wiki/Keybindings.md)
- Remaining packaging + captured-not-wired: [wiki/Gaps.md](../wiki/Gaps.md)
- Browser protocol: [wiki/Protocol.md](../wiki/Protocol.md)
- Non-goals / caveats: [wiki/Tradeoffs-and-Non-Goals.md](../wiki/Tradeoffs-and-Non-Goals.md)

## This fork vs upstream

[agustif/slk](https://github.com/agustif/slk) is a daily-driver fork of [gammons/slk](https://github.com/gammons/slk). Extra work is official-client (OG) parity from HAR / browser protocol only — no invented APIs.

Shipped here on top of upstream’s TUI core:

- Home: Activity, Later, followed Threads, Direct Messages tab, Drafts & sent, Unreads, Starred **messages** + Files-rail starred **files**
- Writable Slack sections, mute, `:notify` mentions/all, `:create` / `:create private`, `:invite` email or `U…`, `:kick`, `:manager` / `:unmanager` (Channel Manager `Rl0A`), Add/remove Starred files, Starred files inbox (`file_ids` + hydrate titles), custom file-section client wrappers (create/rename/delete/sort; no TUI), canvas list/star/open/close/lookup (no editor), `client.dms` on DMs tab, `stars.list` paging, Unreads scientifically sort + section chips, Cmd+K Search, OG recents (`CHANNEL` / `DM` / `FILE`), jump-to-date, share, people search, drafts sync, scheduled send

## Not done (by policy)

See [wiki/Gaps.md](../wiki/Gaps.md). Short list:

- Packaging: AUR unpublished; last tag **v0.20.0** while `main` has unreleased protocol work
- No Files-rail TUI (collections CRUD/sort wrapped; Starred files inbox ships)
- No in-TUI canvas editor (Quip `Collab`; list/star/open/close/lookup wrapped)

Permanent non-goals: huddles, lists, workflows, slash, Slack AI.

## Architecture

Four layers: UI (bubbletea) → service → Slack browser-protocol client → SQLite cache + TOML.

Module path: `github.com/agustif/slk`. ~600 Go files.

Historical design docs (not the live checklist): `docs/superpowers/specs/` and `docs/superpowers/plans/`.
