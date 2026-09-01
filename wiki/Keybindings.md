# Keybindings

| Key | Mode | Action |
|---|---|---|
| `j` / `k` | Normal | Move down/up in channel list or messages |
| `h` / `l` | Normal | Switch focus between panels |
| `Tab` / `Shift+Tab` | Normal | Cycle focus |
| `Enter` | Normal (sidebar) | Open selected channel, Threads, Activity, Later, Direct Messages, Drafts, Unreads, or Starred, or toggle a section header |
| `Ctrl+t` / `Ctrl+p` | Any | Fuzzy channel finder (Home-view shortcuts: Activity, Later, Threads, Direct Messages, Drafts, Unreads, Starred; unjoined public channels join on select) |
| `Ctrl+n` | Any | New message — pick people, open a DM |
| `Ctrl+h` / `Ctrl+k` | Normal | Channel history back / forward |
| `[` / `]` | Normal | Narrow / widen sidebar |
| `?` | Normal | Keybindings overlay |
| `f` / `F` | Normal (Activity) | Next / previous Activity tab (Slack views, including custom Unreads / Reactions / VIP) |
| `s` | Normal (Activity) | Cycle Activity sort (newest ↔ unreads first) |
| `u` | Normal (Activity) | Toggle Activity unread-only |
| `Enter` | Normal (Activity) | Open the selected Activity item in its channel / thread |
| `r` | Normal (Activity) | Open reaction picker on the selected item |
| Click emoji | Activity card | Toggle your reaction on the parent message |
| Click rest of card | Activity card | Open the message (same as Enter) |
| Right-click | Activity card | Open reaction picker (terminals that report `MouseRight`; Apple Terminal may swallow right-click into its own menu) |
| `Enter` | Normal (Later) | Open the selected saved item in its channel / thread |
| `f` / `F` | Normal (Later) | Next / previous Later tab (In progress / Completed / Archived) |
| `c` | Normal (Later) | Mark the selected item complete |
| `z` | Normal (Later) | Archive the selected item |
| `u` | Normal (Later) | Move the selected item back to In progress |
| Enter | Normal (Later, load more) | Fetch the next page of saved items |
| `w` | Normal (message) | Toggle save-for-later on the selected message |
| `W` | Normal (message) | Remind me about this — duration menu, then `reminders.add` |
| `:remind 20m` | Command | Set a reminder on the selected message (`20m` / `1h` / `2d`) |
| `Space` | Normal (sidebar) | Toggle the selected section header (collapse/expand) |
| Double-click | Sidebar section header | Collapse or expand the section (two left-clicks within ~500ms; first click selects the header) |
| `Enter` | Normal (Drafts) | Open the selected draft in its channel / thread (insert mode), or the scheduled message's channel |
| `f` / `F` | Normal (Drafts) | Next / previous Drafts tab (Drafts / Scheduled) |
| `D` | Normal (Drafts) | Delete the selected draft or cancel the scheduled message |
| Enter | Normal (Drafts, load more) | Fetch the next page of drafts |
| `Enter` | Normal (Unreads) | Open the selected message in its channel, or Mark as Read / Undo on a channel header |
| `f` / `F` | Normal (Unreads) | Cycle local sort (sidebar / alphabetical / newest / oldest). Not persisted. |
| Click message | Unreads | Open the message (same as Enter) |
| Click header | Unreads | Mark as Read, or Undo when the header is already marked |
| `Enter` | Normal (Starred) | Open the selected starred message in its channel |
| `*` | Normal (Starred) | Unstar the selected message (`stars.remove`) |
| `x` | Normal (Starred) | Open / Unstar / Share |
| Click card | Starred | Open the message (same as Enter) |
| `*` | Normal | Star / unstar the selected sidebar channel (or the active channel if the message pane is focused). On the Starred **inbox**, unstars the selected message. |
| `J` / `:date` / `:jump` | Normal (channel/DM) | Jump to a calendar date (`YYYY-MM-DD` or `YYYY-MM-DD HH:MM`; no arg opens an overlay) |
| Share | Message actions (`x`) / `:share` | Share the selected message to another channel or DM (posts the permalink; Slack unfurls it) |
| `Enter` | Normal (message) | Open thread |
| `i` | Normal | Enter insert mode |
| `I` | Normal | Show members of the active channel |
| `j` / `k` | Members | Move down/up in the members list |
| type | Members | Filter members |
| `Enter` | Members | Open a DM with the selected member |
| `Esc` | Members | Close the members overlay |
| `Esc` | Insert / Command | Return to normal mode |
| `Enter` | Insert | Send message |
| `Ctrl+Enter` | Insert (thread compose) | Send reply and also post it to the channel |
| `Shift+Enter` | Insert | Newline |
| `Ctrl+V` | Insert | Smart paste — image / file path / text (use `Ctrl+V`, not the terminal's `Ctrl+Shift+V`) |
| `Ctrl+U` | Insert | Clear compose (text + pending attachments) |
| `Ctrl+g` | Insert | Schedule message — opens a duration overlay (20m / 1h / 2h / 4h / 8h / tomorrow 9am / custom). Confirm queues via Slack `chat.scheduleMessage` and clears compose. `Ctrl+Enter` is not used for schedule. |
| `:schedule` | Command | Open the schedule duration overlay for the current compose draft |
| `:schedule 20m` / `:schedule 1h` | Command | Schedule the current compose draft for that duration from now (`tomorrow` = next 9:00 AM). |
| `:scheduled` | Command | List pending scheduled messages; Enter cancels the highlighted one |
| `Ctrl+U` / `Ctrl+D` | Normal | Half-page up / down |
| `Up` | Insert | Previous line; on the first line, jump to start of message |
| `Down` | Insert | Next line; on the last line, jump to end of message |
| `gg` / `G` | Normal | Jump to top / bottom |
| `/` | Normal | Search in channel (vim-style; searches cached history of the current channel) |
| `n` / `N` | Normal | Next / previous search match (wraps) |
| `a` / `A` | Normal | Jump to next / previous unread channel (wraps) |
| `m` | Normal | Mute / unmute the selected sidebar channel, or the active channel if the message pane is focused (not mentions-only; see [[Gaps]]) |
| `Esc` | Normal (search active) | Clear active search |
| `Ctrl+f` | Any | Search workspace (Slack server-side; Messages/Files/People tabs; supports `from:@user`, `in:#channel`, `before:YYYY-MM-DD`) |
| `Tab` / `Shift+Tab` | Workspace search | Switch Messages / Files / People |
| `Enter` | Workspace search | Search; jump to a message; download a file (or open its permalink); load the next page; open a DM from People |
| `Ctrl+b` | Any | Toggle sidebar |
| `Ctrl+]` | Any | Toggle thread panel |
| `o` | Normal (message) | Open links in the selected message |
| `d` | Normal (message) | Download file attachments |
| `L` | Normal (message) | List reactions on the selected message |
| `Ctrl+w s` / `:sp` | Normal | Split window |
| `Ctrl+w v` / `:vsp` | Normal | Vertical split |
| `Ctrl+w h/j/k/l` | Normal | Focus window in that direction |
| `Ctrl+w w` | Normal | Cycle windows |
| `Ctrl+w q` / `:q` | Normal | Close window |
| `Ctrl+w o` / `:only` | Normal | Close other windows |
| `Ctrl+shift+y` | Any | Set default theme (all workspaces) |
| `:ws` | Normal | Workspace picker |
| `:leave` | Normal | Leave the current channel, or close the current DM (with confirmation) |
| `Esc` | Normal (Direct Messages view) | Return to Home (the compact sidebar) |
| Enter | Normal (Direct Messages, Home row) | Return to Home |
| `:reminders` | Command | List pending reminders; Enter marks the highlighted one complete |
| `:move` | Normal | Move the active channel into a Slack sidebar section (picker of existing section names; `:move Engineering` skips the picker) |
| `:section <name>` | Normal | Create an empty Slack sidebar section |
| `:rename <name>` | Normal | Rename the selected custom Slack section header |
| `:section-delete` | Normal | Delete the selected custom Slack section (with confirmation) |
| `:section-up` / `:section-down` | Normal | Reorder the selected custom Slack section |
| `1`–`9` | Normal | Jump to workspace N |
| `x` | Normal (message) | Open message actions menu |
| Right-click | Normal (message) | Open message actions menu (some terminals steal right-click; use `x`) |
| `r` | Normal (message) | Open reaction picker |
| `R` | Normal (message) | Quick-toggle existing reactions |
| `E` | Normal (message) | Edit your own message |
| `D` | Normal (message) | Delete your own message (with confirmation) |
| `U` | Normal (message) | Mark selected message and everything newer as unread |
| `P` | Normal (message) | Pin / unpin the selected message |
| `:pins` | Command | List pinned messages in the current channel; Enter jumps to one |
| Star message | Message actions (`x`) | Star / unstar the selected message (`stars.add` / `stars.remove` with timestamp) |
| `t` | Normal (thread panel) | Follow / unfollow the open thread |
| `S` | Normal (thread) | Save thread to markdown file (`~/.local/share/slk/exports/` or `$XDG_DATA_HOME/slk/exports/`) |
| `yy` | Normal (message) | Yank selected message text |
| `Y` / `C` | Normal (message) | Copy message permalink |
| `O` / `v` | Normal (message or thread) | Open full-screen image preview |
| `Esc` / `q` | Preview | Close preview |
| `Enter` | Preview | Open in system image viewer |
| `h` / `←` | Preview | Previous image (when message has multiple) |
| `l` / `→` | Preview | Next image (when message has multiple) |
| Click | Any (on image) | Open full-screen preview |
| `Ctrl+y` | Any | Switch theme |
| `Ctrl+s` | Any | Set status (Active / Away / DND snooze / custom status) |
| `p` | Normal (message) | Open the selected author's profile |
| `q` | Normal | Quit (with confirmation) |
| `Q` | Normal | Quit immediately |
| `Ctrl+c` | Any | Quit (with confirmation) |

Custom keybinding overrides are on the roadmap. Remaining OG holes (create channel, invite, mentions-only, starred files, Unreads extra sorts): [[Gaps]].
