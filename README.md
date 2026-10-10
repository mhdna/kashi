<div align="center">
  <h1>Kashi</h1>
  <p>The central back office for a multi-branch retail business.</p>
</div>

<p align="center">
<img src="https://github.com/mhdna/kashi/actions/workflows/ci.yml/badge.svg"/>
<img src="https://goreportcard.com/badge/github.com/mhdna/kashi"/>
<img src="https://badges.fw-web.space/github/license/mhdna/kashi"/>
</p>

Kashi is the head office of a chain of shops. Every branch runs its own till
app ([kashi-pos](https://github.com/mhdna/kashi-pos)), which keeps selling
when the internet is down and syncs with Kashi whenever it can. Kashi is where
the owner and staff see every branch at once: sales, stock, money, people,
prices, clients, and the tills themselves.

It runs as a Go API with a PostgreSQL database and a Vue 3 + Vuetify web app,
live at `cloud.example.com`.

---

## Contents

- [What it does](#what-it-does)
- [How it fits together](#how-it-fits-together)
- [Getting started](#getting-started)
- [Configuration](#configuration)
- [Everyday commands](#everyday-commands)
- [Project layout](#project-layout)
- [Working on the code](#working-on-the-code)
- [Access control](#access-control)
- [Deploying](#deploying)
- [License](#license)

---

## What it does

**Analytics**
- Home feed of what's happening across branches and in Kashi itself, with live toasts as sales land.
- Dashboard, daily income (per day, per branch, drill into any day), financials, targets with progress bars, footfall from the door counters, conversion, product revenues, sales analytics and salesperson performance.

**Sales and clients**
- Every branch invoice (sales, returns, exchanges) synced up from the tills, searchable in one place.
- Clients shared across branches, client profiles and spending, loyalty points and tiers, wholesale clients.
- **WhatsApp:** a thank-you message after a sale through Meta's WhatsApp Cloud API. Ordered rules decide which sales get a message and which template (by total, branch, client, item code or category, and how recently the client was messaged). The WhatsApp page shows what was sent, delivered and read, replies, failures, costs and charts. Meta's webhook reports delivery and replies back to Kashi.

**Products and stock**
- Products with variants, colours, sizes, attributes and categories, plus auto-fill rules that read a product code's conventions.
- Barcodes and label printing.
- Inventory per warehouse and branch, stock movements, transfers between branches (requested from the tills, approved here), stock counts, batches, purchase invoices, suppliers.
- Low-stock alerts, stock health and inventory analysis.

**Money**
- Expenses (recurring ones fire on their own), expense and receipt photos, loans and repayments, currencies and payment methods, cashboxes, assets and treemaps of where the value sits.

**People**
- Employees, salaries, shifts and attendance from the tills' clock-ins, with complaints and approvals.

**Pricing**
- Price lists and discount lists per branch, coupons.

**Branches and tills**
- Branches and their registers, branch settings pushed to tills (with defaults for new ones), a till's quick-access cards, remote commands, sync conflicts, and **remote app updates**: publish a signed till build and choose which tills install it.

**System**
- Users and roles with per-page view/edit rights, an audit log, an activity feed, dev logs, storage browser, calendar, and a **Security** page: who reaches the API from where, alerts, country filtering and IP bans.

---

## How it fits together

```
 Branch tills (kashi-pos, offline-first)          Staff browsers
        │  HTTPS + WebSocket, per-register key            │
        ▼                                                  ▼
 ┌──────────────────────────── Caddy (cloud.example.com) ─────────────────┐
 │  /api/*  → Go API on :8090          /*  → built Vue app (ui/dist)       │
 └────────────────────────────────────────────────────────────────────────┘
        │
        ▼
 Go API (Gin) ── PostgreSQL
   ├─ /branch/*      till sync: invoices, shifts, attendance, expenses,
   │                 transfers, settings, commands, app updates, WhatsApp
   ├─ /branch/ws     live push to tills (settings, commands, targets)
   ├─ /admin/ws      live push to open browsers (new sales, alerts…)
   ├─ everything else: the admin API, behind login and role checks
   └─ background: recurring expenses, target series, security guard
```

- **Tills are the source of truth for what happened in a shop.** They sync sales, shifts, attendance and so on up to Kashi. Kashi is the source of truth for shared data: products, prices, clients, settings, users. Rows carry a shared `uid` so the same record matches on both sides.
- **Till authentication:** each register has its own key, sent as `x-branch-code` and `x-branch-key`. Admin users log in with PASETO tokens (access + refresh).
- A gRPC server and a gRPC-gateway with Swagger docs also exist (`proto/`, `gapi/`, `doc/swagger`). The web app and tills use the Gin HTTP API.

---

## Getting started

### Requirements

| Tool | For |
|---|---|
| Go 1.26+ | the API |
| Node 20+ and pnpm | the web app (`ui/`) |
| Docker | a local PostgreSQL |
| [golang-migrate](https://github.com/golang-migrate/migrate) | database migrations |
| [sqlc](https://sqlc.dev) | generating the Go database code from SQL |
| [mockgen](https://github.com/uber-go/mock) | the store mock used by API tests |
| [air](https://github.com/air-verse/air) (optional) | live-reloading API in development |

### Run it locally

```sh
# 1. Database
make postgres          # PostgreSQL in Docker, on port 5433
make createdb
make migrateup

# 2. Config
cp example.app.env app.env    # then edit, see Configuration below

# 3. API
make server            # or `make dev` for live reload

# 4. Web app (in another terminal)
cd ui
pnpm install
VITE_API_URL=http://localhost:8080 pnpm dev
```

The web app opens on `http://localhost:3000`. `VITE_API_URL` must point at the
API's `HTTP_SERVER_ADDRESS`. Optional demo data: `make seed`.

---

## Configuration

Settings come from `app.env` in the working directory. Environment variables
with the same name override it. `app.env` holds secrets and is not committed.

| Key | What it is |
|---|---|
| `DB_DRIVER` | `postgres` |
| `DB_SOURCE` | PostgreSQL connection URL |
| `HTTP_SERVER_ADDRESS` | where the API listens, e.g. `0.0.0.0:8090` |
| `GRPC_SERVER_ADDRESS` | gRPC server address |
| `GATEWAY_SERVER_ADDRESS` | gRPC-gateway (Swagger) address |
| `TOKEN_SYMMETRIC_KEY` | 32-character key that signs login tokens. **Use a random one in production.** |
| `ACCESS_TOKEN_DURATION` | e.g. `15m` |
| `REFRESH_TOKEN_DURATION` | e.g. `24h` |
| `EXPENSE_IMAGE_STORAGE_DIR` | expense receipt photos (default `./storage/expense_images`) |
| `SHIFT_RECEIPT_IMAGE_STORAGE_DIR` | close-shift receipt photos (default `./storage/shift_receipt_images`) |
| `TRANSFER_ATTACHMENT_STORAGE_DIR` | photos on transfer requests (default `./storage/transfer_attachments`) |
| `APP_RELEASE_STORAGE_DIR` | published till builds (default `./storage/app_releases`) |
| `GEOIP_DIR` | IP location databases for the Security page, downloaded automatically (default `./storage/geoip`) |
| `APP_LOG_PATH` | rotating API log (default `logs/kashi.log`) |

Business settings live in the database instead and are edited in the app under **Settings**. That includes WhatsApp credentials, templates and the webhook secret.

The web app reads `VITE_API_URL` at build time: a full origin in development, or `/api` in production behind Caddy.

---

## Everyday commands

| Command | Does |
|---|---|
| `make server` / `make dev` | run the API (once / with live reload) |
| `make test` | run the Go tests |
| `make sqlc` | regenerate `db/sqlc` from `db/query/*.sql` and the migrations |
| `make mock` | regenerate `db/mock/store.go` after the store interface changes |
| `make migrateup` / `make migratedown1` | apply all migrations / roll back the last one |
| `make seed` | load demo data |
| `make proto` | regenerate gRPC code and Swagger docs |
| `cd ui && pnpm dev` | run the web app |
| `cd ui && pnpm lint` | lint and fix the web app |
| `cd ui && pnpm build` | build the web app into `ui/dist` |

---

## Project layout

```
main.go              starts the HTTP API, gateway and background jobs
api/                 HTTP handlers, middleware, access rules, WebSockets
  access.go          sections/pages and which routes each one guards
  server.go          every route
db/
  migrations/        numbered SQL migrations (golang-migrate)
  query/             SQL queries, one file per area
  sqlc/              generated Go from the queries (don't edit)
  mock/              generated store mock for tests (don't edit)
token/               PASETO login tokens
util/                config, logging, helpers
gapi/, proto/, pb/   gRPC server and generated code
doc/                 Swagger docs served by the gateway
cmd/                 one-off tools: seeding, imports, image optimisation, release notes
ui/                  the Vue 3 + Vuetify web app
  src/pages/         one file per page
  src/router/        routes, with the section/page each one belongs to
  src/data/navItems.js  the sidebar
  src/plugins/icons.js  the icons the app may use
  deploy/            Caddyfile, systemd unit and server setup notes
```

---

## Working on the code

**Database changes**
1. Add a migration pair in `db/migrations` with the next number (`000140_x.up.sql` / `.down.sql`). Keep migrations additive (add, don't drop or rename), so an older till build keeps working against a newer database.
2. Write or edit queries in `db/query/*.sql`.
3. Run `make sqlc`, and `make mock` if the store interface changed.

**A new admin page**
1. Add the page in `ui/src/pages` and a route in `ui/src/router/index.js` with its `section` and `page`.
2. Add it to the sidebar in `ui/src/data/navItems.js`.
3. Add the page to its section in `api/access.go`, and a route rule for its API prefix (see below).

**Icons**
The app only bundles the icons it uses. Every `mdi-*` name has to be imported from `@mdi/js` and added to `iconPaths` in `ui/src/plugins/icons.js`; otherwise it shows as text.

**Tests**
API tests use the generated store mock. Run `make test`, or `go test ./api/` for just the API.

---

## Access control

Every user has a role. A role grants each **page** nothing, view, or edit,
and pages are grouped into **sections** (Analytics, Sales, Clients, Branches,
Pricing, Products, Inventory, Accounting, HR, Calendar, System). Admin roles
skip the checks.

The API enforces this in one middleware: `routeRules` in `api/access.go` maps
each route prefix to the section that can write it and the sections that can
read it, and GET vs. other methods decides view vs. edit. **A route that isn't
listed is admin-only**, so forgetting to classify a new route fails closed.
The web app hides pages and buttons to match, but the API is what actually
protects the data.

---

## Deploying

Kashi runs on a server shared with kashi-commerce, behind Caddy (see
`ui/deploy/` for the Caddyfile, the systemd unit and first-time setup). It
needs an SSH alias `kashi` for the server in `~/.ssh/config`.

| Command | Does |
|---|---|
| `make deploy` | API and web app |
| `make deploy-api` | build the Linux binary, upload it with the migrations, run pending migrations, restart the service |
| `make deploy-ui` | build the web app and swap it in |
| `make migrate-prod` | only run pending migrations |
| `make deploy-notes` | post the shipped commits to the home page's release feed |

While the API restarts, open browsers show an "Updating" screen and come back
on their own. After a UI-only deploy, open tabs notice the new version
(`version.json`) and reload into it.

---

## License

[GPL v3](LICENSE)
