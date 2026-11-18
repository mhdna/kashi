import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

// The Referrals page against a fake API: it shows the rules and the list,
// refuses a bad form before sending, and saves the rules in cents.
const calls = []
let program
let referrals
vi.mock('@/composables/useApi', () => ({
  authFetch: async (url, options = {}) => {
    const path = String(url).replace('http://test/api', '')
    calls.push({ path, method: options.method ?? 'GET', body: options.body ? JSON.parse(options.body) : null })
    if (path === '/referral_program' && options.method === 'PUT') {
      program = { ...program, ...JSON.parse(options.body) }
      return { ok: true, status: 200, json: async () => ({ referral_program: program }) }
    }
    if (path === '/referral_program') {return { ok: true, status: 200, json: async () => ({ referral_program: program }) }}
    if (path.startsWith('/referrals')) {return { ok: true, status: 200, json: async () => ({ referrals }) }}
    if (path.startsWith('/branches')) {return { ok: true, status: 200, json: async () => ({ branches: [{ id: 3, name: 'Centro' }, { id: 7, name: 'Hara' }] }) }}
    return { ok: false, status: 404, json: async () => ({}) }
  },
}))
vi.mock('@/config', () => ({ API_BASE: 'http://test/api' }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success: vi.fn(), error: vi.fn() }) }))

let wrapper = null
beforeEach(() => {
  calls.length = 0
  program = { enabled: false, reward_percent: 15, max_reward_cents: 0, min_friend_spend: 0, unlock_after_days: 0, expires_after_days: 0, cancel_on_return: true, max_rewards_per_month: 0 }
  referrals = [
    { id: 1, code: 'K7M4QH', status: 'available', percent: 15, max_cents: 0, owner_name: 'Rami', owner_phone: '+96171000001', friend_name: 'Sara', friend_phone: '+96171000002', branch_name: 'Hara', sale_code: 'SA/1', sale_total: 4000, created_at: '2026-11-17T10:00:00Z' },
    { id: 2, code: 'K7M4QH', status: 'refused', refuse_reason: 'The customer had bought before', owner_name: 'Rami', friend_name: 'Ali', branch_name: 'Hara', sale_code: 'SA/2', sale_total: 2000, created_at: '2026-11-17T11:00:00Z' },
  ]
})
afterEach(() => { wrapper?.unmount(); wrapper = null })

// Vuetify puts data-* either on the field's root or on its input.
function input (w, name) {
  const el = w.find(`[data-test="${name}"]`)
  return el.element.tagName === 'INPUT' ? el : el.find('input')
}

async function open () {
  const { default: Page } = await import('./referrals.vue')
  wrapper = mount(Page, { global: { plugins: [vuetify()] } })
  await flushPromises()
  return wrapper
}

describe('Referrals page', () => {
  it('shows the rules and every referral with its reason', async () => {
    const w = await open()
    expect(w.find('[data-test="summary"]').text()).toMatch(/off/)
    const list = w.find('[data-test="list"]').text()
    expect(list).toContain('Rami')
    expect(list).toContain('Sara')
    expect(list).toContain('Ready to use')
    expect(list).toContain('15% off')
    expect(list).toContain('The customer had bought before')
    expect(list).not.toMatch(/[–—·]/)
  })

  it('refuses a bad reward without sending it', async () => {
    const w = await open()
    await input(w, 'percent').setValue('0')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.find('[data-test="form-error"]').text()).toMatch(/1% to 100%/)
    expect(calls.some(c => c.method === 'PUT')).toBe(false)
  })

  it('saves the rules', async () => {
    const w = await open()
    await input(w, 'branch-7').setValue(true)
    await input(w, 'percent').setValue('20')
    await w.find('form').trigger('submit')
    await flushPromises()
    const put = calls.find(c => c.method === 'PUT')
    expect(put.body).toMatchObject({ enabled: true, branch_ids: [7], reward_percent: 20, max_reward_cents: 0, cancel_on_return: true })
    expect(w.find('[data-test="summary"]').text()).toBe('On at Hara. A new customer\'s first purchase with a client\'s code gives that client 20% off one later visit.')
  })

  it('the reported case: a switch for each branch, and turning the last one off turns referrals off', async () => {
    program = { ...program, enabled: true, branch_ids: [3] }
    const w = await open()
    expect(w.find('[data-test="branches"]').text()).toContain('Centro')
    expect(w.find('[data-test="branches"]').text()).toContain('Hara')
    expect(input(w, 'branch-3').element.checked).toBe(true)
    expect(input(w, 'branch-7').element.checked).toBe(false)
    expect(w.find('[data-test="summary"]').text()).toMatch(/^On at Centro\./)
    await input(w, 'branch-3').setValue(false)
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(calls.find(c => c.method === 'PUT').body).toMatchObject({ enabled: false, branch_ids: [] })
    expect(w.find('[data-test="summary"]').text()).toMatch(/off at every branch/)
  })

  it('filters by status', async () => {
    const w = await open()
    const buttons = w.findAll('.v-btn-toggle .v-btn')
    await buttons.find(b => b.text() === 'No reward').trigger('click')
    await flushPromises()
    expect(calls.at(-1).path).toBe('/referrals?status=refused')
  })
})
