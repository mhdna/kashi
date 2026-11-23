# Plan: a separate hidden screen on a till (read and write, one user at a time)

Builds on `docs/plans/live-view.md` (already shipped: kashi v. `feat/live-view`,
kashi-pos v1.1.189). The mirror shows what the cashier sees. This adds a
**second, hidden copy of the till app** that an admin drives on their own, so
the cashier keeps selling while the admin opens Balance, Reports, etc.

Branches: `feat/virtual-session` in kashi (new worktree) and in kashi-pos
(worktree `/Users/mahdi/Code/kashi-pos-live-view`, already on that branch name,
no commits yet). Never merge kashi-pos in the main folder (wails dev is live,
and it sits on another branch): merge from a worktree, as v1.1.189 was.
One commit per step, strong tests each (see both repos' CLAUDE.md/AGENTS.md).

## Findings (checked in the code, do not re-derive)

- Till router is `createWebHashHistory` (`frontend/src/router/index.ts`); the
  app loads from `index.html`, so an iframe `index.html?virtual=1#/` is a second
  independent app instance with its own router and Pinia.
- **Login is NOT frontend-only.** The screen keeps `currentUser` (`stores/auth.ts`),
  but Go keeps one `a.currentRole` too (`app.go:73`, set by the PIN and admin
  logins at `app.go:633/658`, restored by `resume.go:151`) and the shared PIN
  lockout counters (`failedPINAttempts`, `pinLockedUntil`). 14 server-side gates read
  it via `currentUserIsAdmin()` / `anyoneSignedInThisSession()`: expenses.go,
  branches.go (x3), products.go, attendance.go, management.go,
  conversion_report.go, sync_quick_access.go, records_cancel.go (x2),
  invoice_cancel.go, sync_branch_history.go. Left as is, an admin signing in on the
  hidden copy makes the cashier admin on all of these, and the cashier signing in
  demotes the admin. See "Per-copy sign-in" below.
- The hidden copy starts logged out: the admin logs in with their own PIN/password
  through the screen (attribution and PIN security stay right). Never copy the
  cashier's session.
- Because the iframe calls `parent.go`, **Go cannot tell which copy is calling**.
  Every binding is `window['go']['main']['App']['X'](...)` (`wailsjs/go/main/App.js`),
  so the bridge can swap in a proxy object and route each call where it wants.
- **Go events reach both copies** (about 25 names, e.g. `settings_changed`,
  `app_update_prompt`, `cash_machine_setup_done`, `clock_status_changed`), since the
  iframe's listeners live in the parent's runtime.
- **Focus can be stolen.** If code in the iframe calls `.focus()` (44 `focus()` /
  `autofocus` uses in `frontend/src`), focus moves into the iframe, and the
  cashier's barcode scanner and keyboard then type into the hidden copy.
- **localStorage is shared** by same-origin frames, and the sale draft lives there
  (`kashi-pos.invoice-draft`, `lib/draftSaver.ts`, also `home.vue`). The hidden
  copy must get a **prefixed localStorage** or it clobbers the cashier's cart.
- **Wails events and bindings only reach the main frame.** Go emits with `eval` in
  the main webview, and `window.go` (bindings) is only set there. In the iframe,
  before any app module runs, set `window.go = parent.go` and
  `window.runtime = parent.runtime` (wrap `EventsOnMultiple` to remember the
  cancel functions and call them on teardown). Do it in a side-effect module
  imported **first** in `main.ts`, because composables call `EventsOn` at import
  time (e.g. `usePageUnlocks.ts`).
- Things the hidden copy must NOT do (gate on a `isVirtual` flag): report the live
  cart to kashi (`live_cart.go` caller), mount toasts/banners/clock warning/update
  banner, play sounds, run the mirror's own `startLiveViewWiring`.
- The iframe is positioned off screen (`position:fixed; left:-10000px`) at the main
  window's `innerWidth` x `innerHeight`. Not `display:none` (layout needs size).
- rrweb must run **inside the iframe** to record its document, so the iframe's own
  app instance records (reuse `createLiveView` from `lib/liveView.ts`).

## Design

One more kind of session, keyed by `(register, virtual)`, next to the mirror.

Kashi (Go):
- `liveHub`: key sessions by `liveKey{ID int64, Virtual bool}`. The existing cap of 3
  counts both kinds. A virtual session accepts **one watcher**; a second user gets
  `{"t":"error","msg":"<name> is using this till's separate screen"}`.
- `GET /live/watch/:register_id?mode=virtual&token=`: needs live view on **and the
  control switch on** and System **edit** (it is read/write). Hello:
  `{"t":"hello","control":true,"virtual":true}`. Control messages always forwarded
  for a virtual watcher while the control setting stays on.
- Till stream: `GET /branch/live?mode=virtual`, a separate socket from the mirror's.
- Branch messages: `live_view_start` / `live_view_stop` gain `"virtual":true`
  (`branchWSMessage.Virtual`). Reconnect restart (`branch_ws.go`) covers both kinds.
- Control switch turned off: end every virtual session (stop message + close),
  unlike the mirror, which only loses control.
- `liveSessionInfo` gains `virtual bool`; the admin UI shows "Separate screen used by X".
- Activity feed: log opening a separate screen (user, register).

Admin UI:
- `WatchTillButton.vue`: second button "Open separate screen" when control setting
  is on and the role can edit System; the Watch button stays.
- `live-view.vue` with `?mode=virtual`: same replay, but the control overlay is
  always on (no "Take control" switch), title "Separate screen: <till>", and a note
  that actions are real and the cashier does not see them. Reuse `liveView.js` and
  `liveWatch.js` (add `virtual` to the hello parse and the watch URL).

kashi-pos Go (`live_view.go`):
- `liveView` gets a `prefix` ("live_view" or "live_virtual") used for every event
  name, and a `virtual` flag for the dial (`DialLive(ctx, virtual)` adds
  `?mode=virtual`). Keep the mirror's names and tests unchanged.
- `App.liveV()` second instance; bindings `LiveVirtualSend(batch)` and
  `LiveVirtualStatus()` (add to `wailsjs/go/main/App.js`, `.d.ts`, `models.ts`).
- `handleLiveMessage` routes by `msg.Virtual` (`wsMessage.Virtual`). A virtual start
  is ignored unless the hello offered `live_view_control`.
- Same safety: 2 minute silence stop, never block `send`.

kashi-pos Go, per-copy sign-in (`live_virtual_call.go`, its own commit, first):
- One binding `LiveVirtualCall(method string, args []json.RawMessage) (any, error)`.
  All of the hidden copy's calls go through it (the bridge's `go` proxy). It refuses
  calls while no virtual session is active, then dispatches through a table built once
  with `reflect` from `*App`'s exported methods (args are unmarshalled into the
  parameter types, and a wrong count or type is an error, never a panic).
