# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

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
