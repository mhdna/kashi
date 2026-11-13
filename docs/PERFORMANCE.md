# Performance

Speed and size work, newest first (see the global rules). Each entry: **Symptom**,
**Finding** (measured, or say it wasn't), **Change**, and **Not done** if relevant.

## Known slow spots

Measured or reported slow things that are not fixed yet. Nothing recorded yet.

## Entries

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