- Separate state for the hidden copy, under `pinMu`: `virtualRole` plus its own
  lockout counters. Inside the dispatcher, the PIN and admin login methods and
  Logout run virtual twins that set only `virtualRole`. `SaveResumeRoute` and
  `TakeResumeSession` are no-ops. The session ending clears `virtualRole`.
- Gates, without a shared "who is calling" field (that would race with the
  cashier's calls): each gated method X becomes a thin `X(...)` that calls
  `x(isAdmin bool, ...)` with `currentUserIsAdmin()`, and the dispatcher's table maps
  X to `x(virtualIsAdmin(), ...)`. Same for `anyoneSignedInThisSession`.
- Denied from the hidden copy, because they act on the whole till: `SwitchBranch`,
  `CreateBranch`, `SetDefaultBranch`, `RestartApp`, `RestartComputer`,
  `StartCashMachineSetup`, app update install, and every `LiveView*`/`LiveVirtual*`
  binding. The error says "Not available on the separate screen".

kashi-pos UI:
- `src/virtualBridge.ts` (first import in `main.ts`, runs only when
  `?virtual=1`): set `window.go` to a proxy whose `main.App.X(...args)` calls
  `parent.go.main.App.LiveVirtualCall("X", args)` (never hand the iframe the raw
  parent bindings), bridge `runtime` to the parent, prefix `localStorage`
  (override `window.localStorage` with a prefixing wrapper; `sessionStorage` too),
  and expose `isVirtual`.
- Focus guard: the iframe gets `tabindex="-1"`, and the main frame listens for
  `focusin`/`blur`. When `document.activeElement` becomes the iframe, it puts focus
  straight back where it was. The cashier's keys and scanner must never reach the
  hidden copy.
- Go events in the hidden copy: go through the ~25 names once and gate each listener
  that drives UI meant only for the cashier (update prompt, cash machine, clock
  warning, device status) on `isVirtual`. Data refreshes (`settings_changed`,
  `sync_applied:*`) stay.
- `lib/virtualSession.ts` (main frame): on `live_virtual:start` create the hidden
  iframe (`index.html?virtual=1#/`), on `live_virtual:stop` remove it and call the
  remembered cancel functions; on load ask `LiveVirtualStatus` to resume.
- Inside the iframe: `startVirtualRecorder()` = `createLiveView` with
  `LiveVirtualSend` and the `live_virtual:*` events, started at boot with control on.
- Gate the mirror-only pieces in `App.vue` and the live cart reporter on `isVirtual`.
- No banner for the hidden copy (it is meant to be invisible); the mirror's
  "Being viewed" mark is unchanged.

## Tests (required)

kashi Go: one virtual watcher at a time (second refused with the user's name);
virtual refused when control setting is off or role lacks System edit; mirror and
virtual on the same register at once, each with its own stream, frames never
cross; cap counts both; control switch off ends virtual sessions and stops the
till; till reconnect restarts both; randomised relay identity test over mixed
sessions. Fixture: add the virtual hello and start messages to
`ui/src/test/fixtures/live_view_protocol.json` and read them in `liveView.test.js`.

kashi UI: button shown only with the control setting and edit right; the page in
virtual mode always sends control, shows the title and the note, and refuses a
second user's error.

kashi-pos Go: stand-in kashi test for the `?mode=virtual` dial, event prefix,
no cross-talk between the two `liveView` instances, ignored without
`live_view_control`.

kashi-pos Go, per-copy sign-in:
- The reported case first: admin signs in on the hidden copy while a cashier is signed
  in; every one of the 14 gates still refuses the cashier and allows the admin. And
  the other way round (cashier signs in again, admin keeps admin).
- Wrong PINs on one copy never lock the other out.
- Guard test with `go/ast`: every function that calls `currentUserIsAdmin`,
  `anyoneSignedInThisSession` or reads `currentRole` is reached from an exported
  method that is in the dispatcher's twin table or the deny list. A new gate added
  later fails this test until it gets a twin.
- Deny list: each name exists on `*App` (catches renames), and each is refused.
- Dispatcher: unknown method, wrong arg count or type, and no active session are all
  errors; a randomized test runs hundreds of interleaved cashier/virtual logins and
  gated calls from goroutines (`-race`) and checks that each side sees only its own role.

kashi-pos UI: bridge sets `go`/`runtime` from a fake parent and prefixes storage
(the cashier's `kashi-pos.invoice-draft` is never read or written by the iframe);
iframe created on start and removed on stop with listeners cancelled; the live
cart reporter and toasts do not run when virtual; rrweb still not loaded until a
session starts. The proxy sends every `App` call through `LiveVirtualCall` (no
direct parent binding is reachable). Focus guard: a `.focus()` inside the iframe
leaves the main frame's focused input focused.

Prove the tests bite (break the prefix, break the one-user rule, point a twin at
`currentUserIsAdmin`, remove the focus guard; see failures).

## Risks to check on a real till (cannot be tested here)

- WebView2 on Windows: that `window.parent.go`/`runtime` work from the iframe and
  that the asset server serves `index.html?virtual=1`. If the bridge fails, the
  fallback is a second route in the same page instead of an iframe.
- Throttling: the engine may slow timers, `requestAnimationFrame` and transitions
  in an off-screen iframe, which can stall dialogs and route transitions in the
  hidden copy. If it does, place the iframe on screen at `inset:0; z-index:-1;
  pointer-events:none` under the main app's opaque background.
- Focus on a real till: scan a barcode while the admin types in the hidden copy;
  the scan must land in the cashier's sale.
- Memory: a second app instance. Measure on a till and note it in
  `docs/PERFORMANCE.md`; keep at most one hidden screen per till.
- Printing and hardware actions from the hidden copy are real (receipt prints on the
  till's printer). Say so on the admin page.

## Release

Same as live view: kashi first (deploy from a clean worktree of master), then
kashi-pos from a worktree (`make build-windows` clean, `make bump`, push); the user
publishes. Add a "Separate screen" section to both repos' live view docs and run
`make dev-docs` in each.
