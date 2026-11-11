import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

const calls = []
let reply = { set: 2, unmatched: [] }
let status = 200
const success = vi.fn()
const error = vi.fn()
vi.mock('@/composables/useApi', () => ({
  authFetch: vi.fn(async (url, init) => {
    calls.push({ url, body: JSON.parse(init.body) })
    return { ok: status === 200, status, json: async () => reply }
  }),
}))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success, error }) }))

const { default: VariantPickDialog } = await import('./VariantPickDialog.vue')

let wrapper
afterEach(() => {
  wrapper?.unmount()
  calls.length = 0
  reply = { set: 2, unmatched: [] }
  status = 200
  success.mockClear()
  error.mockClear()
})

function open () {
  wrapper = mount(VariantPickDialog, { global: { plugins: [vuetify()] } })
  return wrapper
}
const type = async value => { await wrapper.find('textarea').setValue(value) }
const save = async () => { await wrapper.find('[data-test="variant-pick-save"]').trigger('click'); await flushPromises() }

describe('VariantPickDialog', () => {
  it('cannot save until a code is entered', async () => {
    open()
    expect(wrapper.find('[data-test="variant-pick-save"]').attributes('disabled')).toBeDefined()
    await type('  ,; ')
    expect(wrapper.find('[data-test="variant-pick-save"]').attributes('disabled')).toBeDefined()
    await type('COC5MSH35')
    expect(wrapper.find('[data-test="variant-pick-save"]').attributes('disabled')).toBeUndefined()
  })

  it('sends the cleaned codes with "always" by default, tells the user and closes', async () => {
    open()
    await type('COC5MSH35, tj66008\nCOC5MSH35')
    await save()
    expect(calls).toHaveLength(1)
    expect(calls[0].url).toMatch(/\/variant_pick$/)
    expect(calls[0].body).toEqual({ entries: ['COC5MSH35', 'tj66008'], mode: 'always' })
    expect(success).toHaveBeenCalledWith('2 items will ask for the colour/size at the till.')
    expect(wrapper.emitted('close')).toBeTruthy()
    expect(wrapper.emitted('saved')).toBeTruthy()
  })

  it('stays open and lists the codes that matched nothing', async () => {
    reply = { set: 1, unmatched: ['NOPE1', 'NOPE2'] }
    open()
    await type('A1 NOPE1 NOPE2')
    await save()
    expect(wrapper.find('[data-test="variant-pick-unmatched"]').text()).toContain('NOPE1, NOPE2')
    expect(wrapper.emitted('close')).toBeFalsy()
  })

  it('shows the server error and does not close', async () => {
    status = 400
    reply = { error: 'no codes given' }
    open()
    await type('A1')
    await save()
    expect(error).toHaveBeenCalledWith('no codes given')
    expect(wrapper.emitted('close')).toBeFalsy()
  })
})
