import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import InvoicePrintButton from './InvoicePrintButton.vue'

const state = vi.hoisted(() => ({
  canEditBranches: true,
  printBranchInvoice: null,
  success: null,
  error: null,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ canEdit: section => section === 'branches' && state.canEditBranches }),
}))
vi.mock('@/composables/useBranchCommands', () => ({
  useBranchCommands: () => ({ printBranchInvoice: state.printBranchInvoice }),
}))
vi.mock('@/composables/useNotify', () => ({
  useNotify: () => ({ success: state.success, error: state.error }),
}))

// RS/CT/2026/00068, the return from the report.
const returnInvoice = {
  id: 123_302,
  branch_id: 1,
  client_ref: '8912abdf-ed49-46cf-be63-84b63ce7f5f2',
  kind: 'return',
  branch_invoice_code: 'RS/CT/2026/00068',
  historical: false,
}

function mountButton (props) {
  return mount(InvoicePrintButton, {
    props: { invoice: returnInvoice, branchName: 'Centro', ...props },
    global: { plugins: [vuetify()] },
  })
}

beforeEach(() => {
  state.canEditBranches = true
  state.printBranchInvoice = vi.fn().mockResolvedValue([{ id: 1 }])
  state.success = vi.fn()
  state.error = vi.fn()
})

describe('InvoicePrintButton', () => {
  it('sends the return to its till and says so', async () => {
    const wrapper = mountButton()
    const button = wrapper.find('button.invoice-print-button')
    expect(button.exists()).toBe(true)
    expect(button.attributes('title')).toBe('Print this receipt on the till that rang it up')

    await button.trigger('click')
    await flushPromises()

    expect(state.printBranchInvoice).toHaveBeenCalledTimes(1)
    expect(state.printBranchInvoice).toHaveBeenCalledWith(returnInvoice)
    expect(state.success).toHaveBeenCalledWith('RS/CT/2026/00068 sent to Centro — it prints within a few seconds if the till is online.')
    expect(state.error).not.toHaveBeenCalled()
  })

  it('shows "Print on till" in the details dialog and an icon in a row', () => {
    expect(mountButton({ text: true }).text()).toContain('Print on till')
    expect(mountButton().text()).not.toContain('Print on till')
  })

  it("doesn't open the row's details when clicked in a row", async () => {
    const rowClick = vi.fn()
    const wrapper = mount({
      components: { InvoicePrintButton },
      setup: () => ({ invoice: returnInvoice, rowClick }),
      template: '<div @click="rowClick"><InvoicePrintButton :invoice="invoice" branch-name="Centro" /></div>',
    }, { global: { plugins: [vuetify()] } })
    await wrapper.find('button.invoice-print-button').trigger('click')
    await flushPromises()
    expect(rowClick).not.toHaveBeenCalled()
    expect(state.printBranchInvoice).toHaveBeenCalledTimes(1)
  })

  it('is hidden for an invoice imported from the old POS', () => {
    const wrapper = mountButton({ invoice: { ...returnInvoice, historical: true } })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('is hidden without Branches edit', () => {
    state.canEditBranches = false
    expect(mountButton().find('button').exists()).toBe(false)
  })

  it('is hidden while the dialog has no invoice yet', () => {
    expect(mountButton({ invoice: null }).find('button').exists()).toBe(false)
  })

  it("shows kashi's refusal and sends nothing twice", async () => {
    state.printBranchInvoice = vi.fn().mockRejectedValue(new Error('this invoice was imported from the old POS; no till has it'))
    const wrapper = mountButton()
    await wrapper.find('button.invoice-print-button').trigger('click')
    await flushPromises()
    expect(state.error).toHaveBeenCalledWith('this invoice was imported from the old POS; no till has it')
    expect(state.success).not.toHaveBeenCalled()
  })

  it('a second click while sending sends one command', async () => {
    let finish
    state.printBranchInvoice = vi.fn(() => new Promise(resolve => {
      finish = resolve
    }))
    const wrapper = mountButton()
    const button = wrapper.find('button.invoice-print-button')
    await button.trigger('click')
    await button.trigger('click')
    finish([])
    await flushPromises()
    expect(state.printBranchInvoice).toHaveBeenCalledTimes(1)
    expect(state.success).toHaveBeenCalledTimes(1)
  })
})
