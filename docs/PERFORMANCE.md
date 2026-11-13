# Performance

Speed and size work, newest first (see the global rules). Each entry: **Symptom**,
**Finding** (measured, or say it wasn't), **Change**, and **Not done** if relevant.

## Known slow spots

Measured or reported slow things that are not fixed yet. Nothing recorded yet.

## Entries

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
