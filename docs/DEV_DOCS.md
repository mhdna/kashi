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
- A branch **hidden from reports** (a test branch) keeps its own clients to itself: other tills get them as inactive.
:::

::: diagram syncChanges

| Topic | Rule |
| --- | --- |
| Identity | Rows carry a `uid`. A till's local ids mean nothing here; invoice lines reference the till's own ids as plain values. |
| Catch-up | `since` omitted = a full snapshot (a fresh till needs no bootstrap). `cursor` supersedes `since`. An old till that omits `limit` still gets one unbounded answer. |
| Hidden branches | A client is the hidden branch's own when only hidden branches linked, homed or sold to them (`client_hidden_branch_only`, migration 238). To a visible branch's till: one made after its branches were hidden (`branches.hidden_since`, `client_withheld_from_visible`, migration 239) is **left out** of the pull, `/branch/sync/ids` and `/branch/sync/fetch`; an older one, which may be on a till already, is sent as `is_active: false`, so the till hides it (it never deletes a client). The hidden branch's own tills get everything as is, and its client pushes wake only its own registers. Hiding or showing a branch bumps those clients' `updated_at` and broadcasts `client_updated`. |
| Client pushes | A client a till sends that central already has unchanged (name, phone) is **not re-saved**: `updated_at` stays, and `client_updated` goes out only when a row was created or really edited. Re-saving every one made every till pull a draining backlog back, batch by batch. |
| Completeness | `/branch/sync/ids` lists uid + `updated_at` (inactive rows included, so a till learns a row went inactive); `/branch/sync/fetch` returns the missing rows. |
| Cursor overlap | `server_time` from catch-up is 2 min behind now (`syncCursorOverlap`): `updated_at` is written before commit, so a pull in between would pass a change it can't see yet. Stamp list rows with `clock_timestamp()`, not `now()`. |
| Lists | A price or discount list the till must stop using is sent **active and empty**, never inactive: tills delete an inactive list, which fails once it has sales and rolls back the whole pull. |
| Sync status | `branch_register_sync_status` (000223) keeps each till's last pull per entity, shown on the Commands page. The same `since` again after a pull that sent items is a failed apply (`retries`, `stuck_since`); two repeats reads as stuck. Tills add the error and their outbox counts via `POST /branch/sync/report`. |
| List delivery | `GET /discount_lists/:id/delivery` (and price lists) compares a fingerprint of what each till holds (`api/list_fingerprint.go`, rows and a hash, same recipe as kashi-pos `sync_report.go`) with what it should hold. A cursor past the change with a different hash is `different`. A till too old to report is judged by its cursor, with `verified: false`. |
| Live push | Postgres `NOTIFY kashi_sync` (migration 000109) batches into one `sync_changed`; a lost listener sends `pull_all`. Any write triggers it: handler, script or SQL. |
| Full sync | A till's "Sync everything" holds a **lease** (renewed every 30 s) so the admin UI shows the branch as syncing; end, closed socket or missed renewal ends it. |
| Colour/size on scan | `branch_settings.variant_pick_required` (000233, default off; "Apply to all" template key `variant_pick_required`) is the branch default; `products.variant_pick` (`''` follow, `always`, `never`) overrides it per item, set by code or barcode from Products, Colour/size on scan (`POST /variant_pick`). It rides on every variant in the product feed (a trigger moves their `updated_at`). The till decides: a scanned barcode whose item has colour/size siblings opens the variant picker. |
| Sync light | The till's nav-rail light shows a pull step only when it received something different (quiet, the default). `branch_settings.sync_verbose` (000235, template key `sync_verbose`) shows every step, changed or not. The till judges "different": more than one page, a batch unlike the last pass (kashi re-sends rows changed in the last 2 minutes), or a whole-set table that really changed. |
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
- **Salesperson Performance** (`/salesperson-performance`) keeps its first score (50% sold, 25% invoices, 25% goods, against the best) and adds a **Performance score** table (`ui/src/utils/salespersonScore.js`): target reached 40, sales per hour 20, average sale 15, items per sale 15, discount rate 5 and return rate 5, each against the team of ranked people and capped at 1.5 times it (the target at 150%). Net sales are line totals sold minus returned; returns and exchanges count for the sale's salesperson. The personal target is the branch target (`targetShare`, like Sales Analysis) shared by attendance hours, or evenly when the branch has none. Ranked from 10 invoices and 40 hours; a part with no data is left out and the score is out of the rest. `GET /salesperson_performance` also returns `returns` (with `original_salesperson_id`) and `branch_targets`; `TestSalespersonPerformanceWithReturnsAndTargets` writes `ui/src/utils/testdata/salesperson_performance.json` for the UI tests.
- A **return or exchange counts for the salesperson of the sale it came back from** (`related_client_ref`), exactly as it was sold, in every report that groups by salesperson (`PeriodReportSalespersons`, `...Daily`, `...Categories`, `ReportProductGroupLines`, `ListDaySalesBySalesperson`); its own salesperson only when kashi doesn't have that sale. Returns made from kashi carry the till's default salesperson, so going by the return's own name split people in two.
- Report **net sales leave out deferred coupons** (coupon types with `calculate_on_checkout`), like kashi-pos: `branch_invoice_deferred_coupons(invoice id)` finds the coupon through `coupons.redeemed_invoice_ref = branch_invoices.client_ref` and returns the "On Coupon" payment, capped at the invoice total. Plain coupons count. Reports in `db/query/reports.sql` subtract it; the dashboard and e-mail reports do not yet.
- The till reads clock skew from `/branch/health` (kashi's clock and the branch zone).
- **Product group report** (`GET /reports/product_group?codes=TT,TX`, `api/report_product_group.go`): products whose code starts with a prefix (any case; `%` and `_` match literally). One query (`ReportProductGroupLines`, capped at 200k lines) returns the group's lines and Go adds them up, so every table sums to the headline figures. Money is `line_total`, as in Sales & goods, and `products` has the `/reports` shape, so `ProductsReport.vue` draws the group's Goods parts unchanged.
- The UI test fixture `ui/src/components/reports/__fixtures__/productGroup.json` is that handler's real answer: after a deliberate change, `UPDATE_FIXTURES=1 go test ./api -run TestGetProductGroupReport`, then run the UI tests.

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
- Every write has a **10s limit** (`wsWriteWait`); a socket that can't take one is closed. Hubs copy their sockets under the lock and **write after releasing it**, so one peer that stopped reading can't freeze connects or the connected list.
- **Connected = socket now, or closed under 45s ago** (`connGrace`). Cloudflare closes long sockets every few minutes to hours and the till redials in seconds; the grace keeps that off the pill. Lapsed graces are swept every 30s and pushed.

::: group Features

## Remote commands {icon=mdiConsole}

::: glance
- Admin creates a **pending** row (audit trail) and nudges the till; the till fetches, runs, acks.
- A new command type needs **both** `allowedBranchCommandTypes` here **and** an executor in kashi-pos `commands.go`, backed by a real validated function.
- Commands about one sale/shift go to **the till that has it**; others to every active register.
- `reprint_invoice` (Invoices page **Print on till**, any kind) is refused for a `historical` old-POS import: no till has it.
- `remote_return_invoice` carries `print_receipt: false` when the dialog's **Print the receipt on the till** is unticked; ticked sends nothing and the till's **Print return receipts** decides.
- Payments go by **method name** and signed USD cents (`payments: [{account_name, amount_cents, amount_units?}]`, units for counted LBP). `remote_return_invoice` may carry them (a return's refund, an exchange's difference); `change_invoice_settlement` re-splits a sale. The till requires them to **equal its own total** (LBP within its rounding) and otherwise fails, naming its figure; kashi's dialog total is only an estimate.
- A refund of a sale still owed **On Account** comes off that debt first (kashi-pos `on_account_returns.go`).
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

**Label reader** (`api/label_reader.go`, `api/unknown_items_label.go`): with `LABEL_READER_PROVIDER` set (`groq` or `anthropic`, plus `GROQ_API_KEY` / `ANTHROPIC_API_KEY`; `LABEL_READER_MODEL` overrides the default `qwen/qwen3.8-27b` / `claude-opus-5-5`) every picture's price label is read in the background: name, code, price, barcode. Off when unset.

| Step | Rule |
| --- | --- |
| Read | Right after upload, off the request. Stored on `unknown_item_images.label_*`; a failed read is `label_status = 'failed'` with the error, and the unit waits for a person. |
| Barcode | The till's scan (`unknown_item_slots.barcode`) wins; else the label's digits, **kept only when the GS1 check digit holds**. No barcode at all: left for a person. |
| Product | Once the sale has landed (whichever of read / landing comes second): the barcode's existing product, else a new one with the label's name and code (the barcode as code when the label's is taken), **the cashier's price** on the default list, `products.source = 'label_photo'` ("From photo"). One picture = one cart line = one product. |
| Price | A label price that isn't the cashier's is only a warning (`label_price_mismatch` on the Unknown items list). |
| Resolve | Same `ResolveUnknownItemsTx` as a person's match, then `auto_resolved`. Two readings of one sale are serialised (`labelMu`); a lost race on the barcode uses the winner's product. |

## Referrals {icon=mdiGiftOutline}

Every client has a code (`clients.referral_code`, migration 000243): 5 random characters and a **check character** from an alphabet without look-alikes (`ACDEFGHJKMNPQRTWXY34679`, 23 is prime, so any one wrong character or two swapped neighbours fails). The database makes them (`new_referral_code()`, the column default; unique index). The till checks them with the same rule (kashi-pos `referrals.go`, `lib/referralCode.ts`); the vectors in `api/referrals_test.go` are shared by all three.

| Step | Rule (`api/referrals.go`) |
| --- | --- |
| Friend's sale | The till sends `referral_code` on a sale. `decideReferral` gives the owner a reward, or refuses with the reason shown on the Referrals page: off, bad code, nobody's code, no customer, own code, **bought before at any branch**, already sent by someone (one reward per friend, `referrals_friend_once`), under the minimum, owner's monthly limit. The friend gets nothing. |
| Reward | `referrals` row, `available`, the program's percent and cap, usable from `unlock_after_days` after the sale until `expires_after_days` later (0 = right away / forever). |
| Branches | On per branch: `referral_program.branch_ids` (migration 000250; `enabled` is kept as "on at any branch"). `referralsOnAt` decides: a code given at a branch that is off earns nothing (refused as off), and that branch's tills are fed `enabled: false`, so they neither take nor print codes nor use rewards. The rules are the same everywhere. |
| Tills | Feeds `referral_program` (the rules a till needs) and `referral_rewards` (refusals left out); `referrals_updated` / `referral_program_updated` on the branch socket. |
| Use | The till uses a reward when it beats the spend tiers and sends `referral_reward_uid`: `used`. A second till that used it offline keeps its sale; `used_again_branch_invoice_id` flags it. |
| Undo | Cancelling the sale that used it makes it `available` again. With `cancel_on_return`, a friend's sale cancelled, or returned down to nothing or under the minimum, cancels a reward not used yet. Exchanges never do. |

Everything here is best-effort after the sale is stored: a failure is logged, never fails the till's push, and a replay is harmless (one row per source invoice).

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
- **Receipt Designer** draws the real receipt live with the till's own code: `receiptrender/` is a copy of kashi-pos's renderer (`make receipt-renderer` there, never edit it here). `GET /receipt_designer/preview` draws a branch's sample receipt (its name, phone, policy, socials, fonts) in a look and mode with the design being edited and says where each part is; `/receipt_designer/catalog` lists the parts, their lines and what each can change. The design is `branch_settings.receipt_texts`, cleaned by the till's own `NormalizeReceiptTextsJSON`. Words are typed on the paper over the line the till reports (`lines`); a whole part takes capitals, size, bold and alignment (`groups`); the Items part also has on/off switches for what each item row prints (name, colour and size, code, before and after price: catalog lines that can only `hide`, or `show` for the code, which is off until switched on; `partsOf` in `receiptDesign.js`), lines above and under and a box (`decor`), and the parts under the totals an `order`. The Shop details tab saves the branch's name, phone, website, policy and socials (the preview takes unsaved ones as `shop`); Make this the default writes the look and design into the default template (`receipt_texts`). `TestReceiptRendererMatchesTheTill` checks the copy draws all 70 looks and modes exactly like the till's pictures.
- **Retail invoice filters** all sit in `components/Invoice/InvoiceFilterDialog.vue`; `utils/invoiceFilters.js` turns them into the `/branch_invoices` query. The product filter (code, name or barcode; contains, starts, ends or exact, built into an escaped ILIKE pattern by `likePattern` in `api/branch_invoice.go`) matches a line through its `product_uid`, so a till line kashi never matched to a variant is never found by it.
- **New Product auto-fill** is data, not code: `app_settings.product_autofill_rules` + `product_autofill_code_layout`, run by `utils/productAutofill.js`. They are generated by `db/tools/gen_autofill_rules.py` into a migration (000232); edit the tables there, regenerate into a new migration. Code convention and agreed fields: `doc/product-rules.md`.
:::

### New admin page

1. Page in `ui/src/pages`; route in `router/index.js` with `meta.section` and `meta.page`.
2. Sidebar entry in `data/navItems.js`, in the section it belongs to (`navSections`: the rail shows one icon per section, the drawer lists the open section's pages).
3. Page added to its section in `api/access.go`, plus a `routeRules` entry for its API prefix.

> [!TIP] The unsaved-changes guard (`useDialogGuard`) compares each dialog field present when the user first touched the dialog (rows arriving later and tab switches don't count). Mark a view control (branch picker, search box, filter) `data-discard-ignore`; a dialog that saves as you go or is all filters gets `data-no-discard-guard`.

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
| A table that changes on every till request | Never leave it on a plain `sync_notify` trigger: the till's pull is a request, so it notifies itself forever. `branches` did (fixed in 000234, `last_seen_at` is ignored). Compare the columns a till pulls instead. |
| Postgres JIT | Off for the app role (000222): compiling took 2-3s on the 1-CPU server, longer than the queries it sped up. Check `SHOW jit` before trusting a timing on another machine. |
| Icon renders as text, app lags | Unregistered `mdi-*` name. Add to `plugins/icons.js`. |
| New route returns 403 for non-admins | Not in `routeRules`. |
| Two websocket writers panic | Writing a raw conn; use `wsConn`. |
| Till never receives a row | It changed without bumping `updated_at`, or the completeness check has not run. |
| Old till breaks after deploy | A migration dropped or renamed something. |
| Zero price rejected | Do not mark amount fields `required` (0 is valid). |
| Receipt/PDF margins | Report PDFs need page margins applied explicitly (fixed 2026-10-31). |
| Dialog asks "Discard your changes?" | Guard sees a field edit; mark view controls `data-discard-ignore`, or the dialog `data-no-discard-guard`. |
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
- 2026-11-09 | Auto-fill rewritten | Rules read the real code convention (EF6LJE10 = outlet lady jeans, 2026, price 10), generated from a script (migration 000232).
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
