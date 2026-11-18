import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const previews = []
vi.mock('@/composables/useApi', () => ({
  authFetch: async url => {
    const path = String(url).replace('http://test/api', '')
    if (path.startsWith('/receipt_designer/catalog')) {return { ok: true, status: 200, json: async () => ({ catalog }) }}
    if (path.startsWith('/receipt_designer/preview')) {
      previews.push(Object.fromEntries(new URLSearchParams(path.split('?')[1] ?? '')))
      return { ok: true, status: 200, json: async () => ({
        image: 'data:image/png;base64,AAAA', width: 576, height: 1000,
        marks: [{ group: 'header', top: 0, bottom: 200 }, { group: 'columns', top: 300, bottom: 600 }, { group: 'thanks', top: 900, bottom: 1000 }],
        texts: '',
      }) }
    }
    return { ok: false, status: 404, json: async () => ({}) }
  },
}))
vi.mock('@/config', () => ({ API_BASE: 'http://test/api' }))
const getBranchSettings = vi.fn()
const updateBranchSettings = vi.fn()
vi.mock('@/composables/useBranchSettings', () => ({ useBranchSettings: () => ({ getBranchSettings, updateBranchSettings }) }))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({ branches: ref([{ id: 7, name: 'Centro' }, { id: 8, name: 'Hara' }]), fetchBranches: async () => {} }),
}))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success: vi.fn(), error: vi.fn() }) }))
const canEdit = ref(true)
vi.mock('@/composables/usePageAccess', () => ({ usePageAccess: () => ({ canEdit }) }))

const { default: ReceiptDesigner } = await import('./receipt-designer.vue')

let wrapper
beforeEach(() => { setActivePinia(createPinia()); canEdit.value = true; previews.length = 0; vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] }) })
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks(); vi.useRealTimers(); document.body.innerHTML = '' })

async function settle () {
  await flushPromises()
  vi.advanceTimersByTime(250)
  await flushPromises()
}
async function open () {
  wrapper = mount(ReceiptDesigner, { global: { plugins: [vuetify()] }, attachTo: document.body })
  await settle()
  return wrapper
}
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const state = () => wrapper.vm.$.setupState
const button = label => wrapper.findAll('button').find(b => b.text().trim() === label)

describe('Receipt Designer', () => {
  it('draws the receipt live from kashi, with a place to click for each part', async () => {
    await open()
    expect(previews.length).toBeGreaterThan(0)
    expect(previews.at(-1)).toMatchObject({ theme: 'classic', layout: 'A', mode: 'sale' })
    expect(find('receipt-image').attributes('src')).toContain('data:image/png')
    expect(find('pick-thanks').exists()).toBe(true)
    expect(find('panel-empty').exists()).toBe(true)
  })

  it('the reported case: click the column titles, reword one, and the receipt is drawn again with it', async () => {
    await open()
    await find('pick-columns').trigger('click')
    expect(find('panel-title').text()).toBe('Items and column titles')
    await find('text-col_item').find('input').setValue('Article')
    await settle()
    expect(JSON.parse(previews.at(-1).texts)).toEqual({ elements: { col_item: { text: 'Article' } } })
  })

  it('every mode is drawn', async () => {
    await open()
    for (const [label, mode] of [['Sale with referral code', 'referral'], ['Sale with a client', 'client'], ['Exchange', 'exchange'], ['Return', 'return']]) {
      await button(label).trigger('click')
      await settle()
      expect(previews.at(-1).mode, label).toBe(mode)
    }
  })

  it('a branch draws with its own data and opens with its design; Save sends the design with the look', async () => {
    getBranchSettings.mockResolvedValue({ receipt_theme: 'modern', receipt_layout: 'B', receipt_texts: '{"elements":{"thanks":{"text":"Bye"}},"hide":["payments"]}' })
    await open()
    state().target = 7
    await settle()
    expect(previews.at(-1)).toMatchObject({ branch_id: '7', theme: 'modern', layout: 'B' })
    expect(JSON.parse(previews.at(-1).texts)).toEqual({ elements: { thanks: { text: 'Bye' } }, hide: ['payments'] })
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings).toHaveBeenCalledWith(7, {
      receipt_theme: 'modern', receipt_layout: 'B', receipt_texts: '{"elements":{"thanks":{"text":"Bye"}},"hide":["payments"]}',
    })
  })

  it('all branches get the same design', async () => {
    await open()
    await find('pick-thanks').trigger('click')
    await find('text-thanks').find('input').setValue('See you')
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings.mock.calls.map(c => c[0])).toEqual([7, 8])
    expect(updateBranchSettings.mock.calls[0][1].receipt_texts).toBe('{"elements":{"thanks":{"text":"See you"}}}')
  })

  it('read-only roles see the receipt but change nothing', async () => {
    canEdit.value = false
    await open()
    expect(find('receipt-image').exists()).toBe(true)
    await find('pick-thanks').trigger('click')
    expect(find('panel-empty').exists()).toBe(true)
    expect(button('Save')).toBeUndefined()
  })
})
