import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { emptyDesign, setElement, toggleSection } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptPartPanel from './ReceiptPartPanel.vue'

let wrapper
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = '' })

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
const button = (sel, label) => find(sel).findAll('button').find(b => b.text().trim() === label)

describe('ReceiptPartPanel', () => {
  it('asks to click a part when none is picked', () => {
    open('')
    expect(find('panel-empty').text()).toContain('Click a part of the receipt')
  })

  it('the reported case: no text fields and no "Look\'s own"; words are typed on the paper', () => {
    open('payments')
    expect(wrapper.findAll('[data-test^="text-"], textarea').length).toBe(0)
    expect(wrapper.text()).not.toContain('Look\'s own')
    expect(find('words-payments_title').text()).toContain('Click its words on the receipt')
  })

  it('the reported case: the paid and change lines can be made bold, and set back with Default', async () => {
    open('payments')
    await button('bold-received', 'Regular').trigger('click')
    expect(design().elements.received).toEqual({ bold: false })
    await button('bold-received', 'Regular').trigger('click')
    expect(design().elements.received).toBeUndefined()
    await button('bold-change', 'Bold').trigger('click')
    expect(find('default-change').exists()).toBe(true)
    await find('default-change').trigger('click')
    expect(design().elements.change).toBeUndefined()
    expect(find('default-change').exists()).toBe(false)
  })

  it('the reported case: a whole part centred and bold at once', async () => {
    open('payments')
    await button('group-align', 'Centre').trigger('click')
    await button('group-bold', 'Bold').trigger('click')
    await find('bigger-group').trigger('click')
    expect(design().groups.payments).toEqual({ align: 'center', bold: true, size: 1 })
    await find('reset-part').trigger('click')
    expect(design().groups.payments).toBeUndefined()
  })

  it('lines and a box: above, under (or none), a rounded box, thicker', async () => {
    open('policy')
    await button('decor-above', 'Dashed').trigger('click')
    await button('decor-below', 'None').trigger('click')
    await button('decor-box', 'Solid').trigger('click')
    await input('decor-rounded').setValue(true)
    await find('bigger-thickness').trigger('click')
    expect(design().decor.policy).toEqual({ above: 'dashed', below: 'none', box: 'solid', rounded: true, thickness: 3 })
    expect(find('value-thickness').text()).toBe('3')
  })

  it('moves a part under the totals up and down; the shop header doesn\'t move', async () => {
    open('thanks')
    expect(find('move-down').attributes('disabled')).toBeDefined()
    await find('move-up').trigger('click')
    expect(design().order).toEqual(['payments', 'policy', 'referral', 'thanks', 'socials'])
    wrapper.unmount()
    open('header')
    expect(find('move-up').exists()).toBe(false)
  })

  it('the branch name gets bigger; column titles can\'t', async () => {
    open('header')
    await find('bigger-branch_name').trigger('click')
    await find('bigger-branch_name').trigger('click')
    expect(design().elements.branch_name).toEqual({ size: 2 })
    expect(find('value-branch_name').text()).toBe('+2')
    wrapper.unmount()
    open('columns')
    expect(find('bigger-col_item').exists()).toBe(false)
    expect(find('case-col_item').exists()).toBe(true)
  })

  it('shows a line\'s own words and their placeholders', () => {
    open('referral', setElement(emptyDesign(), 'referral_title', { text: 'SHARE:' }))
    expect(find('words-referral_title').text()).toContain('"SHARE:"')
    expect(find('words-referral_note').text()).toContain('{percent} prints as the reward')
  })

  it('the part can be left out; its lines grey', async () => {
    open('policy', toggleSection(emptyDesign(), 'policy'))
    expect(find('line-policy_title').classes()).toContain('rpp-line--off')
    await input('part-printed').setValue(true)
    expect(design().hide).toEqual([])
  })

  it('the date format is with the customer, salesperson and date', async () => {
    open('meta')
    expect(find('date-format').exists()).toBe(true)
    wrapper.vm.$.setupState.update({ ...design(), date_format: 'iso' })
    await flushPromises()
    expect(find('reset-part').exists()).toBe(true)
  })

  it('every part opens with each of its lines, nothing in long dashes', () => {
    for (const g of catalog.groups) {
      open(g.key)
      for (const e of catalog.elements.filter(x => x.group === g.key)) {expect(find(`line-${e.key}`).exists(), e.key).toBe(true)}
      expect(wrapper.text()).not.toMatch(/[–—·•]/)
      wrapper.unmount()
      wrapper = null
    }
  })
})
