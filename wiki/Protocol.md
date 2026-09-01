# Protocol

What this fork knows about Slack’s **unofficial browser client** protocol, as implemented. A public Web API name is not enough; the official client’s form, `_x_reason`, envelope, and response shape are the contract.

Product gaps (create channel, invite, mentions-only, starred files, …): [[Gaps]]. Features: [[Features]].

Raw HARs are **not** in the repo (live `xoxc` tokens, `d` cookies, message bodies). Digests and golden tests are.

Last reviewed: 2026-09-01.

## Evidence

| Capture | When | Workspace | What it pinned |
|---|---|---|---|
| 8 HARs, Chrome 150 Linux | 2026-07-30 | Grid (`rands-leadership.slack.com`) | Headers, query envelope, `_x_reason` / `_x_mode` presence, `client.userBoot`, `conversations.view`, `conversations.history` (`limit=28`, `cached_latest_updates`), `client.counts`, `client.shouldReload`, edgeapi, WebSocket upgrade |
| Official web Activity / Later / Drafts / Unreads | 2026-08-31 | Obvious AI (`T099JCA82HJ`) | `activity.feed` / `activity.views`, `saved.*`, `drafts.*`, Unreads `conversations.history` window |
| Threads socket | 2026-08-02 coldboot | — | WebSocket `start_args` (typing delivery) |

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
| no | yes | **0** | never observed — slk must not emit this |

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
| `bookmarks-store/conditional-fetching` (in the 48) | `bookmarks.list` | July 30 |
| `prefs-store/setChannelNotificationOverride` | `users.prefs.setNotifications` | public mute captures; **not** in slk’s 2026-07-30 HARs |

**Fallback:** methods with no table entry and no `WithReason` get `_x_reason=conditional-fetch-manager`. That **string** is attested (sections.list). The **pairing** with e.g. `stars.add` or `activity.feed` is a guess. Emitting nothing is a sharper fingerprint (`_x_mode` without `_x_reason`). `activity.feed` currently uses this fallback (no call-site reason), even though comments mention `fetchActivityFeed`.

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

**Residual:** slk still sends `Authorization: Bearer xoxc-…` on `files.slack.com`. 0/40 captured image requests had `Authorization`. Cookie-only 403’d in production; this is the largest known image-path divergence. See `official-request-shape.json` `authorization_note`.

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
| `GetHistory` / `GetOlderHistory` / `GetHistoryAround` / `GetHistorySince` | **slack-go** — different limit (50/200/500). Residual vs official 28 |

| Method | Form |
|---|---|
| `conversations.mark` | `channel`, `ts` — read or roll back (unread) |
| `subscriptions.thread.mark` | `channel`, `thread_ts`, `ts`, `read` = `"1"` / `"0"` |

### Home surfaces (2026-08-31 unless noted)

| Method | Form | Notes |
|---|---|---|
| `activity.views` | (empty) | Built-in All/DMs/Mentions/Threads + custom views. Selecting a tab does **not** send `view_id`; flatten `entry_types` / `unread_only` / `priority_only` onto the feed |
| `activity.feed` | `limit` (1–100, default 50), `types` (comma list; All-tab list is duplicated as captured), `mode` = `chrono_v1` or `priority_reads_and_unreads_v1`, `archive_only=false`, `unread_only`, `priority_only`, `only_salesforce_channels=false`, `exclude_automations=false`, `automations_only=false`, `is_activity_inbox=true`, optional `sort=vip_unreads_first` | Today’s `_x_reason` is the generic fallback |
| `saved.list` | `limit`, `filter` = `saved` \| `completed` \| `archived`, `include_tombstones=true`, optional `cursor` | `item_type=message` rows used; other types skipped |
| `saved.add` / `.delete` | `item_id` (channel), `item_type=message`, `ts` | |
| `saved.update` | same + `date_due` or `state` = `in_progress` \| `completed` \| `archived` | |
| `drafts.list` | `is_active=true`, `limit`, `next_ts` (not `cursor`) | Page while `has_more` |
| `drafts.listActive` | `limit=1000` | Badge = `active_draft_ids` |
| `drafts.create` | `blocks`, `destinations`, `file_ids`, `attachments`, `client_msg_id`, `is_from_composer=true` | |
| `drafts.update` | + `draft_id`, `client_last_updated_ts` | |
| `drafts.delete` | `draft_id`, `client_last_updated_ts` | |
| `stars.list` | `limit=1000` | `type=channel` → sidebar Starred **section**; `type=message` → inbox (`channel`, `date_create`, `message.{ts,user,text}`). `type=file` / `type=im` unused. Paging fields present; next-page form **not captured** |
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
| `subscriptions.thread.getView` | `limit=100`, `fetch_threads_state=true`, `priority_mode=all`, optional `current_ts` = previous `max_ts` |
| `subscriptions.thread.add` / `.remove` | `channel`, `thread_ts`. Idempotent: `already_subscribed` / `not_subscribed` |

Capture notes (2026-05): official client also sent `limit=8` with `_x_reason=fetch-threads-view-via-refresh` / `…-load-more`. slk uses 100 + generic/default reason.

### Prefs / mute / status

| Method | Form |
|---|---|
| `users.prefs.get` | — | Boot also embeds `all_notifications_prefs` JSON (`channels[id].muted`, `suppress_at_channel`). Mentions-only **write** not captured |
| `users.prefs.setNotifications` | `name=muted`, `value` true/false, `channel_id`, `global=false`, `sync=false` | Only captured write |
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

`chat.postMessage` (incl. `thread_ts`, `reply_broadcast` / `thread_broadcast`), `chat.update`, `chat.delete`, `chat.scheduleMessage`, `chat.scheduledMessages.list`, `chat.deleteScheduledMessage`, `reactions.add` / `.remove`, `pins.add` / `.remove`, `pins.list` (also hand-rolled `PostForm` for header chips: `channel=`), `bookmarks.list` (`channel_id`), `conversations.join` / `.leave`, `conversations.close` (`channel`), `conversations.open` (user ids or reopen `channel`), `conversations.replies`, `files.upload` V2, `auth.test`, `emoji.list`, permalink.

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
| `Authorization: Bearer` on `files.slack.com` | Cookie-only failed with 403 in prod; HAR stripped cookies |
| `Accept-Encoding: gzip` vs Chrome’s four-codec list | stdlib auto-decompress |
| Generic `_x_reason` on most mutating calls (`stars.*`, `activity.feed`, …) | No per-method capture; empty reason is worse |
| slack-go history limits 50/200/500 vs official 28 | Incremental `HistoryWithVersions` used for Unreads + some sync; other paths not fully migrated |
| `GetStarredChannels` / `GetCounts` / some mark paths build HTTP by hand | Same token-in-body + cookie client; still go through `BrowserTransport` if that client is wired |
| `subscriptions.thread.getView` `limit=100` vs captured `8` | Pagination still uses captured `current_ts` / `max_ts` |
| Image DNT | Chrome sent `dnt:1` in those captures (user pref); slk does not |

## How a new method gets in

1. HAR the official **web** client (not a guessed public-API example).
2. Record method, form keys/values (string booleans), `_x_reason`, whether `_x_mode` is present, response keys with denominators (“14/14”), not one sample.
3. Wrap only that shape in `internal/slack`. Prefer `postForm` + `WithReason` when the reason is attested.
4. Golden-test against `official-request-shape.json` if it changes the envelope.
5. No live mutating “probe” calls. No invented prefs.

See [[Gaps]] for methods we **know exist** but have not captured (`conversations.create`, invite, mentions-only pref name, `stars.list` `type=file` item, Unreads extra sorts).
