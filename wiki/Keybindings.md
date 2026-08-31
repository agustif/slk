# Keybindings

| Key | Mode | Action |
|---|---|---|
| `j` / `k` | Normal | Move down/up in channel list or messages |
| `h` / `l` | Normal | Switch focus between panels |
| `Tab` / `Shift+Tab` | Normal | Cycle focus |
| `Enter` | Normal (sidebar) | Open selected channel, Threads, or Activity, or toggle a section header |
| `f` / `F` | Normal (Activity) | Next / previous Activity tab (Slack views, including custom Unreads / Reactions / VIP) |
| `s` | Normal (Activity) | Cycle Activity sort (newest ↔ unreads first) |
| `u` | Normal (Activity) | Toggle Activity unread-only |
| `Enter` | Normal (Activity) | Open the selected Activity item in its channel / thread |
| `Space` | Normal (sidebar) | Toggle the selected section header (collapse/expand) |
| `*` | Normal | Star / unstar the selected sidebar channel (or the active channel if the message pane is focused) |
| `Enter` | Normal (message) | Open thread |
| `i` | Normal | Enter insert mode |
| `I` | Normal | Show members of the active channel |
| `j` / `k` | Members | Move down/up in the members list |
| type | Members | Filter members |
| `Enter` | Members | Open a DM with the selected member |
| `Esc` | Members | Close the members overlay |
| `Esc` | Insert / Command | Return to normal mode |
| `Enter` | Insert | Send message |
| `Shift+Enter` | Insert | Newline |
| `Ctrl+V` | Insert | Smart paste — image / file path / text (use `Ctrl+V`, not the terminal's `Ctrl+Shift+V`) |
| `Ctrl+U` | Insert | Clear compose (text + pending attachments) |
| `Ctrl+g` | Insert | Schedule message — opens a duration overlay (20m / 1h / 2h / 4h / 8h / tomorrow 9am / custom). Confirm queues via Slack `chat.scheduleMessage` and clears compose. `Ctrl+Enter` is not used for schedule. |
| `:schedule` | Command | Open the schedule duration overlay for the current compose draft |
| `:schedule 20m` / `:schedule 1h` | Command | Schedule the current compose draft for that duration from now (`tomorrow` = next 9:00 AM). |
| `Ctrl+U` / `Ctrl+D` | Normal | Half-page up / down |
| `Up` | Insert | Previous line; on the first line, jump to start of message |
| `Down` | Insert | Next line; on the last line, jump to end of message |
| `gg` / `G` | Normal | Jump to top / bottom |
| `/` | Normal | Search in channel (vim-style; searches cached history of the current channel) |
| `n` / `N` | Normal | Next / previous search match (wraps) |
| `a` / `A` | Normal | Jump to next / previous unread channel (wraps) |
| `m` | Normal | Mute / unmute the selected sidebar channel, or the active channel if the message pane is focused |
| `Esc` | Normal (search active) | Clear active search |
| `Ctrl+f` | Any | Search workspace (Slack server-side; supports modifiers like `from:@user`, `in:#channel`, `before:YYYY-MM-DD`) |
| `Ctrl+b` | Any | Toggle sidebar |
| `Ctrl+]` | Any | Toggle thread panel |
| `Ctrl+t` / `Ctrl+p` | Any | Fuzzy channel finder |
| `:ws` | Normal | Workspace picker |
| `:leave` | Normal | Leave the current channel (with confirmation). DMs cannot be left. |
| `:move` | Normal | Move the active channel into a Slack sidebar section (picker of existing section names; `:move Engineering` skips the picker) |
| `:section <name>` | Normal | Create an empty Slack sidebar section |
| `1`–`9` | Normal | Jump to workspace N |
| `x` | Normal (message) | Open message actions menu |
| Right-click | Normal (message) | Open message actions menu (some terminals steal right-click; use `x`) |
| `r` | Normal (message) | Open reaction picker |
| `R` | Normal (message) | Quick-toggle existing reactions |
| `E` | Normal (message) | Edit your own message |
| `D` | Normal (message) | Delete your own message (with confirmation) |
| `U` | Normal (message) | Mark selected message and everything newer as unread |
| `S` | Normal (thread) | Save thread to markdown file (`~/.local/share/slk/exports/` or `$XDG_DATA_HOME/slk/exports/`) |
| `yy` | Normal (message) | Yank selected message text |
| `Y` / `C` | Normal (message) | Copy message permalink |
| `O` / `v` | Normal (message) | Open full-screen image preview |
| `Esc` / `q` | Preview | Close preview |
| `Enter` | Preview | Open in system image viewer |
| `h` / `←` | Preview | Previous image (when message has multiple) |
| `l` / `→` | Preview | Next image (when message has multiple) |
| Click | Any (on image) | Open full-screen preview |
| `Ctrl+y` | Any | Switch theme |
| `Ctrl+s` | Any | Set status (Active / Away / DND snooze) |
| `q` | Normal | Quit (with confirmation) |
| `Q` | Normal | Quit immediately |
| `Ctrl+c` | Any | Quit (with confirmation) |

Custom keybinding overrides are on the roadmap — see [[Tradeoffs and Non-Goals|Tradeoffs-and-Non-Goals]].
