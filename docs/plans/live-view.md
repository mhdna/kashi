# Plan: live view and remote control of tills

Goal: from kashi, watch any connected till's screen live and, if allowed,
click and type on it, so AnyDesk is only needed for things outside kashi-pos.
Model it on bnc-rewrite's live view (`bnc-rewrite/backend/internal/live/hub.go`,
`bnc-rewrite/dashboard/src/lib/live.ts`, `bnc-rewrite/dashboard/src/pages/LivePage.vue`):
the page's DOM is streamed with rrweb and rebuilt in the viewer, no video.

Two repos, one branch in each: `feat/live-view` in `kashi` and in `kashi-pos`.
Merge with `git merge --no-ff`. One commit per finished step, tests in the
same commit. Read `kashi/CLAUDE.md` and `kashi-pos/AGENTS.md` first.

## Hard rules

1. **Off means zero cost.** Two settings in kashi, both default off:
   `live_view_enabled` and `live_view_control_enabled` (control needs view on).
   When view is off: kashi does not advertise the feature, refuses the watch
   endpoint, opens no stream sockets, starts no goroutines; the till never
   loads rrweb (it is a lazy chunk, `import('rrweb')` only on start) and opens
   no extra socket. Nothing runs on the till or server until someone watches.
2. **The existing branch socket stays as it is.** `api/branch_ws.go` is push-only
   and carries sync pushes. Do not stream over it. It only carries two new
   small messages (`live_view_start`, `live_view_stop`). The stream gets its own
   socket, opened only while watched, so a big snapshot never delays a sync push.
3. **Nothing is stored.** The server relays in memory. No DB rows for frames.
4. **Never slow the till.** The till drops stream data rather than block; the
   recorder stops when nobody watches, and on its own after 2 minutes with no
   watcher.
5. **At most 3 tills watched at once** (the droplet has 1 CPU, 2 GB shared with
   bnc). A 4th watch gets a clear error message.

## Flow

```
admin browser --WS /api/live/watch/:register_id--> kashi (liveHub, memory only)
kashi --branch WS {"type":"live_view_start"}--> till Go
till Go --Wails event--> till UI: import('rrweb'), record()
till UI --Wails binding LiveViewSend(batch)--> till Go
till Go --WS /branch/live (branch auth, permessage-deflate)--> kashi --> every watcher
watcher click/type --> kashi (only if control on + permission) --> till Go --> till UI applies it
last watcher leaves --> kashi sends live_view_stop, closes /branch/live --> till stops recording
```

Credentials stay in Go on the till; the webview never talks to kashi directly.

## kashi (server and admin UI)

### 1. Settings (migration + API)
- Check the latest migration number on `master` right before adding (it was
  255 when this plan was written; a clash once bricked tills). New migration:
  `ALTER TABLE app_settings ADD live_view_enabled boolean NOT NULL DEFAULT false,
  ADD live_view_control_enabled boolean NOT NULL DEFAULT false`.
- Add a small query `SetLiveViewSettings` in `db/query/app_settings.sql`
  (do not widen `UpdateAppSettings`). sqlc regen touches unrelated files: keep
  only your hunks, regen the mock.
- Endpoint `PUT /app_settings/live_view` (access area `system`, add to
  `api/access.go`, and an activity label in `api/admin_activity.go`).
  Turning control on while view is off is a 400.
- Keep both flags in memory on the server (load at start, update on PUT) so
  the hot paths never hit the DB.
- On turning view off: send `live_view_stop` to every watched till, close all
  live sockets. On turning control off: drop control rights of open watchers
  at once.

### 2. Advertise to tills
- In `branchWS` hello, add feature `live_view` only when enabled. When the
  setting flips, push `{"type":"live_view_settings"}` via `broadcastAll` so
  tills re-read (or simply include the flag in that message). Add fields to
  `branchWSMessage` as needed (`Control bool` for whether control is allowed).

### 3. liveHub (`api/live_hub.go`, new)
- `map[registerID]*liveSession`; session = till stream conn (may be nil until
  the till connects) + set of watchers (admin user id, name, canControl).
