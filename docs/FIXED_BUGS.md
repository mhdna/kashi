# Fixed bugs

One entry per bug fixed in kashi, newest first: what went wrong, why, and
what changed. Add an entry in the same commit as the fix.

## 2026-11-13 · "Discard your changes?" came up where nothing was edited

**Symptom.** Live cart asked "Discard your changes?" on Escape or a click
outside after only picking another branch to look at. The same happened
after typing in a dialog's search box (a product's stock, a salesperson's
adjustments), after using the Products filter dialog, after picking in
Reports' "Left out of reports" (which saves each pick at once), and in any
dialog after switching its tabs or once live rows arrived.

**Cause.** The guard snapshotted every input and every active button in the
whole dialog as one string. A view control counted as an edit, a tab is an
active button, and rows that arrived with live data (a till's cart) changed
the snapshot on their own.

**Fix.** `ui/src/composables/useDialogGuard.js` now notes each field
(Vuetify input, toggle group, chip group, date picker, bare input) on its
own when the user touches the dialog and compares only those; fields that
appear later count only once touched, and plain buttons and tabs aren't
fields. View controls are marked `data-discard-ignore` (Live cart's title
with its Branch and Online filters, the search boxes in `inventory.vue` and
`salesperson-performance.vue`); the Products filter dialog and Reports'
exclusions dialog opt out with `data-no-discard-guard`. Tests:
`ui/src/composables/useDialogGuard.test.js`, and
`ui/src/components/LiveCartButton.test.js` runs the real guard on the
mounted dialog (branch picker: no question; a typed price: asks).

## 2026-11-13 · Unknown item showed its price twice on the picture page

**Symptom.** On the phone page where the cashier takes pictures, each unknown
item read "Unknown item $25.00 $25.00". The admin Unknown items page's Waiting
tab showed the price in the title and again under it.

**Cause.** The till names each unit "Unknown item $25.00", with the price
already in the name, and both pages print the unit's price next to the name.
The till's own picture dialog shows that name with no separate price, so
the name sent to the till has to keep its price.

**Fix.** Both pages drop a trailing price from the name when it is the unit's
own price; a cashier's name or a different price is left alone.
`unknownLabel` in `api/unknown_upload_assets/upload.html` and
`ui/src/utils/unknownItems.js` (used by `ui/src/pages/unknown-items.vue`).
Tests: `ui/src/utils/unknownItems.test.js` runs the same cases on both copies
and renders the real phone page.

## 2026-11-12 · Every till kept "Updating clients" while a test branch drained its backlog

**Symptom.** Every branch's sync light kept reading "Updating · Clients · 3000
changes", then Waiting, then Updating again, for as long as Test Branch 2's till
was sending its backlog of (fake) clients.

**Cause.** Two things on central. Every client a till sent was saved again with
`updated_at = now()` even when nothing about it had changed, and every batch
broadcast `client_updated` to all tills, so each one pulled the whole batch back.
And a hidden branch's clients, though sent to visible tills as inactive, were
still sent: every till downloaded all of Test Branch 2's fake clients.

**Fix.** An unchanged client is left alone and only a real change wakes the tills
(`api/branch_client.go`, `clientEditChanges`, `notifyClientsChanged`). A hidden
branch's pushes wake only its own registers, and its clients made after it was
hidden are left out of visible tills' pull, completeness ids and fetch altogether
(`branches.hidden_since`, `client_withheld_from_visible`, migration 239;
`api/branch_catchup.go`, `api/branch_sync_ids.go`). Tests:
`api/hidden_branch_clients_test.go` ("UnchangedIsLeftAlone", "WithheldLeftOut"),
`db/sqlc/hidden_branch_clients_test.go` ("TestHiddenBranchWithheldClients").

## 2026-11-12 · App-bar pill: two messages drawn on top of each other

**Symptom.** The app bar's sync pill showed garbled text like "Centrows 5 to fix":
the old and the new message overlapped in the same spot.

**Cause.** The new one-card pill crossfaded old and new messages in the same grid
cell, so both were drawn at once during the fade. A growing group ("5 to fix" ->
"6 to fix") also got a new key on every message, so it swapped the whole card
instead of just changing the number.

**Fix.** The swap is `mode="out-in"` (the old message leaves before the new one
comes) and shorter, and a group keeps one key (`groupKey`), so its count updates in
place (`ui/src/components/SyncCard.vue`, `ui/src/utils/activityPill.js`). Tests:
`activityPill.test.js` ("a growing group keeps its key"), `SyncCard.test.js` ("a
growing conflict count updates in place").

## 2026-11-12 · Towel socks were filed as Home and got the 30% discount

**Symptom.** At the till, "Men Towel long socks" (COC5MTLSO3) showed -30%. Socks are accessories and
get no discount.

**Cause.** The auto-fill reads category from words in the name, first match wins, and the Home row
(which lists "towel") came before Socks & Tights. Five towel-sock products were saved as Home / Home,
so the China & Turkey 30% rule caught them instead of the no-discount Accessories rule.

**Fix.** On prod the five products (COC5MTLSO3, X23MTLSCK3, X23MTSCK2.5, X23KSOCKST3, X23BSOCKST2) were
moved to Accessories / Socks & Tights. In `db/tools/gen_autofill_rules.py` the socks row now comes
before Home; migration 000237 ships the regenerated rules. Covered by
`ui/src/utils/productAutofillRules.test.js` ("socks are accessories, even towel socks"; the five
cases fail against 000236).

## 2026-11-11 · Tills pulled from kashi about twice a second, all day

**Symptom.** The till's new sync light showed "Updating" every half second, even with
nothing changing. On kashi's side every connected till was re-fetching the branch list
non-stop.

**Cause.** Every request a till makes stamps its branch's `last_seen_at`
(`TouchBranchRegister`). The `sync_notify` trigger on `branches` (000109) fired for any
update, so that stamp told every connected till "branches changed, pull". The pull is a
request, which stamps `last_seen_at` again: a loop that sustained itself.

**Fix.** Migration 000234 splits the trigger: inserts and deletes notify as before, an
update notifies only when something other than `last_seen_at` changed. Covered by
`db/sqlc/branches_notify_test.go` (a stamp and `TouchBranchRegister` stay silent; rename,
deactivate, delete, insert and stamp-plus-change still notify). Breaking the trigger
makes the first test fail.

## 2026-11-11 · Failed WhatsApp rows said "poll not sent: WhatsApp is not connected"

**Symptom.** Invoices of the Sour branch showed Failed with "poll not sent: WhatsApp
bridge: WhatsApp is not connected", for three days, with no hint what to do.

**Cause.** WhatsApp logged the Sour number out (code 401) on 2026-11-08 18:19, two
seconds after a send, in the middle of a ramp-up (14, 25, 40, 81 messages a day). A
logged-out number stays down until someone scans its QR again; every send after
that failed. The log text named neither the number nor the cure, and called a
poll-only send "poll not sent" as if a message had gone.

**Fix.** `whatsAppNumberDownError` in `api/whatsapp_bridge.go` rewrites a
"not connected" failure to "<Number>'s WhatsApp is logged out or disconnected: link
it again in Settings > WhatsApp", still marked temporary so the till retries; a
poll-only failure is the send's own failure. Covered by
`TestWhatsAppNumberDownError` and the "isn't linked" case in
`api/whatsapp_bridge_test.go`.

## 2026-11-10 · Centro showed "No branches connected" while its till was syncing

**Symptom.** The Connected branches pill showed a red 0 and "No branches connected"
though Centro's till was sending invoices and syncing normally.

**Cause.** Two things. Pushes to tills and admin tabs were written while holding the
hub's lock and with no time limit, so one socket that stopped reading (a phone tab
asleep behind Cloudflare, a till on a dead network) could block a write forever and
with it every connect and the connected list, which then failed to load and showed
0. Separately, Cloudflare closes long sockets every few minutes to hours
(11m, 1h50m, 20m in the prod log); the till redials in seconds, but in between the
branch counted as gone.

**Fix.** `wsConn.writeJSON` sets a 10s write deadline and closes a socket whose write
fails (`api/ws_conn.go`); both hubs write after releasing their lock
(`api/branch_ws.go`, `api/admin_ws.go`); a register stays connected for 45s after its
socket closes, until it reconnects or the grace is swept. Tests:
`api/branch_hub_test.go` (a till or tab that never reads doesn't freeze the hub, a
timed-out write closes the socket, short gaps stay connected, the grace runs out,
a replaced socket closing late, two registers on one branch).

## 2026-11-09 · Lady jeans (EOC5/EF6) all showed $11

**Symptom.** EOC5LJE25, EOC5LJE35, EF6LJE29 and the other outlet lady jeans were
priced $11 in kashi and on the tills, though the code's last number is the price.

**Cause.** The September product import gave these items the placeholder price
1100 (cents) instead of the price in the code.

**Fix.** On production, the 8 variants were set to code number × 100 in the Default
price list and `product_variants.price` (EOC5LJE7/14/15/20/25/29/35, EF6LJE29/35);
the old values are in table `_bak_lady_jeans_price_20261109`. The sync triggers
push the change to the tills. Other codes that end in 11 (CA22…11) really are $11
and were left alone. Not a code change, so no test.

## 2026-11-09 · Auto-fill filled EF6LJE10 wrongly (category "Lady", Type "Jeans", no origin)

**Symptom.** Auto-fill on EF6LJE10 put "Lady" in Category and "Jeans" in Type and
left Sub-Category, Kind and Origin empty; older codes like CA21LSW filled nothing.

**Cause.** The rules were still in the old layout (category = audience, a Type
field), had no outlet origin, and the code layout required a price at the end of
every code.

**Fix.** Rules and layout rewritten from the production codes (migration 000232,
`db/tools/gen_autofill_rules.py`); see `doc/product-rules.md`. Tests:
`ui/src/utils/productAutofillRules.test.js`, `TestShippedAutofillRules_PassPageValidation`.

## 2026-11-07 · The rating poll showed in the Inbox as text starting with 📊

**Symptom.** In the WhatsApp Inbox, the combined rating poll ("Hi Rena, / Thank
you for shopping… / How was your visit today?" with its options) showed as a
plain text bubble starting with 📊, its options as "•" lines, instead of the
poll bubble.

**Cause.** The bridge logs a poll as "📊 question" then "• option" lines.
`whatsAppPollFromText` took only the first line as the question, so a question
over several lines (the combined poll's) didn't parse, and the raw text was shown.

**Fix.** The options are the "• " lines at the end and everything before them
is the question (`api/whatsapp_chat.go`). The poll bubble keeps the question's
line breaks (`WhatsAppInbox.vue`, `WhatsAppConversation.vue`), and the chat list
previews a poll as its last line, without 📊. Test: `TestWhatsAppPollFromText`.

## 2026-11-07 · Returning both units of a buy-1-get-1 sale failed at the till

**Symptom.** "Return remotely" on a Buy 1 Get 1 sale of two alike shoes failed
with "checkout return: item has already been returned", though nothing had been
returned. The two rows in the dialog also shared one key.

**Cause.** A deal sale keeps the paid and free units of a product as separate
lines, and the till's return code expected one line per product (see
kashi-pos's FIXED_BUGS). The dialog keyed its rows by product id, which repeats.

**Fix.** The dialog keys rows by line and sends each line's unit price
(`ui/src/pages/branch-invoices.vue`); the till spreads a request over the
product's lines (kashi-pos `commands.go`, `exchange_validation.go`). Tests are in
kashi-pos: `TestExecuteRemoteReturnInvoiceCommand_DealLinesOfOneProduct`,
`TestDealRefundOfAlikeUnitsSpansThePaidAndFreeLines`.

## 2026-11-05 · Adding a product failed with "current transaction is aborted"

**Symptom.** Adding a product failed with "pq: current transaction is aborted, commands
ignored until end of transaction block (25P02)". Pressing Add again sometimes worked.

**Cause.** Two things. The barcode sequence sat at 82 while imported products held
nearly every number up to about 10,000, so generated barcodes kept being taken. And
the retry ran inside the transaction the failed insert had already aborted, so it
could never succeed.

**Fix.** A taken barcode now just returns no row (`CreateProductVariantWithFreeBarcode`,
`db/sqlc/tx_product.go`), and the sequence moved past the imported block
(`000227_barcode_sequence_past_imports`). Test: `TestCreateProductSkipsATakenBarcode`.

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
