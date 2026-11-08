# Kashi developer docs

The central back office of a multi-branch retail chain: a Go API, PostgreSQL and a Vue 3 web app. Tills ([kashi-pos](https://github.com/mhdna/kashi-pos)) sync with it. This is the *how it behaves* reference; `README.md` has setup and `docs/FIXED_BUGS.md` the bug history.

**Living document:** when a change adds something a developer must know, update the page in the same commit, then run `make dev-docs` and commit the HTML too.

::: group Start here

## Overview {icon=mdiCompass}

::: glance
- **Tills are the truth about what happened in a shop** (sales, shifts, attendance). **Kashi is the truth for shared data** (products, prices, clients, settings, users).
- Rows match across the two by a shared **`uid`**, never by integer id.
- A **`/branch/*`** API serves tills (per-register key); everything else is the admin API (PASETO login + role checks).
- Changes reach tills by a WebSocket **pointer** (`sync_changed`), then the till pulls. Payloads never ride the socket.
- Migrations must stay **additive**: an older till build runs against a newer database.
- A route missing from `routeRules` is **admin-only** (fails closed).
:::

::: diagram architecture

| Where | What |
| --- | --- |
| `main.go` | Starts the HTTP API, gateway and background jobs. |
| `api/` | Gin handlers, middleware, access rules, WebSockets. `server.go` lists every route. |
| `db/migrations` | Numbered golang-migrate SQL, 218 pairs so far. |
| `db/query` | sqlc input, one file per area. `db/sqlc` and `db/mock` are **generated, never edited**. |
| `token/` | PASETO login tokens. |
| `util/` | Config, logging, image optimisation, helpers. |
| `gapi/`, `proto/`, `pb/`, `doc/swagger` | gRPC server and gateway (little used; the web app and tills use Gin). |
| `cmd/` | One-off tools: seed, imports, image optimisation, release notes. |
| `ui/` | The Vue 3 + Vuetify app. `src/pages`, `router`, `data/navItems.js`, `plugins/icons.js`, `deploy/`. |

## Running and deploying {icon=mdiPlayCircle}

::: glance
- Local: `make postgres`, `createdb`, `migrateup`, copy `example.app.env` to `app.env`, `make server` (or `make dev`), then `cd ui && pnpm dev`.
- `VITE_API_URL` must equal the API's `HTTP_SERVER_ADDRESS` in dev, and be `/api` in production.
- `app.env` holds secrets and is not committed; env vars of the same name override it.
- Deploy needs the SSH alias `kashi`. `make deploy` = API + UI.
:::

| Command | Does |
| --- | --- |
| `make server` / `make dev` | Run the API once / with live reload (air). |
| `make test` | Go tests. `go test ./api/` for just the API. |
| `make sqlc` | Regenerate `db/sqlc` from queries and migrations. |
| `make mock` | Regenerate `db/mock/store.go` after the `Store` interface changes. |
| `make migrateup` / `migratedown1` | Apply all / roll back one. |
| `make seed` | Demo data. |
| `make proto` | gRPC code + Swagger. |
| `cd ui && pnpm lint` / `pnpm build` | Lint-fix / build to `ui/dist`. |
| `cd ui && pnpm test` | UI tests (vitest + happy-dom; config `ui/vitest.config.mjs`, Vuetify helper `src/test/vuetify.js`). |

### Configuration (`app.env`)

| Key | Meaning |
| --- | --- |
| `DB_SOURCE` | PostgreSQL URL (`DB_DRIVER=postgres`). |
| `HTTP_SERVER_ADDRESS` | API listen address, e.g. `0.0.0.0:8090`. |
| `TOKEN_SYMMETRIC_KEY` | 32 chars signing login tokens. **Random in production.** |
| `ACCESS_TOKEN_DURATION` / `REFRESH_TOKEN_DURATION` | e.g. `15m` / `24h`. |
| `*_STORAGE_DIR` | Expense, shift-receipt, transfer photos and app releases (under `./storage`). |
| `GEOIP_DIR` | IP location databases for Security (auto-downloaded). |
| `APP_LOG_PATH` | Rotating log, default `logs/kashi.log`. |

Business settings (WhatsApp credentials, templates, webhook secret) live **in the database**, edited under Settings.

### Deploying

::: diagram deploy

> [!NOTE] While the API restarts, open browsers show an "Updating" screen and return on their own. After a UI-only deploy, open tabs notice `version.json` and reload. Production runs behind Caddy (`ui/deploy/` has the Caddyfile, systemd unit, setup notes) on a server shared with kashi-commerce.

`make migrate-prod` runs only pending migrations. `make deploy-notes` posts shipped commits to the home release feed.

::: group Core concepts

## Till sync {icon=mdiSync}

::: glance
- Auth: headers `x-branch-code` + `x-branch-key` (a register key; **never expires**, a till may be offline for weeks). Not `Authorization: Bearer`, so the two schemes cannot be confused.
- **Push** (till to kashi): invoices, shifts, attendance, expenses, loans, transfers. **Pull**: `GET /branch/sync/changes?since=` per entity.
- Paged only for the two unbounded entities (clients, variants): `limit` + opaque `cursor` `<updated_at>|<id>`.
- A **completeness check** compares every uid, because a cursor only sees changes.
- Conflicts are **staged** for an admin; a push never fails because staging failed.
:::

::: diagram syncChanges

| Topic | Rule |
| --- | --- |
| Identity | Rows carry a `uid`. A till's local ids mean nothing here; invoice lines reference the till's own ids as plain values. |
| Catch-up | `since` omitted = a full snapshot (a fresh till needs no bootstrap). `cursor` supersedes `since`. An old till that omits `limit` still gets one unbounded answer. |
| Completeness | `/branch/sync/ids` lists uid + `updated_at` (inactive rows included, so a till learns a row went inactive); `/branch/sync/fetch` returns the missing rows. |
| Cursor overlap | `server_time` from catch-up is 2 min behind now (`syncCursorOverlap`): `updated_at` is written before commit, so a pull in between would pass a change it can't see yet. Stamp list rows with `clock_timestamp()`, not `now()`. |
| Lists | A price or discount list the till must stop using is sent **active and empty**, never inactive: tills delete an inactive list, which fails once it has sales and rolls back the whole pull. |
| Sync status | `branch_register_sync_status` (000223) keeps each till's last pull per entity, shown on the Commands page. The same `since` again after a pull that sent items is a failed apply (`retries`, `stuck_since`); two repeats reads as stuck. Tills add the error and their outbox counts via `POST /branch/sync/report`. |
| List delivery | `GET /discount_lists/:id/delivery` (and price lists) compares a fingerprint of what each till holds (`api/list_fingerprint.go`, rows and a hash, same recipe as kashi-pos `sync_report.go`) with what it should hold. A cursor past the change with a different hash is `different`. A till too old to report is judged by its cursor, with `verified: false`. |
| Live push | Postgres `NOTIFY kashi_sync` (migration 000109) batches into one `sync_changed`; a lost listener sends `pull_all`. Any write triggers it: handler, script or SQL. |
| Full sync | A till's "Sync everything" holds a **lease** (renewed every 30 s) so the admin UI shows the branch as syncing; end, closed socket or missed renewal ends it. |
| History | A fresh till can pull its own past invoices, expenses, loans and shift closes (`branch_history_sync`). |
| Registers | Each till has its own key, shown **once** at creation or rotation (`bcrypt` hash stored). The code is fixed: it is printed in every invoice number (`SA/<code>/<year>/<n>`). |

> [!WARNING] Two things look like bugs and are not: `UnitPrice`/`LineTotal` have no `required` tag (0 is a valid amount, e.g. 100% discount), and returns use **negative quantities** (`required` accepts them). The till's return route also carries **exchanges**; kashi sends exchanges back down as `return`.

> [!NOTE] The socket hub is an in-memory map, so it assumes **one kashi instance**. Horizontal scaling would break live push (catch-up still works).

## Money and time {icon=mdiCash}

::: glance
- Money is **integer cents** (`BIGINT` amounts; `amount_cents` in payloads). The only exception is anything explicitly noted otherwise in a field name.
- Timestamps are stored as **UTC instants**; calendar days are computed in **Asia/Beirut** (`businessZone`).
- Each branch has its own `branches.timezone` for branch-scoped views.
:::

- Report SQL spells the zone `'Asia/Beirut'`; Go code uses `businessToday` / `dayStart` (UTC-midnight `date` values, `timestamptz` bounds).
- Sale payloads are signed cents: positive revenue, negative refund, an exchange carries its net.
- Report **net sales leave out deferred coupons** (coupon types with `calculate_on_checkout`), like kashi-pos: `branch_invoice_deferred_coupons(invoice id)` finds the coupon through `coupons.redeemed_invoice_ref = branch_invoices.client_ref` and returns the "On Coupon" payment, capped at the invoice total. Plain coupons count. Reports in `db/query/reports.sql` subtract it; the dashboard and e-mail reports do not yet.
- The till reads clock skew from `/branch/health` (kashi's clock and the branch zone).

## Database and migrations {icon=mdiDatabase}

::: glance
- **Additive migrations only** (add, never drop or rename): old tills must keep working.
- Workflow: migration pair, then `db/query/*.sql`, then `make sqlc` (+ `make mock` if `Store` changed).
- Multi-statement work lives in `Store` methods ending `Tx` (`SalesInvoiceTx`, `CreateBranchInvoiceTx`, ...).
- API tests use the generated **mock store**; Postgres integration only where noted.
:::

| Rule | Why |
| --- | --- |
| `000140_x.up.sql` + `.down.sql`, next number | golang-migrate; prod runs pending ones on deploy. |
| Never edit `db/sqlc` or `db/mock` | Generated. |
| Add new write flows as `...Tx` on `Store` | Atomic, mockable. |
| Triggers/NOTIFY for sync | So a write by any path reaches tills (000109). |
| Keep `updated_at` correct on synced tables | The pull cursor depends on it. |

## Access control {icon=mdiShieldLock}

::: glance
- Role grants each **page** none / view / edit; pages group into **sections**; admin roles skip checks.
- The API only knows **sections**; a role's section level is its highest page level. The per-page split is UI-enforced.
- **One middleware** (`permissionMiddleware`) maps the matched route to a section via `routeRules`; GET = view, other methods = edit.
- **Unlisted route = admin only.**
- Keep `meta.section` / `meta.page` in `ui/src/router/index.js` in step with `api/access.go`.
:::

::: diagram access

> [!DANGER] The UI hiding a page is not protection. Always classify a new route in `routeRules`.

> [!NOTE] **App Updates** is its own section on purpose: putting a program on every till is the most powerful thing kashi does, so it is granted deliberately, never as a side effect of editing settings.

## Admin live updates {icon=mdiLightningBolt}

- `/admin/ws` pushes to open browsers: new sales (signed-cents amounts), alerts, presence, sync status.
- `/branch/ws` pushes pointers to tills.
- Both hubs wrap sockets in `wsConn` (a **write mutex**: gorilla panics on two concurrent writers) with ping/pong heartbeats so a dead peer is noticed in seconds.
- Every handler may broadcast from many goroutines, so never write to a raw `*websocket.Conn`.

::: group Features

## Remote commands {icon=mdiConsole}

::: glance
- Admin creates a **pending** row (audit trail) and nudges the till; the till fetches, runs, acks.
- A new command type needs **both** `allowedBranchCommandTypes` here **and** an executor in kashi-pos `commands.go`, backed by a real validated function.
- Commands about one sale/shift go to **the till that has it**; others to every active register.
- `reprint_invoice` (Invoices page **Print on till**, any kind) is refused for a `historical` old-POS import: no till has it.
- `remote_return_invoice` carries `print_receipt: false` when the dialog's **Print the receipt on the till** is unticked; ticked sends nothing and the till's **Print return receipts** decides.
- Branch-wide `update_settings` is also written to `branch_settings` (kashi is the authority).
:::

**Live cart** (`api/live_cart.go`, app-bar cart button): each till reports its open sale to `POST /branch/live_cart` on every change and every 20 s. Only the latest per register is kept, **in memory**; anything older than 75 s counts as offline. `POST /live_carts/:branch_id/override` (pricing rights) sends a `cart_override` for that sale only: every deal off, or a line's price, discount or no deal by barcode. Unknown items all share one barcode, so one goes by its `line_ref` (the till's `unknownRef`, reported as `lineRef`) and can also take `unknown_deal` (`b2g1_30`, `b2g1_20`, `none`); a till too old to report `lineRef` shows "Update this till" instead. The till applies it **past the branch's price/discount/deal settings** and refuses one for a sale it no longer has open. The set replaces the last one, so an empty set undoes everything.

::: diagram commands

## App updates {icon=mdiUpdate}

::: glance
- Developer runs `make publish` (kashi-pos): signed build uploaded with `createAppRelease`.
- Kashi **cannot verify the signature** (the public key lives in kashi-pos); it refuses a file whose SHA-256 differs from the uploader's.
- Admin sets a **target** per register on App Updates; kashi nudges; the till fetches, downloads, installs at a safe moment, reports each step.
- Rollback = the same with an older target. Release files are stored under a fixed on-disk name, never the uploaded one.
- **Small downloads** (`api/app_update_compact.go`): builds are stored **zstd** (~28 MB → ~9 MB), plus **patches** (~1–2 MB) from each build tills run, keyed by that exe's SHA-256. A till asking `accept=zstd,patch&have=<its exe sha>` gets both; an older till gets the plain exe, unpacked once to `release.raw`.
- `sha256`/`size_bytes`/signature are always **the exe's**; on a zstd upload kashi unpacks it to check that. A bad patch fails the till's check and it falls back to the full download.
:::

::: diagram release

## WhatsApp {icon=mdiWhatsapp}

::: glance
- Thank-you messages after a sale through **Meta's Cloud API**; credentials, templates, webhook secret are in Settings (database).
- **Rules** run top to bottom; first enabled rule whose conditions all hold decides: `skip` or `send` (its template, else the default).
- No matching rule = the default. A condition-less `skip` at the bottom disables the default.
- Phones compare by digits; if they differ, the **last 8 digits** (Lebanese number without country code).
- The webhook reports sent, delivered, read, replies and failures; there is also a poll and a bridge.
- A till's invoices list shows each sale's **poll answer** as an emoji: it posts the codes on screen to `/branch/invoice_ratings` (by branch + invoice code; `invoice_ratings_enabled`, on by default).
:::

::: diagram whatsapp

Item codes match exactly or as a prefix written `ABC*`, any case.

## Unknown items {icon=mdiHelpBox}

- A till sells a product with no code after a **phone photographs every unit**; the sale waits until each unit has a picture (or a manager's pass).
- Pictures go **phone to kashi** (QR link, token-gated, public page) and the till polls status.
- When the sale lands it is linked to its session; an admin then matches each picture to a real product and the line is relabelled.
- Pictures of sales that never happened are cleaned up (`unknown_image_cleanup`); the Storage page can delete and returns `deleted_ids`.

## Security guard {icon=mdiShieldAlert}

::: glance
- Every API request passes `securityGuard.middleware`.
- Banned IPs and (if the filter is on) IPs outside allowed countries are refused; **tills skip the country filter** (a till on a foreign-looking link must keep syncing).
- Private networks and the server itself are never refused.
- Counts are flushed every few seconds, not per request.
- The site sits behind **Cloudflare**; the real client IP is taken from its edge ranges.
:::

::: diagram security

## Background jobs {icon=mdiTimerSand}

| Job | What |
| --- | --- |
| Recurring expenses | Templates fire on their own (`FireRecurringExpenseTx`). Pause with `active`, never delete (it would lose `next_due_at`'s place). |
| Target series | `branch_target_series_scheduler` creates targets on a schedule. |
| Sync listener | `listenForSyncChanges` + lease expiry loop, for the life of the process (`Start`). |
| Security flush | Visitor counters written in batches. |
| Image optimise / cleanup | `imageopt`, thumbnails, unknown-image cleanup. |
| Email reports | Scheduled reports (`email_schedule`, xlsx). |

::: group Frontend

## Web app {icon=mdiVuejs}

::: glance
- Vue 3 + Vuetify 4 + Vite, Pinia, file-based layouts; one file per page in `src/pages`.
- **Icons are tree-shaken**: every `mdi-*` name must be imported from `@mdi/js` and added to `iconPaths` in `src/plugins/icons.js`. `pnpm build` runs `scripts/check-icons.mjs` and fails otherwise (an unregistered icon once hung the app).
- Data access goes through `composables/use*.js` (`useApi`, `useApiResource`, per-entity composables).
- Live updates: `useAdminSocket`.
:::

### New admin page

1. Page in `ui/src/pages`; route in `router/index.js` with `meta.section` and `meta.page`.
2. Sidebar entry in `data/navItems.js`.
3. Page added to its section in `api/access.go`, plus a `routeRules` entry for its API prefix.

> [!TIP] The unsaved-changes guard (`useDialogGuard`) watches every dialog's fields; a dialog whose only field is a search box needs `data-no-discard-guard` (or be skipped by class).

::: group US POS landscape

## US POS systems {icon=mdiStoreSearch}

::: glance
- Researched 2026-11-01 from general market knowledge (no live pricing); **verify vendor details before copying a design**.
- Two worlds: **big-box** (Target, Walmart, Kroger: NCR/Toshiba/Diebold Nixdorf lanes, custom software) and **small/mid retail SaaS** (Square, Shopify, Lightspeed, Clover, Toast for food).
- Kashi already covers the core: offline-first tills, multi-branch sync, multi-currency, price/discount lists, coupons, loyalty, stock counts, transfers, purchasing, WhatsApp receipts.
- Biggest US-style gaps: **gift cards, store credit, layaway, split tender, ecommerce/BOPIS, customer-facing display, self-checkout, tips, sales-tax engine, mobile/handheld POS**.
:::

### Popular systems

| System | Typical user | Known for |
| --- | --- | --- |
| Square for Retail | Small shops | Free tier, built-in card reader, fast onboarding, invoices, online store, loyalty and gift cards. |
| Shopify POS | Brands selling online and in store | One catalog and inventory across web and shops, BOPIS, staff roles, exchanges across channels, customer profiles. |
| Lightspeed Retail | Mid-size specialty retail | Deep inventory (variants, matrix, serials, purchase orders, reorder points), multi-store, consignment. |
| Clover | SMB, bank-resold | Own hardware (Station, Mini, Flex), app marketplace, integrated payments. |
| Toast | Restaurants | Handhelds, table service, kitchen display; cited for UX ideas only (food, not retail). |
| Heartland / Global Payments | SMB and enterprise | Payments-first, retail and hospitality, reporting. |
| Revel | Larger SMB, QSR | iPad POS, enterprise controls, inventory, employee management. |
| NCR Voyix (Aloha, Counterpoint, Silver) | Enterprise and mid-market | Lane hardware, self-checkout, large-scale back office. |
| Toshiba (Global Commerce) | Grocery, big-box | Self-checkout and fixed lanes, store-wide software. |
| Diebold Nixdorf | Grocery, big-box | Self-checkout, ATMs, vision-based loss prevention. |
| Oracle Xstore / MICROS | Large chains | Omnichannel, enterprise-grade promotions, tax and returns. |
| Epicor, Retail Pro, Microsoft Dynamics, SAP | Large retail | ERP-tied POS: finance, supply chain, merchandising. |
| Target / Walmart / Kroger (custom) | Own chains | Branded self-checkout, price match, app scan-and-go, store-pickup staff tools. |

### Self-checkout patterns (from the Target lane that prompted this page)

| Pattern | Idea for Kashi |
| --- | --- |
| "Scan items to begin" idle screen with one icon and one sentence | A matching idle state on the customer-facing display. |
| Running list + total always visible, big pay button | Customer-facing second screen mirroring the cart. |
| Weight-verify bagging area, attendant override | Attendant approval already fits Kashi's PIN auth; add a "call staff" flag. |
| Age/ID prompts, restricted items | Product flag "needs ID check" that blocks the sale until staff confirm. |
| Price Match Policy signage | See "Price match" below. |
| App scan-and-go with loyalty id (Target Circle) | Phone number already identifies clients; QR to link a client to a till. |

### What Kashi has vs. what is missing

| Feature (typical in US POS) | Kashi today | Gap / idea | Effort |
| --- | --- | --- | --- |
| Gift cards (sell, load, redeem, balance check) | None | New `gift_cards` table synced to tills; sell as a line, redeem as a payment type. | Medium |
| Store credit | Exchanges and loans exist | Issue credit on returns instead of cash; redeem as a payment type. | Medium |
| Split tender (cash + card + credit in one sale) | Multi-currency payments exist | Verify arbitrary mixed payments with change rules, surface in quick pay UI. | Small |
| Layaway / deposits / special orders | None | Partial payments against a held sale; stock reserved until paid. | Large |
| Parked sales | Not verified | Park/resume carts per register, sync-free. | Small |
| Returns without receipt, restocking fee | Exchanges and cancel exist | Return reason codes, fee, manager approval. | Small |
| Sales tax engine (rates by state, county, product class, exempt customers) | Basic `tax` in shift reports | Tax categories and exemption certificates if Kashi ever sells in the US. | Medium |
| Tips and gratuity | None | Optional tip step on card payment; only for service retail. | Small |
| Customer-facing display | None | Second window mirroring the cart (Wails supports multiple windows). | Medium |
| Self-checkout mode | `kiosk` mode exists for the cash machine | Customer-operated flow: scan, pay, receipt; attendant PIN override. | Large |
| Scan-and-go / mobile POS (phone or handheld as a till) | None | Web till for floor staff to ring up from the aisle. | Large |
| Ecommerce sync, BOPIS, ship-from-store | None | Product feed plus order intake; pick lists per branch. | Large |
| Email/SMS receipts | WhatsApp, email pages exist | Add "receipt by email/SMS" choice at tender time. | Small |
| Loyalty tiers, points expiry, birthday offers | Loyalty page exists | Tiers, expiry and automated offers via WhatsApp. | Medium |
| Promotions engine (BOGO, mix-and-match, spend X get Y, bundles) | Deals, coupons, discount lists | Rule types beyond discount lists; see `doc/product-rules.md`. | Medium |
| Price match | None | Price-override reason code with logged competitor evidence. | Small |
| Reorder points and auto purchase orders | Stock health alerts, purchase invoices | Per-branch min/max and a "suggested PO" page per supplier. | Medium |
| Serial / IMEI tracking | `serial` exists in the tills | Check it covers warranty lookup. | Small |
| Consignment and vendor-owned stock | None | Supplier-owned flag with payout report. | Medium |
| Employee commissions and spiffs | Salesperson performance, targets | Commission rules on top of targets. | Medium |
| Time clock with breaks and overtime | Attendance exists | Break types and weekly overtime summary. | Small |
| Loss prevention (void/discount exceptions) | Audit log, security guard | Exception report: voids, big discounts, no-sale drawer opens per cashier. | Small |
| Cash drawer: paid-in/out, blind close, drops | Shifts with float and close | Safe drops and blind count. | Small |
| Offline mode | Built in (tills are the truth) | None; a selling point against SaaS POS. | None |
| EMV/card terminal integration | Not in scope today | Needed for US card payments (Stripe Terminal, Square Terminal, Clover). | Large |
| EBT/SNAP, WIC | None | Grocery only; skip unless Kashi enters grocery. | Large |

### Suggested order

1. **Quick wins:** split tender check, parked sales, return reason codes, void/discount exception report, safe drops.
2. **Customer-facing:** second display, email/SMS receipts, "needs ID" product flag.
3. **Money products:** gift cards, then store credit (they share the payment-type plumbing).
4. **Big bets:** promotions engine, layaway, self-checkout, ecommerce/BOPIS, card terminal integration.

> [!NOTE] Anything that adds a payment type touches `Money and time` (cents, rounding) and the till `invoice_settlement` flow; read those pages first.

::: group Knowledge base

## Recipes {icon=mdiRocketLaunch}

### Database change

1. Add `NNNNNN_name.up.sql` / `.down.sql` (additive).
2. Edit `db/query/*.sql`.
3. `make sqlc`, then `make mock` if the `Store` interface changed.
4. If tills pull the table, make sure `updated_at` and the NOTIFY trigger cover it.

### New till-facing endpoint

1. Handler under `/branch/*` (auth is `branchAuthMiddleware`).
2. Keep old shapes working: an older till build will call it.
3. If a till should learn of changes, add the entity to the catch-up and NOTIFY set.

### New admin API route

1. Handler + route in `api/server.go`.
2. **Classify it in `routeRules`** (or it is admin-only).
3. Test with the mock store.

### New remote command

1. Add the type to `allowedBranchCommandTypes`.
2. Implement it in kashi-pos `commands.go` first.

### Fix a bug

Add a Symptom / Cause / Fix entry to `docs/FIXED_BUGS.md` in the same commit.

## Gotchas and workarounds {icon=mdiAlertOctagon}

| Symptom | Cause / rule |
| --- | --- |
| Postgres JIT | Off for the app role (000222): compiling took 2-3s on the 1-CPU server, longer than the queries it sped up. Check `SHOW jit` before trusting a timing on another machine. |
| Icon renders as text, app lags | Unregistered `mdi-*` name. Add to `plugins/icons.js`. |
| New route returns 403 for non-admins | Not in `routeRules`. |
| Two websocket writers panic | Writing a raw conn; use `wsConn`. |
| Till never receives a row | It changed without bumping `updated_at`, or the completeness check has not run. |
| Old till breaks after deploy | A migration dropped or renamed something. |
| Zero price rejected | Do not mark amount fields `required` (0 is valid). |
| Receipt/PDF margins | Report PDFs need page margins applied explicitly (fixed 2026-10-31). |
| Dialog asks "Discard your changes?" | Guard sees a field edit; opt out with `data-no-discard-guard`. |
| Live push stops after scale-out | Hubs are in-memory, single instance only. |

### Smaller traps

- Registers' keys are shown **once**; rotation overwrites the hash immediately.
- A register's code is fixed once tills have issued invoices with it.
- `cmd/` tools write real data; check the target database.
- `app.env` and `storage/` hold secrets and customer photos; never commit them.

## How Kashi changed {icon=mdiHistory}

::: timeline
- 2026-01-07 | Project starts as a Vuetify UI | First commit is an empty Vuetify app; the Go API follows.
- 2026-03-27 | Generated store mock | `db/mock` and the first API tests: the pattern every handler test uses.
- 2026-06-05 | Amount naming cleanup | Columns such as `net_amount_in_default_currency` become plain `amount` (cents).
- 2026-09-07 | Serialised websocket writes | Concurrent broadcasts crashed sockets; every write now goes through one mutex.
- 2026-09-29 | Branch history pull | A fresh till can pull its own past invoices, expenses, loans and shift closes.
- 2026-10-19 | Branch settings and designers | Code-only search and per-code price edits for tills; Receipt Designer page.
- 2026-10-23 | Product rules and Centro list | Attributes back-filled on production; discount list stored as rules (migration 000200).
- 2026-10-29 | WhatsApp rating poll | Option to send purchase details inside the rating poll.
- 2026-10-31 | Loading polish | No skeleton/loader flashes on fast loads; report fixes.
:::

> [!NOTE] `doc/product-rules.md` records the agreed discount and product-field rules (what was applied to production on 2026-10-23, with the backup names).

## Glossary {icon=mdiBookAlphabet}

| Term | Meaning |
| --- | --- |
| till / register | A kashi-pos install; a branch has one or more registers, each with its own key. |
| branch | A shop. Has settings pushed to its tills, a timezone, price/discount lists. |
| `uid` | The shared identity of a row across kashi and tills. |
| catch-up | `GET /branch/sync/changes`: what a till pulls. |
| lease | A till's "syncing" claim, renewed every 30 s. |
| section / page | Access-control units: pages group into sections; the API enforces sections. |
| conflict | A till/kashi disagreement staged for an admin to resolve. |
| flavor | A kashi-pos build variant (standard or compact) that updates separately. |

## Keeping these docs current {icon=mdiBookOpenVariant}

> [!TIP] Update this file in the **same commit** as any change a developer must know (a convention, invariant, route rule, command type, trap), then `make dev-docs` and commit `docs/dev-docs.html`.

- Put it in the page it belongs to; start the page's **At a glance** list if it is a headline fact.
- Prefer a table or one line to a paragraph; say *why*, name the file or test that enforces it.
- Callouts: `> [!WARNING]`, `[!DANGER]`, `[!TIP]`, `[!NOTE]`, `[!HISTORY]`. Diagrams live in `docs/dev-docs-diagrams.mjs`.
- Delete text that stopped being true.
