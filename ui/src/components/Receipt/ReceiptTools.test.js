import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { designJSON, emptyDesign, setElement, toggleSection } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptFontSelect from './ReceiptFontSelect.vue'
import ReceiptLineTools from './ReceiptLineTools.vue'
import ReceiptPartTools from './ReceiptPartTools.vue'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

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

  it('the reported case: the whole exchange policy is left out from either of its lines', async () => {
    for (const key of ['policy_title', 'policy_text']) {
      lineTools(key)
      expect(find('line-part-hide').text()).toBe('Leave out the whole exchange policy')
      await find('line-part-hide').trigger('click')
      expect(design().hide).toEqual(['policy'])
      expect(JSON.parse(designJSON(design())).hide).toEqual(['policy'])
      wrapper.unmount()
      wrapper = null
    }
  })

  it('offers leaving out the whole part exactly on lines of a part that can be', () => {
    for (const e of catalog.elements) {
      lineTools(e.key)
      expect(find('line-part-hide').exists(), e.key).toBe(Boolean(catalog.groups.find(g => g.key === e.group).section))
      wrapper.unmount()
      wrapper = null
    }
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

  it('what each item prints: name, colour and size, code, before and after price', async () => {
    partTools('columns')
    const row = find('item-parts')
    expect(row.exists()).toBe(true)
    expect(row.findAll('button').map(b => b.text().trim())).toEqual(['Name', 'Colour and size', 'Code', 'Before and after price'])
    const pressed = key => find(`item-part-${key}`).classes().includes('v-btn--active')
    // As the till prints today: everything but the code.
    expect([pressed('item_name'), pressed('item_variant'), pressed('item_code'), pressed('item_price_note')]).toEqual([true, true, false, true])

    await find('item-part-item_code').trigger('click')
    await find('item-part-item_name').trigger('click')
    await find('item-part-item_price_note').trigger('click')
    expect(design().elements).toEqual({ item_code: { show: true }, item_name: { hide: true }, item_price_note: { hide: true } })
    expect([pressed('item_name'), pressed('item_code'), pressed('item_price_note')]).toEqual([false, true, false])
    expect(JSON.parse(designJSON(design())).elements).toEqual({ item_code: { show: true }, item_name: { hide: true }, item_price_note: { hide: true } })

    // Back on: nothing left over to save.
    await find('item-part-item_code').trigger('click')
    await find('item-part-item_name').trigger('click')
    await find('item-part-item_price_note').trigger('click')
    expect(design().elements).toEqual({})
    expect(designJSON(design())).toBe('')

    // Default puts them back too.
    await find('item-part-item_variant').trigger('click')
    expect(find('reset-part').exists()).toBe(true)
    await find('reset-part').trigger('click')
    expect(design().elements).toEqual({})
  })

  it('only the items part has item switches', () => {
    for (const group of ['header', 'meta', 'totals', 'payments', 'thanks']) {
      partTools(group)
      expect(find('item-parts').exists(), group).toBe(false)
      wrapper.unmount()
    }
  })

  it('a part left out says so', () => {
    partTools('policy', toggleSection(setElement(emptyDesign(), 'policy_title', { text: 'X' }), 'policy'))
    expect(find('part-printed').text()).toContain('Print this part')
    expect(find('reset-part').exists()).toBe(true)
  })
})

describe('fonts', () => {
  it('the reported case: a line takes a font from the list, each name in its own letters, and Default undoes it', async () => {
    lineTools('store_name')
    await find('line-font').find('.v-field').trigger('mousedown')
    await flushPromises()
    const names = [...document.querySelectorAll('.rfont-name')]
    expect(names.length).toBe(catalog.fonts.length)
    const pacifico = names.find(n => n.textContent === 'Pacifico')
    expect(pacifico.style.fontFamily).toContain('kashi-receipt-pacifico')
    pacifico.closest('.v-list-item').click()
    await flushPromises()
    expect(design().elements.store_name).toEqual({ font: 'pacifico' })
    expect(find('line-font').text()).toContain('Pacifico')
    await find('line-default').trigger('click')
    expect(design().elements.store_name).toBeUndefined()
  })

  it('lines that can\'t change font show no list', () => {
    lineTools('invoice_label_sale')
    expect(find('line-font').exists()).toBe(false)
  })

  it('a part sets every line\'s font at once; clearing it is the look\'s own', async () => {
    partTools('totals')
    wrapper.findComponent(ReceiptFontSelect).vm.$emit('update:modelValue', 'bebas-neue')
    await flushPromises()
    expect(design().groups.totals).toEqual({ font: 'bebas-neue' })
    expect(find('reset-part').exists()).toBe(true)
    wrapper.findComponent(ReceiptFontSelect).vm.$emit('update:modelValue', '')
    await flushPromises()
    expect(design().groups.totals?.font ?? '').toBe('')
    expect(designJSON(design())).toBe('')
  })
})

describe('Arabic', () => {
  it('the reported case: a line can be English, Arabic or both', async () => {
    lineTools('policy_text')
    expect(find('line-lang').exists()).toBe(true)
    await button('line-lang', 'Both').trigger('click')
    expect(design().elements.policy_text).toEqual({ lang: 'both' })
    await button('line-lang', 'عربي').trigger('click')
    expect(design().elements.policy_text).toEqual({ lang: 'ar' })
    await button('line-lang', 'EN').trigger('click')
    expect(design().elements.policy_text).toBeUndefined()
  })

  it('a part puts all its lines in Arabic at once', async () => {
    partTools('totals')
    await button('group-lang', 'Both').trigger('click')
    expect(design().groups.totals).toEqual({ lang: 'both' })
  })

  it('a column title has no language choice yet', () => {
    lineTools('col_item')
    expect(find('line-lang').exists()).toBe(false)
  })
})

describe('white on black', () => {
  it('the reported case: a line can go on black and back', async () => {
    lineTools('thanks')
    await find('line-dark').trigger('click')
    expect(design().elements.thanks).toEqual({ dark: true })
    await find('line-dark').trigger('click')
    expect(design().elements.thanks).toBeUndefined()
  })

  it('the referral code is on black by default; switching it off is stored', async () => {
    lineTools('referral_title')
    expect(find('line-dark').text()).toContain('White on black')
    await find('line-dark').trigger('click')
    expect(design().elements.referral_title).toEqual({ dark: false })
  })

  it('a whole part on black or on white', async () => {
    partTools('totals')
    await button('group-dark', 'On black').trigger('click')
    expect(design().groups.totals).toEqual({ dark: true })
    await button('group-dark', 'On white').trigger('click')
    expect(design().groups.totals).toEqual({ dark: false })
  })
})
