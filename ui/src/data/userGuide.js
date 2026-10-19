// The user guide (pages/user-guide.vue): one entry per page of the app, grouped
// like the nav drawer. Keep entries short and in plain words: what the page is
// for, then what you can do on it. Screenshots will come later -- a `figure`
// block can be added to an entry's blocks when they do.
//
// Text is rendered with v-html so <b> and <code> work; never put user data here.

// An entry: `what` is one or two sentences, `can` a short list, `tip` optional.
function entry (id, group, title, what, can = [], tip = '') {
  const blocks = [{ kind: 'p', text: what }]
  if (can.length > 0) {blocks.push({ kind: 'list', items: can })}
  if (tip) {blocks.push({ kind: 'tip', text: tip })}
  return { id, group, title, blocks }
}

export const guideSections = [
  // ---------------------------------------------------------------- Basics
  entry('how-kashi-works', 'Basics', 'How Kashi works',
    'Kashi Cloud is the office side of your shops. Each branch has one or more <b>tills</b> (registers). Sales, shifts, expenses and new clients flow from the tills to the Cloud on their own, and prices, products and settings flow back.',
    [
      'Pages update live when a branch sends something new.',
      'Your <b>role</b> decides which pages you see. Some roles can look but not change anything.',
      'Most lists have a search box and a branch filter at the top.',
    ],
    'If a number looks wrong, check the branch filter and the date range first.'),

  entry('home', 'Basics', 'Home',
    'A quick look at what is happening right now.',
    [
      '<b>Recent activity</b> lists what the branches just did, newest first. Click a row to see the details.',
      'Items marked <b>pinned</b> stay at the top until someone marks them resolved.',
      '<b>Latest attachments</b> shows the newest photos, like receipts.',
    ]),

  // ----------------------------------------------------------------- Sales
  entry('daily-income', 'Sales', 'Daily Income',
    'A grid of every day\'s income, branch by branch.',
    [
      'Pick one or more branches with the chips at the top, or <b>All branches</b>.',
      'Click a day to open it: totals compared to last week, payments, best sellers, salespeople, hour-by-hour chart, invoices, closed shifts and expenses.',
      '<b>Email</b> sends that day\'s report. The arrows move to the previous or next day.',
    ]),

  entry('invoices', 'Sales', 'Invoices',
    'All sales, in two tabs: <b>Retail</b> and <b>Wholesale</b>. Sales made at the tills show here with their branch.',
    [
      'Search, and filter by branch.',
      '<b>Add Invoice</b> makes a sale from the office: choose the cashbox, shift, inventory, client and the items.',
      'Open an invoice to see its items and how it was paid.',
      'You can <b>cancel</b> a till invoice, or start a <b>return</b> or <b>exchange</b>. Returns and exchanges are sent to the till and show up once the branch processes them.',
    ]),

  entry('shifts', 'Sales', 'Shifts',
    'Every cashier shift, open or closed.',
    [
      'Filter by status, cashbox or branch.',
      '<b>Open Shift</b> and <b>Close</b> work for office shifts. Branch shifts are opened and closed on the till.',
      'A closed branch shift can show a photo of its close-shift receipt, and you can ask the till to reprint it.',
      'The <b>Visitors</b> tab shows how many people the door counters counted each day.',
    ]),

  entry('whatsapp', 'Sales', 'WhatsApp & Reviews',
    'Kashi can send a thank-you message on WhatsApp after a sale. This page shows what was sent and how it went.',
    [
      '<b>Dashboard</b> and <b>Messages</b>: what went out, to whom, and what it cost.',
      '<b>Replies</b>: what customers answered.',
      '<b>Rules</b>: decide who gets which message.',
      '<b>Health</b>: warns you when a sending number is at risk or paused.',
    ],
    'The account details and the default message are in <b>Settings &gt; WhatsApp</b>.'),

  entry('emails', 'Sales', 'Emails',
    'Invoice reports sent by email.',
    [
      '<b>Send report</b>: pick a day, a date range or a shift, choose CSV or Excel, add the people to send to.',
      '<b>Schedules</b>: reports sent by themselves at a set time (Beirut time).',
      '<b>History</b>: everything sent, and whether it worked.',
    ],
    'Email needs its key set once in <b>Settings &gt; Email</b>.'),

  // -------------------------------------------------------------- Analysis
  entry('dashboard', 'Analysis', 'Dashboard',
    'The big picture for <b>this month</b>, <b>last month</b> or <b>this year</b>, in tabs.',
    [
      '<b>Overview</b>: revenue, expenses, profit, profit margin and financial position.',
      '<b>Sales</b>: items sold, new clients, invoices, due loans, purchases, targets, average invoice.',
      '<b>Financials</b>, <b>Demand</b> (slow movers and reorder suggestions), <b>Alerts</b> and <b>Health</b> (stock turnover).',
    ],
    'Profit only counts items that have a cost. Items without a cost make profit look higher.'),

  entry('targets', 'Analysis', 'Targets',
    'A scoreboard of the sales targets set for each branch.',
    [
      'Each target shows its goal, what was achieved and a status: <b>met</b>, <b>missed</b> or still running.',
      'Filter by branch or month, and see earlier months.',
    ],
    'Targets are created on the <b>Branches</b> page.'),

  entry('footfall', 'Analysis', 'Footfall',
    'How many customers came in, counted by the door counters.',
    [
      'Daily chart, average per weekday, month over month, and a comparison between branches.',
      'Changes when you pick a different range of months or branches.',
    ]),

  entry('busy-hours', 'Analysis', 'Busy Hours',
    'When customers come in, and when they buy. Everything is an average per day.',
    [
      'Charts by hour and by weekday, plus heatmaps of weekday against hour.',
      'Shows the hour where most visitors end up buying.',
    ],
    'If there are more sales than customers counted, the counter was missed at that time.'),

  entry('product-revenues', 'Analysis', 'Product Revenues',
    'How much each product earned.',
    [
      'Filter by category, sub-category, brand or branch.',
      'Switch to <b>Wholesale</b> to see wholesale sales.',
      'See revenue split by branch.',
    ]),

  entry('conversion', 'Analysis', 'Conversion',
    'The share of visitors who bought something: invoices divided by visitors.',
    [
      'By month, by weekday and by branch.',
      'A month-by-branch table shows where it is strongest and weakest.',
    ]),

  entry('sales-analytics', 'Analysis', 'Sales Analytics',
    'Sales trends and demand: which products are slow and what to reorder.'),

  entry('sales-analysis', 'Analysis', 'Sales Analysis',
    'January up to the month you choose, compared with your targets and the same months last year.',
    [
      'Sales by category and by branch, actual sales against target, payment methods and growth.',
      'Gauges for <b>inventory days</b> and <b>receivable days</b> (how long stock and unpaid money take to turn over).',
    ]),

  entry('financials', 'Analysis', 'Financials',
    'Sales with a forecast, sales by branch, payment methods, gross margin, and when sales happen during the week.'),

  entry('sales-dashboard', 'Analysis', 'Sales Dashboard',
    'Every sale, from the tills and from the office, for a year or a single month.',
    [
      'Monthly sales, profit and profit percent.',
      'Top product, top category, revenue by product, by branch and by payment method, and daily sales.',
    ]),

  // ------------------------------------------------------------- Customers
  entry('clients', 'Customers', 'Clients',
    'Your customers, in two tabs: <b>Retail</b> and <b>Wholesale</b>.',
    [
      '<b>Top Spenders</b> shows your best clients as a pyramid. Sort by most spent, most invoices or most items. Click anyone to open their details.',
      '<b>All Clients</b> lists the rest. Search by name or phone.',
      'The branch filter lets you pick <b>several branches</b> at once.',
      '<b>Add Client</b>, edit, or delete. A client who already has invoices or points is <b>deactivated</b> instead, and you can reactivate them later.',
      'The cog changes the spend <b>tiers</b>.',
    ]),

  entry('clients-export', 'Customers', 'Exporting clients',
    'On the Clients page, <b>Export CSV</b> downloads the clients of the tab you are on.',
    [
      'Tick the <b>fields</b> you want as columns.',
      'A client belongs to a branch if they <b>bought something there</b>. Each branch you pick gets its own file. Several branches come as one zip.',
      'No branch picked: one file with every client.',
      '<b>Came from / until</b> keeps only people who bought in that period. Total spent and Invoices then count only that period.',
      '<b>No duplicates</b>: anyone with the same phone in more than one file is removed from all of them. Inside one file, only one row per phone is kept.',
      '<b>Phone check (Lebanese)</b> adds a column saying Valid or Invalid. Mobiles start with 03, 70, 71, 76, 78, 79 or 81. Landlines start with 01 or 04 to 09. +961 and spaces are fine.',
      '<b>Include cancelled invoices and hidden branches</b> counts invoices that are normally skipped.',
    ],
    'A branch can look short if its old sales were imported without client names, or if cashiers sold without choosing a client.'),

  entry('loyalty', 'Customers', 'Loyalty',
    'Clients are grouped into <b>tiers</b> by how much they have spent in total. This page shows how that looks.',
    [
      'Members and revenue per tier, new members each month, and points not yet used.',
      'A table of every tier, and the clients who are close to the next tier.',
      'Choose a period at the top to see activity for that time.',
    ],
    'Tier names and limits are edited from the cog on the Clients page.'),

  // ---------------------------------------------------------------- People
  entry('employees', 'People', 'Employees',
    'Your staff and their pay.',
    [
      '<b>Add employee</b>: name, role, branch, status, hire date and base salary.',
      '<b>Edit salaries</b> lets you change many rows, then <b>Save all</b>.',
      'The table shows deductions and the net pay.',
    ]),

  entry('attendance', 'People', 'Attendance',
    'When staff clocked in and out.',
    [
      '<b>Sync from POS</b> pulls punches from the tills. <b>Import CSV</b> is for a branch with no clock-in device.',
      'Four views: <b>Timeline</b>, <b>In / out pairs</b>, <b>Monthly grid</b> and <b>Punches</b>.',
      'Edit or delete a wrong punch.',
      '<b>Requests</b>: changes asked for on a till. Approve or reject them.',
    ]),

  entry('salesperson-performance', 'People', 'Salesperson Performance',
    'A ranking of salespeople with a score out of 100.',
    [
      'Filter by period or branch.',
      'Open a person to see their details, including manual discounts and price changes they made.',
      'The <b>Formula</b> button explains how the score is worked out.',
    ]),

  // ----------------------------------------------------------------- Stock
  entry('alerts', 'Stock', 'Alerts',
    'Products that need attention.',
    [
      '<b>Out of stock</b> and <b>Low stock</b>.',
      'Set a default low-stock limit, or a different limit for one product.',
      '<b>Mute alerts</b> for a product you do not care about. It moves to <b>Ignored</b>, and you can resume it.',
    ]),

  entry('stock-health', 'Stock', 'Stock Health',
    'A quick check of how your stock is doing.',
    [
      'Days of stock left, and the value of your stock at cost.',
      '<b>Dead stock</b> (not selling), <b>never sold</b> and <b>overstocked</b> products.',
    ]),

  entry('inventory', 'Stock', 'Inventory',
    'The places that hold stock, like shops and warehouses.',
    [
      '<b>Add Inventory</b> with a name, type and code.',
      'Open one to see each product\'s quantity on hand and average cost.',
      '<b>Adjust</b> sets a counted quantity. <b>Start Count</b> opens a full stock count.',
    ]),

  entry('inventory-analysis', 'Stock', 'Inventory Analysis',
    'Stock numbers in tabs.',
    [
      '<b>Overview</b> and <b>By Branch / Location</b>: value and days of stock.',
      '<b>Low Stock</b> and <b>Slow Movers</b>.',
      '<b>Movements</b>: what went in and out, filtered by reason and date.',
    ]),

  entry('purchase-invoices', 'Stock', 'Purchase Invoices',
    'Stock you buy from suppliers.',
    [
      '<b>Add Purchase</b>: choose the supplier and the inventory it goes to, then add items with quantity and unit cost.',
      '<b>Receive</b> adds the stock to that inventory.',
      '<b>Cancel Purchase</b> if it was a mistake.',
    ]),

  entry('suppliers', 'Stock', 'Suppliers',
    'The companies you buy from: name, phone, country and address.'),

  entry('batches', 'Stock', 'Batches',
    'Stock bought together, so you can see what each purchase earned.',
    [
      'Add a batch from a purchase invoice.',
      'Pick a batch to see what was bought, sold, remaining, total cost, revenue and profit.',
      '<b>Insights</b> lists items that were bought and never sold.',
    ]),

  entry('transfers', 'Stock', 'Transfers',
    'Moving stock from one inventory to another.',
    [
      'Steps: <b>Draft</b>, <b>Requested</b>, <b>Dispatched</b>, <b>Received</b>.',
      '<b>Approve</b> a request from a branch, <b>Dispatch</b> when it leaves, <b>Receive</b> when it arrives.',
      'Cancel or reject a transfer that should not happen.',
    ]),

  entry('stock-counts', 'Stock', 'Stock Counts',
    'Check the real stock against what the system says.',
    [
      '<b>New Count</b>: choose the inventory. The sheet starts with the system quantities.',
      'Change the counted quantity of each item.',
      '<b>Save Draft</b> to continue later, or <b>Post</b> to apply it.',
    ]),

  entry('stock-movements', 'Stock', 'Stock Movements',
    'A read-only history of every stock change: the reason, the document, the quantity before and after, and who did it.'),

  entry('barcodes', 'Stock', 'Barcodes',
    'Print barcode labels.',
    [
      'Search products, set how many labels each, then <b>Generate barcodes</b> to get a PDF.',
      'Label size is set in the dialog (72 points = 1 inch).',
    ]),

  entry('assets', 'Stock', 'Assets',
    'Things the business owns, like vehicles, equipment and furniture. Each has a name, code, category and purchase date.'),

  entry('treemaps', 'Stock', 'Assets & Warehouses Treemaps',
    'Two picture views. Bigger boxes mean more value. One shows your assets, the other shows what is in your warehouses.'),

  // ---------------------------------------------------------------- Branches
  entry('branches', 'Branches & tills', 'Branches',
    'Your shops and their tills.',
    [
      '<b>Add Branch</b> with a name and a short code.',
      '<b>Add Register</b> to give a branch another till (for example CT2).',
      '<b>Rotate Key</b> makes a new key to connect a till. Online tills are marked.',
      'The <b>In reports</b> switch keeps a branch syncing but removes it from every report.',
      'Add sales <b>targets</b>, one-off or repeating every few months.',
    ]),

  entry('branch-settings', 'Branches & tills', 'Branch Settings',
    'Change how the till of one branch behaves, from the office.',
    [
      'Sections: General, Behaviour, Cash &amp; shifts, Page access, Printing, Attendance, Users, Payments, Quick access and Till display.',
      '<b>Save</b> sends only what you changed.',
      '<b>Sync now</b> asks the till to send its latest data.',
    ]),

  entry('default-branch-settings', 'Branches & tills', 'Default Settings',
    'A template of settings you can copy to branches.',
    [
      'Fields left on <b>Leave as is</b> are not touched.',
      '<b>Apply to all branches</b> pushes the template out.',
    ]),

  entry('commands', 'Branches & tills', 'Commands',
    'Send an instruction to a till and follow it.',
    [
      'Examples: <b>Sync now</b>, <b>Print close shift</b>, <b>Reprint invoice</b>, <b>Return invoice</b>, <b>Swap shift balance</b>, <b>Set PIN</b>, <b>Update settings</b>.',
      'Each command is <b>Queued</b>, <b>Done</b>, <b>Failed</b> or <b>Cancelled</b>. Retry a failed one, or cancel a queued one.',
      'A command runs when the till is online.',
    ]),

  entry('conflicts', 'Branches & tills', 'Conflicts',
    'Things from the branches that need a decision.',
    [
      '<b>Oversold stock</b>: a till sold more than the system had. Fix it with an adjustment, transfer or count.',
      '<b>Client conflicts</b>: a name or phone that does not match. Keep the Cloud\'s version, use the branch\'s, merge into an existing client, or create a new one with a phone number.',
    ]),

  entry('app-updates', 'Branches & tills', 'App Updates',
    'Choose which version of the till app each till runs.',
    [
      'Tills download updates by themselves and install when it is safe, or when the cashier taps <b>Install now</b>.',
      '<b>Set version</b> or <b>Deploy</b> a release to chosen tills. <b>Roll back</b> goes to the previous release.',
      'The status shows which tills are up to date or still updating.',
    ]),

  // ---------------------------------------------------- Pricing & products
  entry('price-lists', 'Pricing & products', 'Price Lists & Discount Lists',
    'A list gives certain products a different price or discount.',
    [
      '<b>Add</b> a list, then add products with their price or discount.',
      'Set <b>Valid from / to</b>, and switch the list <b>Active</b> or not.',
      'See which <b>branches</b> use a list. One list can be the <b>Default</b>.',
    ]),

  entry('coupons', 'Pricing & products', 'Coupons',
    'Codes that give a customer a value. Three tabs: <b>Coupons</b>, <b>Types</b>, <b>Lists</b>.',
    [
      '<b>Types</b> define the value and currency, and whether it applies automatically or needs the code typed.',
      '<b>Add Coupon</b>: a code, a type, optionally a client and an end date.',
      '<b>Generate List</b> makes many codes at once. <b>Deactivate</b> stops a coupon.',
    ]),

  entry('products', 'Pricing & products', 'Products',
    'Your product catalog.',
    [
      'Search and filter by status, attributes, dates or whether it has variants.',
      '<b>Add Product</b>, or <b>Import CSV</b> with the columns code, name, price and an optional description.',
      'If any row in the CSV is wrong, nothing is imported. Codes that already exist are skipped.',
    ]),

  entry('colors-sizes', 'Pricing & products', 'Colors & Sizes',
    'The colors and sizes products can come in. Add a color with a name and a hex code, or a size with a type, a name and an order.'),

  entry('attributes', 'Pricing & products', 'Attributes',
    'Extra product details, like brand or material. Pick an attribute on the left, then add or remove its values.'),

  entry('autofill-rules', 'Pricing & products', 'Auto-fill Rules',
    'Rules that fill product details for you from the code or name.',
    [
      '<b>Add rule</b>, then <b>Save</b>. The whole list is saved together.',
      'Use <b>Try it</b> to type a code or name and see what would be filled.',
      'The rules only work if smart auto-fill is on in <b>Settings &gt; Products</b>.',
    ]),

  // ------------------------------------------------------------- Accounting
  entry('expenses', 'Accounting', 'Expenses',
    'Money spent. Three tabs: <b>Central</b>, <b>Branch</b> and <b>Recurring</b>.',
    [
      'Add an expense with a description, category and amount.',
      'Branch expenses are the ones tills can see. Central ones are only in the office.',
      '<b>Recurring</b> repeats every few days, weeks or months, starting at the first due date.',
      'Pick a category on the left to narrow the list.',
    ]),

  entry('loans', 'Accounting', 'Loans',
    'Money borrowed, in <b>Central</b> and <b>Branch</b> tabs.',
    [
      '<b>Add Loan</b> with a lender, amount and category.',
      'Open a loan to add payments. It shows what is still <b>remaining</b>.',
      'Filter by status or branch.',
    ]),

  entry('currencies', 'Accounting', 'Currencies & Payments',
    'The money you accept and how customers pay.',
    [
      '<b>Currencies</b>: the rate (how many units per 1 USD) and the cash rounding step, for example to the nearest 5,000 LL.',
      '<b>Payment Methods</b>: cash, card, transfer and so on, with an order and a color.',
      'Methods marked <b>System</b> cannot be edited.',
    ]),

  // ----------------------------------------------------------------- System
  entry('calendar', 'System', 'Calendar',
    'A month calendar of events.',
    [
      '<b>Add event</b> with dates, a color and an icon.',
      'Built-in events (like holidays) can be turned off.',
      'Upcoming events also show in the top bar.',
    ]),

  entry('storage', 'System', 'Storage',
    'All uploaded photos in one place, like receipts and transfer attachments. Pick a folder, search, and <b>Load more</b>.'),

  entry('security', 'System', 'Security',
    'Who reaches the system, and from where.',
    [
      'See visitors by country, and alerts for odd activity. <b>Mark seen</b> or <b>Resolve</b> an alert.',
      'Allow only some countries, or always allow certain IP addresses.',
      'Block an IP automatically after repeated failed sign-ins, or <b>Ban</b> one by hand.',
    ]),

  entry('audit-log', 'System', 'Audit Log',
    'A record of changes: what was changed, what it was before and after, who did it and when.'),

  entry('dev-logs', 'System', 'Dev Logs',
    'Technical messages for whoever maintains the system. You normally do not need this page.'),

  entry('settings', 'System', 'Settings',
    'App-wide options, in tabs.',
    [
      '<b>General</b>, <b>Financials</b>, <b>Barcode labels</b> and <b>Products</b>.',
      '<b>Users</b> and <b>Roles</b> (admins only): add people, change their role or password, and choose what each role can see and edit.',
      '<b>WhatsApp</b>, <b>Reviews</b> and <b>Email</b>: the accounts and messages Kashi sends with.',
      'Save and Discard light up only when you change something.',
    ]),
]

// The groups in the order they first appear, for the contents list.
export const guideGroups = [...new Set(guideSections.map(s => s.group))]
