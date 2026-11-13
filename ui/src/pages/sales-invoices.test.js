import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'

// happy-dom has no visualViewport; Vuetify's dialog overlay reads it.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

// Every request the page makes goes through authFetch; record the paths.
const requested = []
function bodyFor (path) {
  if (path.startsWith('/branch_invoices')) {return { branch_invoices: [], total: 0 }}
  if (path.startsWith('/sales_invoices')) {return { sales_invoices: [], total: 0 }}
  if (path.startsWith('/barcodes')) {return { variants: [] }}
  return []
}
vi.mock('@/composables/useApi', () => ({
  NO_INTERNET_MESSAGE: 'offline',
  authFetch: async url => {
    const path = String(url).replace(/^.*?\/api|^https?:\/\/[^/]+/, '')
    requested.push(path)
    return { ok: true, status: 200, json: async () => bodyFor(path) }
  },
}))
vi.mock('@/config', () => ({ API_BASE: 'http://test/api' }))
vi.mock('@/composables/useAdminSocket', () => ({
  useAdminSocket: () => ({ status: ref('open'), ensureConnected: () => {}, onMessage: () => () => {} }),
}))
vi.mock('@/composables/usePageAccess', () => ({ usePageAccess: () => ({ canEdit: ref(true) }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAdmin: true, can: () => true }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push: vi.fn(), replace: vi.fn() }) }))


// The New Sales Invoice dialog's lists, which the page used to fetch on open.
const DIALOG_PATHS = ['/cashboxes/', '/cashbox_accounts/', '/shifts', '/inventories/', '/clients/', '/price_lists', '/barcodes']
const hits = prefix => requested.filter(p => p.startsWith(prefix)).length

let wrapper = null
let pinia = null
beforeEach(() => {
  requested.length = 0
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => { wrapper?.unmount(); wrapper = null })

// The lists are cached per module (useApiResource), so each test imports a
// fresh copy of the page and its composables, as a fresh page load would.
async function open () {
  vi.resetModules()
  const { default: SalesInvoices } = await import('./sales-invoices.vue')
  window.innerWidth = 1400
  wrapper = mount(SalesInvoices, { global: { plugins: [vuetify(), pinia] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

function tab (label) {
  return wrapper.findAll('.v-tab').find(t => t.text().includes(label))
}
function button (label) {
  return wrapper.findAll('button').find(b => b.text().includes(label))
}

describe('Invoices page first load', () => {
  it('the reported case: opening it loads the retail list and branches, nothing for the hidden dialog or tab', async () => {
    await open()
    expect(hits('/branch_invoices')).toBe(1)
    expect(hits('/branches')).toBe(1)
    for (const p of DIALOG_PATHS) {expect(hits(p), p).toBe(0)}
    expect(hits('/sales_invoices')).toBe(0)
  })

  it('the wholesale list loads when its tab is first opened, and only once', async () => {
    await open()
    expect(hits('/sales_invoices')).toBe(0)
    await tab('Wholesale').trigger('click')
    await flushPromises()
    expect(hits('/sales_invoices')).toBe(1)
    await tab('Retail').trigger('click')
    await flushPromises()
    await tab('Wholesale').trigger('click')
    await flushPromises()
    expect(hits('/sales_invoices')).toBe(1)
  })

  it('Add Invoice loads every list the dialog picks from, once, however often it opens', async () => {
    await open()
    await tab('Wholesale').trigger('click')
    await flushPromises()
    for (let i = 0; i < 3; i++) {
      await button('Add Invoice').trigger('click')
      await flushPromises()
    }
    for (const p of DIALOG_PATHS) {expect(hits(p), p).toBe(1)}
    expect(document.body.textContent).toContain('New Sales Invoice')
  })
})
