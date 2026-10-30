# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

## 2026-10-30 · Saving a list's rules wiped their price limits

**Symptom.** A discount rule limited to cheap products ("toys up to $1.11:
no discount") lost its limit the next time anyone saved that list's rules,
and then applied to every price.

**Cause.** Saving updates each existing rule in place
(`ReplaceDiscountListRuleGroupsTx`), and that update never passed
`MaxPrice`, so it wrote NULL (no limit). Only a newly created rule kept it.

**Fix.** The update passes `MaxPrice` (`db/sqlc/tx_discount_rules.go`).
Test: `TestDealRuleFallback` saves a rule with a limit twice, by its id.
