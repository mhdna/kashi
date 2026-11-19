import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { emptyDesign } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptFontSelect from './ReceiptFontSelect.vue'
import ReceiptLivePaper from './ReceiptLivePaper.vue'

let wrapper
const fontFile = vi.fn(async () => new ArrayBuffer(8))
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = '' })

const marks = [{ group: 'header', top: 0, bottom: 200 }, { group: 'columns', top: 280, bottom: 600 }, { group: 'payments', top: 650, bottom: 760 }, { group: 'thanks', top: 900, bottom: 1000 }]
const lines = [
  { key: 'col_item', group: 'columns', top: 300, bottom: 330, text: 'ITEM', size_px: 26, align: 'left', left: 24, right: 90 },
  { key: 'col_qty', group: 'columns', top: 300, bottom: 330, text: 'QTY', size_px: 26, align: 'left', left: 340, right: 390 },
  { key: 'payments_title', group: 'payments', top: 660, bottom: 690, text: 'PAYMENT DETAILS', size_px: 26, align: 'center', font: 'belleza', bold: true },
  { key: 'invoice_number', group: 'header', top: 150, bottom: 180, text: 'Sales Invoice: PV-1', size_px: 24, align: 'center' },
  { key: 'policy_text', group: 'policy', top: 800, bottom: 860, text: 'Exchanges accepted within 3 days.', size_px: 22, align: 'center' },
  { key: 'thanks', group: 'thanks', top: 940, bottom: 970, text: 'يسلمو، نورتونا!', size_px: 24, align: 'center', lang: 'ar', font: 'vazirmatn' },
]
function open (props = {}) {
  wrapper = mount(ReceiptLivePaper, {
    props: {
      image: 'data:image/png;base64,AAAA', height: 1000, imageWidth: 576, marks, lines, catalog, design: emptyDesign(),
      'onUpdate:design': d => wrapper.setProps({ design: d }),
      'onPick': g => wrapper.setProps({ picked: g }),
      ...props,
    },
    global: { plugins: [vuetify()], provide: { receiptFontFile: fontFile } },
    attachTo: document.body,
  })
  return wrapper
}
const find = sel => wrapper.find(`[data-test="${sel}"]`)

