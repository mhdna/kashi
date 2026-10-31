# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

## 2026-10-31 · The report appeared dimmed, then faded in, after its animation

**Symptom.** On the Reports page, once the "Preparing your report" picture
went away the sheet appeared dimmed and only then faded to full strength,
which felt like lag.

**Cause.** The sheet is dimmed (`.reports-loading`) while `loading` is true,
meant for reloads where the old sheet stays up. On a first load the sheet
shows as soon as the figures arrive, but `loading` stays true while the staff
list loads afterwards, so the new sheet was dimmed and then faded in.

**Fix.** `ui/src/pages/reports.vue` dims the sheet only when one was already
showing when the load started (`dimSheet`). Checked with a build; no
automated test covers the page.

## 2026-10-31 · Deleting a picture in Storage reloaded the whole grid

**Symptom.** Deleting unknown-item pictures on the Storage page blanked every
thumbnail into a loading state and loaded the whole grid again, as if the
page had reloaded.

**Cause.** After a delete, `deleteSelected` in
`ui/src/components/TreeStorage.vue` called `resetAndLoad()`, which throws
away every thumbnail and fetches the grid from the first page. The server
only answered how many were deleted and kept, not which, so the page couldn't
drop just those.

**Fix.** The delete endpoint also returns `deleted_ids`
(`api/unknown_image_cleanup.go`), and the page removes just those pictures
from the grid and frees their thumbnails; the rest stay as they are. With an
older server (no `deleted_ids`) it still reloads. Test: step 6 of
`TestUnknownItemsFlow` (`api/unknown_items_flow_test.go`) checks that a
picture whose sale is waiting to be matched is kept and the other is named in
`deleted_ids`.

## 2026-10-31 · Closing the command palette asked "Discard your changes?"

**Symptom.** After typing in the command palette (Ctrl/Cmd+K), closing it with
Escape or a click outside brought up "Discard your changes? What you changed
in this window hasn't been saved.", as if it were a form.

**Cause.** The unsaved-changes guard (`ui/src/composables/useDialogGuard.js`)
watches every dialog's fields and asks before closing one whose fields
changed. The palette is a dialog whose only field is its search box, so any
search counted as an unsaved edit. Its dialog is rendered by Vuetify, so it
couldn't carry the guard's `data-no-discard-guard` opt-out.

**Fix.** The guard skips dialogs with Vuetify's `v-command-palette` class.
Checked with a build; no automated test covers the guard.

## 2026-10-31 · Report PDFs ran to the edge of the paper

**Symptom.** An exported report PDF had no margins: the title, cards and
tables touched the left, right and top edges of the A4 page.

**Cause.** The 10mm margins came only from the print page's `@page` rule,
and the print dialog throws that away when its Margins option is "None"
(Safari ignores it too). The sheet's own padding was switched off for print.

**Fix.** `ui/src/utils/reportExport.js` now prints with `@page { margin: 0 }`
and gives the sheet 10mm padding with `box-decoration-break: clone`, so every
page keeps its margins whatever the dialog says. Checked by printing a
three-page test sheet in headless Chrome: each page keeps 10mm on all sides.

## 2026-10-30 · Reports kept a salesperson's old name

**Symptom.** After renaming "Ahmad Robai" to "Ahmad Robaie", the attendance
pages and the salesperson ranking still said "Ahmad Robai", and his punches
at a second branch showed as another person.

**Cause.** The attendance reports took the name the fingerprint device sends
with each punch, which still had the old spelling. Punches at a branch where
he has no salesperson record, and till sales typed with the old name, were
matched to nobody.

**Fix.** A person's name in the reports is kashi's salesperson name. Punches
with the same device number at another branch, and typed sale names, count
for the salesperson whose kashi name starts with them ("Ahmad Robai" ->
"Ahmad Robaie") (`ui/src/components/reports/reportRows.js`,
`AttendanceReport.vue`). Checked against production data; no automated
test covers it.

## 2026-10-30 · Feedback dropped replies from customers who didn't tap an answer

**Symptom.** The feedback report and the Reviews page showed only a handful
of customer replies: in October 13 of the 42 stored ones. Written feedback
like "W sabaya wel 5edmi tamem" or "Great!" was missing.

**Cause.** The earlier small-talk fix (below) counted a reply only after the
follow-up question. The follow-up goes out after a customer taps an answer,
and most customers who wrote back never tapped one, so all their replies
were dropped.

**Fix.** A reply counts from the poll message until a day after it (or after
the follow-up, when that went out later), the first five messages with more
than punctuation (`db/query/review_link.sql`). Long chats are trimmed to
five, and the report's Hide button removes small talk by hand. Checked
against production: 22 of October's 42 replies now show, every one of the
missing feedback messages among them; no automated test covers it.

## 2026-10-30 · Turning a photo twice saved it after one turn

**Symptom.** In the photo viewer (expense, shift and unknown-item photos),
clicking "rotate right" twice saved the photo after the first turn instead
of turning it again.

**Cause.** Undo and Save appeared inside the toolbar after the first turn.
The toolbar is anchored at the bottom right, so it grew to the left: the
rotate buttons slid over and Save landed where "rotate right" had been.

**Fix.** Undo and Save sit in their own bar at the top of the photo
("Turned 90° · Undo · Save"), and the toolbar never changes size
(`ui/src/components/ZoomableImage.vue`). Checked in a browser that "rotate
right" keeps its place after a turn; no automated test covers it.

## 2026-10-30 · WhatsApp Health counted test sends and showed "usual 0"

**Symptom.** On WhatsApp > Health, "Sent today" included test messages
sent from Settings, and a number in daily use could show "15 (usual 0)".

**Cause.** The card showed the bridge's own counts. The bridge counts every
message a number sends, tests too, and works out the usual over days when
the number wasn't linked yet.

**Fix.** kashi now counts sends, failures and one-ticks per number from its
own `whatsapp_messages` log, leaving out test sends
(`WhatsAppNumberSendCounts` in `db/query/whatsapp_bridge.sql`,
`withOwnCounts` in `api/whatsapp_health.go`). The usual is averaged only over
the days of the past week when the number was already sending, and a number
with less than a day of history reads "new number". Disconnects, refusals
and the risk score still come from the bridge. The card now says "Sent, last
24 h", since that's the window, and one tick means 3 days, as on the Messages
tab (`WhatsAppHealth.vue`). Covered by `TestGetWhatsAppHealth` and
`TestWhatsAppHealthNewNumberHasNoUsual` in `api/whatsapp_health_test.go`.

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
