# Protocol

What this fork knows about Slack’s **unofficial browser client** protocol, as implemented. A public Web API name is not enough; the official client’s form, `_x_reason`, envelope, and response shape are the contract.

Product gaps (starred files inbox, Unreads section-filter prefs, …): [[Gaps]]. Features: [[Features]].

Raw HARs are **not** in the repo (live `xoxc` tokens, `d` cookies, message bodies). Digests and golden tests are.

Last reviewed: 2026-09-01.

## Evidence

| Capture | When | Workspace | What it pinned |
|---|---|---|---|
| 8 HARs, Chrome 150 Linux | 2026-07-30 | Grid (`rands-leadership.slack.com`) | Headers, query envelope, `_x_reason` / `_x_mode` presence, `client.userBoot`, `conversations.view`, `conversations.history` (`limit=28`, `cached_latest_updates`), `client.counts`, `client.shouldReload`, edgeapi, WebSocket upgrade |
| Official web Activity / Later / Drafts / Unreads | 2026-08-31 | Obvious AI (`T099JCA82HJ`) | `activity.feed` / `activity.views`, `saved.*`, `drafts.*`, Unreads `conversations.history` window |
| Threads socket | 2026-08-02 coldboot | — | WebSocket `start_args` (typing delivery) |
| Official web create / invite / notify | 2026-09-01 | Test Workspace (`T0BTKN81S79`) | Public `conversations.create`; `users.admin.inviteBulk`; `users.prefs.setNotifications` multi-pref |
| Official web private create / member invite / kick / recents / channel manager | 2026-09-01 | Test Workspace (`T0BTKN81S79`) | Private `conversations.create` `is_private=true` (still **no** `_x_reason`); `conversations.invite` (`force=true`, empty `subteams`); `conversations.kick`; `users.prefs.set` `name=recents`; `admin.roles.addMembers` `role_id=Rl0A` (write returned `no_valid_users` until invitee finished signup) |
| Official web DMs tab + Files rail reload | 2026-09-01 | Test Workspace (`T0BTKN81S79`) | `client.dms` form (`priority_mode=priority` string, `_x_reason=dms-tab-populate`); `files.recentlyDeleted` (`_x_reason=get-deleted-files`); Files Starred is still `files.collections.list` (no `files.favorites.list` on that click) |
| Official web Unified Files unstar | 2026-09-01 | Test Workspace (`T0BTKN81S79`) | `files.favorites.remove` `file_id` + `collection_id` `_x_reason=remove_file_from_collection` (`{"ok":true}` on `F0BUVQHU6NL` → `Fs0BTURTUXK5`). Same-session Move to… Starred radio `aria-checked=true` plus **Remove from Starred**. After reload, radio is unchecked, `collections.list` `files[]` still empty, `files.info` `is_starred=false`. |
| Official web All Unreads sort | 2026-09-01 | Obvious AI (`T099JCA82HJ`) | `users.prefs.set` `name=all_unreads_sort_order` `_x_reason=prefs`. Values: `sidebar`, `alphabetical`, `priority` (UI “Sorted by recommended order”; list `All Unreads, sorted scientifically`), `newest`, `oldest`. JS `sortScientifically`: starred then not; channels-with-mentions, channels, IMs, MPIMs; `channels_priority` desc then name. |
| Official web All Unreads section filter | 2026-09-01 | Obvious AI (`T099JCA82HJ`) | `users.prefs.set` `name=all_unreads_section_filter` `_x_reason=all-unreads-section-filter-menu-item-select`. All conversations = `all_sections`. VIP unreads = `priority`. Starred = `L0ARTKQH049`, Channels = `L0AS91EASGG`, Direct messages = `L0ARBAVLF45` (sidebar section ids; slk looks them up by type). |
| Official web Cmd+K Search (omniswitcher) | 2026-09-01 | Test Workspace (`T0BTKN81S79`) | Open: `search.autocomplete` / `.model` / `.intentModel` / `.offlineFeatures` / `search.precache`. Typing: edge `channels/search` + `users/search` (existing wrappers); `search.inline` (`count=3`, `extract_len=110`, `from_me=true`, `with_me=true`, `recent_channels`, `_x_reason=quick-messages/prototype`); `search.autocomplete.files` (`include_shares=true`, `_x_reason=omniswitcher:suggestions-from-searcher`); `search.autocomplete.triggers` (shortcuts, unwrapped). |

Machine-readable digests (no secrets):