describe('ReceiptLivePaper', () => {
  it('shows the till\'s drawing with each part where it is', () => {
    open()
    expect(find('receipt-image').attributes('src')).toBe('data:image/png;base64,AAAA')
    expect(find('pick-thanks').element.style.top).toBe('90%')
    expect(find('pick-thanks').attributes('aria-label')).toBe('Change Thank-you line')
  })

  it('the reported case: ITEM is its own spot on its row, and its options open right under it', async () => {
    open()
    const item = find('words-col_item')
    expect(item.element.style.left).toBe(`${(24 / 576) * 100 - 0.5}%`)
    expect(find('words-col_qty').element.style.left).not.toBe(item.element.style.left)
    await item.trigger('click')
    expect(wrapper.props('picked')).toBe('columns')
    expect(find('line-tools-title').text()).toBe('Item')
    // The toolbar sits right under the line.
    expect(find('line-tools').element.parentElement.style.top).toBe(`calc(${(330 / 1000) * 100}% + 4px)`)
    const abc = find('line-case').findAll('button').find(b => b.text() === 'ABC')
    await abc.trigger('click')
    expect(wrapper.props('design').elements.col_item).toEqual({ case: 'upper' })
  })

  it('type over words in place: Enter keeps them, the default typed back is no change', async () => {
    open()
    await find('words-payments_title').trigger('click')
    const box = find('edit-words')
    expect(box.element.value).toBe('PAYMENT DETAILS')
    expect(box.element.style.textAlign).toBe('center')
    await box.setValue('  PAID WITH ')
    await box.trigger('keydown', { key: 'Enter' })
    expect(wrapper.props('design').elements.payments_title).toEqual({ text: 'PAID WITH' })
    await find('words-payments_title').trigger('click')
    await find('edit-words').setValue('PAYMENT DETAILS')
    await find('edit-words').trigger('keydown', { key: 'Enter' })
    expect(wrapper.props('design').elements.payments_title).toBeUndefined()
  })

  it('the reported case: the shop\'s long policy cut down to the default sticks', async () => {
    const long = 'Exchanges accepted within 3 days. Items must be unworn, unused, unwashed, undamaged, with tags. Present this receipt.'
    open({ lines: lines.map(l => (l.key === 'policy_text' ? { ...l, text: long } : l)) })
    await find('words-policy_text').trigger('click')
    expect(find('edit-words').element.value).toBe(long)
    await find('edit-words').setValue('Exchanges accepted within 3 days.')
    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await flushPromises()
    expect(wrapper.props('design').elements.policy_text).toEqual({ text: 'Exchanges accepted within 3 days.' })
  })

  it('Esc keeps the words as they were; long words get a box of lines', async () => {
    open()
    await find('words-payments_title').trigger('click')
    await find('edit-words').setValue('Other')
    await find('edit-words').trigger('keydown', { key: 'Escape' })
    expect(find('edit-words').exists()).toBe(false)
    expect(wrapper.props('design').elements.payments_title).toBeUndefined()
    await find('words-policy_text').trigger('click')
    expect(find('edit-words').element.tagName).toBe('TEXTAREA')
  })

  it('a line whose words can\'t change still opens its options, with nothing to type', async () => {
    open()
    await find('words-invoice_number').trigger('click')
    expect(find('edit-words').exists()).toBe(false)
    expect(find('line-tools-title').text()).toBe('Invoice number line')
  })

  it('a click on a part opens the part\'s options under it; the cross closes them', async () => {
    open()
    await find('pick-payments').trigger('click')
    expect(find('part-tools-title').text()).toBe('Payment details')
    expect(find('part-tools').element.parentElement.style.top).toBe(`calc(${(760 / 1000) * 100}% + 4px)`)
    await find('part-close').trigger('click')
    expect(find('part-tools').exists()).toBe(false)
    expect(wrapper.props('picked')).toBe('')
  })

  it('the picked line follows a redrawn receipt', async () => {
    open()
    await find('words-col_item').trigger('click')
    await wrapper.setProps({ lines: lines.map(l => (l.key === 'col_item' ? { ...l, top: 310, bottom: 340 } : l)) })
    await flushPromises()
    expect(find('line-tools').element.parentElement.style.top).toBe(`calc(${(340 / 1000) * 100}% + 4px)`)
  })

  it('read-only: nothing to click', () => {
    open({ editable: false, picked: 'payments' })
    expect(find('words-col_item').exists()).toBe(false)
    expect(find('part-tools').exists()).toBe(false)
    expect(find('pick-thanks').attributes('disabled')).toBeDefined()
  })

  it('says it is drawing before the first picture', () => {
    open({ image: '' })
    expect(wrapper.text()).toContain('Drawing the receipt')
  })

  it('the reported case: the typing box keeps the line\'s own letters, not a stand-in font', async () => {
    open()
    await find('words-payments_title').trigger('click')
    const style = find('edit-words').element.style
    expect(style.fontFamily).toContain('kashi-receipt-belleza-bold')
    expect(style.fontFamily).not.toContain('Cascadia')
    expect(fontFile).toHaveBeenCalledWith('belleza', true)
  })

  it('a line takes a font from its toolbar', async () => {
    open()
    await find('words-payments_title').trigger('click')
    expect(find('line-font').exists()).toBe(true)
    wrapper.findComponent(ReceiptFontSelect).vm.$emit('update:modelValue', 'pacifico')
    await flushPromises()
    expect(wrapper.props('design').elements.payments_title).toEqual({ font: 'pacifico' })
  })

  it('typing over an Arabic line writes its Arabic words, right to left', async () => {
    open()
    await find('words-thanks').trigger('click')
    const box = find('edit-words')
    expect(box.attributes('dir')).toBe('rtl')
    expect(box.element.value).toBe('يسلمو، نورتونا!')
    await box.setValue('شكرا كتير!')
    await box.trigger('keydown.enter')
    expect(wrapper.props('design').elements.thanks).toEqual({ text_ar: 'شكرا كتير!' })
  })

  it('the reported case: cut part of the policy, click anywhere outside, and it is kept', async () => {
    open()
    await find('words-policy_text').trigger('click')
    const box = find('edit-words')
    box.element.value = 'Exchanges accepted.'
    await box.trigger('input')
    // A click outside the box that doesn't take the focus away.
    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await flushPromises()
    expect(wrapper.props('design').elements.policy_text).toEqual({ text: 'Exchanges accepted.' })
    expect(find('edit-words').exists()).toBe(false)
  })

  it('closing the toolbar keeps typed words; Esc drops them', async () => {
    open()
    await find('words-payments_title').trigger('click')
    await find('edit-words').setValue('PAID WITH')
    await find('line-close').trigger('click')
    expect(wrapper.props('design').elements.payments_title).toEqual({ text: 'PAID WITH' })
    await find('words-policy_text').trigger('click')
    await find('edit-words').setValue('Nothing')
    await find('edit-words').trigger('keydown', { key: 'Escape' })
    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await flushPromises()
    expect(wrapper.props('design').elements.policy_text).toBeUndefined()
  })

  it('a click inside the box keeps it open', async () => {
    open()
    await find('words-policy_text').trigger('click')
    find('edit-words').element.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await flushPromises()
    expect(find('edit-words').exists()).toBe(true)
  })
})
