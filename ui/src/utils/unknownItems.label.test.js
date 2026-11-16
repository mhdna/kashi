import { describe, expect, it } from 'vitest'
import rows from './testdata/unknown_item_views.json'
import { labelReading } from './unknownItems'

// rows are real backend views (TestUnknownItemViewsGolden): the user's
// label sold at $12, the same sold at $10, no price, a failed read, reader off.
describe('labelReading', () => {
  it("the user's label, sold at the label's price: name, details, From photo, no warning", () => {
    expect(labelReading(rows[0])).toEqual({
      status: 'read',
      name: 'cup set 6pcs basic-Glasses',
      details: 'CS3850-, 8000000092403, label $12',
      warning: '',
      fromPhoto: true,
    })
  })

  it('sold at $10 while the label says $12: a warning with both prices', () => {
    expect(labelReading(rows[1]).warning).toBe('Label says $12, sold at $10')
  })

  it('no price on the label: no price in the details and no warning', () => {
    const r = labelReading(rows[2])
    expect(r.details).toBe('CS3850-, 8000000092403')
    expect(r.warning).toBe('')
    expect(r.fromPhoto).toBe(false)
  })

  it('a failed read and a reader that is off show nothing read', () => {
    expect(labelReading(rows[3])).toMatchObject({ status: 'failed', name: '', details: '', warning: '' })
    expect(labelReading(rows[4])).toMatchObject({ status: 'none', name: '', details: '', warning: '' })
    expect(labelReading(undefined).status).toBe('none')
  })

  it('one cent off either way is a warning, the same price is not', () => {
    for (const [label, sold, warned] of [[1200, 1199, true], [1200, 1201, true], [1200, 1200, false], [5, 4, true]]) {
      const item = { ...rows[0], label_price: label, unit_price: sold, label_price_mismatch: label !== sold }
      expect(labelReading(item).warning !== '').toBe(warned)
    }
  })

  it('never uses a middle dot or long dash in what it shows', () => {
    for (const row of rows) {
      const r = labelReading(row)
      for (const text of [r.name, r.details, r.warning]) {
        expect(text).not.toMatch(/[·•—–]/)
      }
    }
  })
})
