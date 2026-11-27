import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import fixture from '@/utils/testdata/client_sources.json'

const authFetch = vi.fn()
const notifyError = vi.fn()
vi.mock('@/composables/useApi', () => ({ authFetch: (...args) => authFetch(...args) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ error: notifyError, success: vi.fn() }) }))
vi.mock('@/config', () => ({ API_BASE: 'https://cloud.grandbz.com/api' }))
vi.mock('@/i18n', () => ({
  t: (text, params) => (params ? text.replace(/\{(\w+)\}/g, (_, k) => params[k]) : text),
}))

const { default: ClientSourcesPanel } = await import('./ClientSourcesPanel.vue')

let wrapper = null
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.clearAllMocks()
})

const ok = data => ({ ok: true, json: async () => data })
function server (list = fixture.client_sources) {
  authFetch.mockImplementation(async (url, opts = {}) => {
    if (!opts.method || opts.method === 'GET') {return ok({ client_sources: list })}
    if (opts.method === 'DELETE') {return { ok: false, status: 409, json: async () => ({ error: 'customers gave this answer' }) }}
    return ok({ client_source: JSON.parse(opts.body) })
  })
}
async function show (props = {}) {
  wrapper = mount(ClientSourcesPanel, { props, global: { plugins: [vuetify()] } })
  await flushPromises()
}
const q = sel => wrapper.findAll(`[data-test="${sel}"]`)
const calls = method => authFetch.mock.calls.filter(([, o]) => (o?.method || 'GET') === method)

describe('ClientSourcesPanel', () => {
  it('shows every option from the API, in order, with its customers and share', async () => {
    server()
    await show()
    expect(authFetch.mock.calls[0][0]).toBe('https://cloud.grandbz.com/api/client_sources')
    expect(q('source-name').map(e => e.text())).toEqual(['Social media', 'SMS', 'A friend', 'Walking by', 'Other', 'Radio'])
    expect(q('source-count').map(e => e.text())).toEqual([
      '41 customers, 48%', '12 customers, 14%', '22 customers, 26%', '7 customers, 8%', '0 customers, 0%', '3 customers, 4%',
    ])
    expect(q('sources-total')[0].text()).toContain('85 customers answered')
    expect(q('source-row')[5].text()).toContain('Hidden')
  })

  it('only an option nobody gave can be deleted', async () => {
    server()
    await show()
    expect(q('source-delete')).toHaveLength(1)
    expect(q('source-row')[4].find('[data-test="source-delete"]').exists()).toBe(true)
  })

  it('adds a new option at the end, and refuses a blank or taken name', async () => {
    server()
    await show()
    const input = q('source-new-name')[0].find('input')
    await input.setValue(' sms ')
    await q('source-add')[0].trigger('submit')
    await flushPromises()
    expect(calls('POST')).toHaveLength(0)
    expect(wrapper.text()).toContain('There is already an option with this name')

    await input.setValue('Flyer')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(calls('POST')).toHaveLength(1)
    expect(JSON.parse(calls('POST')[0][1].body)).toEqual({ name: 'Flyer', sort_order: 7, is_active: true })
  })

  it('hiding sends the option with is_active off', async () => {
    server()
    await show()
    q('source-toggle')[1].trigger('click')
    await flushPromises()
    expect(JSON.parse(calls('PUT')[0][1].body)).toEqual({ id: 2, name: 'SMS', sort_order: 2, is_active: false })
  })

  it('renaming saves the trimmed name', async () => {
    server()
    await show()
    q('source-rename')[2].trigger('click')
    await flushPromises()
    await q('source-edit-name')[0].find('input').setValue(' A friend or family ')
    await q('source-save')[0].trigger('click')
    await flushPromises()
    expect(JSON.parse(calls('PUT')[0][1].body)).toEqual({ id: 3, name: 'A friend or family', sort_order: 3, is_active: true })
  })

  it('a refused delete is shown to the user', async () => {
    server()
    await show()
    q('source-delete')[0].trigger('click')
    await flushPromises()
    expect(notifyError).toHaveBeenCalledWith('customers gave this answer')
  })

  it('without edit access there are no buttons to change anything', async () => {
    server()
    await show({ canEdit: false })
    expect(q('source-toggle')).toHaveLength(0)
    expect(q('source-add')).toHaveLength(0)
    expect(q('source-name')).toHaveLength(6)
  })
})
