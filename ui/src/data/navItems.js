import { t } from '../i18n'

// The app's pages in sections. The rail on the far left shows one icon per
// section; the drawer beside it lists only the open section's pages. A
// section's item with `children` is a heading with its pages under it.
export const navSections = [
  {
    key: 'overview',
    get title () { return t('Overview') },
    icon: 'mdi-home',
    items: [
      { get title () { return t('Home') }, icon: 'mdi-home', to: '/' },
      { get title () { return t('Daily Income') }, icon: 'mdi-chart-bar', to: '/daily-income' },
      { get title () { return t('Calendar') }, icon: 'mdi-calendar', to: '/calendar' },
      { get title () { return t('Reports') }, icon: 'mdi-file-chart-outline', to: '/reports' },
      {
        get title () { return t('Alerts') },
        icon: 'mdi-alert-circle-outline',
        to: '/alerts',
        append: 'alert-count',
      },
    ],
  },
  {
    key: 'sales',
    get title () { return t('Sales') },
    icon: 'mdi-cash-register',
    items: [
      { get title () { return t('Invoices') }, icon: 'mdi-format-list-bulleted-square', to: '/invoices' },
      { get title () { return t('Unknown Items') }, icon: 'mdi-help-box-outline', to: '/unknown-items' },
      { get title () { return t('Shifts') }, icon: 'mdi-clock-outline', to: '/shifts' },
      {
        get title () { return t('Pricing') },
        icon: 'mdi-tag-outline',
        children: [
          { get title () { return t('Price Lists') }, icon: 'mdi-format-list-bulleted', to: '/price-lists' },
          { get title () { return t('Discount Lists') }, icon: 'mdi-sale', to: '/discount-lists' },
          { get title () { return t('Coupons') }, icon: 'mdi-ticket-percent', to: '/pricing' },
        ],
      },
    ],
  },
  {
    key: 'customers',
    get title () { return t('Customers') },
    icon: 'mdi-account-heart-outline',
    items: [
      { get title () { return t('Clients') }, icon: 'mdi-account', to: '/clients' },
      { get title () { return t('Loyalty') }, icon: 'mdi-trophy', to: '/loyalty' },
      { get title () { return t('Referrals') }, icon: 'mdi-gift-outline', to: '/referrals' },
      { get title () { return t('WhatsApp & Reviews') }, icon: 'mdi-whatsapp', to: '/whatsapp', append: 'whatsapp-status' },
      { get title () { return t('Emails') }, icon: 'mdi-email-outline', to: '/emails' },
    ],
  },
  {
    key: 'stock',
    get title () { return t('Products and stock') },
    get short () { return t('Products') },
    icon: 'mdi-package-variant',
    items: [
      {
        get title () { return t('Products') },
        icon: 'mdi-cart',
        children: [
          { get title () { return t('Products') }, icon: 'mdi-package-variant', to: '/products' },
          { get title () { return t('Colors & Sizes') }, icon: 'mdi-palette', to: '/colors-sizes' },
          { get title () { return t('Attributes') }, icon: 'mdi-tag-multiple', to: '/attributes' },
          { get title () { return t('Auto-fill Rules') }, icon: 'mdi-auto-fix', to: '/autofill-rules' },
        ],
      },
      {
        get title () { return t('Inventory') },
        icon: 'mdi-folder-network',
        children: [
          { get title () { return t('Inventory') }, icon: 'mdi-warehouse', to: '/inventory' },
          { get title () { return t('Stock Health') }, icon: 'mdi-heart-pulse', to: '/stock-health', append: 'status' },
          { get title () { return t('Inventory Analysis') }, icon: 'mdi-chart-box-outline', to: '/inventory-analysis' },
          { get title () { return t('Batches') }, icon: 'mdi-package-variant-closed', to: '/batches' },
          { get title () { return t('Transfers') }, icon: 'mdi-transfer', to: '/transfers' },
          { get title () { return t('Stock Counts') }, icon: 'mdi-clipboard-list-outline', to: '/stock-counts' },
          { get title () { return t('Stock Movements') }, icon: 'mdi-swap-vertical', to: '/stock-movements' },
          { get title () { return t('Barcodes') }, icon: 'mdi-barcode', to: '/barcodes' },
          { get title () { return t('Assets') }, icon: 'mdi-hammer-wrench', to: '/assets' },
        ],
      },
      {
        get title () { return t('Purchasing') },
        icon: 'mdi-truck',
        children: [
          { get title () { return t('Suppliers') }, icon: 'mdi-truck-delivery', to: '/suppliers' },
          { get title () { return t('Purchase Invoices') }, icon: 'mdi-invoice', to: '/purchase-invoice' },
        ],
      },
    ],
  },
  {
    key: 'money',
    get title () { return t('Money') },
    icon: 'mdi-currency-usd',
    items: [
      { get title () { return t('Expenses') }, icon: 'mdi-currency-usd-off', to: '/expenses' },
      { get title () { return t('Loans') }, icon: 'mdi-hand-coin', to: '/loans' },
      { get title () { return t('Currencies & Payments') }, icon: 'mdi-cash-multiple', to: '/currencies' },
    ],
  },
  {
    key: 'analysis',
    get title () { return t('Analysis') },
    icon: 'mdi-chart-line',
    items: [
      { get title () { return t('Dashboard') }, icon: 'mdi-view-dashboard', to: '/dashboard' },
      { get title () { return t('Sales Dashboard') }, icon: 'mdi-view-dashboard', to: '/sales-dashboard' },
      { get title () { return t('Targets') }, icon: 'mdi-target', to: '/targets' },
      { get title () { return t('Footfall') }, icon: 'mdi-walk', to: '/footfall' },
      { get title () { return t('Busy Hours') }, icon: 'mdi-clock-time-four-outline', to: '/busy-hours' },
      { get title () { return t('Product Revenues') }, icon: 'mdi-package-variant', to: '/product-revenues' },
      { get title () { return t('Conversion') }, icon: 'mdi-percent-outline', to: '/conversion' },
      { get title () { return t('Sales Analytics') }, icon: 'mdi-chart-line', to: '/sales-analytics' },
      { get title () { return t('Sales Analysis') }, icon: 'mdi-chart-box-outline', to: '/sales-analysis' },
      { get title () { return t('Financials') }, icon: 'mdi-currency-usd', to: '/financials' },
      { get title () { return t('Assets Treemap') }, icon: 'mdi-view-gallery', to: '/assets-treemap' },
      { get title () { return t('Warehouses Treemap') }, icon: 'mdi-warehouse', to: '/warehouses-treemap' },
    ],
  },
  {
    key: 'people',
    get title () { return t('People') },
    icon: 'mdi-account-tie-outline',
    items: [
      { get title () { return t('Employees') }, icon: 'mdi-account-group', to: '/employees' },
      { get title () { return t('Attendance') }, icon: 'mdi-account-multiple-check-outline', to: '/attendance' },
      { get title () { return t('Salesperson Performance') }, icon: 'mdi-account-cash', to: '/salesperson-performance' },
    ],
  },
  {
    key: 'branches',
    get title () { return t('Branches') },
    icon: 'mdi-domain',
    items: [
      { get title () { return t('Branches') }, icon: 'mdi-store-outline', to: '/branches' },
      { get title () { return t('Branch Settings') }, icon: 'mdi-store-cog-outline', to: '/branch-settings' },
      { get title () { return t('Default Settings') }, icon: 'mdi-content-copy', to: '/default-branch-settings' },
      { get title () { return t('Commands') }, icon: 'mdi-console-line', to: '/branch-commands', append: 'commands-status' },
      { get title () { return t('Conflicts') }, icon: 'mdi-source-branch-sync', to: '/branch-conflicts', append: 'conflicts-status' },
      { get title () { return t('App Updates') }, icon: 'mdi-update', to: '/app-updates' },
    ],
  },
  {
    key: 'design',
    get title () { return t('Design') },
    icon: 'mdi-palette-outline',
    items: [
      { get title () { return t('Receipt Designer') }, icon: 'mdi-receipt-text', to: '/receipt-designer' },
      { get title () { return t('Barcode Designer') }, icon: 'mdi-barcode', to: '/barcode-designer' },
    ],
  },
  {
    key: 'system',
    get title () { return t('System') },
    icon: 'mdi-cog-outline',
    items: [
      { get title () { return t('Settings') }, icon: 'mdi-cog-outline', to: '/settings' },
      { get title () { return t('Security') }, icon: 'mdi-shield-lock-outline', to: '/security' },
      { get title () { return t('Audit Log') }, icon: 'mdi-history', to: '/audit-log' },
      { get title () { return t('Dev Logs') }, icon: 'mdi-code-tags', to: '/logs' },
      { get title () { return t('Storage') }, icon: 'mdi-google-drive', to: '/storage' },
      { get title () { return t('User Guide') }, icon: 'mdi-book-open-variant', to: '/user-guide' },
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
export const navItems = navSections.map(s => ({ get title () { return s.title }, icon: s.icon, children: navPages(s) }))

export const appendItems = []

// railLabel is the word under a section's icon in the left rail: its short
// name when its title is too long for the rail.
export function railLabel (section) {
  return section.short || section.title
}
