# Architecture

Service-oriented, four layers:

```
UI Layer (bubbletea)   workspace rail · sidebar · Home views · messages · thread · compose · status bar
Service Layer          WorkspaceManager · MessageService · ConnectionManager · section/mute stores
Client Layer           Slack Web API + browser-protocol WebSocket (xoxc + cookie)
Data Layer             SQLite cache · TOML config · desktop-app session mint
```

Home views (Activity, Later, Threads, DMs, Drafts, Unreads, Starred) swap the messages pane; they are not extra Electron windows.

- ~600 Go files (~160k lines including tests). SQLite is a cache — Slack remains authoritative.
- Render cache + item-level selection for snappy scrolling.
- muesli/reflow everywhere for ANSI-correct wrapping and truncation.
- Module path: `github.com/agustif/slk` (fork of `github.com/gammons/slk`).

## Layout (high level)

```
slk/
├── cmd/slk/                 # wiring, onboarding, WS event → UI messages
├── internal/
│   ├── slack/               # browser-protocol client (stars, drafts, activity, saved, …)
│   ├── slackdesktop/        # read Slack desktop app session
│   ├── cache/               # SQLite
│   ├── service/             # sections, mute, VIP, messages
│   └── ui/                  # bubbletea App + pane packages
│       ├── sidebar/         # synth rows + Slack sections
│       ├── activityview/ laterview/ threadsview/ draftsview/
│       ├── unreadsview/ starredview/
│       ├── messages/ thread/ compose/
│       └── channelfinder/   # includes Home-view shortcuts
├── packaging/aur/           # in-tree slk-git PKGBUILD (not published to AUR)
├── wiki/                    # this documentation
└── docs/superpowers/        # historical design specs / plans (not the live feature list)
```

## Further reading

- [[Features]] — what ships
- [[Gaps]] — packaging, captured-not-TUI, permanent non-goals
- [[Protocol]] — unofficial browser protocol (envelope, methods, HARs)
- Design specs (historical): [`docs/superpowers/specs/`](https://github.com/agustif/slk/tree/main/docs/superpowers/specs/)
- Snapshot: [`docs/STATUS.md`](https://github.com/agustif/slk/blob/main/docs/STATUS.md)
