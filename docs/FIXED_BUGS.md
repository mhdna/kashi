# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

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
