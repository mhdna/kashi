# Performance

Speed and size work, newest first (see the global rules). Each entry: **Symptom**,
**Finding** (measured, or say it wasn't), **Change**, and **Not done** if relevant.

## Known slow spots

Measured or reported slow things that are not fixed yet.

- Invoices client box with a 1 or 2 character term: trigrams need 3 characters, so
  such a term still reads every client (about 0.5 s a pass on prod).

## Entries

### 2026-11-23 Live view of tills: what it costs

Not a slow thing, a cost check for a new feature (`docs/plans/live-view.md`).

**Off (the default):** nothing. Tills are not told the feature exists, the
endpoints refuse, `liveHub` is an empty map, and rrweb is never loaded on a till.

**Server, measured** (`LIVE_LOAD=1 go test ./api -run TestLiveViewLoad -v`): 3
tills each streaming 4 KB frames 20 times a second (80 KB/s each, a busy till)
to one watcher for 10 s, after a 300 KB first snapshot. The whole test process,
fake tills and watchers included: heap +3.4 MB, 1.2% of one core, every frame
delivered. A slow watcher is dropped and resynced, never waited on
(`TestLiveHubSlowWatcherNeverBlocks`).

**Not measured yet:** the till's own cost (rrweb recording inside the Wails
webview) and the real droplet's memory while watching. To do on a real kashi-pos
build: watch one till for 5 minutes of normal use and record kashi-api RSS before,
during and after, bytes a second on the stream, and the till's CPU in Task Manager
with and without a watcher. Expected: under 20 MB extra on kashi, a few KB/s, no
visible lag. Add the numbers here.

### 2026-11-17 Filtering retail invoices by client took over a minute

**Symptom:** typing a name or phone in the new client box on Invoices took ages.

**Finding:** the first version checked each of the 35k invoices against the 131k
clients (a correlated `EXISTS` with an `OR` across uid and till link). On prod it
did not finish within 60 s (`EXPLAIN ANALYZE`, cancelled by `statement_timeout`).
Finding the matching clients once and then the invoices by hashed lookup took it to
1.2 s ("maya") and 2.2 s ("70 12"), most of it two sequential `ILIKE` scans of
`clients` (0.5 to 0.8 s each). On a local copy of the same size, trigram indexes cut
those queries from 100 to 210 ms to 5 to 7 ms.

**Change:** `db/query/branch_invoice.sql` finds matching clients first, then invoices
by `client_uid IN` or `(register_id, branch_client_id) IN`. Migration 000242 adds
trigram indexes on `clients.name` and on the phone's digits. The API works out the
digits once (`db/sqlc/client_search.go`), so a term with letters skips the phone
match and the phone `ILIKE` is gone.

**Not done:** the time on prod after the indexes is not measured yet (it needs the
deploy that adds them).

### 2026-11-13 Invoices page fetched the hidden dialog's and tab's data on open

**Symptom:** the Invoices list was slow to show on first load.

**Finding:** opening the page sent 10 requests. Only 3 were for what's on
screen (retail invoices, branches, registers). The other 7 were the New Sales
Invoice dialog's lists (cashboxes, accounts, shifts, inventories, clients,
price lists, 50 barcodes), plus the Wholesale table, whose tab is hidden.
They compete with the retail list for the browser's connections and the
server. `/clients/` was the slowest one (see the entry below). Time in the
browser was not measured.

**Change:** `ui/src/pages/sales-invoices.vue` loads the dialog's lists the first
time Add Invoice is clicked, and mounts the Wholesale table the first time its
tab opens. Opening the page now sends 3 requests. Test:
`ui/src/pages/sales-invoices.test.js`.

### 2026-11-13 Clients list summed every client's till points on each call

**Symptom:** the Invoices page was slow to open. It loads `/clients/` for the
New Sales Invoice dialog, and the Clients page uses the same call.

**Finding:** `ListClientsWithLoyalty` added up loyalty points for every client
across all of `branch_invoices`, then kept only the 10 on the page. On the
local DB with 300k synthetic invoices (rolled back afterwards) it took 141 ms,
and it grows with every sale. The invoice list itself was fine (0.2 ms for the
page, 31 ms for the count). Prod was not measured.

**Change:** `db/query/client.sql` picks the page of clients first, then sums
points for just those clients with a LATERAL join on
`idx_branch_invoices_register_branch_client`: 1.5 ms on the same data. Paging
now breaks name ties by id, so a client can't show up on two pages. Tests:
`db/sqlc/client_loyalty_list_test.go`.

## 2026-11-19: The tills' customer completeness check, 6s to 0.3s a page

**Symptom:** Centro's till never finished checking its 130k customers against kashi, so older customers never got their referral codes and reprints printed none.

**Finding:** each `/branch/sync/ids` page of 5000 customers took 6 to 15 seconds on production; the page query itself took 29 ms, the rest was `ListWithheldClientUIDs` calling `client_withheld_from_visible` once per customer (about 1.2 ms each). Past the till's 10-second timeout the check gave up and started over on its next sync. Prefiltering on "touches a hidden branch" didn't help: Test Branch links 4778 of every 5000.

**Change:** `ListWithheldClientUIDs` is the same rule written as joins over the whole page (`db/query/client.sql`): 291 ms for the same page on production. `TestWithheldClientUIDsMatchesFunction` checks it gives the function's answer on randomized data.

## 2026-11-23: Live view separate screen (not yet measured on a real till)

**Cost on kashi:** one more session and one more stream per till, only while an admin uses it, through the same in-memory relay; it counts toward the cap of 3 watched tills. Nothing when unused.

**Cost on the till:** a second copy of the till app runs in a hidden iframe, so memory roughly doubles for the page while it is open (at most one per till). **To measure on a real till:** the window's memory with and without the separate screen open, whether the off-screen frame's timers and animations are throttled, and that a barcode scan lands in the cashier's sale while the admin types. If the frame is throttled, move it on screen behind the app's opaque background (see the plan, Risks).
