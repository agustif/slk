# slk Implementation Status

Last updated: 2026-09-01 (agustif/slk fork, v0.19.0).

This file is a **snapshot**, not the feature spec. Live docs:

- What ships: [wiki/Features.md](../wiki/Features.md)
- Keys: [wiki/Keybindings.md](../wiki/Keybindings.md)
- Remaining OG holes + packaging: [wiki/Gaps.md](../wiki/Gaps.md)
- Browser protocol: [wiki/Protocol.md](../wiki/Protocol.md)
- Non-goals / caveats: [wiki/Tradeoffs-and-Non-Goals.md](../wiki/Tradeoffs-and-Non-Goals.md)

## This fork vs upstream

[agustif/slk](https://github.com/agustif/slk) is a daily-driver fork of [gammons/slk](https://github.com/gammons/slk). Extra work is official-client (OG) parity from HAR / browser protocol only — no invented APIs.

Shipped here on top of upstream’s TUI core:

- Home: Activity, Later, followed Threads, Direct Messages tab, Drafts & sent, Unreads, Starred **messages**
- Writable Slack sections, mute, `:notify` mentions/all, `:create` / `:create private`, `:invite` email or `U…`, `:kick`, `:manager` (Channel Manager `Rl0A`), Add/remove Starred files (`files.favorites.add` / `.remove`), Unreads recommended/scientifically sort + persisted sort/filter prefs, Cmd+K Search omniswitcher, OG recents write, jump-to-date, share, people search, message actions, drafts sync, scheduled send, double-click section collapse

## Not done (by policy)

See [wiki/Gaps.md](../wiki/Gaps.md). Short list:

- Starred files **inbox** (`files.favorites.add` / `.remove` write shipped; list not HAR’d)
- `client.dms` form captured (`priority_mode=priority`); DMs tab still uses boot cache + history
- Published AUR package (GitHub Release **v0.19.0** has binaries; AUR is still unpublished)

Permanent non-goals: huddles, canvas, lists, workflows, slash, Slack AI.

## Architecture

Four layers: UI (bubbletea) → service → Slack browser-protocol client → SQLite cache + TOML.

Module path: `github.com/agustif/slk`. ~600 Go files.

Historical design docs (not the live checklist): `docs/superpowers/specs/` and `docs/superpowers/plans/`.
