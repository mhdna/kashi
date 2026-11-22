import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { localDateString } from '@/utils/exchangeOpening'

// v-dialog reads the visual viewport, which the test DOM lacks.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const sendCommand = vi.fn(async () => [{ id: 51 }])
let tillResult = { status: 'success', error: '' }
const waitForCommand = vi.fn(async () => tillResult)
const error = vi.fn()
vi.mock('@/composables/useBranchCommands', () => ({ useBranchCommands: () => ({ sendCommand }) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ error, success: vi.fn() }) }))
vi.mock('@/composables/usePaymentEntry', () => ({ usePaymentEntry: () => ({ waitForCommand }) }))

const { default: ExchangeOpeningDialog } = await import('./ExchangeOpeningDialog.vue')

const invoice = { id: 9, branch_id: 1, kind: 'sales', client_ref: 'sale-3022', branch_invoice_code: 'SA/CT/2026/03022' }
function dayFromToday (n) { const d = new Date(); return localDateString(new Date(d.getFullYear(), d.getMonth(), d.getDate() + n)) }

let wrapper
afterEach(() => {
  wrapper?.unmount()
  vi.clearAllMocks()
  tillResult = { status: 'success', error: '' }
  document.body.innerHTML = ''
})

async function open () {
  wrapper = mount(ExchangeOpeningDialog, {
    props: { modelValue: true, invoice, branchName: 'Centro' },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  await flushPromises()
}
const el = sel => document.querySelector(`[data-test="${sel}"]`)
async function pick (sel) { el(sel).querySelector('input').click(); await flushPromises() }
async function typeDays (value) {
  const input = el('days').querySelector('input')
  input.value = value
  input.dispatchEvent(new Event('input'))
  await flushPromises()
}
async function send () { el('send-opening').click(); await flushPromises() }

describe('ExchangeOpeningDialog', () => {
  it('opens the sale for today only by default and tells the till', async () => {
    await open()
    expect(el('until').textContent).toContain('Open until the end of')
    await send()
    expect(sendCommand).toHaveBeenCalledWith(1, 'open_invoice_exchange', { client_ref: 'sale-3022', until_date: dayFromToday(0), reason: '' })
    expect(waitForCommand).toHaveBeenCalledWith(1, 51, expect.any(Function))
    expect(el('result').textContent).toContain('Open for exchange on the till')
  })

  it('opens it for a number of days counting today', async () => {
    await open()
    await pick('mode-days')
    await typeDays('5')
    await send()
    expect(sendCommand.mock.calls[0][2].until_date).toBe(dayFromToday(4))
  })

  it('can not send a day count the till would refuse', async () => {
    await open()
    await pick('mode-days')
    for (const bad of ['0', '-2', '400']) {
      await typeDays(bad)
      expect(el('send-opening').disabled).toBe(true)
      expect(el('until').textContent).toContain('Enter between 1 and')
    }
    expect(sendCommand).not.toHaveBeenCalled()
  })

  it("shows the till's refusal and lets the admin try again", async () => {
    tillResult = { status: 'failed', error: 'this sale was cancelled' }
    await open()
    await send()
    expect(el('result').textContent).toContain('this sale was cancelled')
    const again = [...document.querySelectorAll('button')].find(b => b.textContent.includes('Edit and send again'))
    again.click()
    await flushPromises()
    expect(el('send-opening')).not.toBeNull()
  })

  it('says the till will do it later when it is offline', async () => {
    tillResult = { status: 'pending', error: '' }
    await open()
    await send()
    expect(el('result').textContent).toContain('the till may be offline')
  })

  it('reports a refusal from kashi', async () => {
    sendCommand.mockRejectedValueOnce(new Error('only a sale can be opened for exchange'))
    await open()
    await send()
    expect(error).toHaveBeenCalledWith('only a sale can be opened for exchange')
  })
})
