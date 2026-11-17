import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

// v-dialog reads the visual viewport, which the test DOM lacks.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const sendCommand = vi.fn(async () => [{ id: 41 }])
let tillResult = { status: 'success', error: '' }
const waitForCommand = vi.fn(async () => tillResult)
const error = vi.fn()
vi.mock('@/composables/useBranchCommands', () => ({ useBranchCommands: () => ({ sendCommand }) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ error, success: vi.fn() }) }))
vi.mock('@/composables/usePaymentEntry', () => ({
  usePaymentEntry: () => ({
    loadPaymentSetup: async () => ({
      methods: [{ name: 'Cash $$', currency: 'USD' }, { name: 'On Account', currency: 'USD' }, { name: 'Visa', currency: 'USD' }],
      currencies: [],
    }),
    waitForCommand,
  }),
}))

const { default: SettlementChangeDialog } = await import('./SettlementChangeDialog.vue')

// The reported sale: SA/CT/2026/03022, $65.56 On Account.
const invoice = { id: 9, branch_id: 1, kind: 'sales', client_ref: 'sale-3022', grand_total: 6556, branch_invoice_code: 'SA/CT/2026/03022' }

let wrapper
afterEach(() => {
  wrapper?.unmount()
  vi.clearAllMocks()
  tillResult = { status: 'success', error: '' }
  document.body.innerHTML = ''
})

async function open (payments = [{ account_name: 'On Account', amount: 6556 }]) {
  wrapper = mount(SettlementChangeDialog, {
    props: { modelValue: true, invoice, payments, branchName: 'Centro' },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  await flushPromises()
}
const el = sel => document.querySelector(`[data-test="${sel}"]`)
const tap = async sel => { el(sel).click(); await flushPromises() }
const sendButton = () => el('send-settlement')
const removeAll = async () => {
  for (const b of [...document.querySelectorAll('[aria-label^="Remove "]')]) {
    b.click()
    await flushPromises()
  }
}

describe('SettlementChangeDialog', () => {
  it('starts from how the sale was paid, already adding up', async () => {
    await open()
    expect(el('entries').textContent).toContain('On Account')
    expect(el('remaining').textContent).toBe('Done')
    expect(sendButton().disabled).toBe(false)
  })

  it('moves the sale to cash and Visa and sends that split to the till', async () => {
    await open()
    await removeAll()
    expect(sendButton().disabled).toBe(true)
    for (const k of '50') await tap(`key-${k}`)
    await tap('method-Cash $$')
    expect(sendButton().disabled).toBe(true) // $15.56 still to enter
    await tap('method-Visa')
    expect(sendButton().disabled).toBe(false)
    sendButton().click()
    await flushPromises()
    expect(sendCommand).toHaveBeenCalledWith(1, 'change_invoice_settlement', {
      client_ref: 'sale-3022',
      payments: [{ account_name: 'Cash $$', amount_cents: 5000 }, { account_name: 'Visa', amount_cents: 1556 }],
      reason: '',
    })
    expect(waitForCommand).toHaveBeenCalledWith(1, 41, expect.any(Function))
    expect(document.body.textContent).toContain('Settlement changed.')
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })

  it("shows the till's refusal and lets the admin fix it", async () => {
    tillResult = { status: 'failed', error: 'the client already paid $6.00 of this sale back; keep at least that much On Account' }
    await open()
    sendButton().click()
    await flushPromises()
    expect(document.body.textContent).toContain('keep at least that much On Account')
    expect(wrapper.emitted('changed')).toBeUndefined()
    const again = [...document.querySelectorAll('button')].find(b => b.textContent.includes('Edit and send again'))
    again.click()
    await flushPromises()
    expect(el('entries').textContent).toContain('On Account')
  })

  it('a till that is offline leaves it pending', async () => {
    tillResult = { status: 'pending', error: '' }
    await open()
    sendButton().click()
    await flushPromises()
    expect(document.body.textContent).toContain('the till may be offline')
  })
})