- `POST`-less: watcher opens `GET /api/live/watch/:register_id` (admin auth,
  access area `system`; control also needs `system` write right). Steps:
  setting on? register connected (`branchHub`)? fewer than 3 sessions or this
  register already has one? Then add the watcher and, if this is the first
  watcher, `branchHub.notifyRegister(id, live_view_start)`. If not first, ask
  the till for a fresh full snapshot (`{"t":"snapshot"}`) so the new watcher
  can render.
- Till opens `GET /branch/live` (behind `branchAuthMiddleware`). Refused (409)
  if no session waits for that register. Enable `permessage-deflate` on this
  upgrader only (`EnableCompression: true`); JSON from rrweb shrinks ~5-10x.
  Read limit 8 MB (a full snapshot can be several hundred KB).
- Relay: till message bytes are forwarded as-is to each watcher (do not parse
  rrweb events in Go). Each watcher has a buffered send channel (256). If it is
  full, drop that watcher's queued frames and ask the till for a full snapshot
  so it resyncs; never block the till reader on a slow watcher.
- Control messages from a watcher are forwarded to the till only if the
  setting and that watcher's right allow it; otherwise dropped silently.
- Last watcher leaves: send `live_view_stop`, close the till stream, delete
  the session. Till disconnects: tell watchers `{"t":"ended"}`, keep them
  waiting 30 s for a reconnect, then close.
- Reuse `wsConn` and `startHeartbeat` from `api/ws_conn.go`.
- Push session changes to admin UIs (`adminHub.broadcastAll`, type
  `live_view_sessions`: which registers are watched and by whom) so the till
  list can show "watched by X".
- Log each watch start and each control session in admin activity (who,
  which register, when). Not each click.

### 4. Admin UI (`ui/`)
- `ui/src/pages/settings.vue`: two switches, "Live view of tills" and
  "Allow remote control" (second disabled while first is off). Short hint
  under each. No em/en dashes, no chips.
- Till list (`ui/src/components/ConnectedBranchesCard.vue` / `branches.vue`):
  a "Watch" button per connected register, shown only when the setting is on.
- New page `ui/src/pages/live-view.vue` (route `/live/:registerId`):
  `import('rrweb')` lazily, `new Replayer([], { liveMode: true, mouseTail: false })`,
  `replayer.startLive()`, feed events with `addEvent`. Scale the till's screen
  to fit the panel width. Show connection state: waiting for till, live,
  till disconnected. A "Take control" toggle (only if control is on and the
  user may): when on, capture clicks and key presses on the replay iframe and
  send them (see protocol). Activity list beside it, like bnc's.
- Put protocol encode/decode, batching and click-to-node mapping in a plain
  module `ui/src/utils/liveView.js` and test it exhaustively.
- Icons: treeshaken `@mdi/js`, registered in `ui/src/plugins/icons.js`.
- Add `rrweb` with pnpm, pinned exact (bnc uses `2.2.1`; use the same on both
  sides so recorder and replayer agree).

## kashi-pos (till)

### 5. Go side (`live_view.go`, new)
- In `handleWSMessage` (`sync_ws.go`): on `live_view_start` emit Wails event
  `live_view:start` (with `control` flag) and open `/branch/live` with the same
  auth headers as `DialWS` (`sync_client.go`). On `live_view_stop` emit
  `live_view:stop` and close. Ignore both if the hello had no `live_view`
  feature (older or off kashi).
- Binding `LiveViewSend(batch string) error`: non-blocking hand-off into a
  channel (cap ~64). If full, drop the batch and set a flag so the UI takes a
  full snapshot on the next tick. A single writer goroutine writes to the
  socket; it exists only while the stream is open.
- Messages from kashi on the stream: `{"t":"snapshot"}` → emit
  `live_view:snapshot`; control messages → emit `live_view:control`.
- Safety stop: if the stream is closed or no message arrived from kashi for
  2 minutes (kashi pings), stop and emit `live_view:stop`.
- No till DB migration is needed.

### 6. UI side (`frontend/src/lib/liveView.ts`, new)
- On `live_view:start`: `const { record } = await import('rrweb')`, start with
  `maskInputOptions: { password: true }`, `sampling: { mousemove: 50, scroll: 150, input: 'last' }`,
  `inlineStylesheet: true`, `blockSelector: 'img, video, canvas'` (images become
  same-size placeholders: no photo bytes over the wire), `checkoutEveryNms: 5 min`.
  Batch events every 100 ms and call `LiveViewSend`.
