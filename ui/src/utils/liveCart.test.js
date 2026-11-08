import { describe, expect, it } from 'vitest'
import {
  buildOverrideLines,
  canSetLine,
  draftLinesFromOverrides,
  lineKey,
  lockedLineText,
  unknownDealItems,
} from './liveCart'

const shirt = { barcode: 'bc-A', name: 'Shirt', qty: 1 }
const u1 = { barcode: 'UNKNOWN', name: 'Unknown item', qty: 1, unknown: true, lineRef: 'u1' }
const u2 = { barcode: 'UNKNOWN', name: 'Unknown item', qty: 2, unknown: true, lineRef: 'u2' }
const oldTillUnknown = { barcode: 'UNKNOWN', name: 'Unknown item', qty: 1, unknown: true, lineRef: '' }

describe('which lines kashi can change', () => {
  it('takes products by barcode and unknown items by their ref', () => {
    expect([shirt, u1, u2].map(l => lineKey(l))).toEqual(['bc-A', 'ref:u1', 'ref:u2'])
    expect([shirt, u1, u2].every(l => canSetLine(l))).toBe(true)
  })

  it("can't change an unknown item from a till too old to name it, nor returned or barcode-less lines", () => {
    expect(canSetLine(oldTillUnknown)).toBe(false)
    expect(lockedLineText(oldTillUnknown)).toBe('Update this till to set unknown items')
    expect(canSetLine({ ...shirt, qty: -1 })).toBe(false)
    expect(canSetLine({ ...shirt, barcode: '' })).toBe(false)
    expect(lockedLineText({ ...shirt, qty: -1 })).toBe('—')
  })
})

describe('the override sent to the till', () => {
  it('sends each unknown item on its own, by its ref', () => {
    const lines = buildOverrideLines([shirt, u1, u2], {
      'bc-A': { price: '12', discount: '', noDeal: true },
      'ref:u1': { price: '25.50', discount: '10', unknownDeal: '' },
      'ref:u2': { price: '', discount: '', unknownDeal: 'b2g1_30' },
    }, false)
    expect(lines).toEqual([
      { barcode: 'bc-A', price_cents: 1200, no_deal: true },
      { barcode: 'UNKNOWN', line_ref: 'u1', price_cents: 2550, discount: 10 },
      { barcode: 'UNKNOWN', line_ref: 'u2', unknown_deal: 'b2g1_30' },
    ])
  })

  it('leaves out lines with nothing typed, and deal choices while every deal is off', () => {
    expect(buildOverrideLines([shirt, u1], {
      'bc-A': { price: '', discount: '', noDeal: true },
      'ref:u1': { price: '', discount: '', unknownDeal: 'none' },
    }, true)).toEqual([])
  })

  it("refuses a deal on top of a discount, a price of 0 and a discount past 100", () => {
    expect(() => buildOverrideLines([u1], { 'ref:u1': { discount: '10', unknownDeal: 'b2g1_20' } }, false))
      .toThrow("a deal doesn't go with a discount")
    expect(buildOverrideLines([u1], { 'ref:u1': { discount: '10', unknownDeal: 'none' } }, false))
      .toEqual([{ barcode: 'UNKNOWN', line_ref: 'u1', discount: 10, unknown_deal: 'none' }])
    expect(() => buildOverrideLines([u1], { 'ref:u1': { price: '0' } }, false)).toThrow('price above 0')
    expect(() => buildOverrideLines([shirt], { 'bc-A': { discount: '101' } }, false)).toThrow('0 to 100')
  })

  it('reads back what the till applies into the same draft', () => {
    const sent = [
      { barcode: 'bc-A', price_cents: 1200, no_deal: true },
      { barcode: 'UNKNOWN', line_ref: 'u1', discount: 10, unknown_deal: 'none' },
    ]
    const draft = draftLinesFromOverrides({ lines: sent })
    expect(draft).toEqual({
      'bc-A': { price: '12.00', discount: '', noDeal: true, unknownDeal: '' },
      'ref:u1': { price: '', discount: '10', noDeal: false, unknownDeal: 'none' },
    })
    expect(buildOverrideLines([shirt, u1], draft, false)).toEqual(sent)
  })
})

describe('deal choices', () => {
  it("name the till's deal", () => {
    expect(unknownDealItems('B1G1').map(c => c.title)).toEqual(['As rung', 'B1G1 · else 30%', 'B1G1 · else 20%', 'No deal'])
    expect(unknownDealItems('').map(c => c.value)).toEqual(['', 'b2g1_30', 'b2g1_20', 'none'])
  })
})
