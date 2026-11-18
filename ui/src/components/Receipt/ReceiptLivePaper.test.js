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
