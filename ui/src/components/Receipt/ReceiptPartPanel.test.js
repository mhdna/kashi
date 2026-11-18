import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { emptyDesign, setElement, toggleSection } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptPartPanel from './ReceiptPartPanel.vue'

let wrapper
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

function open (group, design = emptyDesign()) {
  wrapper = mount(ReceiptPartPanel, {
    props: { catalog, design, group, 'onUpdate:design': d => wrapper.setProps({ design: d }) },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  return wrapper
}
const design = () => wrapper.props('design')
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const input = sel => find(sel).find('input, textarea')

describe('ReceiptPartPanel', () => {
  it('asks to click a part when none is picked', () => {
    open('')
    expect(find('panel-empty').text()).toContain('Click any part of the receipt')
  })

  it('the reported case: column titles can be reworded and put in First Letters', async () => {
    open('columns')
    expect(find('panel-title').text()).toBe('Items and column titles')
    for (const key of ['col_item', 'col_qty', 'col_disc', 'col_price', 'col_total', 'col_amount']) {expect(find(`line-${key}`).exists(), key).toBe(true)}
    expect(input('text-col_item').attributes('placeholder')).toBe('ITEM')
    await input('text-col_item').setValue('Article')
    expect(design().elements.col_item).toEqual({ text: 'Article' })
    // A column title has no size, bold or alignment of its own.
    expect(find('bigger-col_item').exists()).toBe(false)
    expect(find('align-col_item').exists()).toBe(false)
    expect(find('case-col_item').exists()).toBe(true)
  })

  it('the reported case: the branch name gets bigger, and the policy text is its own', async () => {
    open('header')
    await find('bigger-branch_name').trigger('click')
    await find('bigger-branch_name').trigger('click')
    expect(design().elements.branch_name).toEqual({ size: 2 })
    expect(find('size-branch_name').text()).toBe('+2')
    await find('smaller-branch_name').trigger('click')
    expect(find('size-branch_name').text()).toBe('+1')
    wrapper.unmount()
    open('policy')
    expect(input('text-policy_text').attributes('placeholder')).toBe('Exchanges accepted within 3 days.')
    await input('text-policy_text').setValue('Swaps within 7 days.')
    expect(design().elements.policy_text.text).toBe('Swaps within 7 days.')
  })

  it('a line can be centred or right, bold or regular, and left out', async () => {
    open('thanks')
    const align = find('align-thanks').findAll('button')
    await align.find(b => b.text() === 'Right').trigger('click')
    expect(design().elements.thanks.align).toBe('right')
    const bold = find('bold-thanks').findAll('button')
    await bold.find(b => b.text() === 'Regular').trigger('click')
    expect(design().elements.thanks.bold).toBe(false)
    await input('printed-thanks').setValue(false)
    expect(design().elements.thanks.hide).toBe(true)
  })

  it('the part can be left out, and Back to default undoes the part', async () => {
    open('payments', setElement(emptyDesign(), 'payments_title', { text: 'Paid with' }))
    await input('part-printed').setValue(false)
    expect(design().hide).toEqual(['payments'])
    await find('reset-part').trigger('click')
    expect(design().hide).toEqual([])
    expect(design().elements.payments_title).toBeUndefined()
    expect(find('reset-part').exists()).toBe(false)
  })

  it('the date format is with the customer, salesperson and date', async () => {
    open('meta')
    expect(find('date-format').exists()).toBe(true)
    wrapper.vm.$.setupState.update({ ...design(), date_format: 'iso' })
    await flushPromises()
    expect(design().date_format).toBe('iso')
    expect(find('reset-part').exists()).toBe(true)
  })

  it('a part left out greys its lines', () => {
    open('policy', toggleSection(emptyDesign(), 'policy'))
    expect(find('line-policy_title').classes()).toContain('rpp-line--off')
  })

  it('explains {percent} and {currency} where a line has them', () => {
    open('referral')
    expect(find('line-referral_note').text()).toContain('{percent} prints as the reward')
    wrapper.unmount()
    open('totals')
    expect(find('line-equivalent').text()).toContain('{currency} prints as the currency')
  })

  it('every part opens with every one of its lines, nothing in long dashes', () => {
    for (const g of catalog.groups) {
      open(g.key)
      for (const e of catalog.elements.filter(x => x.group === g.key)) {expect(find(`line-${e.key}`).exists(), e.key).toBe(true)}
      expect(wrapper.text()).not.toMatch(/[–—·•]/)
      wrapper.unmount()
      wrapper = null
    }
  })
})
