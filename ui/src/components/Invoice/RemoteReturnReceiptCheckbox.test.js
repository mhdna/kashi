import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import RemoteReturnReceiptCheckbox from './RemoteReturnReceiptCheckbox.vue'

function mountBox (modelValue) {
  return mount(RemoteReturnReceiptCheckbox, { props: { modelValue }, global: { plugins: [vuetify()] } })
}

describe('RemoteReturnReceiptCheckbox', () => {
  it('reads "Print the receipt on the till", with what unticking does', () => {
    const wrapper = mountBox(true)
    expect(wrapper.text()).toContain('Print the receipt on the till')
    expect(wrapper.text()).toContain("Unticked, the till doesn't print it.")
    expect(wrapper.find('input[type=checkbox]').element.checked).toBe(true)
    expect(mountBox(false).find('input[type=checkbox]').element.checked).toBe(false)
  })

  it('unticking and ticking reach the dialog', async () => {
    const wrapper = mountBox(true)
    const box = wrapper.find('input[type=checkbox]')
    await box.setValue(false)
    await wrapper.setProps({ modelValue: false })
    await box.setValue(true)
    expect(wrapper.emitted('update:modelValue')).toEqual([[false], [true]])
  })
})
