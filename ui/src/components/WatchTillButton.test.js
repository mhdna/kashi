import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

const authFetch = vi.fn()
vi.mock('@/composables/useApi', () => ({ authFetch: (...a) => authFetch(...a) }))
const listeners = new Set()
vi.mock('@/composables/useAdminSocket', () => ({
  useAdminSocket: () => ({ ensureConnected: () => {}, onMessage: cb => { listeners.add(cb); return () => listeners.delete(cb) } }),
}))
vi.mock('@/config', () => ({ API_BASE: 'https://k.test/api' }))
const canViewPage = vi.fn(() => true)
const canEditPage = vi.fn(() => true)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ canViewPage, canEditPage }) }))

const { default: WatchTillButton } = await import('./WatchTillButton.vue')
const { resetLiveViewState } = await import('@/composables/useLiveView')

let wrapper = null
beforeEach(() => {
  setActivePinia(createPinia())
  resetLiveViewState()
  canViewPage.mockReturnValue(true)
  canEditPage.mockReturnValue(true)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  listeners.clear()
  vi.clearAllMocks()
})

async function show (settings, props = {}) {
  authFetch.mockResolvedValue({ ok: true, json: async () => ({ settings }) })
  wrapper = mount(WatchTillButton, {
    props: { registerId: 10, name: 'Centro CT1', online: true, ...props },
    global: { plugins: [vuetify()], stubs: { RouterLink: { props: ['to'], template: '<a :href="JSON.stringify(to)"><slot /></a>' } } },
  })
  await vi.waitFor(() => expect(authFetch).toHaveBeenCalled())
  await wrapper.vm.$nextTick()
  await new Promise(r => setTimeout(r, 0))
}

describe('WatchTillButton', () => {
  it('shows Watch for an online till when live view is on', async () => {
    await show({ live_view_enabled: true })
    expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(true)
  })
  it('shows nothing while live view is off in Settings', async () => {
    await show({ live_view_enabled: false })
    expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(false)
  })
  it('shows nothing for a till that is not online', async () => {
    await show({ live_view_enabled: true }, { online: false })
    expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(false)
  })
  it('shows nothing to a role without System', async () => {
    canViewPage.mockReturnValue(false)
    await show({ live_view_enabled: true })
    expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(false)
  })
  it('says who is watching, from the admin socket', async () => {
    await show({ live_view_enabled: true })
    for (const cb of listeners) {
      cb({ type: 'live_view_sessions', live_sessions: [{ register_id: 10, branch_id: 1, watchers: ['Ana', 'Sam'] }, { register_id: 11, branch_id: 1, watchers: ['Zed'] }] })
    }
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-test="watchers"]').text()).toBe('Watched by Ana, Sam')
    for (const cb of listeners) {
      cb({ type: 'live_view_sessions' })
    }
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-test="watchers"]').exists()).toBe(false)
  })
  it('icon mode shows only the eye, with no label or watcher text', async () => {
    await show({ live_view_enabled: true }, { icon: true })
    const btn = wrapper.find('[data-test="watch-register"]')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toBe('')
    expect(wrapper.find('[data-test="watchers"]').exists()).toBe(false)
  })
  it('icon mode is hidden while live view is off', async () => {
    await show({ live_view_enabled: false }, { icon: true })
    expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(false)
  })
  describe('separate screen', () => {
    const on = { live_view_enabled: true, live_view_control_enabled: true }

    it('offers the button only with the control switch on and the right to edit System', async () => {
      await show(on)
      const btn = wrapper.find('[data-test="separate-register"]')
      expect(btn.exists()).toBe(true)
      expect(btn.text()).toBe('Open separate screen')
      expect(wrapper.findComponent('[data-test="separate-register"]').props('to')).toEqual({ path: '/live/10', query: { name: 'Centro CT1', mode: 'virtual' } })
      wrapper.unmount()
      resetLiveViewState()
      await show({ live_view_enabled: true, live_view_control_enabled: false })
      expect(wrapper.find('[data-test="separate-register"]').exists()).toBe(false)
      expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(true)
    })

    it('is hidden from a role that can only view System, while Watch stays', async () => {
      canEditPage.mockReturnValue(false)
      await show(on)
      expect(wrapper.find('[data-test="separate-register"]').exists()).toBe(false)
      expect(wrapper.find('[data-test="watch-register"]').exists()).toBe(true)
    })

    it('is hidden for an offline till and in icon mode', async () => {
      await show(on, { online: false })
      expect(wrapper.find('[data-test="separate-register"]').exists()).toBe(false)
      wrapper.unmount()
      resetLiveViewState()
      await show(on, { icon: true })
      expect(wrapper.find('[data-test="separate-register"]').exists()).toBe(false)
    })

    it('says who uses it, and keeps the mirror\'s watchers apart', async () => {
      await show(on)
      for (const cb of listeners) {
        cb({
          type: 'live_view_sessions',
          live_sessions: [
            { register_id: 10, branch_id: 1, watchers: ['Ana'] },
            { register_id: 10, branch_id: 1, watchers: ['Sam'], virtual: true },
          ],
        })
      }
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-test="watchers"]').text()).toBe('Watched by Ana')
      expect(wrapper.find('[data-test="separate-user"]').text()).toBe('Separate screen used by Sam')
    })
  })
})
