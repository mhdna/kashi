// The app's pages in sections. The rail on the far left shows one icon per
// section; the drawer beside it lists only the open section's pages. A
// section's item with `children` is a heading with its pages under it.
export const navSections = [
  {
    key: 'overview',
    title: 'Overview',
    icon: 'mdi-home',
    items: [
      { title: 'Home', icon: 'mdi-home', to: '/' },
      { title: 'Daily Income', icon: 'mdi-chart-bar', to: '/daily-income' },
      { title: 'Calendar', icon: 'mdi-calendar', to: '/calendar' },
      { title: 'Reports', icon: 'mdi-file-chart-outline', to: '/reports' },
      {
        title: 'Alerts',
        icon: 'mdi-alert-circle-outline',
        to: '/alerts',
        append: 'alert-count',
      },
    ],
  },
  {
    key: 'sales',
    title: 'Sales',
    icon: 'mdi-cash-register',
    items: [
      { title: 'Invoices', icon: 'mdi-format-list-bulleted-square', to: '/invoices' },
      { title: 'Unknown Items', icon: 'mdi-help-box-outline', to: '/unknown-items' },
      { title: 'Shifts', icon: 'mdi-clock-outline', to: '/shifts' },
      {
        title: 'Pricing',
        icon: 'mdi-tag-outline',
        children: [
          { title: 'Price Lists', icon: 'mdi-format-list-bulleted', to: '/price-lists' },
          { title: 'Discount Lists', icon: 'mdi-sale', to: '/discount-lists' },
          { title: 'Coupons', icon: 'mdi-ticket-percent', to: '/pricing' },
        ],
      },
    ],
  },
  {
    key: 'customers',
    title: 'Customers',
    icon: 'mdi-account-heart-outline',
    items: [
      { title: 'Clients', icon: 'mdi-account', to: '/clients' },
      { title: 'Loyalty', icon: 'mdi-trophy', to: '/loyalty' },
      { title: 'Referrals', icon: 'mdi-gift-outline', to: '/referrals' },
      { title: 'WhatsApp & Reviews', icon: 'mdi-whatsapp', to: '/whatsapp', append: 'whatsapp-status' },
      { title: 'Emails', icon: 'mdi-email-outline', to: '/emails' },
    ],
  },
  {
    key: 'stock',
    title: 'Products and stock',
    icon: 'mdi-package-variant',
    items: [
      {
        title: 'Products',
        icon: 'mdi-cart',
        children: [
          { title: 'Products', icon: 'mdi-package-variant', to: '/products' },
          { title: 'Colors & Sizes', icon: 'mdi-palette', to: '/colors-sizes' },
          { title: 'Attributes', icon: 'mdi-tag-multiple', to: '/attributes' },
          { title: 'Auto-fill Rules', icon: 'mdi-auto-fix', to: '/autofill-rules' },
        ],
      },
      {
        title: 'Inventory',
        icon: 'mdi-folder-network',
        children: [
          { title: 'Inventory', icon: 'mdi-warehouse', to: '/inventory' },
          { title: 'Stock Health', icon: 'mdi-heart-pulse', to: '/stock-health', append: 'status' },
          { title: 'Inventory Analysis', icon: 'mdi-chart-box-outline', to: '/inventory-analysis' },
          { title: 'Batches', icon: 'mdi-package-variant-closed', to: '/batches' },
          { title: 'Transfers', icon: 'mdi-transfer', to: '/transfers' },
          { title: 'Stock Counts', icon: 'mdi-clipboard-list-outline', to: '/stock-counts' },
          { title: 'Stock Movements', icon: 'mdi-swap-vertical', to: '/stock-movements' },
          { title: 'Barcodes', icon: 'mdi-barcode', to: '/barcodes' },
          { title: 'Assets', icon: 'mdi-hammer-wrench', to: '/assets' },
        ],
      },
      {
        title: 'Purchasing',
        icon: 'mdi-truck',
        children: [
          { title: 'Suppliers', icon: 'mdi-truck-delivery', to: '/suppliers' },
          { title: 'Purchase Invoices', icon: 'mdi-invoice', to: '/purchase-invoice' },
        ],
      },
    ],
  },
  {
    key: 'money',
    title: 'Money',
    icon: 'mdi-currency-usd',
    items: [
      { title: 'Expenses', icon: 'mdi-currency-usd-off', to: '/expenses' },
      { title: 'Loans', icon: 'mdi-hand-coin', to: '/loans' },
      { title: 'Currencies & Payments', icon: 'mdi-cash-multiple', to: '/currencies' },
    ],
  },
  {
    key: 'analysis',
    title: 'Analysis',
    icon: 'mdi-chart-line',
    items: [
      { title: 'Dashboard', icon: 'mdi-view-dashboard', to: '/dashboard' },
      { title: 'Sales Dashboard', icon: 'mdi-view-dashboard', to: '/sales-dashboard' },
      { title: 'Targets', icon: 'mdi-target', to: '/targets' },
      { title: 'Footfall', icon: 'mdi-walk', to: '/footfall' },
      { title: 'Busy Hours', icon: 'mdi-clock-time-four-outline', to: '/busy-hours' },
      { title: 'Product Revenues', icon: 'mdi-package-variant', to: '/product-revenues' },
      { title: 'Conversion', icon: 'mdi-percent-outline', to: '/conversion' },
      { title: 'Sales Analytics', icon: 'mdi-chart-line', to: '/sales-analytics' },
      { title: 'Sales Analysis', icon: 'mdi-chart-box-outline', to: '/sales-analysis' },
      { title: 'Financials', icon: 'mdi-currency-usd', to: '/financials' },
      { title: 'Assets Treemap', icon: 'mdi-view-gallery', to: '/assets-treemap' },
      { title: 'Warehouses Treemap', icon: 'mdi-warehouse', to: '/warehouses-treemap' },
    ],
  },
  {
    key: 'people',
    title: 'People',
    icon: 'mdi-account-tie-outline',
    items: [
      { title: 'Employees', icon: 'mdi-account-group', to: '/employees' },
      { title: 'Attendance', icon: 'mdi-account-multiple-check-outline', to: '/attendance' },
      { title: 'Salesperson Performance', icon: 'mdi-account-cash', to: '/salesperson-performance' },
    ],
  },
  {
    key: 'branches',
    title: 'Branches',
    icon: 'mdi-domain',
    items: [
      { title: 'Branches', icon: 'mdi-store-outline', to: '/branches' },
      { title: 'Branch Settings', icon: 'mdi-store-cog-outline', to: '/branch-settings' },
      { title: 'Default Settings', icon: 'mdi-content-copy', to: '/default-branch-settings' },
      { title: 'Commands', icon: 'mdi-console-line', to: '/branch-commands', append: 'commands-status' },
      { title: 'Conflicts', icon: 'mdi-source-branch-sync', to: '/branch-conflicts', append: 'conflicts-status' },
      { title: 'App Updates', icon: 'mdi-update', to: '/app-updates' },
    ],
  },
  {
    key: 'design',
    title: 'Design',
    icon: 'mdi-palette-outline',
    items: [
      { title: 'Receipt Designer', icon: 'mdi-receipt-text', to: '/receipt-designer' },
      { title: 'Barcode Designer', icon: 'mdi-barcode', to: '/barcode-designer' },
    ],
  },
  {
    key: 'system',
    title: 'System',
    icon: 'mdi-cog-outline',
    items: [
      { title: 'Settings', icon: 'mdi-cog-outline', to: '/settings' },
      { title: 'Security', icon: 'mdi-shield-lock-outline', to: '/security' },
      { title: 'Audit Log', icon: 'mdi-history', to: '/audit-log' },
      { title: 'Dev Logs', icon: 'mdi-code-tags', to: '/logs' },
      { title: 'Storage', icon: 'mdi-google-drive', to: '/storage' },
      { title: 'User Guide', icon: 'mdi-book-open-variant', to: '/user-guide' },
    ],
  },
]

// navPages is every page link in a section, headings' pages included.
export function navPages (section) {
  return section.items.flatMap(item => item.children ?? [item])
}

// sectionForPath is the section holding a page: the one with the longest
// link that the path starts with ('/' only for Home itself).
export function sectionForPath (sections, path) {
  let best = null
  let bestLength = -1
  for (const section of sections) {
    for (const page of navPages(section)) {
      const hit = page.to === '/' ? path === '/' : path === page.to || path.startsWith(`${page.to}/`)
      if (hit && page.to.length > bestLength) {
        best = section.key
        bestLength = page.to.length
      }
    }
  }
  return best
}

// For the command palette: each section as a group of its pages.
export const navItems = navSections.map(s => ({ title: s.title, icon: s.icon, children: navPages(s) }))

export const appendItems = []
