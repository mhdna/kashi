import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

// v-dialog reads the visual viewport, which the test DOM lacks.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const sendCommand = vi.fn()
const success = vi.fn()
const error = vi.fn()
vi.mock('@/composables/useBranchCommands', () => ({ useBranchCommands: () => ({ sendCommand }) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success, error }) }))

const { default: ReturnEditDialog } = await import('./ReturnEditDialog.vue')

const invoice = { kind: 'return', client_ref: 'ref-68', branch_id: 1, grand_total: -4100, branch_invoice_code: 'RS/CT/2026/00068' }
const items = [
  { quantity: -1, unit_price: 900, line_total: 0, barcode: 'b-pants', product_name: 'Kids Boy Jogging Pants', product_code: 'BOC5KBJOP9' },
  { quantity: -1, unit_price: 1500, line_total: -1200, barcode: 'b-hoodie', product_name: 'Kids Boy Hoodie', product_code: 'BOC5KBH15' },
  { quantity: -1, unit_price: 2900, line_total: -2900, barcode: 'b-jacket', product_name: 'Kids Girl Jacket', product_code: 'CA24KGJK29' },
]

let wrapper
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks(); document.body.innerHTML = '' })

function open () {
  wrapper = mount(ReturnEditDialog, {
    props: { modelValue: true, invoice, items, branchName: 'Centro' },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
}
const inputs = () => [...document.querySelectorAll('[data-test="refund-row"] input')]
const sendButton = () => [...document.querySelectorAll('button')].find(b => b.textContent.includes('Send to till'))

describe('ReturnEditDialog', () => {
  it('shows each returned item with what it gives back now', async () => {
    open()
    await flushPromises()
    expect(inputs().map(i => i.value)).toEqual(['0.00', '12.00', '29.00'])
    expect(document.body.textContent).toContain('Kids Boy Hoodie')
    expect(document.body.textContent).toContain('at most 15.00')
    expect(sendButton().disabled).toBe(true) // nothing changed yet
  })

  it('gives the hoodie its full $15 and sends that to the till', async () => {
    open()
    await flushPromises()
    const hoodie = inputs()[1]
    hoodie.value = '15'
    hoodie.dispatchEvent(new Event('input'))
    await flushPromises()
    expect(document.querySelector('[data-test="refund-total"]').textContent).toContain('-44.00')
    expect(document.querySelector('[data-test="refund-total"]').textContent).toContain('3.00 more back')
    sendButton().click()
    await flushPromises()
    expect(sendCommand).toHaveBeenCalledWith(1, 'edit_return_invoice', {
      client_ref: 'ref-68', lines: [{ position: 1, barcode: 'b-hoodie', line_total: -1500 }], reason: '',
    })
    expect(success).toHaveBeenCalled()
  })

  it('won\'t send more than an item\'s price', async () => {
    open()
    await flushPromises()
    const hoodie = inputs()[1]
    hoodie.value = '16'
    hoodie.dispatchEvent(new Event('input'))
    await flushPromises()
    expect(document.body.textContent).toContain('At most 15.00, its price')
    expect(sendButton().disabled).toBe(true)
  })
})
