import { setupLayouts } from 'virtual:generated-layouts'
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/', component: () => import('@/pages/home.vue'), meta: { title: 'Home' } },
  { path: '/dashboard', component: () => import('@/pages/dashboard.vue'), meta: { title: 'Dashboard', section: 'analytics' } },
  { path: '/daily-income', component: () => import('@/pages/daily-income.vue'), meta: { title: 'Daily Income', section: 'analytics' } },
  { path: '/financials', component: () => import('@/pages/financials.vue'), meta: { title: 'Financials', section: 'analytics' } },
  { path: '/targets', component: () => import('@/pages/targets.vue'), meta: { title: 'Targets', section: 'analytics' } },
  { path: '/footfall', component: () => import('@/pages/footfall.vue'), meta: { title: 'Footfall', section: 'analytics' } },
  { path: '/product-revenues', component: () => import('@/pages/product-revenues.vue'), meta: { title: 'Product Revenues', section: 'analytics' } },
  { path: '/conversion', component: () => import('@/pages/conversion.vue'), meta: { title: 'Conversion', section: 'analytics' } },
  { path: '/alerts', component: () => import('@/pages/alerts.vue'), meta: { title: 'Alerts', section: 'inventory' } },
  { path: '/stock-health', component: () => import('@/pages/stock-health.vue'), meta: { title: 'Stock Health', section: 'inventory' } },
  { path: '/inventory-analysis', component: () => import('@/pages/inventory-analysis.vue'), meta: { title: 'Inventory Analysis', section: 'inventory' } },
  { path: '/sales-analytics', component: () => import('@/pages/sales-analytics.vue'), meta: { title: 'Sales Analytics', section: 'analytics' } },
  { path: '/sales-analysis', component: () => import('@/pages/sales-analysis.vue'), meta: { title: 'Sales Analysis', section: 'analytics' } },
  { path: '/sales-dashboard', component: () => import('@/pages/sales-dashboard.vue'), meta: { title: 'Sales Dashboard', section: 'analytics' } },
  { path: '/salesperson-performance', component: () => import('@/pages/salesperson-performance.vue'), meta: { title: 'Salesperson Performance', section: 'analytics' } },
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

  { path: '/products', component: () => import('@/pages/products.vue'), meta: { title: 'Products', section: 'products' } },
  { path: '/inventory', component: () => import('@/pages/inventory.vue'), meta: { title: 'Inventory', section: 'inventory' } },
  { path: '/storage', component: () => import('@/pages/storage.vue'), meta: { title: 'Storage', section: 'system' } },
  { path: '/assets-treemap', component: () => import('@/pages/assets-treemap.vue'), meta: { title: 'Assets Treemap', section: 'accounting' } },
  { path: '/warehouses-treemap', component: () => import('@/pages/warehouses-treemap.vue'), meta: { title: 'Warehouses Treemap', section: 'accounting' } },
  { path: '/branches', component: () => import('@/pages/branches.vue'), meta: { title: 'Branches', section: 'branches' } },
  { path: '/branch-settings', component: () => import('@/pages/branch-settings.vue'), meta: { title: 'Branch Settings', section: 'branches' } },
  { path: '/default-branch-settings', component: () => import('@/pages/default-branch-settings.vue'), meta: { title: 'Default Settings', section: 'branches' } },
  { path: '/branch-invoices', redirect: '/invoices' },
  { path: '/transfers', component: () => import('@/pages/transfers.vue'), meta: { title: 'Transfers', section: 'inventory' } },
  { path: '/stock-movements', component: () => import('@/pages/stock-movements.vue'), meta: { title: 'Stock Movements', section: 'inventory' } },
  { path: '/stock-counts', component: () => import('@/pages/stock-counts.vue'), meta: { title: 'Stock Counts', section: 'inventory' } },
  { path: '/audit-log', component: () => import('@/pages/audit-log.vue'), meta: { title: 'Audit Log', section: 'system' } },
  { path: '/barcode-printing', component: () => import('@/pages/barcode-printing.vue'), meta: { title: 'Barcode Printing', section: 'products' } },
  { path: '/colors-sizes', component: () => import('@/pages/Colors_Sizes.vue'), meta: { title: 'Colors & Sizes', section: 'products' } },
  { path: '/attributes', component: () => import('@/pages/attributes.vue'), meta: { title: 'Attributes', section: 'products' } },
  { path: '/barcodes', component: () => import('@/pages/barcodes.vue'), meta: { title: 'Barcodes', section: 'products' } },
  { path: '/assets', component: () => import('@/pages/assets.vue'), meta: { title: 'Assets', section: 'accounting' } },

  { path: '/invoices', component: () => import('@/pages/sales-invoices.vue'), meta: { title: 'Invoices', section: 'sales' } },
  { path: '/sales-invoices', redirect: '/invoices' },
  { path: '/purchase-invoice', component: () => import('@/pages/purchase-invoice.vue'), meta: { title: 'Purchase Invoices', section: 'inventory' } },
  { path: '/batches', component: () => import('@/pages/batches.vue'), meta: { title: 'Batches', section: 'inventory' } },
  { path: '/expenses', component: () => import('@/pages/expenses.vue'), meta: { title: 'Expenses', section: 'accounting' } },
  { path: '/loans', component: () => import('@/pages/loans.vue'), meta: { title: 'Loans', section: 'accounting' } },
  { path: '/discounts', redirect: '/discount-lists' },
  { path: '/pricing', component: () => import('@/pages/pricing.vue'), meta: { title: 'Coupons', section: 'pricing' } },
  { path: '/price-lists', component: () => import('@/pages/price-lists.vue'), meta: { title: 'Price Lists', section: 'pricing' } },
  { path: '/discount-lists', component: () => import('@/pages/discount-lists.vue'), meta: { title: 'Discount Lists', section: 'pricing' } },
  { path: '/invoice-types', redirect: '/invoices' },
  { path: '/currencies', component: () => import('@/pages/currencies.vue'), meta: { title: 'Currencies & Payment Methods', section: 'accounting' } },
  { path: '/coupons', redirect: '/pricing' },
  { path: '/shifts', component: () => import('@/pages/shifts.vue'), meta: { title: 'Shifts', section: 'hr' } },

  { path: '/clients', component: () => import('@/pages/clients.vue'), meta: { title: 'Clients', section: 'clients' } },
  { path: '/clients/:id', component: () => import('@/pages/client-page.vue'), meta: { title: 'Client', section: 'clients' } },
  { path: '/loyalty', component: () => import('@/pages/loyalty.vue'), meta: { title: 'Loyalty', section: 'clients' } },
  { path: '/suppliers', component: () => import('@/pages/suppliers.vue'), meta: { title: 'Suppliers', section: 'inventory' } },

  { path: '/employees', component: () => import('@/pages/employees.vue'), meta: { title: 'Employees', section: 'hr' } },
  { path: '/employees/:id', component: () => import('@/pages/employee_page.vue'), meta: { title: 'Employee', section: 'hr' } },
  { path: '/attendance', component: () => import('@/pages/Attendance.vue'), meta: { title: 'Attendance', section: 'hr' } },
  { path: '/calendar', component: () => import('@/pages/calendar.vue'), meta: { title: 'Calendar', section: 'calendar' } },

  { path: '/users', component: () => import('@/pages/users.vue'), meta: { title: 'Users', adminOnly: true } },
  { path: '/settings', component: () => import('@/pages/settings.vue'), meta: { title: 'Settings' } },
  { path: '/logs', component: () => import('@/pages/logs.vue'), meta: { title: 'Dev Logs', section: 'system' } },
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
  // (Home, About, Settings, ...) are open to every signed-in user.
  const { section, adminOnly } = to.meta
  if (authStore.isAuthenticated && ((adminOnly && !authStore.isAdmin) || (section && !authStore.canView(section)))) {
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
