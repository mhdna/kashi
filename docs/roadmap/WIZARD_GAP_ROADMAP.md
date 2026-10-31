# kashi vs Wizard Cloud ERP & Odoo: missing features & milestone roadmap

_Last updated 2026-10-31._

## How this was researched

- Logged into thewizardcloud.net/98 as the R1 wholesale user (read-only): every route in their JavaScript, every form, grid column and Actions menu that loads, the full app config (268 document types, 57 document codes, 85 company options, price lists, warehouses, currencies, chart of accounts).
- Every page on wizardsolutions.net from its sitemap (70 pages): home, /modules, all 14 module pages, 23 industry pages, 4 'compare' pages (Odoo, SAP B1, Dynamics BC, QuickBooks), 5 guides, 14 glossary pages, about, customers.
- Every feature card on Wizard's 23 industry pages and 5 guides (about 770 cards; the extra ones are in the 'More Wizard detail' groups).
- Odoo: odoo.com/page/all-apps, the Point of Sale, Accounting, Inventory, Sales and Purchase feature pages, odoo.com/pricing, apps.odoo.com l10n_lb listings.
- A full survey of ~/Code/kashi (Go + Postgres + Vue HQ) and ~/Code/kashi-pos (Wails + SQLite till): routes, 219 kashi migrations, POS migrations, App.d.ts bindings, settings, integrations.

## TL;DR

- **Branch operations & analytics: kashi looks ahead (POS app not yet reviewed).** Offline till with real sync, remote commands, signed updates, deals/discount rules/coupons, loyalty tiers, WhatsApp automation, Google reviews, footfall & conversion, door-device attendance, shift close vs expected with photo, stock-health forecasting, scheduled email reports. Their POS is a separate Flutter app we haven't reviewed yet, so the till-vs-till comparison is pending.
- **Back office: Wizard is far ahead.** They are an accounting-first ERP: general ledger, receivables & payables with credit limits and statements, quote→order→delivery→invoice, purchase orders, landed cost, banks & cheques, VAT, USD/LBP with dated rates and revaluation, payroll. kashi has none of the ledger and only fragments of the rest.
- **Their product is old and fragile.** Yii 1 + jQuery, 268 document types bolted on over 20 years (including one-off customer forms like 'Otis Contract', 'Yarz Member', 'Leave Request-Califood'). Right now a pending DB upgrade breaks most of their screens for our account, error pages leak raw SQL, and every page ships the logged-in user's **plaintext password** inside its JavaScript (`wz.user.pwd`).
- **Odoo is a different threat**: a huge, modern suite (~45 apps, AI, portal, Studio) sold per user through partners, but with no official Lebanese localization or payroll and a POS that isn't built for USD+LBP cash. Part 2 maps it feature by feature with its own O-milestones; Part 3 ranks what actually matters to Lebanese businesses.
- **The plan:** M0 builds the shared document engine, then M1–M2 (receivables + wholesale selling, payables + purchasing) let us switch the R1 wholesale branch off Wizard. M3–M6 make kashi a full trading ERP. M7–M9 make it sellable. After that we pick fights module by module where we can be 10x better (mobile, WhatsApp, analytics, offline).

## Scoreboard

Of **204** capabilities Wizard has (from their app and their website):

| Status | Count | Meaning |
|---|---|---|
| 🟢 kashi ahead | 10 | kashi already does it better |
| ✅ have | 17 | kashi has it |
| 🟡 partial | 49 | kashi has a piece of it |
| ❌ missing | 108 | kashi doesn't have it |
| ⚪ skip | 20 | not worth building for us |

## Priorities & sizes used below

- **P0 — needed to drop Wizard in our own business**
- **P1 — needed to sell kashi to other retailers/wholesalers**
- **P2 — needed to compete across the board**
- **P3 — only for specific verticals / on demand**
- Sizes are rough focused-work estimates: **S** < 1 week, **M** 1–3 weeks, **L** 3–6 weeks, **XL** 6+ weeks.

## Milestone overview

