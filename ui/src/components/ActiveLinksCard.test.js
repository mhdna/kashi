import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

const authFetch = vi.fn()
vi.mock('@/composables/useApi', () => ({ authFetch: (...args) => authFetch(...args) }))
vi.mock('@/config', () => ({ API_BASE: 'https://cloud.grandbz.com/api' }))

const { default: ActiveLinksCard } = await import('./ActiveLinksCard.vue')

let wrapper = null
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.clearAllMocks()
})

const inMinutes = minutes => new Date(Date.now() + minutes * 60_000 + 30_000).toISOString()
function answer (links) {
  authFetch.mockResolvedValue({ ok: true, json: async () => ({ links }) })
}
async function show () {
  wrapper = mount(ActiveLinksCard, { global: { plugins: [vuetify()] } })
  await flushPromises()
  return wrapper
}

describe('ActiveLinksCard', () => {
  it('the reported case: lists the unknown item link a till is showing, with its full URL', async () => {
    answer([
      { kind: 'unknown', path: '/u/9bZAgBdrUOh31zcm', branch_name: 'Centro', ref_id: 3, units: 3, done: 1, expires_at: inMinutes(12) },
      { kind: 'expense', path: '/e/abc', branch_name: 'Hara', ref_id: 9, label: 'Electricity', amount: 4500, currency_code: 'USD', expires_at: inMinutes(50) },
    ])
    await show()
    expect(authFetch).toHaveBeenCalledWith('https://cloud.grandbz.com/api/home/active_links')
    const rows = wrapper.findAll('[data-test="active-link"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('Unknown items')
    expect(rows[0].text()).toContain('Centro, 1 of 3 pictured')
    expect(rows[0].text()).toContain('Expires in 12 min')
    expect(rows[0].find('[data-test="active-link-url"]').text()).toBe('https://cloud.grandbz.com/api/u/9bZAgBdrUOh31zcm')
    expect(rows[1].text()).toContain('Expense photo')
    expect(rows[1].text()).toContain('Hara, Electricity, $45')
    expect(wrapper.find('[data-test="active-links-count"]').text()).toBe('2 open')
  })

  it('Copy puts the link on the clipboard and says Copied', async () => {
    const writeText = vi.fn(async () => {})
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    answer([{ kind: 'shift', path: '/s/tok', branch_name: 'Centro', ref_id: 206, expires_at: inMinutes(5) }])
    await show()
    await wrapper.find('[data-test="active-link-copy"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('https://cloud.grandbz.com/api/s/tok')
    expect(wrapper.find('[data-test="active-link-copy"]').text()).toBe('Copied')
  })

  it('shows nothing when no link is open, or when the server says no', async () => {
    answer([])
    await show()
    expect(wrapper.find('.active-links').exists()).toBe(false)
    wrapper.unmount()
    authFetch.mockResolvedValue({ ok: false, json: async () => ({}) })
    await show()
    expect(wrapper.find('.active-links').exists()).toBe(false)
  })
})
