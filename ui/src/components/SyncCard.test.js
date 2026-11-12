import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'

let handler = null
vi.mock('@/composables/useAdminSocket', () => ({
  useAdminSocket: () => ({ status: ref('open'), ensureConnected: () => {}, onMessage: fn => { handler = fn; return () => {} } }),
}))
vi.mock('@/composables/useApi', () => ({ NO_INTERNET_MESSAGE: 'offline' }))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({ branches: ref([{ id: 1, name: 'Test Branch' }]), fetchBranches: async () => {} }),
}))
vi.mock('@/composables/useBranchInvoices', () => ({ useBranchInvoices: () => ({ listRecentBranchInvoices: async () => [] }) }))
vi.mock('@/composables/useNetworkStatus', () => ({ useNetworkStatus: () => ({ online: ref(true) }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAdmin: true }) }))
vi.mock('@/stores/settings', () => ({ useSettingsStore: () => ({ activityDisplayMode: 'appbar', activityMessageSeconds: 30 }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

const { default: SyncCard } = await import('./SyncCard.vue')

let wrapper = null
afterEach(() => { wrapper?.unmount(); wrapper = null })

async function show () {
  window.innerWidth = 1400
  wrapper = mount(SyncCard, { global: { plugins: [vuetify()] } })
  await flushPromises()
  return wrapper
}
const shift = amount => ({ type: 'branch_shift_closed', branch_id: 1, amount, currency_code: 'USD' })

describe('SyncCard', () => {
  it('the reported case: three shift closes show one card, the newest, with +2', async () => {
    await show()
    handler(shift(0))
    handler(shift(-500))
    handler(shift(200))
    await flushPromises()
    const lines = wrapper.findAll('[data-test="activity-line"]')
    expect(lines.length).toBe(1)
    expect(lines[0].find('strong').text()).toBe('Test Branch')
    expect(lines[0].text()).toContain('shift 2.00')
    expect(lines[0].find('.v-progress-circular').exists()).toBe(true)
    expect(wrapper.find('[data-test="activity-more"]').text()).toBe('+2')
    expect(wrapper.find('.activity-pill--success').exists()).toBe(true)
  })

  it('one message: no +N, tinted by its colour', async () => {
    await show()
    handler(shift(-500))
    await flushPromises()
    expect(wrapper.findAll('[data-test="activity-line"]').length).toBe(1)
    expect(wrapper.find('[data-test="activity-more"]').exists()).toBe(false)
    expect(wrapper.find('.activity-pill--error').exists()).toBe(true)
  })

  it('the reported case: a growing conflict count updates in place, one line, no overlap', async () => {
    await show()
    const conflict = { type: 'branch_sync_conflict', branch_id: 1, kind: 'oversold' }
    for (let n = 0; n < 5; n++) {handler(conflict)}
    await flushPromises()
    const before = wrapper.find('[data-test="activity-line"]').element
    handler(conflict)
    await flushPromises()
    const lines = wrapper.findAll('[data-test="activity-line"]')
    expect(lines.length).toBe(1)
    expect(lines[0].element).toBe(before)
    expect(lines[0].text()).toContain('6 to fix')
  })

  it('no activity: the plain synced status, no spinner', async () => {
    await show()
    expect(wrapper.find('[data-test="activity-line"]').exists()).toBe(false)
    expect(wrapper.find('.v-progress-circular').exists()).toBe(false)
    expect(wrapper.text()).toContain('no sync yet')
  })
})
