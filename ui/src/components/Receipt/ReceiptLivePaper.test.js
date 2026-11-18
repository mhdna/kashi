import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptLivePaper from './ReceiptLivePaper.vue'

let wrapper
afterEach(() => { wrapper?.unmount() })

const marks = [{ group: 'header', top: 0, bottom: 200 }, { group: 'thanks', top: 900, bottom: 1000 }]
function open (props = {}) {
  wrapper = mount(ReceiptLivePaper, {
    props: { image: 'data:image/png;base64,AAAA', height: 1000, marks, groups: catalog.groups, ...props },
    global: { plugins: [vuetify()] },
  })
  return wrapper
}

describe('ReceiptLivePaper', () => {
  it('shows the till\'s drawing and a place to click for each part, where it is', () => {
    open()
    expect(wrapper.find('[data-test="receipt-image"]').attributes('src')).toBe('data:image/png;base64,AAAA')
    const thanks = wrapper.find('[data-test="pick-thanks"]')
    expect(thanks.element.style.top).toBe('90%')
    expect(thanks.element.style.height).toBe('10%')
    expect(thanks.attributes('aria-label')).toBe('Change Thank-you line')
  })
  it('a click picks the part, and the picked one is outlined', async () => {
    open({ picked: 'header' })
    await wrapper.find('[data-test="pick-thanks"]').trigger('click')
    expect(wrapper.emitted('pick')).toEqual([['thanks']])
    expect(wrapper.find('[data-test="pick-header"]').classes()).toContain('rlive-part--picked')
  })
  it('read-only: nothing to pick', async () => {
    open({ editable: false })
    await wrapper.find('[data-test="pick-thanks"]').trigger('click')
    expect(wrapper.emitted('pick')).toBeUndefined()
  })
  it('says it is drawing before the first picture', () => {
    open({ image: '' })
    expect(wrapper.text()).toContain('Drawing the receipt')
  })
})

describe('ReceiptLivePaper typing on the paper', () => {
  const lines = [
    { key: 'payments_title', group: 'payments', top: 300, bottom: 330, text: 'PAYMENT DETAILS', size_px: 26, align: 'center' },
    { key: 'policy_text', group: 'policy', top: 500, bottom: 560, text: 'Exchanges accepted within 3 days.', size_px: 22, align: 'center' },
  ]
  function openLines (props = {}) {
    wrapper = mount(ReceiptLivePaper, {
      props: { image: 'data:image/png;base64,AAAA', height: 1000, marks, lines, groups: catalog.groups, attachTo: document.body, ...props },
      global: { plugins: [vuetify()] },
      attachTo: document.body,
    })
    return wrapper
  }

  it('the reported case: click words, type, Enter: the new words are sent, and the part picked', async () => {
    openLines()
    await wrapper.find('[data-test="words-payments_title"]').trigger('click')
    expect(wrapper.emitted('pick')).toEqual([['payments']])
    const box = wrapper.find('[data-test="edit-words"]')
    expect(box.element.value).toBe('PAYMENT DETAILS')
    expect(box.element.style.textAlign).toBe('center')
    await box.setValue('  PAID WITH ')
    await box.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('words')).toEqual([['payments_title', 'PAID WITH']])
    expect(wrapper.find('[data-test="edit-words"]').exists()).toBe(false)
  })

  it('Esc, or the same words, sends nothing; long words get a box of lines', async () => {
    openLines()
    await wrapper.find('[data-test="words-payments_title"]').trigger('click')
    await wrapper.find('[data-test="edit-words"]').setValue('Other')
    await wrapper.find('[data-test="edit-words"]').trigger('keydown', { key: 'Escape' })
    await wrapper.find('[data-test="words-payments_title"]').trigger('click')
    await wrapper.find('[data-test="edit-words"]').trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('words')).toBeUndefined()
    await wrapper.find('[data-test="words-policy_text"]').trigger('click')
    expect(wrapper.find('[data-test="edit-words"]').element.tagName).toBe('TEXTAREA')
  })

  it('starts from the design\'s own words when there are some', async () => {
    openLines({ wordsOf: key => (key === 'payments_title' ? 'MINE' : '') })
    await wrapper.find('[data-test="words-payments_title"]').trigger('click')
    expect(wrapper.find('[data-test="edit-words"]').element.value).toBe('MINE')
  })

  it('read-only: no words to click', () => {
    openLines({ editable: false })
    expect(wrapper.find('[data-test="words-payments_title"]').exists()).toBe(false)
  })

  it('a part\'s name shows beside the paper, not over its words', () => {
    openLines({ picked: 'thanks' })
    const label = wrapper.find('[data-test="pick-thanks"] .rlive-label')
    expect(label.exists()).toBe(true)
  })
})
