import { setupLayouts } from 'virtual:generated-layouts'
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/', component: () => import('@/pages/home.vue'), meta: { title: 'Home' } },
  { path: '/dashboard', component: () => import('@/pages/dashboard.vue'), meta: { title: 'Dashboard', section: 'analytics', page: 'dashboard' } },
  { path: '/daily-income', component: () => import('@/pages/daily-income.vue'), meta: { title: 'Daily Income', section: 'analytics', page: 'daily-income' } },
  { path: String.raw`/daily-income/:day(\d{4}-\d{2}-\d{2})`, component: () => import('@/pages/daily-income-day.vue'), meta: { title: 'Day Details', section: 'analytics', page: 'daily-income' } },
  { path: '/financials', component: () => import('@/pages/financials.vue'), meta: { title: 'Financials', section: 'analytics', page: 'financials' } },
  { path: '/targets', component: () => import('@/pages/targets.vue'), meta: { title: 'Targets', section: 'analytics', page: 'targets' } },
  { path: '/footfall', component: () => import('@/pages/footfall.vue'), meta: { title: 'Footfall', section: 'analytics', page: 'footfall' } },
  { path: '/product-revenues', component: () => import('@/pages/product-revenues.vue'), meta: { title: 'Product Revenues', section: 'analytics', page: 'product-revenues' } },
  { path: '/conversion', component: () => import('@/pages/conversion.vue'), meta: { title: 'Conversion', section: 'analytics', page: 'conversion' } },
  { path: '/alerts', component: () => import('@/pages/alerts.vue'), meta: { title: 'Alerts', section: 'inventory', page: 'alerts' } },
  { path: '/stock-health', component: () => import('@/pages/stock-health.vue'), meta: { title: 'Stock Health', section: 'inventory', page: 'stock-health' } },
  { path: '/inventory-analysis', component: () => import('@/pages/inventory-analysis.vue'), meta: { title: 'Inventory Analysis', section: 'inventory', page: 'inventory-analysis' } },
  { path: '/sales-analytics', component: () => import('@/pages/sales-analytics.vue'), meta: { title: 'Sales Analytics', section: 'analytics', page: 'sales-analytics' } },
  { path: '/sales-analysis', component: () => import('@/pages/sales-analysis.vue'), meta: { title: 'Sales Analysis', section: 'analytics', page: 'sales-analysis' } },
  { path: '/sales-dashboard', component: () => import('@/pages/sales-dashboard.vue'), meta: { title: 'Sales Dashboard', section: 'analytics', page: 'sales-dashboard' } },
  { path: '/salesperson-performance', component: () => import('@/pages/salesperson-performance.vue'), meta: { title: 'Salesperson Performance', section: 'analytics', page: 'salesperson-performance' } },
  { path: '/about', component: () => import('@/pages/About.vue'), meta: { title: 'About' } },
  {
    path: '/login',
    component: () => import('@/pages/Login.vue'),
    meta: { title: 'Login', layout: 'login', public: true },
  },
  {
    path: '/review',
    component: () => import('@/pages/review.vue'),
    meta: { title: 'Feedback', layout: 'login', public: true },
  },

  { path: '/products', component: () => import('@/pages/products.vue'), meta: { title: 'Products', section: 'products', page: 'products' } },
  { path: '/inventory', component: () => import('@/pages/inventory.vue'), meta: { title: 'Inventory', section: 'inventory', page: 'inventory' } },
  { path: '/storage', component: () => import('@/pages/storage.vue'), meta: { title: 'Storage', section: 'system', page: 'storage' } },
  { path: '/assets-treemap', component: () => import('@/pages/assets-treemap.vue'), meta: { title: 'Assets Treemap', section: 'accounting', page: 'assets-treemap' } },
  { path: '/warehouses-treemap', component: () => import('@/pages/warehouses-treemap.vue'), meta: { title: 'Warehouses Treemap', section: 'accounting', page: 'warehouses-treemap' } },
  { path: '/branches', component: () => import('@/pages/branches.vue'), meta: { title: 'Branches', section: 'branches', page: 'branches' } },
  { path: '/branch-settings', component: () => import('@/pages/branch-settings.vue'), meta: { title: 'Branch Settings', section: 'branches', page: 'branch-settings' } },
  { path: '/default-branch-settings', component: () => import('@/pages/default-branch-settings.vue'), meta: { title: 'Default Settings', section: 'branches', page: 'default-branch-settings' } },
  { path: '/app-updates', component: () => import('@/pages/app-updates.vue'), meta: { title: 'App Updates', section: 'updates', page: 'app-updates' } },
  { path: '/security', component: () => import('@/pages/security.vue'), meta: { title: 'Security', section: 'security', page: 'security' } },
  { path: '/branch-commands', component: () => import('@/pages/branch-commands.vue'), meta: { title: 'Commands', section: 'branches', page: 'branch-commands' } },
  { path: '/branch-conflicts', component: () => import('@/pages/branch-conflicts.vue'), meta: { title: 'Conflicts', section: 'branches', page: 'branch-conflicts' } },
  { path: '/branch-invoices', redirect: '/invoices' },
  { path: '/transfers', component: () => import('@/pages/transfers.vue'), meta: { title: 'Transfers', section: 'inventory', page: 'transfers' } },
  { path: '/stock-movements', component: () => import('@/pages/stock-movements.vue'), meta: { title: 'Stock Movements', section: 'inventory', page: 'stock-movements' } },
  { path: '/stock-counts', component: () => import('@/pages/stock-counts.vue'), meta: { title: 'Stock Counts', section: 'inventory', page: 'stock-counts' } },
  { path: '/audit-log', component: () => import('@/pages/audit-log.vue'), meta: { title: 'Audit Log', section: 'system', page: 'audit-log' } },
  { path: '/barcode-printing', component: () => import('@/pages/barcode-printing.vue'), meta: { title: 'Barcode Printing', section: 'products', page: 'barcode-printing' } },
  { path: '/colors-sizes', component: () => import('@/pages/Colors_Sizes.vue'), meta: { title: 'Colors & Sizes', section: 'products', page: 'colors-sizes' } },
  { path: '/attributes', component: () => import('@/pages/attributes.vue'), meta: { title: 'Attributes', section: 'products', page: 'attributes' } },
  { path: '/autofill-rules', component: () => import('@/pages/autofill-rules.vue'), meta: { title: 'Auto-fill Rules', section: 'products', page: 'autofill-rules' } },
  { path: '/barcodes', component: () => import('@/pages/barcodes.vue'), meta: { title: 'Barcodes', section: 'products', page: 'barcodes' } },
  { path: '/assets', component: () => import('@/pages/assets.vue'), meta: { title: 'Assets', section: 'accounting', page: 'assets' } },

  { path: '/invoices', component: () => import('@/pages/sales-invoices.vue'), meta: { title: 'Invoices', section: 'sales', page: 'invoices' } },
  { path: '/sales-invoices', redirect: '/invoices' },
  { path: '/purchase-invoice', component: () => import('@/pages/purchase-invoice.vue'), meta: { title: 'Purchase Invoices', section: 'inventory', page: 'purchase-invoice' } },
  { path: '/batches', component: () => import('@/pages/batches.vue'), meta: { title: 'Batches', section: 'inventory', page: 'batches' } },
  { path: '/expenses', component: () => import('@/pages/expenses.vue'), meta: { title: 'Expenses', section: 'accounting', page: 'expenses' } },
  { path: '/loans', component: () => import('@/pages/loans.vue'), meta: { title: 'Loans', section: 'accounting', page: 'loans' } },
  { path: '/discounts', redirect: '/discount-lists' },
  { path: '/pricing', component: () => import('@/pages/pricing.vue'), meta: { title: 'Coupons', section: 'pricing', page: 'pricing' } },
  { path: '/price-lists', component: () => import('@/pages/price-lists.vue'), meta: { title: 'Price Lists', section: 'pricing', page: 'price-lists' } },
  { path: '/discount-lists', component: () => import('@/pages/discount-lists.vue'), meta: { title: 'Discount Lists', section: 'pricing', page: 'discount-lists' } },
  { path: '/invoice-types', redirect: '/invoices' },
  { path: '/currencies', component: () => import('@/pages/currencies.vue'), meta: { title: 'Currencies & Payment Methods', section: 'accounting', page: 'currencies' } },
  { path: '/coupons', redirect: '/pricing' },
  { path: '/shifts', component: () => import('@/pages/shifts.vue'), meta: { title: 'Shifts', section: 'hr', page: 'shifts' } },

  { path: '/clients', component: () => import('@/pages/clients.vue'), meta: { title: 'Clients', section: 'clients', page: 'clients' } },
  { path: '/clients/:id', component: () => import('@/pages/client-page.vue'), meta: { title: 'Client', section: 'clients', page: 'clients' } },
  { path: '/loyalty', component: () => import('@/pages/loyalty.vue'), meta: { title: 'Loyalty', section: 'clients', page: 'loyalty' } },
  { path: '/suppliers', component: () => import('@/pages/suppliers.vue'), meta: { title: 'Suppliers', section: 'inventory', page: 'suppliers' } },

  { path: '/employees', component: () => import('@/pages/employees.vue'), meta: { title: 'Employees', section: 'hr', page: 'employees' } },
  { path: '/employees/:id', component: () => import('@/pages/employee_page.vue'), meta: { title: 'Employee', section: 'hr', page: 'employees' } },
  { path: '/attendance', component: () => import('@/pages/Attendance.vue'), meta: { title: 'Attendance', section: 'hr', page: 'attendance' } },
  { path: '/calendar', component: () => import('@/pages/calendar.vue'), meta: { title: 'Calendar', section: 'calendar', page: 'calendar' } },

  { path: '/users', component: () => import('@/pages/users.vue'), meta: { title: 'Users', adminOnly: true } },
  { path: '/settings', component: () => import('@/pages/settings.vue'), meta: { title: 'Settings' } },
  { path: '/whatsapp', component: () => import('@/pages/whatsapp.vue'), meta: { title: 'WhatsApp', section: 'system', page: 'whatsapp' } },
  { path: '/logs', component: () => import('@/pages/logs.vue'), meta: { title: 'Dev Logs', section: 'system', page: 'logs' } },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: setupLayouts(routes),
})

