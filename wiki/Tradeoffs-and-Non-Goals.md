# Tradeoffs and Non-Goals

slk is intentionally not a 1:1 port of the desktop client. Some Slack features are deferred or out of scope.

## On the roadmap

- Custom keybinding overrides

## Not planned

- Huddles, Slack Connect, Workflow Builder, Canvas, Lists, Slack AI
- Bot/app management, slash commands, custom emoji management
- Animated reactions

Search (`Ctrl+f`), file upload/download, quiet hours, per-channel mute, link unfurls, and in-app toasts shipped.

## Markdown caveats

- Editing a message you originally formatted with markdown may flatten the rich_text formatting on Slack clients that prefer blocks. The mrkdwn fallback (`*bold*`, etc.) still renders correctly everywhere.
- Headings (`# Title`) and blockquotes (`> quote`) are passed through verbatim — Slack has no heading construct and `>` is already valid mrkdwn.
- Tables, footnotes, task lists, and reference-style links are not translated.

## Image rendering caveats

- iTerm2 ≥ 3.5 implements kitty graphics but does not support unicode placeholders, so it falls back to half-block.
- Animated GIFs render as a static first frame.
- Threads side panel renders images inline using the same pipeline as the main messages pane (kitty, sixel, and half-block). Click-to-preview and `O` / `v` work from a thread reply.

## Auth caveat

Browser-cookie auth means tokens expire when you log out of the browser or
Slack rotates them. Re-run `--add-workspace` and you're back in business. See
[[Setup]] for the token-extraction walkthrough.

## Unofficial / TOS caveat

slk talks to Slack via the same internal browser protocol the official web
client uses. This is unofficial and not sanctioned by Slack — using it may
violate Slack's [API](https://slack.com/terms-of-service/api) and
[user](https://slack.com/terms-of-service/user) Terms of Service, and Slack
may break the protocol or invalidate tokens at any time. Use at your own risk
on workspaces where that's acceptable to you and your admins.
