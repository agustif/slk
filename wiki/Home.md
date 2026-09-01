# slk Wiki

A blazingly fast Slack TUI. Keyboard-driven, beautifully themed, under 20MB,
one static binary, no Electron required.

**This fork:** [github.com/agustif/slk](https://github.com/agustif/slk) · **Upstream:** [gammons/slk](https://github.com/gammons/slk) · Upstream marketing site: [getslk.sh](https://getslk.sh)

OG-parity daily driver: only reverse-engineered official-client APIs. No invented endpoints, prefs, or sorts.

## Getting started

1. **[[Installation]]** — Homebrew `agustif/tap/slk` (last released tag v0.20.0), Go `@v0.20.0`, in-tree AUR PKGBUILD, GitHub Release binaries.
2. **[[Setup]]** — Slack desktop app session (`--add-workspace`). No DevTools, no tokens to copy.
3. **[[Configuration]]** — `config.toml`, custom themes, XDG paths, per-workspace settings.

## Using slk

- **[[Features]]** — full feature breakdown (Home surfaces, messaging, compose, images, threads, reactions, channels, notifications, status).
- **[[Keybindings]]** — every key, every mode.
- **[[Terminal Compatibility|Terminal-Compatibility]]** — which terminals do what (kitty graphics, sixel, half-block, OSC 52).
- **[[Clipboard and OSC 52|Clipboard-and-OSC-52]]** — getting copy/paste working under tmux, screen, and stricter terminals.

## Reference

- **[[Gaps]]** — packaging, captured-not-TUI, permanent non-goals.
- **[[Protocol]]** — browser protocol: envelope, `_x_reason`, methods, HARs.
- **[[Tradeoffs and Non-Goals|Tradeoffs-and-Non-Goals]]** — roadmap, permanent non-goals, caveats, TOS.
- **[[Architecture]]** — service-oriented layout, data layer, render pipeline.

## Project

- License: [MIT](https://github.com/agustif/slk/blob/main/LICENSE)
- `slk` is an independent, unofficial project. Not affiliated with Slack Technologies, LLC. See the [[Tradeoffs and Non-Goals|Tradeoffs-and-Non-Goals]] page for the TOS caveat.
