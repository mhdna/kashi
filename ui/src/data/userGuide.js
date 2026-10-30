// The user guide (pages/user-guide.vue): one entry per page of the app, grouped
// like the nav drawer. Keep entries short and in plain words: what the page is
// for, then what you can do on it. Screenshots will come later -- a `figure`
// block can be added to an entry's blocks when they do.
//
// Text is rendered with v-html so <b> and <code> work; never put user data here.

// Block builders for the longer entries.
const h = text => ({ kind: 'heading', text })
const steps = items => ({ kind: 'steps', items })
const bullets = items => ({ kind: 'list', items })

// An entry: `what` is one or two sentences, `can` a short list, `more` extra
// blocks (a heading, steps, a list) after it, and `tip` an optional note.
function entry (id, group, title, what, can = [], tip = '', more = []) {
  const blocks = [{ kind: 'p', text: what }]
  if (can.length > 0) {blocks.push({ kind: 'list', items: can })}
  blocks.push(...more)
  if (tip) {blocks.push({ kind: 'tip', text: tip })}
  return { id, group, title, blocks }
}

export const guideSections = [
  // ---------------------------------------------------------------- Basics
  entry('how-kashi-works', 'Start here', 'How Kashi works',
    'Kashi Cloud is the office side of your shops. Each branch has one or more <b>tills</b> (registers) running the POS app. The two sides keep each other up to date.',
    [
      '<b>Tills to Cloud:</b> sales, shifts, expenses, door-counter visits and new clients.',
      '<b>Cloud to tills:</b> products, prices, clients, settings and app updates.',
      'Pages refresh live when a branch sends something new.',
    ],
    'If a till is offline, nothing is lost. It sends everything when it is back online.',
    [
      h('Who sees what'),
      bullets([
        'Your <b>role</b> decides which pages you see. A page can be hidden, <b>view only</b>, or <b>editable</b>.',
        'With view only you can look and search, but there are no Add, Edit or Delete buttons.',
        'Admins also see Users and Roles in Settings.',
      ]),
      h('Tips for every page'),
      bullets([
        'Most lists have a <b>search box</b> and a <b>branch filter</b> at the top. If a number looks wrong, check these first.',
        'Money is in dollars unless a currency is shown next to it.',
        'Branches switched off under <b>In reports</b> are left out of every report.',
      ]),
    ]),

  entry('glossary', 'Start here', 'Words used in Kashi',
    'A short list of the words you will see on many pages.',
    [],
    '',
    [
      bullets([
        '<b>Branch:</b> a shop. It can have several tills.',
        '<b>Till / register:</b> one POS computer. Each has a short code like CT or CT2.',
        '<b>Cashbox:</b> the drawer and money accounts a till uses.',
        '<b>Shift:</b> the time between a cashier opening and closing the till. Closing it counts the cash.',
        '<b>Inventory:</b> a place that holds stock, such as a shop or a warehouse.',
        '<b>Product and variant:</b> a product comes in variants, one per color and size. Each variant has its own barcode and stock. Also called an <b>SKU</b>.',
        '<b>Invoice kinds:</b> a <b>sale</b>, a <b>return</b> (customer gives goods back) or an <b>exchange</b>.',
        '<b>Client tier:</b> a level (like Silver or Gold) based on how much a client has spent in total.',
        '<b>Target:</b> a sales goal for a branch and a period.',
        '<b>Command:</b> an instruction sent from the office to a till.',
        '<b>Conflict:</b> something from a branch that does not match the Cloud and needs your decision.',
      ]),
    ]),

  // ------------------------------------------------------------- Workflows
  entry('wf-new-product', 'Workflows', 'Add a product and put it on sale',
    'From a new item to something a cashier can sell.',
    [],
    'Tills pick up new products and prices by themselves. You do not need to send anything.',
    [
      steps([
        'Make sure its <b>colors</b> and <b>sizes</b> exist in <b>Colors &amp; Sizes</b>, and its brand or similar details in <b>Attributes</b>.',
        'Open <b>Products</b> and click <b>Add Product</b>.',
        'Enter the <b>code</b> and <b>name</b>. Choose the colors and sizes: you get one variant for every color and size combination.',
        'Add a <b>barcode</b> (optional) and a <b>price</b>. The price goes into the price list you choose, usually the default one.',
        'Click <b>Add Product</b>, or <b>Save &amp; Add Another</b> to keep going.',
        'Bring in stock with a <b>Purchase Invoice</b> (see the next workflow).',
        'Print labels from <b>Barcodes</b> if you need them.',
      ]),
      h('Many products at once'),
      bullets([
        'Use <b>Import CSV</b> on the Products page: columns <code>code</code>, <code>name</code>, <code>price</code> and an optional description.',
        'If one row is wrong, nothing is imported. Codes that already exist are skipped.',
      ]),
    ]),

  entry('wf-buy-stock', 'Workflows', 'Buy stock from a supplier',
    'Record what you bought and add it to your stock.',
    [],
    'Receiving also updates each item\'s average cost, which feeds profit numbers.',
    [
      steps([
        'Add the company in <b>Suppliers</b> if it is new.',
        'Open <b>Purchase Invoices</b> and click <b>Add Purchase</b>.',
        'Choose the <b>supplier</b>, the <b>inventory</b> the goods go to, and the date.',
        'Add each item: search by SKU or barcode, then enter the <b>quantity</b> and <b>unit cost</b>.',
        'Click <b>Receive</b>. The stock now shows in that inventory.',
        'Optional: in <b>Batches</b>, add a batch from this purchase to track what it earns.',
      ]),
      bullets(['A purchase stays a <b>draft</b> until you click Receive, so you can fix it before then.']),
    ]),

  entry('wf-transfer', 'Workflows', 'Move stock between places',
    'Send stock from one inventory to another, for example warehouse to branch.',
    [],
    '',
    [
      h('From the office'),
      steps([
        'Open <b>Transfers</b> and click <b>Add Transfer</b>.',
        'Choose <b>From inventory</b> and <b>To inventory</b>, and add the items and quantities.',
        'Click <b>Dispatch</b> when the goods leave. The stock is taken from the source and is "in transit".',
        'Click <b>Receive</b> when they arrive. The stock lands in the destination.',
      ]),
      h('When a branch asks for stock'),
      bullets([
        'A branch can propose a transfer from its till. It shows as <b>Requested</b>.',
        '<b>Approve</b> does the dispatch and the receive in one step. <b>Reject</b> turns it down.',
      ]),
    ]),

  entry('wf-count', 'Workflows', 'Count your stock',
    'Compare what is on the shelf with what the system says, and fix the difference.',
    [],
    'Count when the shop is quiet, so sales do not change the numbers while you count.',
    [
      steps([
        'Open <b>Inventory</b> and click <b>Start Count</b> on the place you will count. Or use <b>Stock Counts</b> and <b>New Count</b>.',
        'The sheet starts with the system quantity for every item.',
        'Change the <b>counted</b> quantity wherever the shelf is different.',
        'Click <b>Save Draft</b> to stop and continue later.',
        'Click <b>Post</b> when done. Every difference becomes a stock movement you can see in <b>Stock Movements</b>.',
      ]),
    ]),

  entry('wf-new-till', 'Workflows', 'Connect a new till',
    'Link a POS computer to Kashi Cloud.',
    [],
    'The code is also the start of that till\'s invoice numbers, so it cannot be changed once the till is in use.',
    [
      steps([
        'Open <b>Branches</b>. For a new shop click <b>Add Branch</b>. For another till in an existing shop open its <b>Registers</b> and click <b>Add Register</b>.',
        'Give the till a short <b>code</b> of 1 to 8 letters or digits, like CT2, and optionally a name.',
        'Kashi shows the <b>API key</b> <b>once</b>. Copy it now.',
        'On the till, open the kashi-pos settings and enter the <b>code</b> and the <b>key</b>.',
        'Back on the Registers list, the till shows <b>Online</b> when it connects.',
        'Check its behaviour in <b>Branch Settings</b>, and its app version in <b>App Updates</b>.',
      ]),
      bullets(['Lost the key? Use <b>Rotate Key</b> to make a new one. The old key stops working.']),
    ]),

  entry('wf-day-end', 'Workflows', 'Check the day',
    'A simple end-of-day routine.',
    [],
    '',
    [
      steps([
        'Open <b>Shifts</b> and check that every shift that should be closed is closed.',
        'Open <b>Daily Income</b> and click the day. Look at the totals against last week, the payments and the hour-by-hour chart.',
        'Check <b>Expenses</b> for the day.',
        'Click <b>Email</b> to send the day\'s report to the people who need it. You can also set a <b>Schedule</b> in Emails to do this by itself.',
        'Look at <b>Conflicts</b> and <b>Commands</b> for anything waiting or failed.',
      ]),
    ]),

  entry('wf-return', 'Workflows', 'Return or exchange a sale from the office',
    'Take goods back, or swap them, for a sale made at a till.',
    [],
    'If nothing happens, check <b>Commands</b>. The till may be offline, and the command runs when it is back.',
    [
      steps([
        'Open <b>Invoices</b> and find the sale. Search or filter by branch.',
        'Open it and click <b>Return</b> or <b>Exchange</b>.',
        'Choose the lines and quantities coming back. For an exchange, add the items you give instead.',
        'Add a reason if you like, then queue it.',
        'The request is sent to the till. The return or exchange shows in Invoices once the branch has processed it.',
      ]),
      bullets(['To cancel a sale completely, use <b>Cancel invoice</b> in the same dialog.']),
    ]),

  entry('wf-campaign', 'Workflows', 'Make a customer list for a campaign',
    'Get phone numbers of the people who bought at a branch.',
    [],
    'Send each file from its own channel. Because of No duplicates, nobody is in two files.',
    [
      steps([
        'Open <b>Clients</b> and choose the <b>Retail</b> or <b>Wholesale</b> tab.',
        'Click <b>Export CSV</b>.',
        'Tick <b>Name</b>, <b>Phone</b> and <b>Phone check (Lebanese)</b>.',
        'Choose the branches, and a date range if you only want recent buyers.',
        'Tick <b>No duplicates</b> if the same person should not get two messages.',
        'Click <b>Export</b>. Open the file and remove the rows marked <b>Invalid</b>.',
      ]),
    ]),

  entry('wf-loyalty-coupon', 'Workflows', 'Reward customers with tiers and coupons',
    'Two ways to keep customers coming back.',
    [],
    '',
    [
      h('Tiers (automatic)'),
      bullets([
        'A client\'s tier depends on how much they have spent in total.',
        'Change the tier names and limits with the cog on the <b>Clients</b> page.',
        'See how many people are in each tier on <b>Loyalty</b>.',
      ]),
      h('Coupons'),
      steps([
        'Open <b>Coupons</b>, then the <b>Types</b> tab, and add a type: its value, its currency, and whether it applies by itself or needs the code typed.',
        'Go to the <b>Coupons</b> tab and click <b>Add Coupon</b>. Choose the code, the type, and optionally one client and an end date.',
        'For many codes at once, use <b>Generate List</b>.',
        'Click <b>Deactivate</b> to stop a coupon.',
      ]),
    ]),

  entry('wf-targets', 'Workflows', 'Set a sales target',
    'Give a branch a goal and watch it.',
    [],
    '',
    [
      steps([
        'Open <b>Branches</b> and the <b>Targets</b> of the branch.',
        'For a single goal click <b>Add Target</b>: choose <b>From</b>, <b>To</b> and the amount in dollars.',
        'For a goal that repeats, use <b>Add Recurring Target</b>: the day each period starts, and repeat every 1 month, 3 months for a quarter, and so on.',
        'Watch progress on the <b>Targets</b> page.',
      ]),
      bullets(['You can pause a recurring target. Targets it already made are kept.']),
    ]),

  entry('wf-settings', 'Workflows', 'Change a till setting',
    'Settings live in two places.',
    [],
    'Money settings here are in dollars.',
    [
      h('One branch'),
      steps([
        'Open <b>Branch Settings</b> and choose the branch.',
        'Pick the section, change the values, and click <b>Save</b>. Only what you changed is sent.',
      ]),
      h('Every branch'),
      steps([
        'Open <b>Default Settings</b>.',
        'Set only the values you want to push. Leave the others on <b>Leave as is</b>.',
        'Click <b>Apply to all branches</b>.',
      ]),
    ]),

  entry('wf-money', 'Workflows', 'Record an expense or a loan payment',
    'Keep the money going out in one place.',
    [],
    'Pick the right tab first. A branch expense is one the till can see, and a central one stays in the office.',
    [
      h('Expense'),
      steps([
        'Open <b>Expenses</b> and choose <b>Central</b> or <b>Branch</b>.',
        'Make sure the category exists in the left list. Use <b>Add Category</b> if not.',
        'Click <b>Add Expense</b> and fill in the description, category and amount.',
        'For rent or salaries that repeat, use <b>Add Recurring</b> with the first due date and how often.',
      ]),
      h('Loan payment'),
      steps([
        'Open <b>Loans</b> and find the loan.',
        'Open its payments and click <b>Add Payment</b>.',
        'The loan shows what is still remaining.',
      ]),
    ]),

  entry('wf-user', 'Workflows', 'Add a user and limit what they can see',
    'Admins only.',
    [],
    'A new role starts with nothing switched on, so turn on what the person needs.',
    [
      steps([
        'Open <b>Settings</b> and the <b>Roles</b> tab. Click <b>Add role</b>.',
        'Turn each section <b>On</b> or <b>Off</b>. Inside a section, set each page to <b>None</b>, <b>View</b> or <b>Edit</b>.',
        'Save the role.',
        'Open the <b>Users</b> tab, add the person, and choose the role.',
      ]),
      bullets([
        'You can change a person\'s role, name or password later, or switch them off.',
        'A role that still has users cannot be deleted. Move its users first.',
      ]),
    ]),

  entry('wf-attendance', 'Workflows', 'Fix attendance',
    'Get a branch\'s clock-in times right.',
    [],
    '',
    [
      steps([
        'Open <b>Attendance</b> and choose the branch.',
        'Click <b>Sync from POS</b> to pull the latest punches. For a branch with no clock-in device, use <b>Import CSV</b> instead.',
        'Open <b>Requests</b> and approve or reject changes asked for on the till.',
        'Fix a wrong punch with <b>Edit</b>, or delete it.',
        'Check the <b>Monthly grid</b> to see the result.',
      ]),
      bullets(['Staff are set up per branch in <b>Branches</b> under <b>Staff</b>, with their attendance ID from the clock-in device.']),
    ]),

  entry('wf-conflicts', 'Workflows', 'Resolve a conflict',
    'Deal with things from the branches that need a decision.',
    [],
    '',
    [
      h('A client name or phone does not match'),
      steps([
        'Open <b>Conflicts</b>.',
        'Look at what the branch says and what Kashi has.',
        'Choose: keep Kashi\'s name, use the branch\'s name, <b>merge</b> into an existing client, or <b>create a new client</b> with a phone number.',
        'Click <b>Resolve</b>, or <b>Dismiss</b> if it does not matter.',
      ]),
      h('Oversold stock'),
      bullets([
        'A till sold more than the system had. Correct it with a stock adjustment, a transfer or a stock count.',
      ]),
    ]),

  entry('home', 'Start here', 'Home',
    'A quick look at what is happening right now. Open it first thing in the day.',
    [
      '<b>Recent activity</b> lists what the branches just did, newest first: sales, shifts closed, new clients and more. Click a row to see the details.',
      'Items marked <b>pinned</b> stay at the top until someone clicks <b>Mark resolved</b>.',
      '<b>Latest attachments</b> shows the newest photos, like receipts. <b>Show more</b> loads older ones.',
    ],
    'The counts on this page cover the last week.'),

  // ----------------------------------------------------------------- Sales
  entry('daily-income', 'Sales', 'Daily Income',
    'A grid of every day\'s income, branch by branch. It is the quickest way to see how a day or a month went.',
    [
      'Pick one or more branches with the chips at the top, or <b>All branches</b>. Your choice stays in the address, so you can bookmark it.',
      'Click a day to open it.',
    ],
    'Branches switched off under <b>In reports</b> do not appear here.',
    [
      h('Inside a day'),
      bullets([
        'Totals compared with the same weekday last week.',
        '<b>Payments</b>: how much came in by each payment method.',
        '<b>Best sellers</b> and <b>salespeople</b> for the day.',
        'An hour-by-hour chart of net income and customers.',
        'Tabs for <b>Invoices</b> and for <b>Shifts &amp; Expenses</b>. A closed shift can be reprinted on the till that closed it.',
        '<b>Email</b> sends this day\'s report. The arrows move to the previous or next day.',
      ]),
    ]),

  entry('invoices', 'Sales', 'Invoices',
    'All sales, in two tabs: <b>Retail</b> and <b>Wholesale</b>. Sales made at the tills show here with their branch.',
    [
      'Search by number or name, and filter by branch.',
      'Open an invoice to see its items, discounts, loyalty points and how it was paid.',
    ],
    'To return or exchange a sale, see the workflow <b>Return or exchange a sale from the office</b>.',
    [
      h('Add a sale from the office'),
      steps([
        'Click <b>Add Invoice</b>.',
        'Choose the <b>cashbox</b>, the <b>shift</b>, the <b>inventory</b> the goods come from and the <b>client</b>. A salesperson and a price list are optional.',
        'Click <b>Add Line</b> and search by SKU or barcode. Set quantity, price and any discount.',
        'Add an invoice discount if needed, check the totals, and click <b>Create Invoice</b>.',
      ]),
      h('Good to know'),
      bullets([
        'A cancelled till invoice is marked <b>Cancelled</b> and no longer counts in reports.',
        'A return shows its <b>Original Sale</b>, so you can trace it back.',
      ]),
    ]),

  entry('shifts', 'Sales', 'Shifts',
    'Every cashier shift. A shift starts when a cashier opens the till and ends when they close it and count the cash.',
    [
      'Filter by status, cashbox or branch, or search a cashbox name or shift number.',
      'Branch shifts are opened and closed on the till. The page shows what each one reported.',
      '<b>Open Shift</b> and <b>Close</b> are for shifts in the office.',
    ],
    '',
    [
      h('Close-shift receipt'),
      bullets([
        'A closed branch shift can have a <b>photo of its receipt</b>. You can rotate or replace it.',
        'Click the print button to ask the till that closed it to <b>reprint</b> the report.',
      ]),
      h('Visitors tab'),
      bullets(['Shows how many people the door counter counted each day, per branch. It updates live.']),
    ]),

  entry('whatsapp', 'Sales', 'WhatsApp & Reviews',
    'Kashi can send a thank-you message on WhatsApp after a sale. This page shows what was sent, what it cost and how it went.',
    [
      '<b>Dashboard</b>: totals, cost and review results.',
      '<b>Messages</b>: each message, who got it and its status. Open one to see the chat.',
      '<b>Replies</b>: what customers answered.',
      '<b>Rules</b>: decide who gets which message.',
      '<b>Health</b>: warns you when a sending number is at risk or paused. A red dot on the tab means look here.',
    ],
    'The account details and the default message are in <b>Settings &gt; WhatsApp</b>. Costs only show when you send through Meta.',
    [
      h('Good to know'),
      bullets([
        'The top of the page shows whether sending is <b>on</b>.',
        'If a till fails to send, it is told to retry temporary failures.',
        'Use the <b>From / To</b> dates and <b>Refresh</b> to look at a period.',
      ]),
    ]),

  entry('emails', 'Sales', 'Emails',
    'Invoice reports sent by email.',
    [
      '<b>Send report</b>: choose a day, a date range or a shift, choose <b>CSV</b> or <b>Excel</b>, pick the registers, add who to send to and an optional note.',
      '<b>Schedules</b>: reports sent by themselves at a set time (Beirut time). Use <b>New schedule</b>, <b>Send now</b> or <b>Edit</b>.',
      '<b>History</b>: everything sent, and whether it worked.',
    ],
    'Email needs its key set once in <b>Settings &gt; Email</b>. Until then the page says it is not set up.'),

  // -------------------------------------------------------------- Analysis
  entry('dashboard', 'Analysis', 'Dashboard',
    'The big picture for <b>this month</b>, <b>last month</b> or <b>this year</b>, with a branch filter.',
    [
      '<b>Overview</b>: revenue, expenses, profit, profit margin and your financial position.',
      '<b>Sales</b>: items sold, new clients, invoices, due loans, purchases, targets, average invoice and revenue per client.',
      '<b>Financials</b>: sales with a forecast, by branch and by payment method.',
      '<b>Demand</b>: slow movers and reordering suggestions.',
      '<b>Alerts</b> and <b>Health</b>: low and out-of-stock products, overstock, and how fast stock turns over.',
    ],
    'Profit only counts items that have a cost. Items without a cost make profit look higher. The page tells you how much is covered.',
    [
      h('How the main numbers work'),
      bullets([
        '<b>Profit</b> = revenue minus the cost of goods sold minus expenses.',
        '<b>Financial position</b> = revenue minus expenses minus purchases. Purchases are office-only, so under a single branch they count as zero.',
      ]),
    ]),

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

  entry('reports', 'Analysis', 'Reports',
    'End-of-month (or any period) reports to print or save as PDF, in English or Arabic: monthly overview, customers (with busy times), customer feedback, salespersons & attendance, and sales & goods.',
    [
      'Pick a report on the left and tick the parts you want -- for example only the attendance parts of the salespersons report.',
      'Choose the period, branches and (for salespersons) employees, then <b>Save as default</b>: the page opens like that from then on, for everyone.',
      '<b>Leave out products or customers</b> keeps them out of every report: a customer\'s sales drop out of all figures, a product drops out of best sellers, categories, sizes and item counts.',
      'The preview is an A4 page: zoom in and out, or <b>Fit width</b>, above it. The <b>EN / عربي</b> switch draws the report in Arabic, right to left.',
      '<b>Export PDF</b> opens the print dialog: choose <b>Save as PDF</b>. <b>CSV</b> downloads the tables of the ticked parts for Excel.',
      'Customer feedback shows the poll\'s own answers (Loved it, It was okay, Not great) and only the replies to the follow-up question.',
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
      '<b>Top Spenders</b> shows your best clients as a pyramid. Sort by <b>most spent</b>, <b>most invoices</b> or <b>most items</b>, and show the top 15, 50 or 100.',
      '<b>All Clients</b> lists everyone else. Search by name or phone.',
      'The branch button lets you pick <b>several branches</b>. The numbers then add up across those branches. With a branch picked you see the clients who bought there, plus those whose <b>home branch</b> it is.',
      'The <b>date range</b> button keeps only clients who bought in that period, and the numbers count that period only. Click <b>Clear</b> for any time.',
      'With a date range picked, <b>First-timers</b> shows clients whose first purchase was inside the period, and <b>Returning</b> those who had already bought before it. With a branch picked, "first purchase" means at those branches.',
    ],
    'The circle colour is the client\'s tier. Change tier names and limits with the cog.',
    [
      h('Client details'),
      bullets([
        'Click anyone, in the pyramid or in the list, to open a side panel: total spent, invoices, items, phone, type, loyalty points and member since.',
        '<b>Open full profile</b> shows the whole history.',
        '<b>Edit</b> changes the name, phone or type. <b>Delete</b> removes the client.',
      ]),
      h('Add or delete'),
      bullets([
        '<b>Add Client</b> needs a name and a phone of at least 7 digits. It starts on the tab you are on.',
        'A client who already has invoices, points or a branch link is not erased. They are <b>deactivated</b> instead and tills stop offering them. Open them and click <b>Reactivate</b> to bring them back.',
      ]),
    ]),

  entry('clients-export', 'Customers', 'Exporting clients',
    'On the Clients page, <b>Export CSV</b> downloads the clients of the tab you are on.',
    [
      'Tick the <b>fields</b> you want as columns.',
      'A client belongs to a branch if they <b>bought something there</b>, so someone who shopped at two branches is in both files. Each client also has a <b>home branch</b>: the first one they bought from (or the one they were imported with). They are in that branch\'s file even if they have no purchase there yet. Each branch you pick gets its own file. Several branches come as one zip.',
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
      'Choose a branch, or <b>All</b>. Importing a CSV needs one branch.',
      '<b>Sync from POS</b> pulls punches from the tills. <b>Import CSV</b> is for a branch with no clock-in device.',
      'Four views: <b>Timeline</b>, <b>In / out pairs</b>, <b>Monthly grid</b> and <b>Punches</b>.',
    ],
    'See the workflow <b>Fix attendance</b>.',
    [
      h('Requests'),
      bullets([
        'A cashier can ask for a change on the till. It shows under <b>Requests</b>.',
        '<b>Approve</b> or <b>Reject</b>. It stays in the list until the till has received your answer.',
      ]),
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
    'Products that need attention. A red number on the menu shows how many.',
    [
      '<b>Out of stock</b>: nothing left.',
      '<b>Low stock</b>: at or below its limit.',
      '<b>Ignored</b>: products you muted.',
    ],
    '',
    [
      h('Set limits'),
      bullets([
        'The <b>default low-stock limit</b> applies to every product without its own.',
        'Click a product to give it its own <b>threshold</b>, or <b>Reset to default</b>.',
        '<b>Mute alerts</b> hides a product from the lists. <b>Resume alerts</b> brings it back.',
      ]),
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
      '<b>Add Inventory</b> with a name, a type and a code. A location (latitude and longitude) is optional.',
      'Click one to see each item\'s <b>barcode</b>, quantity <b>on hand</b> and <b>average cost</b>. Search by SKU or barcode.',
      '<b>Start Count</b> opens a stock count for that place.',
    ],
    '',
    [
      h('Quick adjust'),
      steps([
        'Find the item and type the real quantity in <b>set count</b>.',
        'Click <b>Set</b> to update the quantity.',
      ]),
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
    [],
    'Steps in the workflow <b>Move stock between places</b>.',
    [
      h('Statuses'),
      bullets([
        '<b>Draft</b>: being prepared. You can still edit the items.',
        '<b>Requested</b>: a branch asked for it. <b>Approve</b> or <b>Reject</b>.',
        '<b>Dispatched</b>: left the source. Stock is in transit.',
        '<b>Received</b>: arrived. Stock is in the destination.',
      ]),
      h('Good to know'),
      bullets([
        'A transfer can have a photo attached.',
        'Use <b>Cancel Transfer</b> for one that should not happen.',
      ]),
    ]),

  entry('stock-counts', 'Stock', 'Stock Counts',
    'Check the real stock against what the system says.',
    [
      '<b>New Count</b>: choose the inventory. The sheet starts with the system quantities.',
      'Change the counted quantity of each item.',
      '<b>Save Draft</b> to continue later, or <b>Post</b> to apply it.',
    ]),

  entry('stock-movements', 'Stock', 'Stock Movements',
    'A read-only history of every stock change. Use it to find out why a quantity is what it is.',
    [
      'Filter by inventory and by reason.',
      'Each row shows the <b>reason</b>, the <b>document</b> behind it (a sale, a purchase, a transfer or a count), the quantity <b>before</b> and <b>after</b>, and who did it.',
      'Click a row for more details.',
    ]),

  entry('barcodes', 'Stock', 'Barcodes',
    'Print barcode labels.',
    [
      'Search products, set how many labels each, then <b>Generate barcodes</b> to get a PDF.',
      'Label size is set in the dialog (72 points = 1 inch).',
    ]),

  entry('receipt-designer', 'Branches & tills', 'Receipt Designer',
    'Choose how the printed invoice looks.',
    [
      'Pick a <b>theme</b> and one of its <b>layouts</b>; the picture is a sample receipt.',
      '<b>Apply to</b> one branch or all of them, then <b>Save</b>. Each register changes within a few seconds if online.',
    ]),

  entry('barcode-designer', 'Stock', 'Barcode Designer',
    'Choose how the barcode labels look.',
    [
      'Pick a <b>theme</b>, then change anything by hand: which parts show, their size and position, the font and the alignment.',
      'The picture beside the controls is the real label. <b>Save</b> to use the design on every label you print from Barcodes.',
      '<b>Reset to theme</b> goes back to the theme you started from. The Classic theme is the original label.',
    ]),

  entry('assets', 'Stock', 'Assets',
    'Things the business owns, like vehicles, equipment and furniture. Each has a name, code, category and purchase date.'),

  entry('treemaps', 'Stock', 'Assets & Warehouses Treemaps',
    'Two picture views. Bigger boxes mean more value. One shows your assets, the other shows what is in your warehouses.'),

  // ---------------------------------------------------------------- Branches
  entry('branches', 'Branches & tills', 'Branches',
    'Your shops, their tills, their staff and their targets.',
    [
      '<b>Add Branch</b> with a name and a short code. Its API key is shown once.',
      'Each branch row opens <b>Registers</b>, <b>Staff</b> and <b>Targets</b>.',
      'The <b>In reports</b> switch keeps a branch syncing but removes it from every report.',
    ],
    '',
    [
      h('Registers (tills)'),
      bullets([
        'Each till has a <b>code</b>, a name, a status (<b>Online</b> or when it was last seen), and two switches: <b>Active</b> and <b>In reports</b>.',
        '<b>Rotate Key</b> makes a new key for a till. See <b>Connect a new till</b>.',
      ]),
      h('Staff'),
      bullets([
        'The people who work at the branch: name, <b>attendance ID</b> (their number on the clock-in device), a colour, and <b>Sells</b>.',
        'Turn on <b>Sells</b> for people who should appear as a salesperson at checkout.',
      ]),
      h('Targets'),
      bullets(['One-off or repeating sales goals. See <b>Set a sales target</b>.']),
    ]),

  entry('branch-settings', 'Branches & tills', 'Branch Settings',
    'Change how one branch\'s till behaves, without going there.',
    [
      'Choose the branch, then a section: <b>General</b>, <b>Behaviour</b>, <b>Cash &amp; shifts</b>, <b>Page access</b>, <b>Printing</b>, <b>Attendance</b>, <b>Users</b>, <b>Payments</b>, <b>Quick access</b> and <b>Till display</b>.',
      '<b>Save</b> sends only what you changed. Everything else stays as the till has it.',
      '<b>Sync now</b> asks the till to send its latest data. <b>Commands</b> opens the Commands page.',
    ],
    'To change many branches the same way, use <b>Default Settings</b> instead.',
    [
      h('Examples of what you can set'),
      bullets([
        'The branch name, phone and website printed on receipts.',
        'The tax rate, how totals are rounded, and the exchange rate.',
        'Whether cashiers can give custom discounts or set custom prices, and whether every invoice needs a client.',
        'Whether closing a shift asks for a photo of the receipt.',
        'What each till user can open.',
      ]),
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
      '<b>Send command</b>, then choose the branch, the till and the type.',
      'Types: <b>Sync now</b>, <b>Print close shift</b>, <b>Reprint invoice</b>, <b>Return invoice</b>, <b>Swap shift balance</b>, <b>Set PIN</b>, <b>Update settings</b> and <b>Transfer decision</b>.',
      'Cards at the top show how many are <b>Queued</b>, how many <b>Failed</b> in the last 24 hours, and how many tills are online.',
    ],
    'A command runs when the till is online. If it is not, it waits in the queue.',
    [
      h('Statuses'),
      bullets([
        '<b>Queued</b>: waiting for the till. You can <b>Cancel</b> it.',
        '<b>Done</b>: the till did it.',
        '<b>Failed</b>: it did not work. Check the message, then <b>Retry</b>.',
        '<b>Cancelled</b>: you stopped it.',
      ]),
    ]),

  entry('conflicts', 'Branches & tills', 'Conflicts',
    'Things from the branches that need a decision.',
    [
      '<b>Oversold stock</b>: a till sold more than the system had. Fix it with an adjustment, transfer or count.',
      '<b>Client conflicts</b>: a name or phone that does not match. Keep the Cloud\'s version, use the branch\'s, merge into an existing client, or create a new one with a phone number.',
    ]),

  entry('app-updates', 'Branches & tills', 'App Updates',
    'Choose which version of the till app each till runs. You publish a build from the kashi-pos project, then decide who gets it here.',
    [
      'Tills download an update in the background and install it when it is safe: when the cashier taps <b>Install now</b>, or by itself while no shift is open.',
      'If a new version does not start, the till goes back to the old one by itself.',
      'The top cards show the <b>latest release</b>, how many tills are <b>on their target</b>, and how many are updating.',
    ],
    '',
    [
      h('Common jobs'),
      bullets([
        '<b>Deploy</b> a release to chosen tills.',
        '<b>Set version</b> on one till. Choosing an older release is how you <b>roll back</b>.',
        'Tick <b>Install now, even during a shift</b> only when it cannot wait.',
        '<b>Edit notes</b> for what cashiers see after an update. You cannot delete a release tills are still set to.',
      ]),
    ]),

  // ---------------------------------------------------- Pricing & products
  entry('price-lists', 'Pricing & products', 'Price Lists & Discount Lists',
    'A list gives certain products a different price, or a discount in percent. Price Lists and Discount Lists work the same way.',
    [
      'Pick a list on the left, or <b>Add</b> a new one.',
      '<b>Default</b> marks the list used when nothing else applies.',
      'Set <b>Valid from / to</b> dates and switch the list <b>Active</b> or not.',
    ],
    'A branch only uses a list if it is assigned to it.',
    [
      h('Fill a list'),
      steps([
        'Open the list.',
        'Search for a product variant and click <b>Add</b>.',
        'Enter its price (or discount %).',
        'To remove products, select them and delete. Delete the whole list to remove it and its entries.',
      ]),
      h('Branches'),
      bullets(['Click <b>Branches</b> on a list to see and choose which branches use it.']),
    ]),

  entry('coupons', 'Pricing & products', 'Coupons',
    'Codes that give a customer a value. Three tabs: <b>Coupons</b>, <b>Types</b> and <b>Lists</b>.',
    [
      '<b>Types</b> say what a coupon is worth: a value and a currency.',
      'A type can <b>apply automatically</b> at checkout, or need its <b>code typed</b>. It can also be <b>reference only</b>, with no money taken off.',
      'A <b>coupon</b> has a code, a type, a category, optionally one client, a status and an end date.',
    ],
    'See the workflow <b>Reward customers with tiers and coupons</b>.',
    [
      h('Bulk'),
      bullets([
        '<b>Generate List</b> makes many codes at once and lets you copy them.',
        'Select several coupons to delete them together.',
      ]),
    ]),

  entry('products', 'Pricing & products', 'Products',
    'Your product catalog. Every product has one or more <b>variants</b>, one for each color and size.',
    [
      'Search by name or code.',
      '<b>Filters</b>: status (active or not), attributes, created between two dates, and whether it has variants.',
      '<b>Add Product</b> opens the form. <b>Import CSV</b> adds many at once.',
    ],
    'New products and price changes reach the tills by themselves.',
    [
      h('The Add Product form'),
      bullets([
        '<b>Code</b>, <b>name</b> and an optional description.',
        '<b>Colors</b> and <b>sizes</b>: pick several of each. You get one variant for each combination.',
        '<b>Barcode</b> (optional) and <b>Price</b>. Choose which price list the price goes into.',
        '<b>Auto-fill</b> fills attributes and price from the code and name, using your Auto-fill Rules.',
        '<b>Save &amp; Add Another</b> keeps the form open for the next product.',
      ]),
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
      '<b>Central</b> expenses are only in the office. <b>Branch</b> expenses are the ones tills can see and add.',
      '<b>Recurring</b> repeats by itself. Choose how often (days, weeks or months) and the first due date.',
      'The list on the left is the <b>categories</b>. Click one to narrow the list. Add, rename or delete categories there.',
    ],
    'Central and branch expenses have their own categories.',
    [
      h('Add an expense'),
      steps([
        'Choose the tab, then click <b>Add Expense</b> (or <b>Add Branch Expense</b>).',
        'Enter the description, category and amount. For a branch expense, choose the branch and the date.',
        'Save. Select several rows to delete them together.',
      ]),
    ]),

  entry('loans', 'Accounting', 'Loans',
    'Money borrowed, in <b>Central</b> and <b>Branch</b> tabs.',
    [
      '<b>Add Loan</b> with a lender (like Owner or External), amount, currency and category.',
      'Open a loan to see its <b>Payments</b> and what is still <b>remaining</b>.',
      'Filter by status or branch, or search.',
    ],
    'Kashi will not accept a payment bigger than what is left when it is in the loan\'s own currency.',
    [
      h('Add a payment'),
      steps([
        'Expand the loan.',
        'Enter the amount, the date and an optional note.',
        'Click <b>Add Payment</b>. The remaining amount updates.',
      ]),
    ]),

  entry('currencies', 'Accounting', 'Currencies & Payments',
    'The money you accept and the ways customers pay. Two tabs.',
    [
      '<b>Currencies</b>: a code (like USD), name and symbol, the <b>rate</b> as units per 1 USD, and a <b>cash rounding unit</b>. For example, 90,000 for LL means 90,000 LL to the dollar.',
      'The <b>cash rounding unit</b> rounds cash in that currency to the nearest multiple. 1 means no rounding.',
      '<b>Payment Methods</b>: cash, card, transfer and so on, each with an order and an optional color.',
    ],
    'Methods marked <b>System</b> (like On Account and On Coupon) are built into the till and cannot be edited. The default currency cannot be deleted.'),

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
    'App-wide options, in tabs. Save and Discard light up only when you change something.',
    [
      '<b>General</b>: how long messages show, how many days ahead the calendar cards look, and which months are the high season.',
      '<b>Financials</b> and <b>Products</b>: money options, and turning on smart auto-fill.',
      '<b>Barcode labels</b>: the default label width and height.',
      '<b>Users</b> and <b>Roles</b> (admins only). See <b>Add a user and limit what they can see</b>.',
    ],
    '',
    [
      h('The accounts Kashi sends with'),
      bullets([
        '<b>WhatsApp</b>: the phone number ID, access token and message template, and whether to send a thank-you after each sale.',
        '<b>Reviews</b>: the review request settings.',
        '<b>Email</b>: the API key, sender and reply-to, and default recipients.',
      ]),
      h('Your own look'),
      bullets(['Your avatar color is saved on this device, and is always yours to change.']),
    ]),
]

// The groups in the order they first appear, for the contents list.
export const guideGroups = [...new Set(guideSections.map(s => s.group))]

// Icons for the contents tree. Only icons already registered in
// plugins/icons.js (most are the nav drawer's own icons for the same page).
export const guideGroupIcons = {
  'Start here': 'mdi-rocket-launch-outline',
  'Workflows': 'mdi-check-circle-outline',
  'Sales': 'mdi-cart',
  'Analysis': 'mdi-chart-bar',
  'Customers': 'mdi-account',
  'People': 'mdi-account-group',
  'Stock': 'mdi-warehouse',
  'Branches & tills': 'mdi-store-outline',
  'Pricing & products': 'mdi-sale',
  'Accounting': 'mdi-currency-usd',
  'System': 'mdi-cog-outline',
}

export const guideIcons = {
  'how-kashi-works': 'mdi-information',
  'glossary': 'mdi-format-list-bulleted',
  'home': 'mdi-home',
  'wf-new-product': 'mdi-package-variant',
  'wf-buy-stock': 'mdi-truck-delivery',
  'wf-transfer': 'mdi-transfer',
  'wf-count': 'mdi-clipboard-list-outline',
  'wf-new-till': 'mdi-store-outline',
  'wf-day-end': 'mdi-clock-outline',
  'wf-return': 'mdi-swap-horizontal',
  'wf-campaign': 'mdi-file-delimited-outline',
  'wf-loyalty-coupon': 'mdi-ticket-percent',
  'wf-targets': 'mdi-target',
  'wf-settings': 'mdi-store-cog-outline',
  'wf-money': 'mdi-currency-usd',
  'wf-user': 'mdi-account-group',
  'wf-attendance': 'mdi-account-multiple-check-outline',
  'wf-conflicts': 'mdi-source-branch-sync',
  'daily-income': 'mdi-chart-bar',
  'invoices': 'mdi-format-list-bulleted-square',
  'shifts': 'mdi-clock-outline',
  'whatsapp': 'mdi-whatsapp',
  'emails': 'mdi-email-outline',
  'dashboard': 'mdi-view-dashboard',
  'targets': 'mdi-target',
  'footfall': 'mdi-walk',
  'reports': 'mdi-file-chart-outline',
  'busy-hours': 'mdi-clock-time-four-outline',
  'product-revenues': 'mdi-package-variant',
  'conversion': 'mdi-percent-outline',
  'sales-analytics': 'mdi-chart-line',
  'sales-analysis': 'mdi-chart-box-outline',
  'financials': 'mdi-currency-usd',
  'sales-dashboard': 'mdi-view-dashboard',
  'clients': 'mdi-account',
  'clients-export': 'mdi-file-delimited-outline',
  'loyalty': 'mdi-trophy',
  'employees': 'mdi-account-group',
  'attendance': 'mdi-account-multiple-check-outline',
  'salesperson-performance': 'mdi-account-cash',
  'alerts': 'mdi-alert-circle-outline',
  'stock-health': 'mdi-heart-pulse',
  'inventory': 'mdi-warehouse',
  'inventory-analysis': 'mdi-chart-box-outline',
  'purchase-invoices': 'mdi-invoice',
  'suppliers': 'mdi-truck-delivery',
  'batches': 'mdi-package-variant-closed',
  'transfers': 'mdi-transfer',
  'stock-counts': 'mdi-clipboard-list-outline',
  'stock-movements': 'mdi-swap-vertical',
  'barcodes': 'mdi-barcode',
  'barcode-designer': 'mdi-palette',
  'receipt-designer': 'mdi-receipt-text',
  'assets': 'mdi-hammer-wrench',
  'treemaps': 'mdi-view-gallery',
  'branches': 'mdi-store-outline',
  'branch-settings': 'mdi-store-cog-outline',
  'default-branch-settings': 'mdi-content-copy',
  'commands': 'mdi-console-line',
  'conflicts': 'mdi-source-branch-sync',
  'app-updates': 'mdi-update',
  'price-lists': 'mdi-format-list-bulleted',
  'coupons': 'mdi-ticket-percent',
  'products': 'mdi-package-variant',
  'colors-sizes': 'mdi-palette',
  'attributes': 'mdi-tag-multiple',
  'autofill-rules': 'mdi-auto-fix',
  'expenses': 'mdi-currency-usd-off',
  'loans': 'mdi-hand-coin',
  'currencies': 'mdi-cash-multiple',
  'calendar': 'mdi-calendar',
  'storage': 'mdi-google-drive',
  'security': 'mdi-shield-lock-outline',
  'audit-log': 'mdi-history',
  'dev-logs': 'mdi-code-tags',
  'settings': 'mdi-cog-outline',
}
