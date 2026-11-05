# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

## 2026-11-05 · The Barcodes page showed NaN as the price

**Symptom.** Every product with a price showed "NaN" in the Barcodes page's Price
column. The printed labels were right.

**Cause.** `GET /barcodes` sends the price as `{"Int64": 2900, "Valid": true}`, as it
can be empty, and the page treated that object as a number.

**Fix.** The page reads `Int64` when `Valid` is set (`ui/src/pages/barcodes.vue`).

## 2026-11-05 · Picking a product on a price or discount list showed its number

**Symptom.** After choosing a product in "Product variant" on a price or discount
list, the box showed a number like 4174 instead of the product's name.

**Cause.** Picking one clears the box's search, which loads the first page of
products. A product found by typing usually isn't on it, so the box had only its id
to show.

**Fix.** The box holds the picked variant itself and skips that search
(`ui/src/components/PriceDiscountListManager.vue`). The UI has no test runner, so this
was checked in Chrome with a stubbed search: the old box showed 4174, the new one the
name.

## 2026-11-05 · A till never got a list change that saved slowly, yet showed "Reached"

**Symptom.** A crocs rule at 21% was saved and Centro showed "Reached", but the till
kept selling Men Crocs at 30%.

**Cause.** The list was stamped with `now()`, when the save's transaction began
(12:50:07), and committed three minutes later (12:53:10). The till pulled at 12:52:55,
so its cursor passed the stamp before the change was visible. After the commit the
list looked older than the till's cursor and was never sent. "Reached" compares the
same cursor, so it was wrong too.

**Fix.** List stamps use `clock_timestamp()`, the cursor kept 2 minutes behind now
(`syncCursorOverlap` in `api/branch_catchup.go`), and the migration re-sends every
list to repair what was missed (`000225_list_stamps_at_write_time`). A pull that sent
nothing no longer counts towards "Reached". Tests: `TestListStampIsWriteTime`,
`TestBranchSyncCursorKeepsAnOverlap`, `TestListDeliveryState`.

## 2026-11-05 · Saving a list's rules timed out with "status 524"

**Symptom.** Saving rules spun for minutes and then failed with "Request failed
with status 524", though the save went on to work on the server.

**Cause.** Every save re-matches the rules, and that joined every candidate match
to the price table with no index: 455 million comparisons, for a price limit most
rules don't have. It took 2m30s on the server, past Cloudflare's 100s limit.

**Fix.** The price is looked up only for a rule with a limit
(`db/migrations/000224_faster_rule_refresh.up.sql`). On a copy of production's
tables, the same save went from 35s to 1s, with identical rows on every list.
Test: `TestDiscountListRuleMaxPrice`.

## 2026-11-05 · A branch moved to another discount list never got it

**Symptom.** Centro was moved from "Buy 2 Get 1" to "buy 1 get 1" in kashi, but
its till kept selling on the old deal, with no error anywhere in kashi.

**Cause.** A list the till must stop using was sent as inactive, and tills delete
such a list. Once it has sales the delete fails, because `invoice_products` points
at it. That rolled back the whole pull, so the new list was lost too, the cursor
stayed put and the till retried forever.

**Fix.** Such a list now goes as active with no items, which prices nothing and
always applies (`api/branch_sync_lists.go`). Tests:
`TestBranchSyncChanges_DiscountLists_RemovedListGoesEmpty`,
`TestBranchSyncChanges_PriceLists_InactiveListGoesEmpty`. The payload was also
run through the real till sync code against a list with sales.

## 2026-11-05 · Picking a list failed with "PageID failed on the 'min' tag"

**Symptom.** After the fix below, picking a price or discount list showed "Failed
to load data: Key: 'listPageQuery.PageID' Error:Field validation for 'PageID'
failed on the 'min' tag".

**Cause.** That fix reset the table's row count to -1 when the URL changed. The
data table works out its pages from that count, so -1 gave it no pages, it
dropped to page 0 and asked for a negative `page_id`.

**Fix.** The count resets to 0 (`ui/src/components/Tables/ServerSideTable.vue`).
The UI has no test runner, so this was checked in Chrome against a stub server
that refuses a negative `page_id`.

## 2026-11-05 · Picking a price or discount list kept showing the first list's rows

**Symptom.** On Discount Lists and Price Lists, picking another list changed the
header and rules but not the table, so the page seemed to load forever. The
"buy 1 get 1" list showed the "Buy 2 Get 1" rows.

**Cause.** `ServerSideTable` read its URL only when it fetched, and nothing
fetched when the URL changed. These are the only pages whose URL changes, so the
rows stayed until something else, like paging or a search, forced a reload.

**Fix.** A new URL now empties the table and loads page 1, and only the newest
request may fill it, so a late answer can't bring back the old list
(`ui/src/components/Tables/ServerSideTable.vue`). The UI has no test runner, so
this was checked by running the old and new table in Chrome with stubbed answers.

## 2026-11-05 · A picture taken on the phone sometimes never uploaded

**Symptom.** On the unknown items phone page, a picture was taken for an item
but the card stayed on "Needs a picture", with no thumbnail and no error, while
the items around it showed "Sent".

**Cause.** Each card built its own camera input, and the page rebuilt every card
every 2 seconds, even with nothing new. The input the camera was opened from was
usually gone from the page by the time the picture came back, and a removed input
is not reliable on phones, so the picture could be lost before any upload started.
Worked out from the code and the screen state; not reproduced on a phone.

**Fix.** One camera input now stays in the page outside the cards and remembers
which item it was opened for, and the refresh only rebuilds the cards when
something changed (`api/unknown_upload_assets/upload.html`). Covered by
`TestUnknownUploadPageKeepsCameraInputOutOfTheCards`
(`api/unknown_upload_page_test.go`).

## 2026-11-02 · Pausing a recurring target did nothing on the till, and Add Target errored without dates

**Symptom.** (1) Switching a recurring target off or on in a branch's Targets
dialog changed nothing on the till's Balance page until something else
refreshed it. (2) Pressing Add in the Add Target dialog without picking dates
showed `parsing time "T00:00:00Z" as "2006-01-02T15:04:05Z07:00"`.

**Cause.** (1) `setBranchTargetSeriesActive` only flipped the series flag: it
never touched the periods already generated and never told the branch's
tills to re-pull. (2) The dialog sent the empty dates as `T00:00:00Z`, which
the API can't parse.

**Fix.** (1) `api/branch_target_series.go`: pausing hides the periods running
or still to come, resuming restores them (and creates the current one if
missing, via `ensureCurrentPeriod` in `api/branch_target_series_scheduler.go`),
and the branch is notified with `target_updated`. New query
`SetBranchTargetsActiveBySeriesSince`; test
`TestPausingASeriesHidesItsCurrentTarget`. (2) `ui/src/pages/branches.vue`
asks for the dates before sending.

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