- Mask the PIN entry and any credential screens with `rr-mask` / `rr-block`
  classes. Check the login, PIN, settings keys and payment card screens.
- On `live_view:snapshot`: `record.takeFullSnapshot(true)`.
- On `live_view:stop`: call the stop function, drop references.
- Control (only when start said `control: true`): apply `{"t":"click","id":n}`
  via `record.mirror.getNode(id)` → `el.focus(); el.click()`;
  `{"t":"input","id":n,"v":"..."}` → set value and dispatch `input`/`change`;
  `{"t":"key","key":"Enter"}` → dispatch `keydown`/`keyup` on the focused
  element. Unknown ids are ignored. While controlled, show a clear top banner
  on the till: "Kashi is controlling this screen (name)". While only watched,
  a small unobtrusive "Being viewed" mark.
- Keep all of this in the lazy chunk so the main bundle does not grow.

## Protocol (JSON text frames)

| Direction | Message |
|---|---|
| kashi → till (branch WS) | `{"type":"live_view_start","control":bool}`, `{"type":"live_view_stop"}`, `{"type":"live_view_settings",...}` |
| till → kashi (stream) | `{"e":[rrweb events],"a":[actions]}` (actions as bnc's `LiveAction`) |
| kashi → till (stream) | `{"t":"snapshot"}`, `{"t":"click","id":n}`, `{"t":"input","id":n,"v":s}`, `{"t":"key","key":s}` |
| kashi → watcher | relayed till frames, `{"t":"waiting"}`, `{"t":"ended"}`, `{"t":"error","msg":s}` |
| watcher → kashi | the same control messages as kashi → till |

Write a Go test that emits one of each message to a JSON fixture under
`ui/src/test/fixtures/` and a UI test that decodes it, so a shape change on
one side fails the other.

## Tests (required, both repos)

kashi Go (`api/live_hub_test.go`):
- setting off: watch endpoint refused, hello has no `live_view`, no message
  sent to the till, `/branch/live` refused;
- on: first watcher triggers exactly one `live_view_start`; second watcher
  triggers a snapshot request, not a second start; last leave sends stop;
- cap: 4th distinct register refused, a second watcher on a watched register
  allowed;
- control: dropped when setting off, when watcher lacks the right, and right
  after the setting is turned off mid-session; forwarded otherwise;
- slow watcher: filling its queue does not block the till reader or other
  watchers, and triggers a snapshot request;
- till disconnect: watchers get `ended`; reconnect within 30 s resumes;
- turning view off closes every session;
- permissions: a user without `system` gets 403; another branch's register id
  is fine for an admin but an unknown id is 404;
- `go test -race` on the hub.

kashi UI (vitest, happy-dom): `liveView.js` encode/decode, batching, control
message building; mount `live-view.vue` with a fake socket: waiting, live,
ended states; control toggle hidden when not allowed; settings switches
disable control while view is off.

kashi-pos Go: start/stop messages open/close the stream; ignored without the
feature; `LiveViewSend` never blocks when the socket is stalled (test with a
server that never reads); safety stop after silence.

kashi-pos UI (vitest): recorder loads only on start (assert the dynamic
import is not called before), stops on stop, control applier clicks/types on
the right node and ignores unknown ids, banner shows only while controlled.

Prove the tests bite: break the off-switch and the cap once, see tests fail,
restore.

## Measure before calling it done

On a dev setup with a real kashi-pos build, watch one till for 5 minutes of
normal use and record: kashi-api RSS before/during/after, bytes per second on
the stream, till CPU in Task Manager with and without a watcher. Expected:
under 20 MB extra on kashi while watched, a few KB/s, no visible lag on the
till. Write the numbers in `docs/PERFORMANCE.md`.

## Docs

Add a "Live view" page to `docs/DEV_DOCS.md` (glance list: off by default and
free when off, separate stream socket, memory only, cap 3, control rules),
run `make dev-docs`, commit `docs/dev-docs.html` with it.

## Release

kashi deploys first (old tills ignore the new messages and never see the
feature). Then bump, tag and push kashi-pos; the user builds and publishes.
Run `make build-windows` in a clean worktree on that commit before saying it
is ready.

## Out of scope

Recording history, watching anything outside kashi-pos, file transfer, and
the Windows desktop. AnyDesk stays for those.
