import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const previews = []
const puts = []
let template = { tax_rate: 11 }
vi.mock('@/composables/useApi', () => ({
  authFetch: async (url, options = {}) => {
    const path = String(url).replace('http://test/api', '')
    if (path.startsWith('/receipt_designer/catalog')) {return { ok: true, status: 200, json: async () => ({ catalog }) }}
    if (path.startsWith('/invoice_discounts')) {
      return { ok: true, status: 200, json: async () => ({ invoice_discounts: [{ id: 5, name: 'Students 15%', kind: 'percent', percent: 15, amount: 0, is_active: true }] }) }
    }
    if (path.startsWith('/default_branch_settings')) {
      if (options.method === 'PUT') {
        puts.push(JSON.parse(options.body))
        return { ok: true, status: 200, json: async () => ({ settings: JSON.parse(options.body) }) }
      }
      return { ok: true, status: 200, json: async () => ({ settings: template }) }
    }
    if (path.startsWith('/receipt_designer/preview')) {
      previews.push(Object.fromEntries(new URLSearchParams(path.split('?')[1] ?? '')))
      return { ok: true, status: 200, json: async () => ({
        image: 'data:image/png;base64,AAAA', width: 576, height: 1000,
        marks: [{ group: 'header', top: 0, bottom: 200 }, { group: 'columns', top: 300, bottom: 600 }, { group: 'thanks', top: 900, bottom: 1000 }],
        lines: [
          { key: 'col_item', group: 'columns', top: 300, bottom: 330, text: 'ITEM', size_px: 26, align: 'left', left: 24, right: 90 },
          { key: 'thanks', group: 'thanks', top: 910, bottom: 940, text: 'Thank you for shopping with us!', size_px: 26, align: 'center' },
        ],
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
beforeEach(() => {
  setActivePinia(createPinia())
  canEdit.value = true
  previews.length = 0
  puts.length = 0
  template = { tax_rate: 11 }
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
})
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
  it('draws the receipt live from kashi, with a place to click for each part and its words', async () => {
    await open()
    expect(previews.at(-1)).toMatchObject({ theme: 'classic', layout: 'A', mode: 'sale' })
    expect(find('receipt-image').attributes('src')).toContain('data:image/png')
    expect(find('pick-thanks').exists()).toBe(true)
    expect(find('words-thanks').exists()).toBe(true)
  })

  it('the reported case: type a column title right on the receipt, and it is drawn again with it', async () => {
    await open()
    await find('words-col_item').trigger('click')
    expect(find('line-tools-title').text()).toBe('Item')
    await find('edit-words').setValue('Article')
    await find('edit-words').trigger('keydown', { key: 'Enter' })
    await settle()
    expect(JSON.parse(previews.at(-1).texts)).toEqual({ elements: { col_item: { text: 'Article' } } })
  })

  it('typing the default back is no change', async () => {
    await open()
    await find('words-thanks').trigger('click')
    await find('edit-words').setValue('Bye')
    await find('edit-words').trigger('keydown', { key: 'Enter' })
    await find('words-thanks').trigger('click')
    await find('edit-words').setValue('Thank you for shopping with us!')
    await find('edit-words').trigger('keydown', { key: 'Enter' })
    expect(state().design.elements.thanks).toBeUndefined()
  })

  it('the reported case: Alt+Up moves the picked part up', async () => {
    await open()
    await find('pick-thanks').trigger('click')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', altKey: true }))
    await settle()
    expect(JSON.parse(previews.at(-1).texts).order).toEqual(['payments', 'policy', 'referral', 'thanks', 'socials'])
  })

  it('the receipt sits beside the themes, its options on it, no side panel', async () => {
    await open()
    const work = find('receipt-image').element.closest('.rd-work')
    expect([...work.children].map(c => c.className.split(' ')[0])).toEqual(['rd-picker', 'rd-paper-col'])
    expect(work.className).not.toContain('rd-work--shop')
    await find('tab-shop').trigger('click')
    await settle()
    expect(find('receipt-image').element.closest('.rd-work').className).toContain('rd-work--shop')
  })

  it('the reported case: each deal has a view of its own, drawn with the deal', async () => {
    await open()
    await button('Students 15%').trigger('click')
    await settle()
    expect(previews.at(-1).mode).toBe('client')
    expect(JSON.parse(previews.at(-1).deal)).toEqual({ name: 'Students 15%', kind: 'percent', percent: 15, amount: 0 })
    await button('Sale').trigger('click')
    await settle()
    expect(previews.at(-1).deal).toBeUndefined()
  })

  it('every mode is drawn', async () => {
    await open()
    for (const [label, mode] of [['Sale with referral code', 'referral'], ['Sale with a client', 'client'], ['Exchange', 'exchange'], ['Return', 'return']]) {
      await button(label).trigger('click')
      await settle()
      expect(previews.at(-1).mode, label).toBe(mode)
    }
  })

  it('a branch opens with its design and details; Save sends both', async () => {
    getBranchSettings.mockResolvedValue({
      receipt_theme: 'modern', receipt_layout: 'B', receipt_texts: '{"elements":{"thanks":{"text":"Bye"}}}',
      market_name: 'Bazaar', branch_phone: '03 1', social_platforms: ['instagram'], social_handles: { instagram: '@bz' },
    })
    await open()
    state().target = 7
    await settle()
    expect(previews.at(-1)).toMatchObject({ branch_id: '7', theme: 'modern', layout: 'B' })
    expect(previews.at(-1).shop).toBeUndefined()
    await find('tab-shop').trigger('click')
    await settle()
    await find('shop-store-name').find('input').setValue('Bazaar Store')
    await settle()
    expect(JSON.parse(previews.at(-1).shop)).toMatchObject({ store_name: 'Bazaar Store', phone: '03 1', social_handles: { instagram: '@bz' } })
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings).toHaveBeenCalledWith(7, expect.objectContaining({
      receipt_theme: 'modern', receipt_texts: '{"elements":{"thanks":{"text":"Bye"}}}',
      market_name: 'Bazaar Store', branch_phone: '03 1', social_platforms: ['instagram'], social_handles: { instagram: '@bz' },
    }))
  })

  it('the reported case: Follow us platforms and handles for every branch, each keeping its phone', async () => {
    await open()
    await find('tab-shop').trigger('click')
    await settle()
    expect(find('shop-phone').exists()).toBe(false)
    await find('social-tiktok').find('input').setValue(true)
    await find('handle-tiktok').find('input').setValue('@bazaar')
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings.mock.calls.map(c => c[0])).toEqual([7, 8])
    const sent = updateBranchSettings.mock.calls[0][1]
    expect(sent).toMatchObject({ social_platforms: ['tiktok'], social_handles: { tiktok: '@bazaar' } })
    expect(sent).not.toHaveProperty('branch_phone')
  })

  it('without touching the shop details, Save sends only the receipt', async () => {
    await open()
    await button('Save').trigger('click')
    await flushPromises()
    expect(Object.keys(updateBranchSettings.mock.calls[0][1]).toSorted()).toEqual(['receipt_layout', 'receipt_texts', 'receipt_theme'])
  })

  it('the reported case: Make this the default writes the receipt into the default template, keeping the rest', async () => {
    await open()
    await find('words-thanks').trigger('click')
    await find('edit-words').setValue('Bye')
    await find('edit-words').trigger('keydown', { key: 'Enter' })
    await find('make-default').trigger('click')
    await flushPromises()
    expect(puts).toEqual([{ tax_rate: 11, receipt_theme: 'classic', receipt_layout: 'A', receipt_texts: '{"elements":{"thanks":{"text":"Bye"}}}' }])
  })

  it('read-only roles see the receipt but change nothing', async () => {
    canEdit.value = false
    await open()
    expect(find('receipt-image').exists()).toBe(true)
    expect(find('words-thanks').exists()).toBe(false)
    expect(button('Save')).toBeUndefined()
    expect(find('make-default').exists()).toBe(false)
  })
})