| # | Milestone | Priority | Size | Goal |
|---|---|---|---|---|
| M0 | [Document engine & money foundations](#m0) | P0 | L | Build once what every later document needs, so M1–M6 are mostly screens, not plumbing. |
| M1 | [Receivables & wholesale selling](#m1) | P0 | L | Run the R1 wholesale branch's selling and debt collection in kashi. |
| M2 | [Purchasing, payables & landed cost](#m2) | P0 | L | Buy, receive, cost and pay suppliers in kashi. |
| M3 | [Bale opening, units & inventory depth](#m3) | P1 | M | Model how the clothing-bale business really works, plus the inventory controls Wizard sells on. |
| M4 | [Quote → Order → Delivery chain](#m4) | P1 | M | Full wholesale order flow for bigger clients and dealers. |
| M5 | [General ledger & financial statements](#m5) | P1 | XL | Become an ERP, not 'POS + back office'. Required to fully drop Wizard and to sell to anyone with an accountant. |
| M6 | [Banks, cheques & VAT](#m6) | P1 | M | Cover how Lebanese businesses actually get paid and file taxes. |
| M7 | [Approvals, permissions & control](#m7) | P1 | M | Give owners the control Wizard sells to managers, with our better UX. |
| M8 | [Payroll & HR](#m8) | P2 | L | Turn our attendance lead into real payroll. |
| M9 | [Sell kashi (productize)](#m9) | P1 | L | Make kashi something another company can buy and be running in a week. |
| M10 | [Mobile sales & warehouse app](#m10) | P2 | L | Beat Wizard's mobile app with our offline sync experience. |
| M11 | [CRM, customer portal & loyalty 2.0](#m11) | P2 | M | Turn our WhatsApp + loyalty lead into a CRM Wizard can't match. |
| M12 | [E-commerce & marketplaces](#m12) | P2 | M | One stock number for shops and online. |
| M13 | [Fixed assets & the rest](#m13) | P3 | M | Fill remaining module checkboxes only when a customer needs them. |

### Suggested order

- **Phase A: Replace Wizard for R1 (M0 → M1 → M2).** Run kashi in parallel with Wizard for one month on the wholesale branch and compare client balances, supplier balances and stock. When they match, stop paying for Wizard's Backoffice.
- **Phase B: Become a full trading ERP (M3 → M4 → M5 → M6).** Bale opening is our unique feature; the ledger is the big one. After M5 our accountant works in kashi too.
- **Phase C: Sell it (M7 + M9).** Approvals/permissions plus multi-tenant, Arabic, Wizard import, transparent pricing.
- **Phase D: Out-feature them (M8, M10, M11, M12).** Payroll from real attendance, offline mobile sales, WhatsApp CRM/portal, e-commerce.
- **M13** only on customer demand.

## Where kashi already wins (amplify these)

- **Offline till with real sync**: kashi-pos runs on local SQLite with an outbox and reconciles. Their Flutter POS also claims offline; compare once reviewed.
- **Branch control from HQ**: Remote commands (reprint, return, replace shift balance, sync now, set PIN, cancel record), signed app updates per register, sync conflict screen.
- **Promotions engine**: Deals (buy X get Y, N for $), rule-group discount lists per branch, coupons with types/categories/lists, sale options. Wizard has basic promotion types.
- **WhatsApp-native**: Meta Cloud API with rules, chats, polls, thank-you/welcome, cost tracking. Wizard only says 'SMS & notifications'.
- **Analytics**: Dashboard tabs, 90-day forecast, stock health, slow movers, product revenue, salesperson performance, targets, busy hours, footfall-to-sales conversion.
- **Staff & cash discipline**: Door-device attendance, shift close vs expected with photo, short-close alerts, manager approvals, PIN lockout.
- **Audit & security**: Before/after audit log, admin activity feed, IP bans, alerts. Wizard leaks passwords and SQL.
- **Modern stack & speed**: Go + Vue 3, WebSocket live updates, fast iteration. Wizard is Yii 1 + jQuery with 20 years of one-off customer forms.

## What Wizard is set up with for R1 today

| Area | What we found |
|---|---|
| Licensed modules | Backoffice, POS Cloud, Reports (the other 11 marketed modules exist in their code but aren't licensed) |
| Company / branch | R1, branch WHOLESALE; 2 warehouses (R1 WholeSale, R1 Retail); 2 cost centers (Default, Retail) |
| Currencies | LBP and USD, USD is the base; per-currency rounding/precision; price factor 20000 |
| Price lists | 7: Bazaar, Default, LL, Outlet, Retail, USD, Wholesale (some marked TTC = VAT-inclusive) |
| Product structure | 4 family levels (Kind / Family 1 / Family 2 / Year), brands used as grades (Crème, Extra, L1–L4, Original, Outlet, UKStock…), seasons (Summer/Winter), units Pcs/KG, color & size ON, supplier item codes |
| Sales options on | advanced pricing, auto-settlement, salesman on documents, discount categories, quotation transfer, color/size on lines |
| Accounting | 5-level Lebanese chart of accounts in use (41110 clients, 40110 suppliers, 428xx staff loans, 4659xx service suppliers, fixed asset & amortization accounts, NSSF payable, taxes due…) |
| Clients | hundreds of wholesale clients incl. many '-Employee' accounts (staff buying on credit), outlets, online, regional dealers |

## Full feature matrix: Wizard vs kashi

Legend: 🟢 kashi ahead · ✅ have · 🟡 partial · ❌ missing · ⚪ skip

### Sales (wholesale documents)

_5 partial, 15 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Quotation / Sales Offer (status, validity days, terms, revise, close, duplicate) | ❌ missing | kashi-pos 'quote' is only a cart price preview |
| Quote → Order → Invoice conversion, one quote to many orders/invoices | ❌ missing |  |
| Sales Order: open / partial / multi-invoice, preparation status (not assigned → preparing → prepared) | ❌ missing |  |
| Shipping info on orders (truck, container, vessel, ETD/ETA, ports, insurance, origin) | ❌ missing | export-oriented; low priority for us |
| Consignment sales (goods at client, unsold visibility) | ❌ missing |  |
| Delivery OUT / Preparation delivery / Packing list documents | ❌ missing |  |
| Sales invoice at HQ for wholesale clients (sub-account, address, terms, price list, salesman, warehouse, cost center) | 🟡 partial | kashi has central sales_invoices but no wholesale workflow; real sales come from tills |
| Price list follows the client | ❌ missing | price lists are per branch; client_type=wholesale is only a label |
| Discount automation by client / product / group / category | 🟡 partial | rule-based discount lists, deals and coupons are strong but per branch, not per client |
| Credit limit checked when the order/invoice is raised | ❌ missing | no credit_limit anywhere in the schema |
| Payment terms, due dates, installments (Sales Bills with interest %) | ❌ missing |  |
| Invoice aging & automated reminder emails | ❌ missing | we have WhatsApp + email infra to do this better |
| Sales commissions by rep / product / order value | 🟡 partial | salesperson per line + performance pages; employees have a commission field; no rules engine |
| Sales order shortage → purchase request / PO | ❌ missing |  |
| Upload invoice/order lines from Excel | ❌ missing |  |
| Recurring invoices | ❌ missing |  |
| Return request with approval, credit notes, debit notes | 🟡 partial | POS returns/exchanges + remote return are good; no credit-note document, no approval |
| Side panel on every document: unpaid invoices, balance, statement, sales history, pending orders | 🟡 partial | client profile page has spend & history; no balances |
| Email the printed document to the client | ❌ missing | we email reports, not documents |
| Discountable bills / bills of exchange | ❌ missing | Lebanese-market instrument; P3 |

### Receivables & payables

_1 partial, 8 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Client balances at HQ (per currency) | ❌ missing | 'On Account' receivables live only in each till's SQLite |
| Receipt voucher allocated to specific invoices (auto-settlement option) | 🟡 partial | PayClientAccount on the till; nothing at HQ |
| Statement of account (client & supplier), dual currency | ❌ missing |  |
| Receivables aging (monthly buckets) & not-yet-due schedule | ❌ missing |  |
| Supplier balances, payment vouchers, supplier aging | ❌ missing |  |
| Down payments (client & supplier) linked to later invoices | ❌ missing |  |
| Payment requests → approval → payment | ❌ missing |  |
| Credit facility types (cash / 30 days / …) per client & supplier | ❌ missing |  |
| Opening balances for clients & suppliers | ❌ missing |  |

### Purchasing / procurement

_1 have, 5 partial, 9 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Supplier records with terms, contacts, product lists | 🟡 partial | name/phone/country/address + costs per product; suppliers can't even be edited (update handler commented out) |
| Supplier price lists, last price paid, compare | 🟡 partial | product_supplier_costs exists; no UI to compare |
| Internal requisition → transfer | 🟡 partial | till transfer requests with HQ approval (good) |
| Purchase requisition / supplier request (SR, RU) | ❌ missing |  |
| RFQ to several suppliers, compare quotes, quote → PO | ❌ missing |  |
| Purchase Order with expected date & delay tracking, consignment POs | ❌ missing |  |
| Delivery IN / goods receipt against PO (partial receipts, 3-way match) | ❌ missing | purchase 'receive' exists but no PO to match against |
| Purchase invoice (receive into warehouse, cost update) | ✅ have | draft/received/cancelled, moving-average cost, FIFO batches |
| Landed cost: allocate freight/customs/clearing over a shipment | ❌ missing | critical for imported bales/containers |
| Generate selling prices from cost + markup % on a purchase | ❌ missing |  |
| Purchase returns / debit notes | ❌ missing |  |
| Multi-level approvals by value / department / role | ❌ missing |  |
| Reorder automation (below min → PO suggestion) | 🟡 partial | low-stock alerts + stock health, no PO generation |
| Shipments & containers, letters of credit (LC) | ❌ missing | P3 unless we import directly |
| Expense purchase orders, purchase expenses, expenses on goods | 🟡 partial | expenses with categories, recurring, photos; not tied to suppliers/POs |

### Accounting / finance

_1 kashi ahead, 1 have, 15 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Chart of accounts (multi-level), posting rules, predefined accounts | ❌ missing |  |
| Automatic posting from every document (sales, COGS, purchases, expenses, payroll) | ❌ missing | entries table only links cash to a source |
| Journal vouchers (duplicate, reverse, account distribution) | ❌ missing |  |
| Trial balance, balance sheet, P&L, cash flow, journal report | ❌ missing | financials page gives sales/gross profit/expenses + forecast |
| Cost centers (multi-level) | ❌ missing | branches/inventories could map to cost centers |
| Deferred expenses & accruals, provisions | ❌ missing |  |
| Budget plan | ❌ missing |  |
| Close year, period lock, opening balances | ❌ missing |  |
| Exchange-rate table by date, rate stored on every document | ❌ missing | one current rate per currency (units_per_usd_micros) |
| Realised / unrealised FX gain-loss, period-end revaluation | ❌ missing |  |
| Bank accounts, deposits, withdrawals, cash-to-bank | ❌ missing | payment methods (cash/Visa/Whish) + cashboxes only |
| Bank reconciliation (bank feed / statement import) | ❌ missing |  |
| Cheques received / paid / post-dated, checkbooks, maturity tracking | ❌ missing | very common in Lebanese wholesale |
| VAT categories per product, VAT-inclusive (TTC) price lists, VAT reports, MOF numbers | ❌ missing | one tax_rate per till |
| Bulk upload of journal entries / invoices | ❌ missing |  |
| Expenses with categories, recurring expenses, photos | 🟢 kashi ahead | phone-photo upload, from-central flag, recurring auto-post |
| Staff/partner loans with payments | ✅ have | loans + loan payments + photos |

### Inventory

_3 kashi ahead, 4 have, 8 partial, 5 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Multi-warehouse stock with warehouse groups | ✅ have | inventories of type warehouse/store, treemaps |
| Transfers between locations | 🟢 kashi ahead | request → approve → dispatch → receive with received qty and photos |
| Units of measure, packing, conversions, decimal quantities (KG) | ❌ missing | quantities are bigint; bales by KG need this |
| Barcodes & label printing | 🟢 kashi ahead | label designer, barcode PDF |
| Min / max per item per location, below-min, reorder suggestions | 🟡 partial | low-stock threshold + alerts |
| Stock count by item / group / brand / category | 🟡 partial | stock counts with count sheets per inventory (draft/post) |
| Count & compare: several counters, merge, approve differences | ❌ missing |  |
| Stock adjustment with reasons & upload from file | ✅ have | adjustments with reason + stock_movements ledger |
| Product catalogue with images & specs | 🟡 partial |  |
| Categorisation by many attributes | 🟢 kashi ahead | colors, sizes, attributes, autofill rules (11k products auto-categorised) |
| Expiry, lot & serial tracking | 🟡 partial | FIFO purchase batches only |
| Bulk product upload via Excel | ✅ have | product CSV import/export |
| Warehouse replenishment suggestions between branches | 🟡 partial | stock health / slow movers is a good base |
| Item aging report | 🟡 partial | slow movers |
| Valuation: average cost / landed cost; COGS posted as goods move | 🟡 partial | moving average yes; landed no; no ledger posting |
| Unpacking / disassembly (one item → many, e.g. bale → graded pieces) | ❌ missing | very relevant to the bale business |
| Waste / damage document | 🟡 partial | adjustment with reason |
| Inventory with orders: balance incl. open sales & purchase orders | ❌ missing | needs orders first |
| Closing inventory per fiscal year | ❌ missing |  |
| Movement log with who/when/where | ✅ have | stock_movements + audit_logs |

### Warehouse management

_2 partial, 3 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Bins / zones / shelf locations | ❌ missing |  |
| Barcode-driven receive / pick / pack / dispatch | ❌ missing |  |
| Inbound / outbound documents with approval flows | 🟡 partial | transfers have approval; no DO/DI documents |
| Warehouse staff raising procurement requests | 🟡 partial | till transfer requests |
| Mobile scanning for all warehouse ops | ❌ missing | phones only used for photo uploads today |

### Products & pricing

_1 kashi ahead, 5 partial, 3 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Secondary name (Arabic), HS code, season, year level | 🟡 partial | attributes can hold some of these |
| Supplier item code, supplier price + markup % | 🟡 partial | supplier costs per product |
| Weight / volume / density / dimensions | ❌ missing |  |
| Warning message shown when the item is sold | ❌ missing |  |
| 'Authorize sales below cost' per product | 🟡 partial | oversell checks exist; below-cost guard unclear |
| Price history per product & bulk change prices with history | 🟡 partial | price lists editable; no history view |
| Product types: stock / service / produced / composite (bundles) | 🟡 partial | deals cover bundles at the till |
| Promotions: by barcode, quantity, client, gift product, rebates | 🟢 kashi ahead | deals, discount rules, coupons, sale options |
| Scale barcodes (price/weight embedded) | ❌ missing |  |

### POS (their Flutter app not reviewed yet — statuses are provisional)

_6 have, 2 partial, 2 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Works offline, syncs with HQ | ✅ have | true local SQLite + outbox; their Flutter POS app not reviewed yet |
| Multi cash & currencies at the till | ✅ have | USD/LBP, Visa, Whish, LBP rounding |
| Link branches / see other branches' stock | ✅ have |  |
| Runs on Android / any tablet | 🟡 partial | Wails desktop (Windows/macOS); their POS is Flutter (Android/Windows) |
| POS settlement, POS shift | ✅ have | expected vs counted, photo, Z-report, auto-close, HQ alerts |
| POS vouchers | ✅ have | coupons |
| POS sales order / customer order from the till | ❌ missing |  |
| Loyalty points check & redeem at POS | 🟡 partial | points earned per invoice; redemption flow to verify |
| Cash-drawer kick | ❌ missing | no drawer-open ESC/POS command found |
| Remote commands, signed auto-updates, sync conflicts | ✅ have | not visible in their web app; check their POS app |

### CRM

_2 kashi ahead, 5 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Leads with custom stages, pipeline by stage/rep/region | ❌ missing |  |
| Tasks, calls, meetings, reminders | ❌ missing | calendar_events exists |
| Visit planning, check-ins, missed visits | ❌ missing |  |
| Bulk lead import, lead → prospect → client conversion | ❌ missing |  |
| Win/loss, lead source, sales-cycle reports | ❌ missing |  |
| Customer messaging | 🟢 kashi ahead | WhatsApp Cloud API: rules, chats, thank-you/welcome, polls, cost tracking |
| Reputation | 🟢 kashi ahead | Google reviews + review links |

### Loyalty

_2 have, 1 partial, 2 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Membership tiers, automatic tiering | ✅ have | loyalty page with tiers |
| Point rules by amount / product / frequency | 🟡 partial |  |
| Campaigns: double points for holidays / items | ❌ missing |  |
| Customer portal: balance & history on phone/browser | ❌ missing |  |
| Real-time point sync across branches | ✅ have |  |

### Payroll & HR

_1 kashi ahead, 1 have, 1 partial, 5 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Employee records | ✅ have |  |
| Attendance | 🟢 kashi ahead | Akuvox door device, change requests with approval, complaints, timesheets |
| Leave requests (absent, day off, sick, half-day, vacation) with balances | ❌ missing | attendance change requests only |
| Salary history with effective dates | ❌ missing | single salary snapshot |
| Payroll run from attendance, payslips, bonuses, deductions | ❌ missing | late units typed by hand |
| Advances on salary / loan deductions from payroll | 🟡 partial | loans exist, not deducted |
| NSSF & income-tax reports, end-of-service indemnity | ❌ missing |  |
| Appraisals, recruitment, CVs, interviews, resignation | ❌ missing | P3 |

### Mobile sales app

_7 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Floor sales / sales orders from a phone | ❌ missing |  |
| Van sales: stock loaded in vehicle, cash sales | ❌ missing |  |
| Collections with printed receipt | ❌ missing |  |
| Routes, check-ins, salesman GPS, visit reports | ❌ missing |  |
| Delivery trips, drivers, close delivery at client | ❌ missing |  |
| Offline counting, merge counters, submit for approval | ❌ missing |  |
| Mobile CRM (prospects, quotations) | ❌ missing |  |

### E-commerce

_3 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Push products, prices, discounts, images to website | ❌ missing |  |
| Live stock to Shopify / WooCommerce / Magento / Amazon | ❌ missing |  |
| Import web orders → invoice + delivery note | ❌ missing |  |

### Fixed assets

_1 have, 1 partial, 3 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Asset register with categories & location | ✅ have | assets + categories + treemap |
| Depreciation methods, auto-posted monthly | ❌ missing |  |
| Disposal / sale / write-off | ❌ missing |  |
| QR tags + scan audit | 🟡 partial | barcode infra exists |
| Costs logged against the asset | ❌ missing |  |

### Platform & administration

_2 kashi ahead, 1 have, 7 partial, 2 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Generic approval engine for any document type | 🟡 partial | transfers, attendance, manager password on till |
| Access per document type, per user restrictions (branch/warehouse/salesman/cost center) | 🟡 partial | per-page none/view/edit roles |
| Audit trail per document | 🟢 kashi ahead | before/after JSON audit_logs, admin activities, home feed |
| Attachments with folders on any document | 🟡 partial | phone photos for expenses/loans/shifts/transfers |
| Print template designer for every document | 🟡 partial | receipt designer + label designer |
| Every grid: column chooser, saved templates, PDF/Excel export | 🟡 partial | XLSX in reports; not universal |
| Global search | ❌ missing |  |
| Notifications | ✅ have | home activity feed + WebSocket push |
| Languages | 🟡 partial | Wizard: EN/FR UI. kashi: English UI, Arabic in reports |
| Multi-company / company groups, per-company licensing | ❌ missing | needed to sell kashi |
| Data import from another system (COA, clients, products, balances) | 🟡 partial | product & client CSV imports |
| Security | 🟢 kashi ahead | IP bans, country filter, security alerts; Wizard leaks plaintext passwords in page source |

### Other Wizard modules (vertical)

_1 missing, 5 skip_

| Wizard capability | kashi | Notes |
|---|---|---|
| Production / BOM / production orders / waste / QC | ⚪ skip | only 'bale sorting' matters to us, covered by unpacking |
| Maintenance contracts & service tickets by serial | ⚪ skip | for elevators/solar/electronics |
| Project management (divisions, budgets, material per project) | ⚪ skip |  |
| Rent contracts, property | ⚪ skip |  |
| Freight forwarding shipments | ⚪ skip |  |
| Client portal (statements, invoices, points) | ❌ missing | worth building as part of loyalty/receivables |

### More Wizard detail: warehouse & stock discipline (industry/guide pages)

_5 partial, 9 missing_

| Wizard capability | kashi | Notes |
|---|---|---|
| Blind count sheets (counter doesn't see expected qty) | ❌ missing | easy win on our stock counts |
| Two counters on valuable items, compare before posting | ❌ missing | fits the multi-counter count in M3 |
| Reason required on every adjustment | 🟡 partial | reasons exist; make them mandatory with a list |
| Quarantine on receipt (received but not sellable until checked) | ❌ missing | useful for bales before sorting |
| Bin / shelf locations, zones that match the building, receiving that ends at a bin | ❌ missing |  |
| First-expiry-first-out picking, expiry alerts by window | 🟡 partial | FIFO batches only |
| Cases and pieces (buy cartons, sell singles) | ❌ missing | units & packing (M0) |
| Bulk loading of vehicles / trips | ❌ missing | van sales (M10) |
| Thresholds per branch (min/max per location) | 🟡 partial | global low-stock threshold |
| Stock by location including customer sites (consignment) + ageing on consigned units | ❌ missing |  |
| Lead time per supplier, purchase orders raised from a shortfall | ❌ missing |  |
| Return rate by product, damage & returns tracking | 🟡 partial | returns exist; no return-rate report |
| Margin per item / per shipment / per customer | 🟡 partial | gross profit overall; not per shipment or client |
| Price history per item | ❌ missing |  |

### More Wizard detail: money & billing

_2 partial, 7 missing, 1 skip_

| Wizard capability | kashi | Notes |
|---|---|---|
| Partial payments across different exchange rates | ❌ missing | needs dated rates (M0) |
| Balances viewable in the trading currency + translated balance | ❌ missing |  |
| Reporting currency independent of trading currency | ❌ missing |  |
| Retention held as a visible balance on invoices | ❌ missing | seen as retention_pc on their invoice form |
| Advance payments recovered against later invoices | ❌ missing | down payments |
| Milestone billing, recurring retainers, unbilled time | ⚪ skip | services businesses |
| Subscription & membership billing with renewals | ❌ missing | could matter for loyalty memberships |
| Automated payment reminders & invoice aging emails | ❌ missing | do it on WhatsApp (M1) |
| Credit note posted together with the stock movement | 🟡 partial | returns move stock; no credit note doc |
| Investor / owner financial reporting | 🟡 partial | dashboard + financials, no statements |

### More Wizard detail: people, service & channels

_4 partial, 4 missing, 2 skip_

| Wizard capability | kashi | Notes |
|---|---|---|
| Staff & shift scheduling (rota) | ❌ missing | we have attendance but no planned schedule to compare against |
| Workforce / field service scheduling | ⚪ skip |  |
| SMS notifications to customers | 🟡 partial | WhatsApp instead (better in Lebanon) |
| Warranty tracking & warranty cost | ⚪ skip | electronics |
| AI-powered sales analytics (claimed on online-retail page) | 🟡 partial | we have analytics + forecast; no AI layer yet |
| Demand forecasting & order planning | 🟡 partial | stock health + 90-day financial forecast |
| Route optimisation & delivery scheduling | ❌ missing | M10 |
| Automated carrier selection, shipping & fulfilment | ❌ missing | local delivery companies (M12) |
| Multi-channel order processing (B2B + B2C, web + marketplace + shop) | 🟡 partial | shops only |
| Customs & trade compliance, duty and clearance, inland transport | ❌ missing | landed cost covers our need |

### More Wizard detail: vertical modules they sell (from 23 industry pages)

_12 skip_

| Wizard capability | kashi | Notes |
|---|---|---|
| Restaurants: menus, recipes & sub-recipes, yield & waste factors, kitchen workflow, tables | ⚪ skip |  |
| Hotels: room reservations, guest profiles, guest loyalty | ⚪ skip |  |
| Clinics: patients, appointments, insurance billing, medical inventory | ⚪ skip |  |
| Gyms: memberships, renewals, staff scheduling | ⚪ skip |  |
| Real estate / property: listings, leases, tenants, rent & maintenance, income per property | ⚪ skip |  |
| Contracting: projects, divisions, budgets, material per site | ⚪ skip |  |
| Freight forwarding: multi-modal shipments, margin per shipment/lane, customs | ⚪ skip |  |
| Solar: installations, energy monitoring, maintenance | ⚪ skip |  |
| Events: RSVP, ticketing, vendors | ⚪ skip |  |
| SaaS / IT: subscriptions, churn analytics, agile projects, timesheets | ⚪ skip |  |
| Pharma: batch & expiry, cold storage, GMP/FDA compliance reports | ⚪ skip |  |
| Customization business model: one-off forms per client (e.g. 'Otis Contract', 'Yarz Member', 'Leave Request-Califood') | ⚪ skip | they say yes to everything; we shouldn't |

## Milestones in detail

<a id="m0"></a>
### M0 · Document engine & money foundations

**Priority:** P0 · **Size:** L (3–6 weeks)

**Goal:** Build once what every later document needs, so M1–M6 are mostly screens, not plumbing.

**Why:** Wizard's strength is that every document (offer, order, invoice, receipt, PO, transfer…) behaves the same: numbering, status, convert-to-next, duplicate, Excel upload, print, email, attachments, audit. If we build each document by hand we will never catch up; if we build the engine, each new document is a week.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Generic document header/lines model** | P0 | M | doc_types table (code like SA/SO/SF/PO/RV, name, numbering prefix per branch/year, default warehouse, posting behaviour). documents(id, type, number, date, branch, warehouse, party_type/party_id, currency, rate, status, ref, notes, created_by) + document_lines(product/variant, qty decimal, unit, unit_price, discount, tax_code, line_total, warehouse) + document_links(from_id, to_id, line mapping, qty) for partial conversions. |
| **Status machine + convert-to-next** | P0 | M | draft → confirmed → partially fulfilled → closed/cancelled; 'Transfer to order/invoice/PO' copies open quantities and links lines, so partial delivery/invoicing is tracked automatically. |
| **Exchange-rate table by date** | P0 | S | currency_rates(currency, date, rate, source, entered_by); every document stores original currency, amount, rate and rate date. Keep the current units_per_usd_micros as 'today's rate' for the till. Allow correcting a rate and re-processing affected docs. |
| **Decimal quantities + units of measure** | P0 | M | units(name, symbol, precision), product_units(product, unit, factor_to_base, barcode, is_purchase_default, is_sales_default). Quantities become numeric(14,3) at HQ; till keeps integers for pieces but supports KG items. Needed for bales sold by the kilo. |
| **Universal grid** | P0 | M | One Vue grid component: column chooser, saved column templates per user, filters (Today/Week/Month/Quarter/Year/All), totals row, PDF + Excel export, row click → document. Wizard uses this everywhere and users love it. |
| **Excel line upload on any document** | P1 | S | Paste or upload code/barcode + qty + price; validate; preview; insert. |
| **Print templates per document type + email/WhatsApp the PDF** | P1 | M | Extend the receipt designer into A4 templates (header, logo, columns, totals, footer, Arabic). Send via email (Resend) or WhatsApp document message. |
| **Attachments on any document** | P1 | S | Reuse the phone-upload/QR flow; attach images or PDFs to any document id. |
| **Per-document audit & history tab** | P1 | S | We already have audit_logs; show it on every document with who/when/what changed. |

**Done when:** A test document type can be created, numbered, converted, partially fulfilled, printed, emailed, exported and audited without type-specific code.


<a id="m1"></a>
### M1 · Receivables & wholesale selling

**Priority:** P0 · **Size:** L (3–6 weeks)

**Goal:** Run the R1 wholesale branch's selling and debt collection in kashi.

**Why:** This is the daily pain Wizard solves for us: who owes what, statements, credit limits, wholesale prices by client. It's also the #1 thing every Lebanese wholesaler asks for (their site: 'Cash, cheques and long collection cycles').

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Client accounts at HQ** | P0 | M | client_ledger(client, date, doc_type, doc_id, currency, debit, credit, rate, due_date). Sync till 'On Account' entries up so HQ sees every client's balance per currency. Opening balances import. |
| **Credit limit, payment terms, credit facility** | P0 | S | clients.credit_limit (+currency), clients.payment_terms_days / terms table (cash, 30, 60, installments). Block or require manager approval when an order/invoice would exceed the limit or the client has overdue invoices — at HQ and on the till. |
| **Price list per client / client group** | P0 | S | clients.price_list_id and client_groups with default price list + discount list. Till and HQ invoice pick the client's list automatically. Makes client_type=wholesale real. |
| **Wholesale invoice at HQ** | P0 | M | On the M0 engine: client, address, salesman, warehouse, currency + rate, price list, line discount, invoice discount, VAT, notes, delivery date/address. Fast keyboard line entry by code/barcode, color/size grid entry, Excel upload, KG items. |
| **Receipt voucher with allocation** | P0 | M | Receive cash/cheque/Whish/bank in any currency, allocate to oldest or chosen invoices (auto-settlement option), change rate, print receipt, WhatsApp receipt to client. |
| **Statement of account** | P0 | S | Per client, date range, per currency columns, running balance, opening balance; PDF + WhatsApp + email. |
| **Aging & not-yet-due** | P0 | S | Buckets 0–30/31–60/61–90/90+ and future due schedule, by client, salesman, region; dashboard card. |
| **Return sales invoice & credit note** | P0 | S | Return against an invoice or free return; credit note for price adjustments; both hit the client ledger. |
| **Client side panel** | P1 | S | On every sales document: balance, overdue, last 10 invoices, last price for this item, pending orders, quick statement. |
| **WhatsApp collection reminders** | P1 | S | Rules: 'X days overdue → send statement'; one-click 'send statement' to all overdue clients. This beats Wizard's email reminders in Lebanon. |
| **Installment plans (Sales Bills)** | P2 | S | Split an invoice into N bills with optional interest %; due-date tracking. |

**Done when:** R1 wholesale can invoice, collect, send statements and see aging entirely in kashi for a full month in parallel with Wizard, and the balances match.


<a id="m2"></a>
### M2 · Purchasing, payables & landed cost

**Priority:** P0 · **Size:** L (3–6 weeks)

**Goal:** Buy, receive, cost and pay suppliers in kashi.

**Why:** Our real margin is decided at purchase time: bale/container cost + freight + customs + clearing. Without landed cost and supplier balances we can't price correctly or know what we owe.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Supplier CRUD done properly** | P0 | S | Edit/deactivate, contacts, currency, payment terms, supplier categories (China, Dubai, Turkey…), products & item codes, notes. |
| **Purchase Order** | P0 | M | PO with expected date, delay days, partial receipts, close/cancel; PO → receipt → invoice conversion on the M0 engine. |
| **Goods receipt against PO (3-way match)** | P1 | M | Receive by scan, record differences, invoice only what was received; warn when invoice qty/price differs from PO/receipt. |
| **Landed cost allocation** | P0 | M | 'Expenses on goods' document linked to one or more purchase invoices; distribute by value, quantity, weight or manual; updates item cost and batch cost. Clearing/freight suppliers get their own payables. |
| **Generate selling prices from cost** | P0 | S | On a received purchase: cost × markup % (per category/price list) → proposed prices for each price list, review then apply. Keeps price history. |
| **Supplier ledger, payment vouchers, supplier aging** | P0 | M | Mirror of M1 for suppliers: balances per currency, pay by cash/cheque/bank, allocate to invoices, statements, aging. |
| **Purchase returns & debit notes** | P0 | S | Return against a purchase; reverses stock and supplier balance. |
| **Payment requests with approval** | P1 | S | Request → manager approves → becomes a payment voucher. |
| **Reorder suggestions → PO** | P1 | S | From min/max and stock-health velocity, propose quantities per supplier; one click to create POs. |
| **Supplier price history & compare** | P2 | S | Last price paid per supplier per item, quote comparison table. |

**Done when:** A container/bale purchase can be ordered, received, landed-costed, priced and paid in kashi, and the supplier statement matches the supplier's.


<a id="m3"></a>
### M3 · Bale opening, units & inventory depth

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Model how the clothing-bale business really works, plus the inventory controls Wizard sells on.

**Why:** Wizard has generic Unpacking/Disassembly; we can build a purpose-made 'bale opening' flow that no generic ERP has, and that is a selling point to every bale importer in Lebanon.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Bale opening / sorting document** | P1 | M | Input: a bale (KG or piece, with landed cost). Output: graded products (Crème, Extra, L1–L4…) with quantities; cost allocated by weight, by grade ratio or by expected selling value; waste line. Stats per supplier: grade yield %, cost per graded piece. |
| **Unpacking (carton → pieces) and kitting** | P1 | S | Generic version of the above using product_units factors. |
| **Min / max per item per location** | P1 | S | Per inventory; below-min list; feeds reorder and replenishment. |
| **Branch replenishment suggestions** | P1 | S | From warehouse to shops using sales velocity (stock health already computes it) → draft transfers. |
| **Count & compare** | P1 | M | Count by brand/category/group slices; several counters (phones) in parallel; merge; show differences; approve → adjustment. |
| **Stock aging report** | P1 | S | Age buckets by receipt batch; markdown suggestions for old stock (our sale options/discount lists can apply them). |
| **Balance incl. orders** | P2 | S | Available = on hand − reserved (open SO) + incoming (open PO). |
| **Product extras** | P2 | S | Secondary/Arabic name, HS code, season, year, weight, warning message on sale, 'allow below cost', price history tab. |
| **Serial / lot / expiry (optional per product)** | P3 | M | Only if we target electronics/pharma/food. |

**Done when:** A real bale can be opened into graded stock with correct unit costs, and weekly counts are done from phones by several staff.


<a id="m4"></a>
### M4 · Quote → Order → Delivery chain

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Full wholesale order flow for bigger clients and dealers.

**Why:** Their 'Sales' module pitch is exactly this. Dealers order, we prepare, deliver in parts, invoice.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Quotation** | P1 | S | Validity days, terms, status (open/sent/won/lost/expired), revise, duplicate, convert to order; WhatsApp the PDF. |
| **Sales order** | P1 | M | Reserve stock, partial delivery & invoicing, consignment flag, preparation status (assigned picker → preparing → prepared), close/cancel, return order. |
| **Delivery note / preparation** | P1 | S | Pick list per warehouse, scan to confirm, delivery note signed (photo on phone), then invoice. |
| **Shortage → purchase** | P2 | S | Lines not in stock become a supplier requisition/PO linked to the order. |
| **Commissions** | P2 | S | Rules by rep/product/category/collected amount; monthly commission report feeding payroll (M8). |
| **Recurring invoices** | P3 | S | For rent/service style clients. |

**Done when:** A dealer order can be quoted, confirmed, prepared, delivered in two parts and invoiced twice with stock and balances right.


<a id="m5"></a>
### M5 · General ledger & financial statements

**Priority:** P1 · **Size:** XL (6+ weeks)

**Goal:** Become an ERP, not 'POS + back office'. Required to fully drop Wizard and to sell to anyone with an accountant.

**Why:** Every Wizard page leads back to accounting ('Accounting sits underneath everything'). The ledger is the moat; it's also the biggest single build. Do it after M1–M2 so sub-ledgers already exist and posting rules are obvious.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Chart of accounts with Lebanese template** | P1 | M | Multi-level accounts (class → group → account → auxiliary); ship the Lebanese standard plan as a template; client/supplier/employee auxiliaries map to 411/401/42x. |
| **Journal & posting rules** | P1 | L | journal_entries + journal_lines (account, debit, credit, currency, original amount, rate, cost center, doc link). Posting rules per document type (sales → 411/701/VAT; COGS → 601/37; purchase; expenses; loans; shift cash differences; payroll). Idempotent re-posting. |
| **Journal voucher** | P1 | S | Manual JV with duplicate/reverse, recurring JVs, Excel upload. |
| **Cost centers** | P1 | S | Map branches/inventories/departments; every line can carry one; P&L by cost center. |
| **Reports** | P1 | M | Trial balance (with auxiliaries), general ledger, account statement, P&L, balance sheet, cash flow, journal report; in USD or LBP; compare periods; PDF/Excel. |
| **FX revaluation** | P1 | M | Period-end revaluation of open monetary balances at closing rate; realised vs unrealised gain/loss accounts; reversible journal. |
| **Period lock & year close** | P1 | S | Lock dates, closing entries, opening balances of next year, closing inventory. |
| **Deferred expenses / accruals / provisions** | P2 | S | Spread an expense over N months automatically. |
| **Budget vs actual** | P3 | S |  |

**Done when:** Our accountant can produce the year's trial balance and P&L from kashi and it matches Wizard's for the same period.


<a id="m6"></a>
### M6 · Banks, cheques & VAT

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Cover how Lebanese businesses actually get paid and file taxes.

**Why:** Wizard's 'built for Lebanon' pitch = USD/LBP in one ledger, VAT that reconciles, cheques and long collection cycles.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Bank accounts & movements** | P1 | S | Bank accounts per currency, deposit, withdraw, cash-to-bank, transfers between accounts, bank charges. |
| **Cheques** | P1 | M | Cheques received (from clients) and issued (checkbooks), post-dated register, maturity calendar, deposit/collect/bounce/return flows, notifications on due dates. |
| **Bank reconciliation** | P2 | M | Import statement CSV/Excel, auto-match by amount/date/ref, manual match, reconciliation report. |
| **VAT** | P1 | M | Tax codes per product/category, VAT-inclusive (TTC) price lists, VAT on purchase & sale lines, MOF number on clients/suppliers, VAT return report (sales/purchases, collected/deductible), per-till tax_rate replaced by codes. |
| **Fiscal e-invoicing readiness** | P3 | S | Keep the document model ready for a future Lebanese e-invoicing mandate (sequential numbers, QR, immutable after post). |

**Done when:** A quarter's VAT return and all cheque movements come out of kashi.


<a id="m7"></a>
### M7 · Approvals, permissions & control

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Give owners the control Wizard sells to managers, with our better UX.

**Why:** Wizard: 'multi-level approvals by value, department, or user roles', access per document type, per-user branch/warehouse limits.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Generic approval engine** | P1 | M | Rules per document type: condition (amount > X, discount > Y%, below cost, credit limit exceeded) → approver role/user, multi-step. Approve from the web, the till or a WhatsApp link. |
| **Document-level permissions** | P1 | S | Per role per document type: view/create/edit/post/cancel/print/export; field-level hide cost & margin. |
| **Scope restrictions** | P1 | S | Users limited to branches, warehouses, salesmen, cost centers, price lists. |
| **Global search** | P2 | S | Search any client, product, document number, cheque, phone from one box. |
| **Notifications center** | P2 | S | Approvals waiting, overdue clients, cheques due, low stock, short shifts; push + WhatsApp. |

**Done when:** A discount above the threshold on any document stops and waits for the right approver, and a branch user can only see their branch.


<a id="m8"></a>
### M8 · Payroll & HR

**Priority:** P2 · **Size:** L (3–6 weeks)

**Goal:** Turn our attendance lead into real payroll.

**Why:** We already capture attendance from door devices, which Wizard doesn't. Payroll on top of it (with NSSF and end-of-service) closes the loop.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Salary history & contracts** | P2 | S | Effective-dated salary, allowances, transport, commission rules, working schedule. |
| **Leave management** | P2 | S | Leave kinds (absent, day off, sick, half-day, vacation), balances accrued by policy, request/approve from the till. |
| **Payroll run** | P2 | M | Pull attendance (late/overtime computed, not typed), leave, commissions (M4), advances & loan deductions; payslips; post to ledger; pay by cash/bank. |
| **Payslips by WhatsApp** | P2 | S | PDF payslip to each employee. |
| **NSSF & tax reports, end-of-service** | P2 | M | Lebanese NSSF contributions, income tax, end-of-service indemnity accrual. |

**Done when:** The month's payroll is calculated from attendance and paid from kashi.


<a id="m9"></a>
### M9 · Sell kashi (productize)

**Priority:** P1 · **Size:** L (3–6 weeks)

**Goal:** Make kashi something another company can buy and be running in a week.

**Why:** To be Wizard's 'nightmare competitor' it has to be installable for others, not only for us. Their weak spots: price on request, slow onboarding, old UI, outages like the current upgrade.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Multi-tenant / multi-company** | P1 | L | Tenant isolation (schema or tenant_id), company groups, per-company settings, branding, subdomains. |
| **Module licensing & plans** | P1 | S | Switch modules on per tenant (POS, wholesale, purchasing, accounting, payroll…), per-register/per-user pricing, public price list (Wizard hides prices: be transparent). |
| **Migration from Wizard (and Excel)** | P1 | M | Import clients (with balances), suppliers, products with family levels/brands/units/barcodes/prices per list, chart of accounts, opening stock & cost, open invoices. We know their exact data model now. |
| **Onboarding wizard** | P1 | S | Company, currencies & rates, branches, warehouses, taxes, price lists, users; demo data. |
| **Languages: Arabic UI + French** | P1 | M | Full i18n with RTL; Arabic documents and receipts. |
| **Android / tablet till** | P2 | L | Same Vue frontend packaged for Android (Capacitor) with SQLite, or a PWA offline mode. |
| **Reliability & trust** | P1 | S | Zero-downtime migrations (already additive), status page, daily backups with export, 2FA, SSO later. |
| **Docs & sales site** | P2 | M | User guide (exists), comparison pages vs Wizard/Odoo, videos, demo booking. |

**Done when:** A new retailer signs up, imports from Wizard/Excel and sells on day one.


<a id="m10"></a>
### M10 · Mobile sales & warehouse app

**Priority:** P2 · **Size:** L (3–6 weeks)

**Goal:** Beat Wizard's mobile app with our offline sync experience.

**Why:** Their mobile app is a big part of the wholesale pitch (van sales, collection, routes, offline count). We already know how to do offline sync.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Salesman app (PWA/Android)** | P2 | L | Client list with balance/credit/overdue, catalogue with stock & client price, orders & quotations, offline with sync. |
| **Collections** | P2 | S | Receive payment on the spot, Bluetooth receipt or WhatsApp receipt, allocates in M1 ledger. |
| **Van sales** | P2 | M | Vehicle = inventory; load/unload transfers; cash sales from van; end-of-day van settlement. |
| **Routes & visits** | P3 | M | Planned visits, check-in with GPS, missed / no-order visits report. |
| **Warehouse scanning** | P2 | M | Receive against PO, pick sales orders, count, transfer — all by phone camera or Bluetooth scanner. |

**Done when:** A rep can take orders and collect money offline for a day and everything syncs cleanly.


<a id="m11"></a>
### M11 · CRM, customer portal & loyalty 2.0

**Priority:** P2 · **Size:** M (1–3 weeks)

**Goal:** Turn our WhatsApp + loyalty lead into a CRM Wizard can't match.

**Why:** Their CRM is classic leads/stages; ours can live inside WhatsApp where Lebanese customers actually are.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Leads & pipeline** | P2 | M | Leads from WhatsApp chats, forms, imports; custom stages; tasks & reminders; convert to client; win/loss reports. |
| **Customer portal** | P2 | M | Phone-number login (WhatsApp OTP): loyalty points, invoices, statement & balance, receipts, reviews. |
| **Loyalty rules & campaigns** | P2 | S | Earn by amount/product/category, double-points periods, birthday rewards, redemption at POS, tier upgrades automatic. |
| **Segments & broadcasts** | P2 | S | Segments by spend/recency/branch (we already export VCF lists by hand) → WhatsApp campaigns with opt-out and cost estimate. |

**Done when:** A WhatsApp enquiry becomes a lead, then a client, and that client checks their points and balance from a link.


<a id="m12"></a>
### M12 · E-commerce & marketplaces

**Priority:** P2 · **Size:** M (1–3 weeks)

**Goal:** One stock number for shops and online.

**Why:** Wizard sells Shopify/WooCommerce/Magento sync; online clothing in Lebanon runs on Instagram + WhatsApp + Shopify/WooCommerce.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Shopify & WooCommerce connectors** | P2 | M | Push products/variants/images/prices/discounts, live stock per location, pull orders → sales order → delivery → invoice. |
| **Online stock allocation** | P3 | S | Which inventories feed the web; safety stock. |
| **Delivery companies** | P3 | S | COD reconciliation with local delivery companies. |

**Done when:** An online order reserves stock in kashi within a minute and is invoiced from the same flow as a shop order.


<a id="m13"></a>
### M13 · Fixed assets & the rest

**Priority:** P3 · **Size:** M (1–3 weeks)

**Goal:** Fill remaining module checkboxes only when a customer needs them.

**Why:** Low value for retail; good for completeness in sales conversations.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Depreciation & disposal** | P3 | S | Methods, useful life, monthly auto-posting, disposal/sale with gain/loss. |
| **Asset QR audit** | P3 | S | Scan tags by phone, compare with register by location. |
| **After-sales tickets** | P3 | M | Only if we target electronics: tickets, warranty by serial, parts & labour. |
| **Production / projects** | P3 | L | Skip unless a paying customer needs it. |

## Part 2 — kashi vs Odoo

- **Odoo is the giant.** ~45 apps in one suite (website, sales, finance, supply chain, HR, marketing, services, productivity, AI, Studio). In Lebanon it's sold and implemented through partners (e.g. Circosoft), not by Odoo directly.
- **Pricing (odoo.com/pricing):** one app free with unlimited users; 'Standard' all apps and 'Custom' (Studio, agentic AI, multi-company, API) are billed per user per month (the page shows roughly US$ 9–21 per user per month depending on plan and discount). Implementation, custom code, Odoo.sh hosting and in-app credits (SMS, AI bill scanning) are extra — the real cost in Lebanon is the partner's implementation.
- **Lebanon fit is weak out of the box.** We found no official Lebanese localization: the only `l10n_lb` modules on apps.odoo.com are community charts of accounts for Odoo 13 and 14. No Lebanese payroll (NSSF, end-of-service). Multi-currency is strong in accounting (daily automated rates) but the POS only lists currency rounding, not USD+LBP cash at the till.
- **Where Odoo is stronger than Wizard:** modern UI, customer portal with online payment & e-signature, AI invoice scanning, bank sync, marketing apps, website/eCommerce, Studio (no-code customization), huge app marketplace. **Where Wizard is stronger than Odoo:** Lebanese accounting habits (USD/LBP ledger, cheques, VAT reports, NSSF payroll), one vendor doing support.
- **Where kashi beats both:** offline-first multi-branch retail with real sync, USD/LBP cash at the till, WhatsApp-native customer flows, door-device attendance, footfall/conversion, retail analytics, and the speed to build Lebanon-specific features they won't.

Of **89** Odoo capabilities: 🟢 10 kashi ahead, ✅ 7 have, 🟡 28 partial, ❌ 35 missing, ⚪ 9 skip.

### Odoo · Point of Sale (Odoo POS Shop)

| Odoo capability | kashi | Notes |
|---|---|---|
| Offline mode, offline payments synced on reconnect | ✅ have | kashi-pos is offline-first by design (local SQLite + outbox) |
| Connect devices: scanners, printers, cash drawers, payment terminals (IoT box) | 🟡 partial | ESC/POS printers, pole display, scanner detection; no card terminals, no drawer kick |
| PC and tablet, browser support | 🟡 partial | Windows/macOS desktop; no Android/iPad |
| Payment methods, split payments, currency rounding | 🟢 kashi ahead | USD/LBP/Visa/Whish with LBP rounding and change in LBP |
| Card payments via external terminals | ❌ missing |  |
| Bill splitting, tips | ⚪ skip | restaurant features |
| Invoice from POS, B2B customers, credit limits at POS | 🟡 partial | On Account exists; no credit limit |
| Prevent cash payment-method re-use / cash control | 🟢 kashi ahead | shift close vs expected, manager approvals, PIN lockout |
| Prices & discounts, pricelists | 🟢 kashi ahead | deals, rule-based discount lists, coupons |
| Parallel orders | ✅ have | hold invoices |
| Customized receipts | 🟢 kashi ahead | receipt designer with themes |
| Weighing at the counter (scale integration), weighed barcodes | ❌ missing |  |
| Sell on the move (phone/tablet POS) | ❌ missing |  |
| Dynamic barcodes (price/weight/loyalty embedded) | ❌ missing |  |
| Order history, daily sales, cash flows | ✅ have |  |
| Franchises / multi-store | 🟢 kashi ahead | remote commands, signed updates, per-branch settings |
| Self-service kiosk, preparation display | ⚪ skip |  |
| Customer registration at POS, identify customers | ✅ have | phone lookup, required-client setting |
| Loyalty cards (points) | ✅ have | loyalty points & tiers |
| eWallet (store credit topped up or from refunds) | ❌ missing | useful for exchanges/refunds |
| Gift cards | 🟡 partial | coupons; no sellable gift card with balance |
| Units of measure, multiple barcodes, variants | 🟡 partial | variants & barcodes yes; no UoM |
| Product availability across shops | ✅ have | other branches' stock from the till |
| Ship later (order now, deliver later) | ❌ missing |  |

### Odoo · Accounting & Invoicing

| Odoo capability | kashi | Notes |
|---|---|---|
| Full double-entry accounting, journal entries, period closing, lock dates | ❌ missing | M5 |
| Multi-company | ❌ missing | M9 |
| AI invoice digitization (OCR vendor bills & receipts) | ❌ missing | we already photograph expenses; add AI extraction |
| AI assistant | ❌ missing |  |
| Invoice templates, automated draft invoices | 🟡 partial | receipt designer only |
| Credit notes & refunds | 🟡 partial | returns; no credit note doc |
| Sales credit limits | ❌ missing | M1 |
| Multi-currency with daily automated exchange-rate updates | 🟡 partial | one current rate, typed by hand |
| Online payments, customer portal, QR mobile payments, direct debit | ❌ missing |  |
| Automated payment follow-ups (dunning levels) | ❌ missing | M1, WhatsApp |
| Batch payments, automated wire transfers, check printing | ❌ missing |  |
| Bank synchronization, statement import, smart reconciliation, partial/outstanding payments | ❌ missing | M6 |
| Cash register management | 🟢 kashi ahead | shifts per payment method |
| Dynamic financial statements, tax engine, tax reports, tax audit report, cash-basis tax | ❌ missing | M5/M6 |
| Analytic accounting: multi-dimensional plans, distribution, budgets | ❌ missing |  |
| Assets, deferred revenue & expense, period cut-off | 🟡 partial | asset register only |
| Expenses app (employee expenses, receipts) | 🟢 kashi ahead | phone-photo expenses, recurring, from-central |
| Web services API, data import/export | 🟡 partial | internal REST + CSV imports; no public API |
| Documents, Sign (e-signature), Spreadsheets (BI) | 🟡 partial | photos + XLSX exports; no e-sign, no spreadsheet BI |

### Odoo · Inventory & Purchase

| Odoo capability | kashi | Notes |
|---|---|---|
| Locations, putaway rules, smart removal, packages | ❌ missing |  |
| Receipts / pick / pack / ship operations, role-based to-do list | ❌ missing |  |
| Scrap, returns, inventory adjustments, stock allocation/reservation | 🟡 partial | adjustments + returns |
| Routes: drop-ship, cross-dock, push/pull, subcontracting, multi-warehouse | 🟡 partial | multi-warehouse + transfers only |
| Barcode app (scan to receive, pick, count) | ❌ missing |  |
| Replenishment: forecast, reordering rules (min/max), lead times, make-to-order | 🟡 partial | low stock + stock health |
| Lots / serials / expiry dates | 🟡 partial | FIFO batches |
| Variants, custom fields, vendor pricelists, shipping labels | 🟡 partial | variants & attributes strong; no shipping labels |
| Costing FIFO / AVCO / standard, perpetual valuation, landed costs | 🟡 partial | moving average + FIFO batches; no landed cost, no ledger |
| Stock aging report | 🟡 partial | slow movers |
| Quality checks, PLM, Manufacturing, Maintenance | ⚪ skip |  |
| RFQs, purchase tenders, blanket orders, vendor bill control (3-way match) | ❌ missing | M2 |
| Vendor quality rating, purchase dashboards & forecasts | ❌ missing |  |
| Inter-company rules, automatic inter-company reconciliation | ❌ missing |  |

### Odoo · Sales & CRM

| Odoo capability | kashi | Notes |
|---|---|---|
| Quotation builder & templates, pro forma, upselling/optional products | ❌ missing | M4 |
| Electronic signature & online payment of quotes | ❌ missing |  |
| Sales orders → invoicing, order-specific routes, contracts | ❌ missing | M4 |
| Sales warnings on customers/products | ❌ missing |  |
| Variant grid entry | 🟡 partial | variant pickers on the till |
| Coupons, promotions, loyalty, gift cards & eWallet | 🟢 kashi ahead | except eWallet/gift card balance |
| Pricelists with rules, discounts, product configurator | 🟡 partial | price lists per branch; no client price lists |
| Commissions | 🟡 partial | salesperson performance, no commission rules |
| CRM pipeline, activities, AI lead scoring | ❌ missing | M11 |
| Subscriptions, rental | ⚪ skip |  |

### Odoo · Website, Marketing & Communication

| Odoo capability | kashi | Notes |
|---|---|---|
| Website builder, eCommerce, blog, forum, eLearning | ❌ missing | M12 (connectors first) |
| Live chat | 🟡 partial | WhatsApp chats in kashi |
| WhatsApp app (templates, messages from records) | 🟢 kashi ahead | deeper: rules, polls, thank-you/welcome, cost tracking, review requests |
| Email marketing, SMS marketing, social marketing | ❌ missing |  |
| Marketing automation (journeys) | 🟡 partial | WhatsApp rules |
| Events, surveys | 🟡 partial | WhatsApp polls + Google review flow |

### Odoo · HR & Services

| Odoo capability | kashi | Notes |
|---|---|---|
| Employees | ✅ have |  |
| Attendances (kiosk / badge) | 🟢 kashi ahead | door-device punches, approvals, complaints |
| Time off with accrual | ❌ missing | M8 |
| Payroll (country localizations) | ❌ missing | M8 — and Odoo has no Lebanese payroll |
| Planning (shift scheduling) | ❌ missing | very useful for shops |
| Recruitment, appraisals, referrals, fleet | ⚪ skip |  |
| Project, timesheets, field service, helpdesk, appointments | ⚪ skip |  |

### Odoo · Productivity & Platform

| Odoo capability | kashi | Notes |
|---|---|---|
| Discuss (internal chat), VoIP | ⚪ skip | staff use WhatsApp |
| Approvals app | 🟡 partial | transfers, attendance, manager password |
| Internet of Things (scales, terminals, printers via IoT box) | 🟡 partial | printers + display |
| Knowledge base | 🟡 partial | user guide page |
| AI: tools and agents in every app | ❌ missing | big opportunity for us (Claude) |
| Studio: no-code custom fields, views, reports, automations | ❌ missing |  |
| ESG reporting | ⚪ skip |  |
| Arabic UI with RTL | 🟡 partial | Arabic in reports only |
| Third-party app store (thousands of modules) | ⚪ skip | can't match; don't try |

### Odoo-driven milestones (O-series)

These come on top of M0–M13; where an M-milestone already covers an Odoo feature it is not repeated.

| # | Milestone | Priority | Size | Goal |
|---|---|---|---|---|
| O1 | [AI layer (our answer to Odoo AI)](#o1) | P1 | M | Make AI the reason to pick kashi over both. |
| O2 | [Customer portal, online payment & e-signature](#o2) | P1 | M | Match Odoo's portal, built WhatsApp-first. |
| O3 | [POS hardware & retail parity](#o3) | P1 | M | Close the till gaps Odoo POS covers. |
| O4 | [Warehouse operations](#o4) | P2 | L | Odoo-level stock operations for the wholesale warehouse. |
| O5 | [Marketing suite on our WhatsApp lead](#o5) | P2 | M | Beat Odoo's Email/SMS/Social/Automation apps where Lebanese customers actually are. |
| O6 | [Staff planning (shifts) + time off](#o6) | P2 | M | Odoo Planning + Time Off, wired to our attendance. |
| O7 | [Custom fields, reports & dashboards (Studio-lite)](#o7) | P2 | M | Let customers adapt kashi without us writing code each time. |

<a id="o1"></a>
#### O1 · AI layer (our answer to Odoo AI)

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Make AI the reason to pick kashi over both.

**Why:** Odoo now sells 'AI tools and agents' and AI bill scanning as paid credits; Wizard has nothing. We can use Claude directly.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Scan supplier bills & expense receipts** | P1 | S | Photo (we already collect them) → supplier, date, lines, totals, currency; user confirms; creates the purchase/expense. |
| **Ask kashi** | P1 | M | Chat box (web + WhatsApp for the owner): 'how much did Hara sell this week vs last', 'who owes more than $500', 'what should I reorder'. Read-only tools over our API. |
| **Smart product entry** | P2 | S | From a photo or name, propose category, attributes, Arabic name, price band (extends autofill rules). |
| **WhatsApp AI replies** | P2 | M | Answer customer questions about stock, prices, opening hours, order status; hand off to staff. |
| **Anomaly alerts** | P2 | S | Unusual discounts, voids, short shifts, price edits, stock drops pushed to the owner. |

**Done when:** An expense photo becomes a posted expense with one tap, and the owner gets correct answers from 'Ask kashi'.


<a id="o2"></a>
#### O2 · Customer portal, online payment & e-signature

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Match Odoo's portal, built WhatsApp-first.

**Why:** Odoo's portal lets clients see/pay invoices and sign quotes; wholesale clients in Lebanon would rather get a WhatsApp link.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Client portal via WhatsApp link (OTP)** | P1 | M | Invoices, statement & balance (M1), quotations (M4), loyalty points, receipts; Arabic/English. |
| **Accept & sign quotations** | P2 | S | Signature pad, accepted quote → sales order. |
| **Online payment / payment links** | P2 | M | Whish payment links where available, card gateway (e.g. Areeba/local acquirer) later; record against the client ledger. |

**Done when:** A wholesale client opens a link, sees what they owe, signs a quote and pays.


<a id="o3"></a>
#### O3 · POS hardware & retail parity

**Priority:** P1 · **Size:** M (1–3 weeks)

**Goal:** Close the till gaps Odoo POS covers.

**Why:** These are small individually but show up in every POS comparison.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Cash drawer kick** | P0 | S | ESC/POS pulse on cash payments and on manager 'open drawer' (logged). |
| **Scales & weighed barcodes** | P1 | M | Serial scale reading at checkout; EAN-13 price/weight embedded barcodes (prefix 2x). |
| **Gift cards & eWallet / store credit** | P1 | S | Sell gift cards with balance; refunds and exchanges can go to store credit; usable in any branch. |
| **Units of measure at the till** | P1 | S | Sell by KG/metre; depends on M0. |
| **Card terminal integration** | P2 | M | Pass amount to the bank terminal and read the result (depends on local acquirer APIs). |
| **Android / tablet till** | P2 | L | Same as M9; Odoo and Wizard both run on tablets. |
| **Ship later / customer orders from the till** | P2 | S | Take deposit, reserve stock, deliver later. |

<a id="o4"></a>
#### O4 · Warehouse operations

**Priority:** P2 · **Size:** L (3–6 weeks)

**Goal:** Odoo-level stock operations for the wholesale warehouse.

**Why:** Odoo Inventory is its strongest app; Wizard sells bins/zones too. We need a lean version, not all of Odoo's routes.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Locations & bins, putaway rules** | P2 | M | Optional per inventory; receiving ends at a bin. |
| **Pick / pack / ship with phone barcode scanning** | P2 | M | From sales orders (M4) and transfers. |
| **Reservation & availability** | P2 | S | Reserve on order; available-to-promise. |
| **Lots/expiry & quarantine** | P3 | S | For non-clothing customers. |
| **Drop-ship & cross-dock** | P3 | S | Only if a customer asks. |

<a id="o5"></a>
#### O5 · Marketing suite on our WhatsApp lead

**Priority:** P2 · **Size:** M (1–3 weeks)

**Goal:** Beat Odoo's Email/SMS/Social/Automation apps where Lebanese customers actually are.

**Why:** Odoo splits marketing into 5 paid apps; we already run WhatsApp at scale.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Segments & campaigns** | P2 | S | Segments by spend, recency, branch, category bought; WhatsApp + email campaigns; opt-out; cost estimate (we already export VCF lists by hand). |
| **Journeys / automation** | P2 | M | Welcome → first purchase → 30-day no visit → birthday → points expiring. |
| **Surveys & reviews** | P2 | S | Extend WhatsApp polls + Google reviews into NPS per branch/salesperson. |
| **Email & SMS fallback** | P3 | S | For customers without WhatsApp. |

<a id="o6"></a>
#### O6 · Staff planning (shifts) + time off

**Priority:** P2 · **Size:** M (1–3 weeks)

**Goal:** Odoo Planning + Time Off, wired to our attendance.

**Why:** We record when staff arrive; without a planned rota we can't say who is late or short-staffed. Odoo has Planning; Wizard has staff scheduling on some industry pages.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Weekly rota per branch** | P2 | S | Drag shifts; publish to staff by WhatsApp. |
| **Late/absent vs plan** | P2 | S | Computed from door punches vs rota; feeds payroll deductions (M8) instead of typed late units. |
| **Time off requests & balances** | P2 | S | Request from the till or WhatsApp; approve; accrue. |
| **Staffing vs footfall** | P3 | S | Use busy hours/visitor counts to suggest staffing — no one else can do this. |

<a id="o7"></a>
#### O7 · Custom fields, reports & dashboards (Studio-lite)

**Priority:** P2 · **Size:** M (1–3 weeks)

**Goal:** Let customers adapt kashi without us writing code each time.

**Why:** Odoo Studio and Spreadsheet BI are why companies pick Odoo; Wizard does one-off custom forms by hand.

| Feature | Priority | Size | What to build |
|---|---|---|---|
| **Custom fields on products/clients/documents** | P2 | M | Typed fields, shown in forms, grids, filters, exports, print templates. |
| **Report builder** | P2 | M | Pick a dataset, group/pivot, chart, save, schedule by email/WhatsApp. |
| **Automations** | P3 | M | When X happens → do Y (notify, set field, send WhatsApp). |
| **Public API & webhooks** | P2 | M | Token-based API for partners and integrations. |

## Part 3 — What matters most in the Lebanese market

| Need | Importance | Wizard | Odoo | kashi today | What to build |
|---|---|---|---|---|---|
| USD + LBP in the same sale, price in USD, take cash in both, give change in LBP | **Critical** | Strong in ledger; POS app not reviewed | Accounting multi-currency yes; POS only rounding | ahead at the till; no dated rates/ledger | Dated rate table, dual-currency statements & ledger (M0, M1, M5) |
| Cash-heavy economy, wallets & money-transfer apps (Whish, OMT, BOB Finance), few card terminals | **Critical** | Cash & cheques focus | Card/online gateways focus | have: cash per currency, Whish method, shift control | OMT/BOB methods, wallet reconciliation reports, payment links (O2) |
| Unreliable power & internet | **Critical** | Web app (POS app not reviewed) | POS offline mode; back office online only | ahead: offline-first till + outbox sync | Offline mobile app (M10), keep sync bullet-proof |
| Selling on credit (dayn), collectors, statements, reminders | **Critical** | Strong: credit limits, statements, aging | Credit limits, follow-ups by email | partial: till-only On Account | M1 + WhatsApp statements/reminders |
| WhatsApp is the main customer & staff channel | **High** | SMS/notifications | WhatsApp app (templates) | ahead | Put every document and approval on WhatsApp (O1, O2, O5) |
| VAT (11%), MOF numbers, VAT-inclusive prices, quarterly VAT returns; check stamp-duty rules with the accountant | **High** | Claims Lebanese VAT compliance | No official Lebanese localization (community l10n_lb for v13/14 only) | missing: one tax rate per till | M6 VAT codes + return report; verify stamp duty |
| Payroll: NSSF, income tax, end-of-service indemnity, transport allowance | **High** | Has NSSF rules, end-of-service | No Lebanese payroll | partial: attendance only | M8 with Lebanese rules — a clear edge over Odoo |
| Imports through Beirut port: freight, customs, clearing agents, landed cost | **High** | Landed cost, shipments, LC | Landed costs | missing | M2 landed cost; bale opening (M3) |
| Frequent repricing (inflation history), USD price lists, bulk price updates | **High** | Generate price lists, price history | Pricelists with rules | partial: price lists per branch | Bulk reprice by %/category with history (M2/M3) |
| Arabic, English & French; Arabic receipts and documents | **High** | EN/FR UI | Full Arabic RTL | partial: Arabic in reports | Arabic UI + documents (M9) |
| Cheques & post-dated cheques in B2B (less since 2019, still common with suppliers) | **Medium** | Strong | Check printing, batch payments | missing | M6 cheque register |
| Delivery companies & cash-on-delivery for online sales | **Medium** | Generic shipping | Shipping connectors (mostly international carriers) | missing | COD reconciliation with local couriers (M12) |
| Instagram/WhatsApp commerce rather than full web stores | **Medium** | Shopify/WooCommerce/Magento sync | Own website + eCommerce | partial: WhatsApp | WhatsApp catalog + order links before a full store (M12) |
| Price-sensitive SMBs, per-user pricing hurts, want local support in Arabic | **High** | Price on request, local team | Per-user pricing + partner implementation fees | own team, fast changes | Per-branch flat pricing published openly (M9) |
| Secondhand clothing / bale trade (bala), graded stock sold by piece or kilo | **High (our niche)** | Generic unpacking/disassembly | Generic manufacturing/kits | missing | Bale opening with grade yields & landed cost (M3) |

## How we become their nightmare

- **The Lebanon-native ERP**: USD+LBP at the till and in the books, WhatsApp everywhere, NSSF payroll, offline-first — the gap between Wizard's old stack and Odoo's foreign one.
- **AI that Odoo charges credits for**: bill scanning, 'Ask kashi', anomaly alerts (O1).
- **'Switch from Wizard in a weekend'**: an importer that reads their exact data model (we've seen it: g_aux clients/suppliers, in_* documents, family levels, price lists, COA).
- **WhatsApp everything**: statements, receipts, payslips, approvals, reminders, quotations sent as PDFs, customer portal by WhatsApp login. Wizard only does email/SMS.
- **Offline everywhere**: till and mobile both truly offline-first with our proven outbox sync.
- **Bale-business features**: bale opening with grade yields and landed cost per graded piece. No generic ERP has this.
- **Transparent pricing**: Wizard hides prices ('contact us'). Publish ours.
- **Live owner app**: today's sales, cash per branch, overdue clients and approvals on the phone, live over WebSocket.
- **Security as a feature**: we never ship passwords to the browser; audit everything; 2FA.
- **Speed of change**: weekly signed releases to every till vs their 'Upgrade statements are available' screens that break the app.

## Risks & rules for ourselves

- **The ledger is the hardest part.** Get posting rules reviewed by our accountant before coding; build M1/M2 sub-ledgers so they post cleanly later.
- **Two sources of sales.** kashi has central sales_invoices and till-synced branch_invoices. The document engine (M0) should become the single model; migrate carefully (migrations must stay additive for till rollback).
- **Integer quantities on tills.** Decimal/KG support must not break existing till sync; add new columns rather than change types.
- **Scope creep.** Wizard has 268 document types because it says yes to every customer. Say no to verticals (production, projects, rent) unless paid for.
- **Parallel run discipline.** Don't switch R1 off Wizard until a full month of balances match.
