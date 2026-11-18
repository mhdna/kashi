import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { emptyDesign, setElement, toggleSection } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptLineTools from './ReceiptLineTools.vue'
import ReceiptPartTools from './ReceiptPartTools.vue'

let wrapper
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = '' })

const info = key => catalog.elements.find(e => e.key === key)
function lineTools (key, design = emptyDesign()) {
  wrapper = mount(ReceiptLineTools, {
    props: { catalog, design, info: info(key), 'onUpdate:design': d => wrapper.setProps({ design: d }) },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  return wrapper
}
function partTools (group, design = emptyDesign()) {
  wrapper = mount(ReceiptPartTools, {
    props: { catalog, design, group, 'onUpdate:design': d => wrapper.setProps({ design: d }) },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  return wrapper
}
const design = () => wrapper.props('design')
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const button = (sel, label) => find(sel).findAll('button').find(b => b.text().trim() === label || b.attributes('aria-label') === label)

describe('ReceiptLineTools', () => {
  it('the reported case: ITEM\'s capitals right under it, and back with Default', async () => {
    lineTools('col_item')
    expect(find('line-tools-title').text()).toBe('Item')
    await button('line-case', 'Abc').trigger('click')
    expect(design().elements.col_item).toEqual({ case: 'title' })
    await button('line-case', 'Abc').trigger('click')
    expect(design().elements.col_item).toBeUndefined()
    await button('line-case', 'abc').trigger('click')
    await find('line-default').trigger('click')
    expect(design().elements.col_item).toBeUndefined()
    // A column title has no size, weight or alignment of its own.
    expect(find('size-line').exists()).toBe(false)
    expect(find('line-bold').exists()).toBe(false)
    expect(find('line-align').exists()).toBe(false)
  })

  it('a line with everything: size, bold, alignment, leave out', async () => {
    lineTools('thanks')
    await find('bigger-line').trigger('click')
    await button('line-bold', 'Regular').trigger('click')
    await button('line-align', 'Right').trigger('click')
    await find('line-hide').trigger('click')
    expect(design().elements.thanks).toEqual({ size: 1, bold: false, align: 'right', hide: true })
    expect(find('line-hide').text()).toContain('Print it')
  })

  it('explains placeholders, and closes', async () => {
    lineTools('referral_note')
    expect(wrapper.text()).toContain('{percent} prints as the reward')
    await find('line-close').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('every line has its tools, nothing in long dashes', () => {
    for (const e of catalog.elements) {
      lineTools(e.key)
      expect(find('line-tools-title').text(), e.key).toBe(e.label)
      expect(wrapper.text()).not.toMatch(/[–—·•]/)
      wrapper.unmount()
      wrapper = null
    }
  })
})

describe('ReceiptPartTools', () => {
  it('the reported case: a whole part centred and regular at once', async () => {
    partTools('payments')
    await button('group-align', 'Centre').trigger('click')
    await button('group-bold', 'Regular').trigger('click')
    await find('bigger-group').trigger('click')
    expect(design().groups.payments).toEqual({ align: 'center', bold: false, size: 1 })
    await find('reset-part').trigger('click')
    expect(design().groups).toEqual({})
  })

  it('leaves the part out and back, and moves it', async () => {
    partTools('thanks')
    await find('part-printed').trigger('click')
    expect(design().hide).toEqual(['thanks'])
    expect(find('part-printed').text()).toContain('Print this part')
    expect(find('move-down').attributes('disabled')).toBeDefined()
    await find('move-up').trigger('click')
    expect(design().order).toEqual(['payments', 'policy', 'referral', 'thanks', 'socials'])
  })

  it('lines and box, with thickness and round corners', async () => {
    partTools('policy', { ...emptyDesign(), decor: { policy: { box: 'solid' } } })
    expect(find('decor-rounded').exists()).toBe(true)
    await find('decor-rounded').find('input').setValue(true)
    await find('bigger-thickness').trigger('click')
    expect(design().decor.policy).toEqual({ box: 'solid', rounded: true, thickness: 3 })
    wrapper.vm.$.setupState.update({ ...design(), decor: { policy: { ...design().decor.policy, below: 'none' } } })
    await flushPromises()
    expect(design().decor.policy.below).toBe('none')
  })

  it('the date format is with the date part only; the header doesn\'t move', () => {
    partTools('meta')
    expect(find('date-format').exists()).toBe(true)
    expect(find('move-up').exists()).toBe(false)
    wrapper.unmount()
    partTools('header')
    expect(find('date-format').exists()).toBe(false)
    expect(find('part-printed').exists()).toBe(false)
  })

  it('a part left out says so', () => {
    partTools('policy', toggleSection(setElement(emptyDesign(), 'policy_title', { text: 'X' }), 'policy'))
    expect(find('part-printed').text()).toContain('Print this part')
    expect(find('reset-part').exists()).toBe(true)
  })
})
