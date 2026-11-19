import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { navPages, navSections } from '@/data/navItems'
import { vuetify } from '@/test/vuetify'
import NavigationDrawer from './NavigationDrawer.vue'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }
globalThis.ResizeObserver ??= class { observe () {} unobserve () {} disconnect () {} }

const blocked = ref(new Set())
const security = ref(0)
const commands = ref('healthy')
vi.mock('@/composables/useAllowedNav', () => ({
  useAllowedNav: () => ({
    allowed: () => true,
    allowedSections: computed(() => navSections
      .map(s => ({ ...s, items: s.items.map(i => (i.children ? { ...i, children: i.children.filter(c => !blocked.value.has(c.to)) } : i)).filter(i => (i.children ? i.children.length > 0 : !blocked.value.has(i.to))) }))
      .filter(s => s.items.length > 0)),
  }),
}))
vi.mock('@/composables/useLowStockAlerts', () => ({ useLowStockAlerts: () => ({ fetchSummary: async () => ({ out_of_stock_count: 0, low_stock_count: 0 }) }) }))
vi.mock('@/composables/useStockHealth', () => ({ useStockHealth: () => ({ fetchStockHealth: async () => ({ status: 'healthy' }) }) }))
vi.mock('@/composables/useBranchCommands', () => ({ useBranchCommands: () => ({ commandsStatus: commands, refreshCommandSummary: async () => {} }) }))
vi.mock('@/composables/useBranchSyncConflicts', () => ({ useBranchSyncConflicts: () => ({ conflictsStatus: ref('healthy'), refreshConflictSummary: async () => {} }) }))
vi.mock('@/composables/useWhatsAppFailures', () => ({ useWhatsAppFailures: () => ({ whatsAppFailureStatus: ref(null), refreshWhatsAppFailures: async () => {} }) }))
vi.mock('@/composables/useSecurityAlerts', () => ({ useSecurityAlerts: () => ({ openSecurityAlerts: security, refreshOpenSecurityAlerts: async () => {} }) }))
vi.mock('@/composables/useAdminSocket', () => ({ useAdminSocket: () => ({ onMessage: () => () => {}, ensureConnected: () => {} }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ canView: () => true }) }))

let wrapper
let router
beforeEach(() => {
  blocked.value = new Set()
  security.value = 0
  commands.value = 'healthy'
  const pages = navSections.flatMap(s => navPages(s)).map(p => ({ path: p.to, component: { template: '<div />' } }))
  router = createRouter({ history: createMemoryHistory(), routes: [...pages, { path: '/invoices/:id', component: { template: '<div />' } }] })
})
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = '' })

async function open (path = '/', props = {}) {
  await router.push(path)
  wrapper = mount({ components: { NavigationDrawer }, template: '<v-app><NavigationDrawer v-bind="$attrs" /></v-app>' }, {
    props: { modelValue: true, ...props },
    global: { plugins: [vuetify(), router] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const pageLinks = () => wrapper.findAll('[data-test^="nav-/"]').map(l => l.attributes('data-test').slice(4))

describe('NavigationDrawer', () => {
  it('the reported case: a rail of sections, and only the open section\'s pages beside it', async () => {
    await open('/receipt-designer')
    expect(navSections.every(s => find(`rail-${s.key}`).exists())).toBe(true)
    expect(find('nav-section-title').text()).toBe('Design')
    expect(pageLinks()).toEqual(['/receipt-designer', '/barcode-designer'])
    expect(find('rail-design').attributes('aria-current')).toBe('true')
  })

  it('every rail icon says its section in words, short enough for the rail', async () => {
    await open('/')
    const labels = navSections.map(s => find(`rail-label-${s.key}`).text())
    expect(labels).toEqual(['Overview', 'Sales', 'Customers', 'Products', 'Money', 'Analysis', 'People', 'Branches', 'Design', 'System'])
    for (const label of labels) {
      expect(label.length, label).toBeLessThanOrEqual(10)
      expect(label).not.toMatch(/[—–·•]/)
    }
    // The full name stays for screen readers.
    expect(find('rail-stock').attributes('aria-label')).toBe('Products and stock')
  })

  it('a click on a section shows its pages, with their headings', async () => {
    await open('/')
    await find('rail-stock').trigger('click')
    expect(find('nav-section-title').text()).toBe('Products and stock')
    expect(pageLinks()).toEqual(navPages(navSections.find(s => s.key === 'stock')).map(p => p.to))
    expect(wrapper.findAll('.nav-heading').map(h => h.text())).toEqual(['Products', 'Inventory', 'Purchasing'])
    expect(find('rail-stock').attributes('aria-current')).toBe('true')
    expect(find('rail-overview').attributes('aria-current')).toBeUndefined()
  })

  it('the open section follows the page, deeper pages too', async () => {
    await open('/')
    await router.push('/invoices/42')
    await flushPromises()
    expect(find('nav-section-title').text()).toBe('Sales')
  })

  it('collapsed, only the rail shows; a section click opens the drawer', async () => {
    await open('/', { rail: true })
    expect(find('nav-pages').exists()).toBe(false)
    expect(find('nav-rail').exists()).toBe(true)
    await find('rail-money').trigger('click')
    const drawer = wrapper.findComponent(NavigationDrawer)
    expect(drawer.emitted('update:rail')).toEqual([[false]])
  })

  it('pages a role can\'t open are gone, and a section left empty too', async () => {
    blocked.value = new Set(['/receipt-designer', '/barcode-designer', '/expenses'])
    await open('/')
    expect(find('rail-design').exists()).toBe(false)
    await find('rail-money').trigger('click')
    expect(pageLinks()).toEqual(['/loans', '/currencies'])
  })

  it('a section whose page wants a look shows a dot on its icon', async () => {
    commands.value = 'warning'
    security.value = 2
    await open('/')
    expect(find('rail-dot-branches').exists()).toBe(true)
    expect(find('rail-dot-system').exists()).toBe(true)
    expect(find('rail-dot-design').exists()).toBe(false)
  })
})