function updateDocumentTitle (route) {
  const pageTitle = route.meta.title
  document.title = pageTitle ? `Kashi Cloud - ${pageTitle}` : 'Kashi Cloud'
}

router.beforeEach(async to => {
  const authStore = useAuthStore()
  if (to.path !== '/login' && !to.meta.public && !authStore.isAuthenticated) {
    return '/login'
  }
  if (to.path === '/login' && authStore.isAuthenticated) {
    return '/'
  }

  // Sessions from before roles existed have no cached access yet; without it
  // every gated page would look forbidden until the layout's refresh lands.
  if (authStore.isAuthenticated && !authStore.access) {
    await authStore.refreshAccess()
  }

  // Hides pages the role can't see. The API refuses the data regardless; this
  // just avoids landing on a page full of 403 errors. Pages with no `section`
  // (Home, About, Settings, ...) are open to every signed-in user. `page` keys
  // match the page list in api/access.go.
  const { adminOnly } = to.meta
  if (authStore.isAuthenticated && ((adminOnly && !authStore.isAdmin) || !authStore.canViewRoute(to.meta))) {
    return '/'
  }
})

router.afterEach(to => {
  updateDocumentTitle(to)
})

router.onError((err, to) => {
  if (err?.message?.includes?.('Failed to fetch dynamically imported module')) {
    if (localStorage.getItem('vuetify:dynamic-reload')) {
      console.error('Dynamic import error, reloading page did not fix it', err)
    } else {
      console.log('Reloading page to fix dynamic import error')
      localStorage.setItem('vuetify:dynamic-reload', 'true')
      location.assign(to.fullPath)
    }
  } else {
    console.error(err)
  }
})

router.isReady().then(() => {
  updateDocumentTitle(router.currentRoute.value)
  localStorage.removeItem('vuetify:dynamic-reload')
})

export default router
