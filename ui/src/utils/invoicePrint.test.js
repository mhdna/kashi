import { describe, expect, it } from 'vitest'
import { canPrintOnTill, printOnTillPayload, printOnTillSentMessage } from './invoicePrint'

// RS/CT/2026/00068 as kashi's branch_invoices row has it.
const returnInvoice = {
  id: 123_302,
  branch_id: 1,
  register_id: 3,
  client_ref: '8912abdf-ed49-46cf-be63-84b63ce7f5f2',
  kind: 'return',
  branch_invoice_code: 'RS/CT/2026/00068',
  grand_total: -4100,
  historical: false,
  cancelled_at: { Time: '0001-01-01T00:00:00Z', Valid: false },
}

describe('canPrintOnTill', () => {
  it('prints the return from the report', () => {
    expect(canPrintOnTill(returnInvoice)).toBe(true)
  })

  it('prints every kind, cancelled or not', () => {
    for (const kind of ['sales', 'return', 'exchange']) {
      expect(canPrintOnTill({ ...returnInvoice, kind })).toBe(true)
      expect(canPrintOnTill({ ...returnInvoice, kind, cancelled_at: { Time: '2026-11-07T18:00:00Z', Valid: true } })).toBe(true)
    }
  })

  it('never prints an invoice imported from the old POS', () => {
    expect(canPrintOnTill({ ...returnInvoice, historical: true })).toBe(false)
  })

  it('needs a client_ref to name the invoice to the till', () => {
    for (const client_ref of [undefined, null, '', '   ', 42, { String: 'x', Valid: true }]) {
      expect(canPrintOnTill({ ...returnInvoice, client_ref })).toBe(false)
    }
  })

  it('is false for no invoice at all', () => {
    for (const invoice of [null, undefined, '', 0, 'RS/CT/2026/00068']) {
      expect(canPrintOnTill(invoice)).toBe(false)
    }
  })

  // Hundreds of random rows: printable exactly when not historical and the
  // client_ref has text, and then the payload is that trimmed ref and nothing else.
  it('holds for random rows', () => {
    let seed = 7
    const rand = n => {
      seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648
      return seed % n
    }
    const refs = ['', ' ', 'abc', '  abc  ', null, undefined, 'e2c1-77']
    for (let i = 0; i < 500; i++) {
      const invoice = {
        ...returnInvoice,
        kind: ['sales', 'return', 'exchange'][rand(3)],
        historical: [true, false, undefined][rand(3)],
        client_ref: refs[rand(refs.length)],
      }
      const hasRef = typeof invoice.client_ref === 'string' && invoice.client_ref.trim() !== ''
      const want = invoice.historical !== true && hasRef
      expect(canPrintOnTill(invoice)).toBe(want)
      if (want) {
        expect(printOnTillPayload(invoice)).toEqual({ client_ref: invoice.client_ref.trim() })
      }
    }
  })
})

describe('printOnTillSentMessage', () => {
  it('names the invoice and the branch', () => {
    expect(printOnTillSentMessage(returnInvoice, 'Centro'))
      .toBe('RS/CT/2026/00068 sent to Centro — it prints within a few seconds if the till is online.')
  })

  it('still reads without a code or a branch name', () => {
    expect(printOnTillSentMessage({}, '')).toBe('The invoice sent to the till — it prints within a few seconds if the till is online.')
  })
})

// Rows as GET /branch_invoices really sends them, written by the Go test
// TestBranchInvoiceRowsFixtureForPrintOnTill.
describe('rows from the API', () => {
  it('prints the till return and not the old-POS import', async () => {
    const { default: rows } = await import('./testdata/branch_invoice_rows.json')
    const [ret, imported] = rows
    expect(ret.branch_invoice_code).toBe('RS/CT/2026/00068')
    expect(canPrintOnTill(ret)).toBe(true)
    expect(printOnTillPayload(ret)).toEqual({ client_ref: '8912abdf-ed49-46cf-be63-84b63ce7f5f2' })
    expect(canPrintOnTill(imported)).toBe(false)
  })
})
