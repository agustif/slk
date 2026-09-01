# slk Implementation Status

Last updated: 2026-09-01 (agustif/slk fork, v0.18.0).

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
- Writable Slack sections, mute, jump-to-date, share, people search, message actions, drafts sync, scheduled send, double-click section collapse

## Not done (by policy)

See [wiki/Gaps.md](../wiki/Gaps.md). Short list:

- Create channel, invite members (Join / Leave exist)
- Mentions-only notification prefs
- Starred files (`type=file`; IM/MPIM conversation stars ship in the Starred section)
- Unreads recommended / scientifically sort; persisted Unreads sort/section prefs
- Published AUR package (GitHub Release **v0.18.0**)

Permanent non-goals: huddles, canvas, lists, workflows, slash, Slack AI.

## Architecture

Four layers: UI (bubbletea) → service → Slack browser-protocol client → SQLite cache + TOML.

Module path: `github.com/agustif/slk`. ~600 Go files.

Historical design docs (not the live checklist): `docs/superpowers/specs/` and `docs/superpowers/plans/`.
