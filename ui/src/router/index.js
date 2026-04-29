import { createRouter, createWebHistory } from 'vue-router'
import { setupLayouts } from 'virtual:generated-layouts'

const routes = [
  { path: '/', component: () => import('@/pages/index.vue'), meta: { title: 'Home' } },
  { path: '/dashboard', component: () => import('@/pages/dashboard.vue'), meta: { title: 'Dashboard' } },
  { path: '/about', component: () => import('@/pages/About.vue'), meta: { title: 'About' } },
  {
    path: '/login',
    component: () => import('@/pages/Login.vue'),
    meta: { title: 'Login', layout: 'auth' }
  },

  { path: '/products', component: () => import('@/pages/products.vue'), meta: { title: 'Products' } },
  { path: '/inventory', component: () => import('@/pages/inventory.vue'), meta: { title: 'Inventory' } },
  { path: '/storage', component: () => import('@/pages/storage.vue'), meta: { title: 'Storage' } },
  { path: '/warehouses', component: () => import('@/pages/warehouses.vue'), meta: { title: 'Warehouses' } },
  { path: '/transfers', component: () => import('@/pages/transfers.vue'), meta: { title: 'Transfers' } },
  { path: '/barcode-printing', component: () => import('@/pages/barcode-printing.vue'), meta: { title: 'Barcode Printing' } },
  { path: '/colors-sizes', component: () => import('@/pages/Colors_Sizes.vue'), meta: { title: 'Colors & Sizes' } },

  { path: '/sales-invoices', component: () => import('@/pages/sales-invoices.vue'), meta: { title: 'Sales Invoices' } },
  { path: '/purchase-invoice', component: () => import('@/pages/purchase-invoice.vue'), meta: { title: 'Purchase Invoices' } },
  { path: '/expenses', component: () => import('@/pages/expenses.vue'), meta: { title: 'Expenses' } },
  { path: '/discounts', component: () => import('@/pages/discounts.vue'), meta: { title: 'Discounts' } },

  { path: '/clients', component: () => import('@/pages/clients.vue'), meta: { title: 'Clients' } },
  { path: '/clients/:id', component: () => import('@/pages/client-page.vue'), meta: { title: 'Client' } },

  { path: '/employees', component: () => import('@/pages/employees.vue'), meta: { title: 'Employees' } },
  { path: '/employees/:id', component: () => import('@/pages/employee_page.vue'), meta: { title: 'Employee' } },
  { path: '/attendance', component: () => import('@/pages/Attendance.vue'), meta: { title: 'Attendance' } },
  { path: '/calendar', component: () => import('@/pages/calendar.vue'), meta: { title: 'Calendar' } },

  { path: '/stats', component: () => import('@/pages/stats.vue'), meta: { title: 'Statistics' } },
  { path: '/users', component: () => import('@/pages/users.vue'), meta: { title: 'Users' } },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: setupLayouts(routes),
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
  localStorage.removeItem('vuetify:dynamic-reload')
})

export default router