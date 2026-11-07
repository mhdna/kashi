import { describe, expect, it } from 'vitest'
import { remoteReturnPayload } from './remoteReturnPayload'

// The sale RS/CT/2026/00068 returned from: SA/CT/2026/02923.
const sale = { id: 123_286, client_ref: '46a9e27e-1ad6-48c4-a798-3d95121e3c01' }
const items = [
  { branch_product_id: 11_429, quantity: 1, unit_price: 900 },
  { branch_product_id: 11_684, quantity: 1, unit_price: 1500 },
  { branch_product_id: 7934, quantity: 1, unit_price: 2900 },
]

describe('remoteReturnPayload', () => {
  it('ticked (the default) sends no print_receipt: the till decides', () => {
    expect(remoteReturnPayload({ invoice: sale, reason: '', items })).toEqual({
      branch_invoice_id: 123_286,
      client_ref: sale.client_ref,
      reason: '',
      items,
      replacements: [],
    })
    expect(remoteReturnPayload({ invoice: sale, items, printReceipt: true })).not.toHaveProperty('print_receipt')
  })

  it('unticked tells the till not to print', () => {
    expect(remoteReturnPayload({ invoice: sale, items, printReceipt: false }).print_receipt).toBe(false)
  })

  it('an exchange carries its replacements and the choice', () => {
    const replacements = [{ variant_id: 77, quantity: 2 }]
    const payload = remoteReturnPayload({ invoice: sale, reason: 'size', items: items.slice(0, 1), replacements, printReceipt: false })
    expect(payload).toEqual({
      branch_invoice_id: 123_286,
      client_ref: sale.client_ref,
      reason: 'size',
      items: items.slice(0, 1),
      replacements,
      print_receipt: false,
    })
  })

  it('a cancel sends cancel and no items, with the choice', () => {
    expect(remoteReturnPayload({ invoice: sale, reason: 'mistake', cancel: true, printReceipt: false })).toEqual({
      branch_invoice_id: 123_286,
      client_ref: sale.client_ref,
      reason: 'mistake',
      cancel: true,
      print_receipt: false,
    })
    expect(remoteReturnPayload({ invoice: sale, cancel: true })).not.toHaveProperty('print_receipt')
  })

  it('every combination: print_receipt is there exactly when unticked', () => {
    for (const cancel of [true, false]) {
      for (const printReceipt of [true, false, undefined]) {
        for (const reason of ['', undefined, null, 'x']) {
          const payload = remoteReturnPayload({ invoice: sale, reason, items, cancel, printReceipt })
          expect('print_receipt' in payload).toBe(printReceipt === false)
          expect(payload.reason).toBe(reason || '')
          expect('items' in payload).toBe(!cancel)
          expect(payload.client_ref).toBe(sale.client_ref)
        }
      }
    }
  })
})