- [`internal/slackhttp/testdata/official-request-shape.json`](https://github.com/agustif/slk/blob/main/internal/slackhttp/testdata/official-request-shape.json) — header sets, query order, body envelope, 48 observed `_x_reason` values
- [`internal/slackhttp/testdata/capture-evidence.json`](https://github.com/agustif/slk/blob/main/internal/slackhttp/testdata/capture-evidence.json) — 279 `/api/` requests, 0 non-empty `Referer`
- [`internal/slack/testdata/phase2-api-contracts.json`](https://github.com/agustif/slk/blob/main/internal/slack/testdata/phase2-api-contracts.json) — redacted `userBoot` / `conversations.view` / history / counts samples

Code that *must* match those files: `internal/slackhttp/` (`golden_test.go`, `reason.go`, `mode.go`, `envelope.go`, `transport.go`).

## Identity

Not OAuth, not a Slack App, not RTM, not Socket Mode.

| Piece | Where it rides | Notes |
|---|---|---|
| `xoxc-…` token | **POST body** field `token=` (workspace API). JSON `token` on edgeapi. Query `token=` on the WebSocket URL | **Not** `Authorization: Bearer` on `/api/` — that pairing with browser headers is a Grid anomaly signature (upstream #5) |
| `d` cookie | Cookie jar on HTTP + WebSocket | Session cookie from the Slack **desktop app** (`--add-workspace`). Chrome HAR export often strips cookies; do not infer cookie absence from a HAR |
| Team id | Envelope `slack_route` after boot; `gateway_server={team}-1` on WS | Pre-boot requests omit `slack_route` |

Minting: slk reads the desktop app’s local session and re-mints; it does not scrape DevTools. See [[Setup]].

## Hosts

| Host | Use |
|---|---|
| `{workspace}.slack.com/api/{method}` (fallback `slack.com/api/`) | Workspace Web API, `application/x-www-form-urlencoded` |
| `edgeapi.slack.com/cache/{teamId}/{endpoint}` | Edge cache JSON-as-`text/plain` (CORS simple request) |
| `wss-primary.slack.com` | Browser-protocol events |
| `files.slack.com` | File / thumbnail / avatar bytes |

## Transports

`internal/slackhttp.BrowserTransport` is the chokepoint under both slack-go and hand-rolled `postForm`. It adds browser headers and the `_x_*` envelope. Image loads use a **different** header set (`DestImage`).

### Workspace API (`/api/`)

1. Caller form (business fields). `postForm` injects `token`.
2. Transport appends trailing body fields (order is the contract): `_x_reason`, `_x_mode`, `_x_sonic`, `_x_app_name` — except exclusions below.
3. Query params (order is the contract, **not** alphabetical — 0/163 captured requests were alpha-sorted):

Post-boot: `_x_id`, `_x_csid`, `slack_route`, `_x_version_ts`, `_x_foreground`, `_x_frontend_build_type`, `_x_desktop_ia`, `_x_gantry`, `_x_b3_traceid`, `_x_b3_spanid`, `_x_b3_sampled`, `fp`, `_x_num_retries`.

Pre-boot (no team id): `_x_id` prefix `noversion-`; omit `_x_csid`, `slack_route`, and the three B3 params.

`_x_id` is `{prefix}-{unix}.{ms}` and is **not** unique: the official client collides on the same millisecond.

### `_x_reason` / `_x_mode`

Measured on 163 form-body requests (2026-07-30):

| `_x_reason` | `_x_mode` | Count | Methods |
|---|---|---|---|
| yes | yes (`online`) | 149 | majority |
| yes | no | 4 | `client.shouldReload`, `client.userBoot` |
| no | no | 10 | `api.features`, `client.getWebSocketURL`, `conversations.view`, `experiments.getByUser`, `features.access.policies.list` |
| no | yes | **0** of 163 (2026-07-30) | never in that corpus. **Exception (2026-09-01):** `conversations.create` (`_x_mode=online`, no `_x_reason`) |

**Causal caveat:** every “omit `_x_mode`” observation is boot-time. Captures cannot tell “these methods never send `_x_mode`” from “nothing sends it until boot finishes.” slk encodes the measured per-method table.

**Default `_x_reason` (attested pairing, 2026-07-30):**

| Method | `_x_reason` |
|---|---|
| `client.userBoot` | `initial-data` |
| `client.shouldReload` | `boot` |
| `client.counts` | `fetchClientCountsOnConnect` |
| `conversations.history` | `message-pane/requestHistory` (override: `unread-counts/onLastReadUpdated`) |
| `conversations.mark` | `viewed` |
| `conversations.genericInfo` | `fallback:fetchAndUpsertChannelsById` |
| `users.prefs.get` | `fetch-frecency-prefs` |
| `users.channelSections.list` | `conditional-fetch-manager` |
| `dnd.info` | `fetchAndUpsertDndForUsers-getDndTimesFor:self` |

Call-site `WithReason` beats the table. Explicit reasons used from **later** HARs (not in the July 30 48-value set unless noted):

| Reason | Method | Capture |
|---|---|---|
| `saved-api/savedList` (in the 48) | `saved.list` | Later |
| `saved-api/addSavedMessage` | `saved.add` | Later |
| `saved-api/deleteSavedMessage` | `saved.delete` | Later |
| `saved-api/updateSavedMessage` | `saved.update` | Later |
| `drafts-api/list` | `drafts.list` | 2026-08-31 |
| `drafts-api/listActive` | `drafts.listActive` | 2026-08-31 |
| `drafts-api/create` / `update` / `delete` | `drafts.*` | 2026-08-31 |
| `fetchActivityFeed` | `activity.feed` | 2026-08-31 |
| `bookmarks-store/conditional-fetching` (in the 48) | `bookmarks.list` | July 30 |
| `prefs-store/setChannelNotificationOverride` | `users.prefs.setNotifications` | public mute captures; **not** in slk’s 2026-07-30 HARs |
| `prefs-store/setMultiChannelNotificationOverride` | `users.prefs.setNotifications` (multi-pref `channel_ids` + `prefs` JSON) | 2026-09-01 Test Workspace; all / mentions / mute-from-channel-menu |
| `send-workspace-invites-from-channel-invite` | `users.admin.inviteBulk` | 2026-09-01 Test Workspace channel-invite flow |
| `submit-invite-channel-invite-modal` | `conversations.invite` | 2026-09-01 add existing member to `#slk-har-priv2` |
| `submitKickFromChannel` | `conversations.kick` | 2026-09-01 remove member from `#slk-har-priv2` |
| `add-channel-managers` | `admin.roles.addMembers` | 2026-09-01 Make Channel Manager (`role_id=Rl0A`) |
| `fetch-channel-managers` | `admin.roles.entity.listAssignments` | 2026-09-01 channel details |
| `prefs-api/setUserPrefByApi` | `users.prefs.set` `name=recents` | 2026-09-01 channel switch (Home recents) |
| `client_redux_store` | `conversations.listPrefs` | 2026-09-01 channel/DM open |
| `fetch-channel-prefixes` | `conversations.channelPrefixes.list` | 2026-09-01 create-channel name picker (slk skips) |

**Fallback:** methods with no table entry and no `WithReason` get `_x_reason=conditional-fetch-manager`. That **string** is attested (sections.list). The **pairing** with e.g. `stars.add` is a guess. Emitting nothing is a sharper fingerprint (`_x_mode` without `_x_reason`).

48 attested `_x_reason` values: `official-request-shape.json` → `x_reason_observed_values`.

### Edge API

`POST https://edgeapi.slack.com/cache/{teamId}/{endpoint}`

- Body: JSON object + `token`, `Content-Type: text/plain;charset=UTF-8` (no CORS preflight). **No** `_x_*` in the body (116/116).
- Query (116/116 this order): `_x_app_name`, `_x_b3_traceid`, `_x_b3_spanid`, `_x_b3_sampled`, `fp`, `_x_num_retries`.
- **Absent** vs workspace API: `_x_id`, `_x_version_ts`, `slack_route`, `_x_csid`, `_x_frontend_build_type`, `_x_desktop_ia`, `_x_gantry`, `_x_foreground`.

On Enterprise Grid, `channels/info` is keyed by the **owning** team, not always the session team.

### WebSocket

`wss://wss-primary.slack.com/?token=…&sync_desync=1&slack_client=desktop&start_args=…&no_query_on_subscribe=1&flannel=3&lazy_channels=1&gateway_server={team}-1&batch_presence_aware=1`

`start_args` (2026-08-02 coldboot, key-for-key): `agent=client&org_wide_aware=true&agent_version={version_ts}&eac_cache_ts=true&cache_ts=0&name_tagging=true&only_self_subteams=true&connect_only=true&ms_latest=true`.

Upgrade headers are a **smaller** set than XHR: User-Agent, Accept-Language, Cache-Control, Pragma, Origin. No `Accept`, no `Sec-Fetch-*`, no `sec-ch-ua*`, no `Priority`, no `Referer`.

Outbound: typing (`SendTyping`), presence subscribe (`SubscribePresence`).

### Files / images

Chrome `<img>` loads: `Sec-Fetch-Dest: image`, `Sec-Fetch-Mode: no-cors`, `Priority: i`, **Referer present**, **Origin absent**. slk’s `DestImage` matches that set.

**files.slack.com:** cookie-only first (matches 0/40 captured `Authorization` headers). If that returns 403 or HTML login, retry once with Bearer + cookie.

**Residual:** `Accept-Encoding` — Chrome sends `gzip, deflate, br, zstd`; Go’s transport advertises `gzip`. Setting Chrome’s list would disable stdlib decompression.

## Workspace API methods slk calls

`token` is always injected. Boolean form fields are the strings `"true"` / `"false"`, never JSON booleans.

### Boot / session

| Method | Form (business) | Envelope notes |
|---|---|---|
| `client.shouldReload` | `team_ids`, `build_version_ts` | `_x_reason=boot`, **no** `_x_mode`. Sources `_x_version_ts` |
| `client.userBoot` | `version_all_channels=false`, `return_all_relevant_mpdms=true`, `omit_extras=feature_usage_data,plan_info,salesforce_features` | `_x_reason=initial-data`, **no** `_x_mode`. Channels + IMs + prefs blob + DND + self. `stars` / `subteams` in boot were empty in captures — slk still calls `stars.list` / `usergroups.list` |
| `conversations.view` | `channel`, `count=28`, `canonical_avatars`, `no_user_profile`, `ignore_replies`, `no_self`, `include_full_users`, `include_use_case`, `include_stories`, `no_members`, `include_mutation_timestamps`, `include_free_team_extra_messages` (all `"true"`) | **Neither** `_x_reason` nor `_x_mode`. Empty `channel=` is a forbidden shape. Verify `result.channel.id` — Slack may silently return the last-viewed channel |
| `client.counts` | (token only on some paths) | Default reason `fetchClientCountsOnConnect`. Channels / MPIMs / IMs `has_unreads`, `activity_v2`, `saved` |

### History / read watermarks

Captured `conversations.history` (14/14): `limit=28`, `ignore_replies=true`, `include_pin_count=true`, `inclusive=true`, `no_user_profile=true`, `include_stories=true`, `include_free_team_extra_messages=true`, `include_date_joined` always present, `cached_latest_updates` always present (`{}` if empty — never omit, never `null`). At most one of `oldest` / `latest` (both together: 0/14).

Response modelled: `messages`, `unchanged_messages`, `latest_updates`, `has_more`. Versions are opaque 17-char strings to echo, not timestamps.

| Call | Shape |
|---|---|
| `HistoryWithVersions` | Full captured form. Default `_x_reason=message-pane/requestHistory` |
| `UnreadHistory` | Same + `oldest=last_read`, `include_date_joined=false` (2026-08-31 Home Unreads) |
| `GetHistory` / `GetOlderHistory` / `GetHistoryAround` / `GetHistorySince` | Same captured form. Default limit 28. `GetHistorySince` paginates with `latest` (not `next_cursor`) |

| Method | Form |
|---|---|
| `conversations.mark` | `channel`, `ts` — read or roll back (unread) |
| `subscriptions.thread.mark` | `channel`, `thread_ts`, `ts`, `read` = `"1"` / `"0"` |

### Home surfaces (2026-08-31 unless noted)

| Method | Form | Notes |
|---|---|---|
| `activity.views` | (empty) | Built-in All/DMs/Mentions/Threads + custom views. Selecting a tab does **not** send `view_id`; flatten `entry_types` / `unread_only` / `priority_only` onto the feed |
| `activity.feed` | `limit` (1–100, default 50), `types` (comma list; All-tab list is duplicated as captured), `mode` = `chrono_v1` or `priority_reads_and_unreads_v1`, `archive_only=false`, `unread_only`, `priority_only`, `only_salesforce_channels=false`, `exclude_automations=false`, `automations_only=false`, `is_activity_inbox=true`, optional `sort=vip_unreads_first` | `_x_reason=fetchActivityFeed` (2026-08-31). Not in the July 30 48-value golden list; call-site `WithReason` |
| `saved.list` | `limit`, `filter` = `saved` \| `completed` \| `archived`, `include_tombstones=true`, optional `cursor` | `item_type=message` rows used; other types skipped |
| `saved.add` / `.delete` | `item_id` (channel), `item_type=message`, `ts` | |
| `saved.update` | same + `date_due` or `state` = `in_progress` \| `completed` \| `archived` | |
| `drafts.list` | `is_active=true`, `limit`, `next_ts` (not `cursor`) | Page while `has_more` |
| `drafts.listActive` | `limit=1000` | Badge = `active_draft_ids` |
| `drafts.create` | `blocks`, `destinations`, `file_ids`, `attachments`, `client_msg_id`, `is_from_composer=true` | |
| `drafts.update` | + `draft_id`, `client_last_updated_ts` | |
| `drafts.delete` | `draft_id`, `client_last_updated_ts` | |
| `stars.list` | `limit=1000`; next page `page=2` (live) or `cursor` from `response_metadata.next_cursor` (JS + live empty `next_cursor=""`) | `type=channel|im|mpim|group` with `channel` → sidebar Starred **section**; `type=message` → inbox (`channel`, `date_create`, `message.{ts,user,text}`). `type=file` unused. slk follows `next_cursor`, else `paging.total` with `page=N+1` |
| `stars.add` / `stars.remove` | Channel star: `channel` only. Message star: `channel` + `timestamp`. Idempotent errors `already_starred` / `no_star` | |

### Sidebar sections

REST `users.channelSections.list`: `cursor` for more **sections**. Membership is `channel_ids_page` (first page; sections with >10 channels may be truncated until WS). Types: `standard`, `channels`, `direct_messages`, `recent_apps`, `stars` (empty `channel_ids` — fill from `stars.list`), `slack_connect` / `salesforce_records` / `agents` (hidden).

Writes:

| Method | Form |
|---|---|
| `users.channelSections.channels.bulkUpdate` | `insert` / `remove` = JSON arrays `[{"channel_section_id","channel_ids"}]`. Empty `remove` omitted (empty `[]` unverified) |
| `users.channelSections.create` | `name`, optional `emoji` |
| `users.channelSections.update` | `channel_section_id`, `name` / `emoji` / `next_channel_section_id` |
| `users.channelSections.delete` | section id |

Linked-list order is `next_channel_section_id`.

### Threads follow set

| Method | Form |
|---|---|
| `subscriptions.thread.getView` | `limit=8`, `fetch_threads_state=true`, `priority_mode=all`, optional `current_ts` = previous `max_ts` |
| `subscriptions.thread.add` / `.remove` | `channel`, `thread_ts`. Idempotent: `already_subscribed` / `not_subscribed` |

Capture notes (2026-05): official client sent `limit=8` with `_x_reason=fetch-threads-view-via-refresh` (first page) / `fetch-threads-view-via-load-more` (when `current_ts` is set). slk matches that pairing.

### Prefs / mute / status

| Method | Form |
|---|---|
| `users.prefs.get` | — | Boot also embeds `all_notifications_prefs` JSON (`channels[id].muted`, `suppress_at_channel`) |
| `users.prefs.setNotifications` (mute) | `name=muted`, `value` true/false, `channel_id`, `global=false`, `sync=false` | `_x_reason=prefs-store/setChannelNotificationOverride` |
| `users.prefs.setNotifications` (all / mentions) | `global=false`, `channel_ids`, `prefs` JSON, `sync=true` | Mentions: `desktop=mentions_dms` + `follow_all_threads=false` + `suppress_at_channel=false`. All: `desktop=everything` + `follow_all_threads=false` + `badge_all_unreads=false`. `_x_reason=prefs-store/setMultiChannelNotificationOverride` (2026-09-01) |
| `users.profile.set` | `profile` = JSON `{status_text,status_emoji,status_expiration}` | |
| DND / presence | slack-go: `dnd.setSnooze`, `dnd.endSnooze`, `dnd.endDnd`, `users.setPresence` | |

### Search / people / members (edge + Web API)

| Call | Payload |
|---|---|
| `search.messages` | slack-go; `query` + optional `count`/`page`. Modifiers `from:`, `in:`, `before:` pass through |
| `search.files` | same form via `postForm` |
| edge `channels/search` | `query`, `count=30`, `fuzz=1` (number, not bool), `include_record_channels=true`, `check_membership=true`, `default_workspace`, optional `top_channels` (omit key if empty). Debounce; empty query = no request |
| edge `users/search` | `query`, `count=30`, `fuzz=1`, `enable_workspace_ranking=true`, `search_email=true`, `include_profile_only_users=true`, `default_workspace`, optional `top_users`, optional `current_channel` |
| edge `channels/info` | `updated_ids` map, `check_membership=true` → top-level `member_channels` |
| edge `users/info` | `updated_ids`, `check_interaction=true`, `include_profile_only_users=true` |
| edge `users/list` | channel members; `count` observed 20 **and** 30 |
| edge `channels/membership` | `channel`, `user_ids` |
| edge `users/counts` | `channel` |

Do **not** walk `conversations.list` / `users.list` — that is the Grid “data scraping” signature. `SlackAPI` in code omits those methods on purpose.

### Messaging (mostly slack-go)

Still go through the same cookie client + `BrowserTransport` (envelope applies):

`chat.postMessage` (incl. `thread_ts`, `reply_broadcast` / `thread_broadcast`), `chat.update`, `chat.delete`, `chat.scheduleMessage`, `chat.scheduledMessages.list`, `chat.deleteScheduledMessage`, `reactions.add` / `.remove`, `pins.add` / `.remove`, `pins.list` (also hand-rolled `PostForm` for header chips: `channel=`), `bookmarks.list` (`channel_id`), `conversations.join` / `.leave` / `.create` / `.invite` / `.kick`, `conversations.close` (`channel`), `conversations.open` (user ids or reopen `channel`), `conversations.replies`, `users.admin.inviteBulk`, `users.prefs.set` (`name=recents`), `admin.roles.addMembers` / `admin.roles.entity.listAssignments`, `files.upload` V2, `auth.test`, `emoji.list`, permalink.

### Channel create / members / recents (2026-09-01 Test Workspace)

Official client used `multipart/form-data` for these; slk sends the same fields as `application/x-www-form-urlencoded` via `postForm` (same as other workspace methods).

| Method | Form | Envelope / notes |
|---|---|---|
| `conversations.create` (public) | `name`, `validate_name=true`, `team_id` | **No** `_x_reason`, `_x_mode=online`. Do **not** send `is_private`. |
| `conversations.create` (private) | same + `is_private=true` | Same envelope. Response `channel.is_private=true`. Captured name `slk-har-priv2` → `C0BTX6N7JRK`. |
| `conversations.channelPrefixes.list` | `team_id` | `_x_reason=fetch-channel-prefixes`. Name-picker only; slk does not call it. |
| `users.admin.inviteBulk` | `source=invite_emails_to_channel`, `restricted=false`, `ultra_restricted=false`, `invites` JSON `[{email,type:regular}]`, optional `channels`, `team_id` | `_x_reason=send-workspace-invites-from-channel-invite`. Workspace invite from the channel-invite modal. |
| `conversations.invite` | `channel`, `invite_all=false`, `users` (member id), `subteams` (empty key present), `force=true` | `_x_reason=submit-invite-channel-invite-modal`. Existing member `U0BU3458TTK` into `C0BTX6N7JRK`. Response `ok=true` + channel object; join message `subtype=channel_join`. |
| `conversations.kick` | `channel`, `user` | `_x_reason=submitKickFromChannel`. Response `{"ok":true,"errors":{}}`. |
| `admin.roles.entity.listAssignments` | `entity_id` = channel id | `_x_reason=fetch-channel-managers`. Response `role_assignments:[{role_id:"Rl0A", users:[creator]}]`. `Rl0A` is Channel Manager. |
| `admin.roles.addMembers` | `role_id=Rl0A`, `role_scopes` = channel id, `user_ids` | `_x_reason=add-channel-managers`. slk `:manager U…` posts this form. Official picker still “No matches” for Workspace Tester after signup; success extra JSON uncaptured (`ok` ack only). |
| `users.prefs.set` recents | `name=recents`, `value` JSON `{"navigation":[{"id","object_type":"CHANNEL","timestamp":ms},…]}` | `_x_reason=prefs-api/setUserPrefByApi`. Posted on channel switch (move-to-front). Captured `object_type` is `CHANNEL` for `C…` ids only. Opening a DM from the DMs tab (`D0BU4SLGVE0`) did **not** POST recents. |
| `users.prefs.set` All Unreads sort | `name=all_unreads_sort_order`, `value` | `_x_reason=prefs`. Captured values `sidebar` / `alphabetical` / `priority` / `newest` / `oldest`. slk `f`/`F` writes this and applies it from `client.userBoot` prefs. Scientifically is **not** a server sort — `priority` plus client `sortScientifically`. |
| `users.prefs.set` All Unreads section filter | `name=all_unreads_section_filter`, `value` | `_x_reason=all-unreads-section-filter-menu-item-select`. `all_sections` (All), `priority` (VIP unreads), or a sidebar section id `L…` (Starred / Channels / DMs / custom sections). slk chips write All/VIP/Starred/Channels/DMs; custom sections are not extra chips. |
| `conversations.listPrefs` | `channel` | `_x_reason=client_redux_store`. Prefs include `who_can_post` / `can_thread` `{type:["ra"]}`. Not a slk write. |
| `files.collections.list` | (token only) | `_x_reason=fetch_file_collections`. Response `collections:[{id:"Fs0BTURTUXK5", name:"Starred", position:"5000000000", type:"starred", sort:"date_added", files:[]}]`. `files[]` stayed **empty** after a successful `files.favorites.add` of `F0BUVQHU6NL`, including after reload. Collection ids are `Fs…`. With `custom_file_sections=on` this is the Starred-files list source; empty `files[]` means the official Files-rail Starred view is also empty after reload. |
| `files.favorites.add` | `file_id`, `collection_id` | `_x_reason=add_file_to_collection`. Unified Files `star-file-button` → Move to… **Starred**. Own snippet `F0BUVQHU6NL` → collection `Fs0BTURTUXK5` returned `{"ok":true}`. Slack template canvas returned `internal_error`. slk message menu **Add to Starred files** posts this after listing the starred collection. Same-session Redux marks the radio checked; that does **not** survive reload. |
| `files.favorites.remove` | `file_id`, `collection_id` | `_x_reason=remove_file_from_collection`. Same-session Move to… Starred when `aria-checked=true` (menu also shows **Remove from Starred**). `F0BUVQHU6NL` → `Fs0BTURTUXK5` returned `{"ok":true}`. slk message menu **Remove from Starred files**. |
| `files.favorites.list` | `type=all` | `_x_reason=starred_unified_files`. JS `maybeFetchStarredUnifiedFiles` (skipped when `custom_file_sections=on`). Live POST 2026-09-01 Test Workspace: `{ok:true, favorites:[], file_ids:[]}` even after `files.favorites.add` `ok=true`. slk lists `file_ids` reversed (JS `slice().reverse()`). `favorites[]` item JSON uncaptured (empty). |
| `files.recentlyDeleted` | (token only) | `_x_reason=get-deleted-files`. Response `{ok:true, files:[]}`. Files rail support; slk does not wrap it. |
| `files.getShares` | `file_id` | `_x_reason=file-shares-store.ConditionalFetchManager.fetch`. File peek. slk does not wrap it. |
| `files.info` (peek) | `file`, `page=1`, `count=500`, `truncate=true`, `public_shared=false`, `skip_shares=true` | `_x_reason=file-subscription.fetchFileInfo`. File id `F0BUVQHU6NL` (`slk-har-star.txt`). |
| `files.list` (hydrate) | `files` = comma-separated file ids | `_x_reason=files-store-unknown-fetch`. Browser hydrates known ids. Not the public `files.list` channel/user filter. |
| `client.dms` | `count=250`, `include_closed=true`, `include_channel=true`, `exclude_bots=true`, `priority_mode=priority` | `_x_reason=dms-tab-populate` (Files-rail reload) or `_x_reason=dms` (clicking the DMs rail tab). Response `{ims, mpims}`. slk’s DMs tab still uses boot cache + `conversations.history` limit=1; this form is attested, not wired. **`priority_mode` is the string `priority`, not a boolean**. |
| `search.modules.files` | `module=files`, `query` type filters, `page=1`, `count=50`, `sort=last_engaged`, `search_context=desktop_files_browser`, plus highlight/extract flags | Files rail (`_x_reason=fetch-current-browser`). Canvas probe: `query=type:quip creator:U…`, `count=1`, `_x_reason=user-created-canvas-query`. Empty workspace returned `items:[]`. |
| `search.inline` | `query`, `count=3`, `page=1`, `extract_len=110`, `from_me=true`, `with_me=true`, `recent_channels`, `search_session_id`, `client_req_id`, `max_ts`, empty `thread_replies` | `_x_reason=quick-messages/prototype`. Cmd+K message snippets. Envelope `{ok, query, pagination, items}`. slk wraps this. |
| `search.autocomplete.files` | `query`, `include_shares=true` | `_x_reason=omniswitcher:suggestions-from-searcher`. Cmd+K file suggestions. |
| `users.priority.list` | (token only) | `_x_reason=fetch-priority-users-on-boot-render`. Response `users.manual_provenance=[]`. |
| `im.list` | `get_latest=true`, `get_read_state=true` | `_x_reason=people-empty-state`. IM ids + `is_open` / `latest`. slk already uses boot/client.channels for IMs. |
| `activity.feed.scoreEntries` | `types` (long comma list), `limit=50`, `scoring_method=two_pass_min_msg_spp`, `score_threshold=0.8` | `_x_reason=fetch-score-entries`. Activity inbox scoring, **not** Home All Unreads scientifically sort. |

Deferred-data boot (full client reload, `_x_reason=deferred-data`, often **no** `_x_mode` on `client.init` / `client.extras`): `client.init` (`include_relevant_onboarding=true`), `client.extras` (`extras=cache_timestamps,features,…`), `client.channels` (`version_all_channels=false`, `return_all_relevant_mpdms=true`, `min_channel_updated`). slk continues to use `client.userBoot` for the first paint; these are documented, not a second boot path.

## Specs vs browser (do not mix)

There is **no** OpenAPI for the unofficial xoxc+cookie client. Public specs cover OAuth Web API only.

| Source | What it is | Use for slk |
|---|---|---|
| [docs.slack.dev/reference/methods](https://docs.slack.dev/reference/methods/) | Official Web API catalog (xoxb/xoxp, Enterprise admin) | Names/args **hints** only. Forms, `_x_reason`, and which method the **browser** actually posts can differ. |
| [slackapi/slack-api-specs](https://github.com/slackapi/slack-api-specs) (archived 2024) | OpenAPI 2.0 for public Web API + AsyncAPI Events | Same: public methods. No `_x_*` envelope, no `client.userBoot`, no `files.collections`. |
| [ErikKalkoken/slackApiDoc](https://github.com/ErikKalkoken/slackApiDoc) | Legacy-token “undocumented” methods (`users.prefs.get`/`set` as a **prefs JSON blob**) | Historical. Current web client `users.prefs.set` for recents is `name` + `value`, not that blob. Explicitly **out of scope** for web-UI/runtime-token methods. |
| [slack-ruby/slack-api-ref](https://github.com/slack-ruby/slack-api-ref) `methods/_undocumented` | Manually maintained since 2017: `chat.command`, `files.edit`, `files.share`, `users.admin.invite`, `users.admin.setInactive`, `users.prefs.get` | Same legacy set as ErikKalkoken. No `client.*`, no `_x_*`, no Files collections. |
| [karbassi/slack-mcp](https://github.com/karbassi/slack-mcp) [`docs/undocumented-endpoints.md`](https://github.com/karbassi/slack-mcp/blob/main/docs/undocumented-endpoints.md) | 2026-07 Chrome interceptor + live `xoxc`/`xoxd` probes on **their** test workspace. 101 distinct `/api/` + `/cache/` paths. Best public **session-method name list**. | Names and *some* args as **hints**. Forms, `_x_reason`, and which method the **current** web client posts can differ (see corrections below). Do **not** wrap from this list. |
| [rusq/slackdump](https://github.com/rusq/slackdump) `internal/edge` | Edge cache client + a handful of webclient forms (`im.list`, `conversations.view`/`genericInfo`, `search.modules.channels`, `files.list` channel page) | Confirms `edgeapi.slack.com/cache/{teamId}/…` JSON host. Their `_x_reason` strings (`guided-search-people-empty-state`, `browser-query`, `about-modal/sharedFiles`) are **their** captures, not ours. |
| [rgmz Grid HAR gist](https://gist.github.com/rgmz/25a62e3f49ef38f34161094e2b52937c) (also [a7993n](https://gist.github.com/a7993n/4db56cb679028a4414cc12319e708453)) | Enterprise Grid teardown: Flannel (edge), Loom (index), Gantry (SPA), Sonic (cold boot), Quip, HHVM | Architecture names for hosts we already send (`_x_gantry`, `_x_sonic`, `edgeapi`). Not method forms. |
| Official `admin.roles.addAssignments` | Enterprise: `role_id`, `entity_ids[]`, `user_ids[]`. **Channel Manager = `Rl0A`**. Error `no_valid_users` is documented. | Role id `Rl0A` matches the browser. The **web client** posts `admin.roles.addMembers` with `role_scopes`=channel id, not `addAssignments`. Do not swap the method name. |
| Official `stars.add` / `stars.list` | Docs now say **“Save an item for later. Formerly known as adding a star.”** | Explains the file overflow menu: **Save for later** (`saved.*`), not Add star. File “Starred” in the Files rail is `files.collections.list` `type=starred`. |

### Chinese / Gitee / GitCode

Searched Gitee, GitCode, CSDN, Zhihu, 掘金, 52pojie, 看雪 for `xoxc`, `client.userBoot`, `files.collections`, Slack 未公开/逆向 API. Result: **no session-method catalog**. Hits are (1) incoming-webhook bots, (2) English-language token-format explainers (`xoxc`+`d` cookie) rehosted or translated. Nothing to merge.

### karbassi vs this repo (corrections)

| They say | Official web client on Test Workspace (2026-09-01) |
|---|---|
| `client.dms` `priority_mode` boolean | `priority_mode=priority` (**string enum**), `count=250`, `include_closed=true`, `include_channel=true`, `exclude_bots=true`, `_x_reason=dms-tab-populate` |
| `files.favorites.list` `type` required = favorited files | Files rail **Starred** still posts `files.collections.list` (`files:[]`). Live `files.favorites.list` `type=all` `_x_reason=starred_unified_files` returns `{ok, favorites:[], file_ids:[]}`. JS skips the call when `custom_file_sections=on`. slk wraps the live list form. |
| `admin.roles.addAssignments` (Enterprise docs) | Browser: `admin.roles.addMembers` `role_scopes`=channel |
| `users.prefs.set` as a prefs blob (ErikKalkoken / slack-ruby) | Recents write is `name=recents` + JSON `value` |
| `client.boot` as session health-check | slk boots with `client.userBoot`. `client.init` / `client.extras` / `client.channels` are **deferred-data** after a full reload, not a second first-paint path. |
| `threads.getView` | slk’s followed-threads inbox is `subscriptions.thread.getView` (our HAR). `threads.getView` is a different, uncaptured method. |

### karbassi session methods vs slk

**Already wrapped here from our HARs** (do not re-wrap from their args): `client.userBoot`, `client.counts`, `client.shouldReload`, `conversations.view`, `conversations.history`, `conversations.listPrefs`, `conversations.mark`, `subscriptions.thread.getView` / `.mark` / follow, `activity.feed` / `activity.views`, `drafts.list` / `.create` / `.update` / `.delete`, `saved.list` / `.add` / `.delete` / `.update`, `users.channelSections.list` + writes, `users.prefs.get` / `.set` / `setNotifications`, `stars.add` / `.remove` / `.list`, `search.modules.files` (workspace search Files tab), `emoji.list`, reminders, pins, bookmarks, join/leave/create/invite/kick, `users.admin.inviteBulk`.

**Form captured here, not wrapped** (presence in OG, no slk product surface, or DMs already served another way):

| Method | Our form | Why unwrapped |
|---|---|---|
| `client.dms` | see table above (`_x_reason=dms-tab-populate` or `dms`) | DMs tab already lists IMs from boot + `conversations.history` limit=1 |
| `files.collections.list` | token; collection `type=starred`, `files:[]` | List source when `custom_file_sections=on`; empty `files[]`. Inbox uses `files.favorites.list` `file_ids` |
| `files.recentlyDeleted` | token, `_x_reason=get-deleted-files`, `files:[]` | Files-rail support, not a Home surface |
| `files.list` hydrate | `files=id,id,…` | Internal store fill; slk uses `files.info` / search |
| `im.list` | `get_latest=true`, `get_read_state=true` | Boot already has IMs |
| `users.priority.list` | token | People ranking; not a TUI row |
| `activity.feed.scoreEntries` | `scoring_method=two_pass_min_msg_spp` | Activity scoring, not Unreads scientifically |
| `client.init` / `.extras` / `.channels` | `_x_reason=deferred-data` | Not first paint |
| `search.modules.files` Files-rail | `search_context=desktop_files_browser`, `sort=last_engaged` | Workspace search Files tab already ships; this is the Files **rail** |
| `admin.roles.addMembers` | `role_id=Rl0A` | **Shipped** as `:manager U…`. Success extra JSON still uncaptured. |

**Named in karbassi, no form in our HARs** — hints only, do not wrap: `client.boot`, `threads.getView`, `saved.get`, `messages.list`, `search.modules.messages` / `.channels` / `.people` / `.dms`, `search.save`, `today.items.list` (they saw `unknown_method` where Today is not rolled out), `ai.alpha.summarize.unreadsSnapshot`, `ai.alpha.digest.list`, `emoji.collections.list`, `emoji.add` / `.remove` / `.adminList`, `conversations.suggestions`, `conversations.teamConnections`, `conversations.bulkReacjiTriggers`, `connectInvites.list`, `calendar.getInstalledCalendars`, `calendar.user.status`, `users.profile.getExtras`, `enterpriseSearch.getConnectors`.

**Named in official-client JS (gantry boot-async, 2026-09-01), form not HAR’d:** `files.collections.create` / `.update`. `files.favorites.list` / `.add` / `.remove` **are HAR’d** (see table).

**karbassi “Tier 3 skip” we actually wrap or captured:** `activity.views` (Activity tabs), `admin.roles.entity.listAssignments` (Channel Manager list). Their skip list is *their* product cut, not a protocol fact.

**Permanent non-goals** even when the name appears (canvas, lists, huddles, Slack AI, workflows, billing, megaphone, onboarding, Salesforce): `canvases.*`, `lists.*`, `huddles/*`, `aiApps.list`, `functions.workflows.list`, `workflows.triggers.*`, `retail.getAvailablePlans`, `payments.*`, `quip.lookupThreadIds`, `sfdc.*`.

### rusq edge paths (Flannel)

`POST https://edgeapi.slack.com/cache/{teamId}/{resource}/{action}` JSON body + `token`, `Content-Type: text/plain` (CORS simple). slk already uses this host for people search (`users/search`) and related lookups. rusq also wraps `users/info`, `users/list`, `users/counts`, `channels/info`, `channels/membership`, `channels/search`, `huddles/list`, `huddles/info`, `permissions/info`. karbassi independently lists the same set plus `emojis/list` on emoji-picker open. Huddles stay a non-goal.

## 2026-09-01 a11y map (Test Workspace, owner)

Rail: **Home / DMs / Activity / Files / Agents & tools / Admin**.

| Surface | A11y | APIs observed when opening |
|---|---|---|
| **Home** sidebar | Huddles, Directories, **Starred** (section), Channels, DMs, Invite people. **No** Activity/Later/Threads/Drafts/Unreads rows on this workspace (new IA). | Recents write on channel switch; `conversations.history` / `listPrefs` |
| **DMs** tab | Unreads filter, New message, search, list of IMs. Header: star, details, huddle, **Mute conversation**, search | `client.dms` (`count=250`, `priority_mode=priority`, `_x_reason=dms-tab-populate` or `dms` → `ims`/`mpims`), `conversations.history`, `listPrefs`, `bookmarks.list` |
| **Activity** | Tabs All / DMs / Mentions / Threads, **Filter unreads**, Filters, Search, Detailed/Dense | `activity.views`, `activity.feed` `mode=chrono_v1`, `activity.feed.scoreEntries` |
| **Files** | All files / Canvases / Lists / Starred. File overflow: Save for later, View raw, Copy link, Add to folder, Delete | `search.modules.files` (`search_context=desktop_files_browser`), `files.collections.list`, `files.recentlyDeleted`. Starred tab does **not** call `files.favorites.list`. |
| **Admin** menu | Workspace settings, Manage members, **Manage roles**, Slack Connect, Apps, exports, analytics | Not walked to a loaded admin page this session (menu only) |
| Channel details | About / Members / Agents / Automations / Tabs / Settings. **Managed by** → Channel Managers modal | `admin.roles.entity.listAssignments`, `admin.roles.addMembers` |

## Performance-resource method inventory (this session)

Unique `/api/` + `edgeapi` pathnames from `performance.getEntriesByType('resource')` on the owner client (66). **Presence ≠ form capture.** Do not wrap from this list alone.

`activity.feed`, `activity.feed.scoreEntries`, `activity.views`, `admin.advisor.recommendations.list`, `aiApps.list`, `api.features`, `bookmarks.list`, `calendar.getInstalledCalendars`, `canvases.getCannedTemplates`, `client.channels`, `client.counts`, `client.dms`, `client.extras`, `client.init`, `client.shouldReload`, `connectInvites.list`, `conversations.history`, `conversations.listPrefs`, `conversations.view`, `dnd.info`, `dnd.teamInfo`, `drafts.list`, `experiments.getByUser`, `features.access.policies.list`, `files.channelSettingsAllowDownloads`, `files.collections.list`, `files.info`, `files.list`, `files.recentlyDeleted`, `functions.workflows.list`, `help.issues.ticketStats`, `lists.getMyItems`, `lists.records.list`, `lists.templates`, `megaphone.notifications.list`, `megaphone.setNotificationAsSeen`, `onboarding.fetch`, `quip.lookupThreadIds`, `retail.getAvailablePlans`, `search.appDirectory`, `search.autocomplete.*`, `search.modules.files`, `search.precache`, `sfdc.integration.listOrgs`, `sharedInvites.canGetLink`, `team.targetingCriteria`, `teams.trials.info`, `ublockworkaround.history`, `users.customStatus.list`, `users.interactions.list`, `users.interactions.set`, `users.isEarlyJoiner`, `users.prefs.get`, `users.priority.list`, `users.profile.getSections`, `workflows.triggers.list`, `workflows.triggers.listRecentlyRunForUser`.

Edge: `channels/membership`, `huddles/info`, `huddles/list`, `permissions/info`, `users/counts`, `users/info`, `users/list`.

Permanent non-goals still skip huddles / canvas / lists / Slack AI / admin billing even when they appear here.

### Reminders

| Method | Form |
|---|---|
| `reminders.add` | `text`, `time` (unix / offset / phrase), optional `channel` |
| `reminders.list` | — |
| `reminders.complete` / `.delete` | id |

## WebSocket events consumed

`message` (incl. changed / deleted subtypes), `reaction_added` / `reaction_removed`, `presence_change` / `manual_presence_change`, `user_typing`, `dnd_updated`, `channel_marked` / `im_marked` / `group_marked`, `thread_marked`, `thread_subscribed` / `thread_unsubscribed`, `channel_joined` / `im_open` / `group_open` (conversation opened), `member_joined_channel` / `member_left_channel`, `pref_change`, `channel_section_upserted` / `_deleted`, `channel_sections_channels_upserted` / `_removed`.

## Residual divergences (known, not invented features)

| Divergence | Why it remains |
|---|---|
| Bearer fallback on `files.slack.com` | Cookie-only still 403/HTML on some workspaces; retry Bearer+cookie only then |
| `Accept-Encoding: gzip` vs Chrome’s four-codec list | stdlib auto-decompress |
| Generic `_x_reason` on `stars.*` (and other unmapped methods) | No per-method capture; empty reason is worse |
| Image DNT | Chrome sent `dnt:1` in those captures (user pref); slk does not |

## How a new method gets in

1. HAR the official **web** client (not a guessed public-API example, not karbassi/rusq/slack-ruby args).
2. Record method, form keys/values (string booleans), `_x_reason`, whether `_x_mode` is present, response keys with denominators (“14/14”), not one sample.
3. Wrap only that shape in `internal/slack`. Prefer `postForm` + `WithReason` when the reason is attested.
4. Golden-test against `official-request-shape.json` if it changes the envelope.
5. No live mutating “probe” calls. No invented prefs. Third-party method lists are discovery, not a contract.

See [[Gaps]] for methods we **know exist** but have not captured (recents `object_type` for DMs).
