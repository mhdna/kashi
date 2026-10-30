# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

## 2026-10-30 · Small talk showed up as poll feedback

**Symptom.** The Reviews page and the customer feedback report listed
"Hi", "Kifc", "." and other chat as customers' replies to the rating poll.

**Cause.** The WhatsApp bridge reports every message a client writes after
a poll, and `PollReplies` / `LowPollRatings` showed all of them, however
late and whatever they said.

**Fix.** Only answers to the follow-up question count: written after it was
sent, within 24 hours, the client's first three messages with more than
punctuation (`db/query/review_link.sql`). The messages are still stored.
Checked with throwaway rows in a rolled-back transaction against a local
database; no automated test covers it.

## 2026-10-30 · Salespersons couldn't be renamed

**Symptom.** Renaming a salesperson ("Ahmad Robai" to "Ahmad Robaie") failed
with "Field validation for 'Name' failed on the 'required' tag", from the
page and from the API alike.

**Cause.** `updateSalesperson` bound the URL and the body into one struct.
Binding the URL validates the whole struct, and the required name is only in
the body, so it refused every rename before reading the body.

**Fix.** The id is bound from the URL on its own struct, then the name from
the body (`api/salesperson.go`). Test: `TestUpdateSalesperson_Renames`.

## 2026-10-30 · Mixed carts got no free item on B2G1

**Symptom.** At the till, a boy's jeans (BOC5…) and two CA girl's jeans on
"Buy 2 get 1 free" got nothing free.

**Cause.** CA/EOC5/EF6 leftovers were to get 30% and the rest 20%, and a
deal has one fallback, so the rules put them on two deals; the till only
makes sets within one deal.

**Fix.** A rule naming a deal can give its own "Otherwise %"
(`deal_fallback_percent`, 000216), written onto its rows and sent to the
tills in its own field, so one deal pools the whole shop. Test:
`TestDealRuleFallback`. The till side is in kashi-pos's log.

## 2026-10-30 · Salespersons in one branch shared the same colour

**Symptom.** On the till's salesperson picker three people were orange and
two pink, so the colour didn't tell them apart.

**Cause.** A new salesperson's colour was a random one of 12
(`000112_salesperson_color`), which collides quickly in a branch.

**Fix.** A trigger picks the palette colour fewest of the branch's
salespersons have (`000217_salesperson_distinct_color`), so colours repeat
only past twelve people. The existing duplicates in production were
reassigned by hand. Test: `TestNewSalespersonGetsAnUnusedColor`.

## 2026-10-30 · Saving a list's rules wiped their price limits

**Symptom.** A discount rule limited to cheap products ("toys up to $1.11:
no discount") lost its limit the next time anyone saved that list's rules,
and then applied to every price.

**Cause.** Saving updates each existing rule in place
(`ReplaceDiscountListRuleGroupsTx`), and that update never passed
`MaxPrice`, so it wrote NULL (no limit). Only a newly created rule kept it.

**Fix.** The update passes `MaxPrice` (`db/sqlc/tx_discount_rules.go`).
Test: `TestDealRuleFallback` saves a rule with a limit twice, by its id.

## 2026-10-30 — `make seed` creates no branches and no branch invoices

**Symptom:** on a fresh database `make seed` logged "failed to ensure branch …:
column api_key_hash does not exist", then "no branches available, skipping
branch invoices". With branches added by hand, every invoice failed with
"null value in column register_id". The dashboard, home feed and every
branch report were empty.

**Cause:** `cmd/seed/main.go` predated the move of API keys from `branches` to
`branch_registers`: it still wrote `branches.api_key_hash` and never set
`branch_invoices.register_id`, which is now NOT NULL. Its branch codes
(`BR-SEED-01`) also contained a dash, which a till refuses as a branch code.

**Fix:** `ensureSeedBranches` creates branches with letters-only codes, a
register for every branch that has none (sharing the branch's code, like a
branch's first register), and returns branch and register pairs;
`seedBranchInvoices` sets `register_id`. Checked by running the seed against a
freshly migrated throwaway database: 5 branches and 1,000 branch invoices. No
automated test covers the seed command.
