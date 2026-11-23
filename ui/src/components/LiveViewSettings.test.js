import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

const authFetch = vi.fn()
vi.mock('@/composables/useApi', () => ({ authFetch: (...args) => authFetch(...args) }))
vi.mock('@/composables/useAdminSocket', () => ({
  useAdminSocket: () => ({ ensureConnected: () => {}, onMessage: () => () => {} }),
}))
vi.mock('@/config', () => ({ API_BASE: 'https://k.test/api' }))

const { default: LiveViewSettings } = await import('./LiveViewSettings.vue')
const { resetLiveViewState } = await import('@/composables/useLiveView')

let wrapper = null
beforeEach(() => resetLiveViewState())
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.clearAllMocks()
})

function server (initial) {
  let state = initial
  authFetch.mockImplementation(async (url, options) => {
    if (options?.method === 'PUT') {
      const body = JSON.parse(options.body)
      if (body.control_enabled && !body.enabled) {
        return { ok: false, status: 400, json: async () => ({ error: 'remote control needs live view to be on' }) }
      }
      state = { live_view_enabled: body.enabled, live_view_control_enabled: body.control_enabled }
    }
    return { ok: true, json: async () => ({ settings: state }) }
  })
}

async function show (canEdit = true) {
  wrapper = mount(LiveViewSettings, { props: { canEdit }, global: { plugins: [vuetify()] } })
  await flushPromises()
  return wrapper
}
const input = name => wrapper.find(`[data-test="${name}"] input`)

describe('LiveViewSettings', () => {
  it('starts off, with control switched off and unavailable', async () => {
    server({ live_view_enabled: false, live_view_control_enabled: false })
    await show()
    expect(input('live-view-enabled').element.checked).toBe(false)
    expect(input('live-view-control').element.checked).toBe(false)
    expect(input('live-view-control').element.disabled).toBe(true)
  })

  it('turning view on saves it and unlocks control', async () => {
    server({ live_view_enabled: false, live_view_control_enabled: false })
    await show()
    await input('live-view-enabled').setValue(true)
    await flushPromises()
    const put = authFetch.mock.calls.find(([, o]) => o?.method === 'PUT')
    expect(put[0]).toBe('https://k.test/api/app_settings/live_view')
    expect(JSON.parse(put[1].body)).toEqual({ enabled: true, control_enabled: false })
    expect(input('live-view-control').element.disabled).toBe(false)
  })

  it('turning control on, then view off, sends control off with it', async () => {
    server({ live_view_enabled: true, live_view_control_enabled: false })
    await show()
    await input('live-view-control').setValue(true)
    await flushPromises()
    await input('live-view-enabled').setValue(false)
    await flushPromises()
    const bodies = authFetch.mock.calls.filter(([, o]) => o?.method === 'PUT').map(([, o]) => JSON.parse(o.body))
    expect(bodies).toEqual([
      { enabled: true, control_enabled: true },
      { enabled: false, control_enabled: false },
    ])
    expect(input('live-view-control').element.checked).toBe(false)
    expect(input('live-view-control').element.disabled).toBe(true)
  })

  it('shows control as off whenever view is off, even if the server row says otherwise', async () => {
    server({ live_view_enabled: false, live_view_control_enabled: true })
    await show()
    expect(input('live-view-control').element.checked).toBe(false)
  })

  it('without the right to edit, both switches are locked', async () => {
    server({ live_view_enabled: true, live_view_control_enabled: true })
    await show(false)
    expect(input('live-view-enabled').element.disabled).toBe(true)
    expect(input('live-view-control').element.disabled).toBe(true)
  })

  it('shows the server error and keeps the switch where it was', async () => {
    authFetch.mockImplementation(async (url, options) => options?.method === 'PUT'
      ? { ok: false, status: 500, json: async () => ({ error: 'boom' }) }
      : { ok: true, json: async () => ({ settings: { live_view_enabled: false, live_view_control_enabled: false } }) })
    await show()
    await input('live-view-enabled').setValue(true)
    await flushPromises()
    expect(wrapper.find('[data-test="live-view-error"]').text()).toBe('boom')
    expect(input('live-view-enabled').element.checked).toBe(false)
  })
})
