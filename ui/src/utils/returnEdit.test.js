import { describe, expect, it } from 'vitest'
import { canEditRefund, creditProblem, parseCredit, refundEdit, refundEditPayload, refundRows } from './returnEdit'

// RS/CT/2026/00068 as kashi shows it.
const invoice = { kind: 'return', client_ref: 'ref-68', branch_id: 1, grand_total: -4100, branch_invoice_code: 'RS/CT/2026/00068' }
const items = [
  { quantity: -1, unit_price: 900, line_total: 0, deal_discount: -900, barcode: '8000000113368', product_name: 'Kids Boy Jogging Pants', product_code: 'BOC5KBJOP9' },
  { quantity: -1, unit_price: 1500, line_total: -1200, barcode: '8000000115959', product_name: 'Kids Boy Hoodie', product_code: 'BOC5KBH15' },
  { quantity: -1, unit_price: 2900, line_total: -2900, barcode: '8000000072931', product_name: 'Kids Girl Jacket', product_code: 'CA24KGJK29' },
]

describe('edit refund', () => {
  it('makes RS/CT/2026/00068 give back $44', () => {
    const rows = refundRows(items)
    expect(rows.map(r => [r.position, r.credit, r.max])).toEqual([[0, 0, 900], [1, 1200, 1500], [2, 2900, 2900]])
    const edit = refundEdit(invoice, rows, { 0: 0, 1: 1500, 2: 2900 })
    expect(edit).toEqual({ lines: [{ position: 1, barcode: '8000000115959', line_total: -1500 }], oldTotal: -4100, newTotal: -4400, moved: -300 })
    expect(refundEditPayload(invoice, edit.lines, '  Customer keeps the deal ')).toEqual({
      client_ref: 'ref-68', lines: edit.lines, reason: 'Customer keeps the deal',
    })
  })

  it('sends nothing when nothing changed', () => {
    const rows = refundRows(items)
    expect(refundEdit(invoice, rows, { 0: 0, 1: 1200, 2: 2900 }).lines).toEqual([])
  })

  it('can give back less too', () => {
    const edit = refundEdit(invoice, refundRows(items), { 2: 2000 })
    expect(edit.newTotal).toBe(-3200)
    expect(edit.lines).toEqual([{ position: 2, barcode: '8000000072931', line_total: -2000 }])
  })

  it('only lists returned lines, keeping their position among all lines', () => {
    const rows = refundRows([{ quantity: 1, unit_price: 1900, line_total: 1900 }, ...items])
    expect(rows.map(r => r.position)).toEqual([1, 2, 3])
  })

  it.each([
    ['15', 1500], ['15.5', 1550], ['15.00', 1500], ['0', 0], [' 3 ', 300],
    ['', null], ['-3', null], ['abc', null], ['1.234', null], ['1,5', null],
  ])('reads %j as %j', (text, cents) => {
    expect(parseCredit(text)).toBe(cents)
  })

  it('never gives back more than the price', () => {
    const hoodie = refundRows(items)[1]
    expect(creditProblem(hoodie, 1500)).toBe('')
    expect(creditProblem(hoodie, 1501)).toBe('At most 15.00, its price')
    expect(creditProblem(hoodie, -1)).toBe('Can\'t be negative')
    expect(creditProblem(hoodie, null)).not.toBe('')
  })

  it('only offers it on a till return', () => {
    expect(canEditRefund(invoice)).toBe(true)
    expect(canEditRefund({ ...invoice, kind: 'sales' })).toBe(false)
    expect(canEditRefund({ ...invoice, kind: 'exchange' })).toBe(true)
    expect(canEditRefund({ ...invoice, historical: true })).toBe(false)
    expect(canEditRefund({ ...invoice, client_ref: '' })).toBe(false)
    expect(canEditRefund(null)).toBe(false)
  })

  it('the preview total always equals the old total plus what the lines moved', () => {
    let seed = 3
    const rand = () => ((seed = (seed * 1_103_515_245 + 12_345) & 0x7f_ff_ff_ff) / 0x7f_ff_ff_ff)
    for (let run = 0; run < 2000; run++) {
      const its = Array.from({ length: 1 + Math.floor(rand() * 4) }, () => {
        const price = 100 + Math.floor(rand() * 5000)
        return { quantity: -1, unit_price: price, line_total: -Math.floor(rand() * (price + 1)) }
      })
      const total = its.reduce((s, i) => s + i.line_total, 0)
      const rows = refundRows(its)
      const credits = Object.fromEntries(rows.map(r => [r.position, Math.floor(rand() * (r.max + 1))]))
      const edit = refundEdit({ grand_total: total }, rows, credits)
      const after = its.reduce((s, i, idx) => s - (credits[idx] ?? -i.line_total), 0)
      expect(edit.newTotal).toBe(after)
      for (const l of edit.lines) {expect(l.line_total).toBeLessThanOrEqual(0)}
    }
  })
})
