import { afterEach, describe, expect, it } from 'vitest'
import {
  emptyItemFilter, hasItemFilter, itemCompareRows, itemDays, itemFigures, itemFilterFromQuery, itemFilterKey,
  itemFilterLabel, itemFilterToQuery, itemQuery, itemSalespersons, itemStock, normalizeItemFilter, parseCodes,
} from './itemReport'
import { setReportLang } from './reportI18n'
import fixture from './testdata/item_report.json'

afterEach(() => setReportLang('en'))

describe('parseCodes', () => {
  it('the user\'s case: codes starting with X', () => {
    expect(parseCodes(['X'])).toEqual(['X'])
  })
  it('trims, upper-cases and keeps each once, in the order typed', () => {
    expect(parseCodes([' x ', 'BZ', 'bz', 'x'])).toEqual(['X', 'BZ'])
  })
  it('splits a chip holding several codes, by comma, space, Arabic comma or semicolon', () => {
    expect(parseCodes(['x, bz', 'toc5 tdc5', 'a،b;c'])).toEqual(['X', 'BZ', 'TOC5', 'TDC5', 'A', 'B', 'C'])
    expect(parseCodes('X,BZ')).toEqual(['X', 'BZ'])
  })
  it('blank entries and nothing at all are no codes', () => {
    expect(parseCodes([])).toEqual([])
    expect(parseCodes(['', '  ', ',', null, undefined])).toEqual([])
  })
})

describe('the filter', () => {
  it('needs a code, a category or a supplier; codes left out alone pick nothing', () => {
    expect(hasItemFilter(emptyItemFilter())).toBe(false)
    expect(hasItemFilter({ skipCodes: ['X'] })).toBe(false)
    expect(hasItemFilter({ codes: [' '] })).toBe(false)
    expect(hasItemFilter({ codes: ['x'] })).toBe(true)
    expect(hasItemFilter({ categoryIds: [3] })).toBe(true)
    expect(hasItemFilter({ supplierIds: ['4'] })).toBe(true)
    expect(hasItemFilter({ categoryIds: [0, -1, 'shoes'] })).toBe(false)
  })

  it('the query the API takes, only what is set', () => {
    expect(itemQuery({ from: '2026-09-01', to: '2026-09-30', filter: { codes: ['x'] } }))
      .toEqual({ from: '2026-09-01', to: '2026-09-30', codes: 'X' })
    expect(itemQuery({ from: 'a', to: 'b', branchIds: [1, 2], filter: { codes: ['x', 'bz'], skipCodes: ['xy'], categoryIds: [3, 3], supplierIds: [9] } }))
      .toEqual({ from: 'a', to: 'b', branch_ids: '1,2', codes: 'X,BZ', skip_codes: 'XY', category_ids: '3', supplier_ids: '9' })
  })

  it('goes into a link and comes back the same', () => {
    const filter = { codes: ['X', 'BZ'], skipCodes: ['XY'], categoryIds: [3, 4], supplierIds: [9] }
    const query = itemFilterToQuery(filter)
    expect(query).toEqual({ codes: 'X,BZ', skip: 'XY', cats: '3,4', sups: '9' })
    expect(itemFilterFromQuery(query)).toEqual(filter)
    expect(itemFilterFromQuery({})).toEqual(emptyItemFilter())
    expect(itemFilterFromQuery({ codes: ['X'], cats: 'nope' })).toEqual(emptyItemFilter())
  })

  it('a key that changes with the filter, not with how it was typed', () => {
    expect(itemFilterKey({ codes: ['x, bz'] })).toBe(itemFilterKey({ codes: ['X', 'BZ'] }))
    expect(itemFilterKey({ codes: ['X'] })).not.toBe(itemFilterKey({ codes: ['X'], skipCodes: ['XY'] }))
    expect(itemFilterKey(undefined)).toBe(itemFilterKey(emptyItemFilter()))
  })

  it('says on the sheet which items the report is on', () => {
    const names = { categories: { 3: 'Shoes', 4: 'Bags' }, suppliers: { 9: 'Acme' } }
    expect(itemFilterLabel({ codes: ['X'] }, names)).toBe('Codes starting with X')
    expect(itemFilterLabel({ codes: ['X', 'BZ'], skipCodes: ['XY'] }, names)).toBe('Codes starting with X or BZ, not XY')
    expect(itemFilterLabel({ categoryIds: [3, 4], supplierIds: [9] }, names)).toBe('Category: Shoes or Bags. Supplier: Acme')
    expect(itemFilterLabel({ codes: ['T'], categoryIds: [77] }, names)).toBe('Codes starting with T. Category: #77')
    expect(itemFilterLabel(emptyItemFilter())).toBe('')
  })

  it('in Arabic too', () => {
    setReportLang('ar')
    const label = itemFilterLabel({ codes: ['X', 'BZ'], skipCodes: ['XY'] })
    expect(label).toContain('X')
    expect(label).toContain('BZ')
    expect(label).not.toContain('Codes starting with')
    expect(label).not.toMatch(/[—–·•]/)
  })

  it('normalizing twice changes nothing', () => {
    const f = normalizeItemFilter({ codes: ['x, bz'], skipCodes: ['xy'], categoryIds: ['3'], supplierIds: [9, 9] })
    expect(normalizeItemFilter(f)).toEqual(f)
    expect(f).toEqual({ codes: ['X', 'BZ'], skipCodes: ['XY'], categoryIds: [3], supplierIds: [9] })
  })
})

// The backend's real answer (api/item_report_test.go writes it): codes
// starting with X but not XY, 10 to 12 Sep. X01 sold 2 at $10 for $18 and
// one came back for $9; X02 sold one for $7; a Z item ($5) shares the scope.
describe('the backend\'s figures', () => {
  const f = itemFigures(fixture.summary, fixture.stock)

  it('headline figures', () => {
    expect(f.netSales).toBe(1600)
    expect(f.share).toBeCloseTo(1600 / 3000, 10)
    expect(f.netUnits).toBe(2)
    expect(f.unitsOut).toBe(3)
    expect(f.unitsBack).toBe(1)
    expect(f.returnRate).toBeCloseTo(1 / 3, 10)
    expect(f.returnedValue).toBe(900)
    expect(f.salesCount).toBe(2)
    expect(f.avgUnitPrice).toBeCloseTo(2500 / 3, 10)
    expect(f.discount).toBe(200)
    expect(f.discountRate).toBeCloseTo(200 / 2700, 10)
    expect(f.productsSold).toBe(2)
    expect(f.productsInGroup).toBe(4)
    expect(f.unitsPerDay).toBe(1)
  })
  it('profit only on goods with a known cost', () => {
    expect(f.costed).toBe(true)
    expect(f.profit).toBe(500)
    expect(f.margin).toBeCloseTo(500 / 900, 10)
    expect(f.costedShare).toBeCloseTo(900 / 1600, 10)
  })
  it('stock left and sell-through', () => {
    expect(f.onHand).toBe(19)
    expect(f.sellThrough).toBeCloseTo(2 / 21, 10)
  })
  it('the pieces add up to the headline figures', () => {
    const sum = (rows, key) => rows.reduce((a, r) => a + r[key], 0)
    expect(sum(fixture.daily, 'net_sales')).toBe(fixture.summary.net_sales)
    expect(sum(fixture.daily, 'net_units')).toBe(fixture.summary.net_units)
    expect(sum(fixture.branches, 'net_sales')).toBe(fixture.summary.net_sales)
    expect(sum(fixture.branches, 'discount')).toBe(f.discount)
    expect(sum(fixture.salespersons, 'net_sales')).toBe(fixture.summary.net_sales)
    expect(sum(fixture.products, 'revenue')).toBe(fixture.summary.net_sales)
    expect(sum(fixture.products, 'quantity')).toBe(fixture.summary.net_units)
    for (const kind of ['size', 'color']) {
      expect(sum(fixture.variant_attributes.filter(a => a.kind === kind), 'revenue')).toBe(fixture.summary.net_sales)
    }
  })

  it('compared with the period before', () => {
    const rows = Object.fromEntries(itemCompareRows(fixture.summary, fixture.previous.summary).map(r => [r.key, r]))
    expect(rows.net_sales).toMatchObject({ now: 1600, before: 1000 })
    expect(rows.net_sales.delta).toBeCloseTo(0.6, 10)
    expect(rows.net_units.delta).toBeCloseTo(1, 10)
    // A share changes by points: 53.3% now against 100% then.
    expect(rows.share.delta).toBeCloseTo(1600 / 3000 - 1, 10)
    // Nothing came back before: no percent to give.
    expect(rows.units_back.before).toBe(0)
    expect(Number.isNaN(rows.units_back.delta)).toBe(true)
    expect(rows.units_back.lowerIsBetter).toBe(true)
  })

  it('every day of the period, the quiet ones at zero', () => {
    const days = itemDays(fixture.daily, '2026-09-09', '2026-09-13')
    expect(days.map(d => [d.day, d.net_sales])).toEqual([
      ['2026-09-09', 0], ['2026-09-10', 1800], ['2026-09-11', 700], ['2026-09-12', -900], ['2026-09-13', 0],
    ])
  })

  it('stock left, and what sits unsold', () => {
    const { stockLeft, unsold } = itemStock(fixture.stock, fixture.products)
    expect(stockLeft.map(r => [r.product_code, r.sold, r.on_hand])).toEqual([
      ['QBASEX01', 1, 10], ['QBASEX02', 1, 0], ['QBASEX04', 0, 7], ['QBASEX05', 0, 2],
    ])
    expect(stockLeft[0].sell_through).toBeCloseTo(1 / 11, 10)
    expect(stockLeft[1].sell_through).toBe(1)
    expect(unsold.map(r => r.product_code)).toEqual(['QBASEX04', 'QBASEX05'])
  })

  it('who sold them, the unnamed ones by branch', () => {
    const rows = itemSalespersons(fixture.salespersons, { 1: 'Centro' })
    expect(rows.map(r => [r.name, r.net_sales])).toEqual([['No salesperson (Centro)', 900], ['Rana', 700]])
  })
})

describe('one cent off anywhere is caught', () => {
  const base = itemFigures(fixture.summary, fixture.stock)
  for (const key of ['net_sales', 'gross_sales', 'list_value', 'cost', 'costed_revenue', 'all_net_sales', 'returned_value']) {
    it(key, () => {
      const changed = itemFigures({ ...fixture.summary, [key]: fixture.summary[key] + 1 }, fixture.stock)
      expect(changed).not.toEqual(base)
    })
  }
  it('a unit on hand', () => {
    expect(itemFigures(fixture.summary, [...fixture.stock, { product_id: 99, on_hand: 1 }]).onHand).toBe(base.onHand + 1)
  })
})

// Randomized: made-up periods whose figures are known, through every
// function, checking the identities the sheet relies on.
describe('randomized', () => {
  const rand = n => Math.floor(Math.random() * n)
  it('hundreds of groups', () => {
    for (let round = 0; round < 300; round++) {
      const products = Array.from({ length: 1 + rand(8) }, (_, i) => ({ product_id: i + 1, product_code: `C${i}`, quantity: rand(9) - 2, revenue: rand(90_000) - 10_000 }))
      const stock = [
        ...products.filter(() => Math.random() < 0.8).map(p => ({ product_id: p.product_id, product_code: p.product_code, on_hand: rand(30) - 3 })),
        ...Array.from({ length: rand(4) }, (_, i) => ({ product_id: 100 + i, product_code: `U${i}`, on_hand: rand(10) })),
      ]
      const unitsOut = rand(50)
      const gross = rand(500_000)
      const list = gross + rand(50_000)
      const summary = {
        net_sales: products.reduce((a, p) => a + p.revenue, 0),
        net_units: products.reduce((a, p) => a + p.quantity, 0),
        units_out: unitsOut, units_back: rand(5), gross_sales: gross, list_value: list,
        all_net_sales: rand(2_000_000), returned_value: rand(9000), sales_count: rand(40), products_sold: products.length,
        selling_days: rand(31), customers: rand(20), cost: rand(100_000), costed_revenue: rand(200_000), discounted_lines: rand(9),
      }
      const f = itemFigures(summary, stock)
      expect(f.discount).toBe(list - gross)
      expect(f.discount).toBeGreaterThanOrEqual(0)
      expect(f.onHand).toBe(stock.reduce((a, s) => a + Math.max(0, s.on_hand), 0))
      expect(f.sellThrough).toBeGreaterThanOrEqual(0)
      expect(f.sellThrough).toBeLessThanOrEqual(1)
      for (const v of Object.values(f)) {expect(Number.isNaN(v)).toBe(false)}

      const { stockLeft, unsold } = itemStock(stock, products)
      for (const r of stockLeft) {
        expect(r.sell_through).toBeGreaterThanOrEqual(0)
        expect(r.sell_through).toBeLessThanOrEqual(1)
      }
      // Every unsold product has stock and no sale; the sold ones keep their units.
      for (const r of unsold) {
        expect(r.on_hand).toBeGreaterThan(0)
        expect(products.some(p => p.product_id === r.product_id)).toBe(false)
      }
      const listed = new Map(stockLeft.map(r => [r.product_id, r]))
      for (const p of products.filter(p => stock.some(s => s.product_id === p.product_id) && p.quantity !== 0)) {
        expect(listed.get(p.product_id).sold).toBe(p.quantity)
      }

      const days = itemDays([{ day: '2026-09-02T00:00:00Z', net_units: summary.net_units, net_sales: summary.net_sales, sales_count: 1 }], '2026-09-01', '2026-09-05')
      expect(days).toHaveLength(5)
      expect(days.reduce((a, d) => a + d.net_sales, 0)).toBe(summary.net_sales)

      const rows = itemCompareRows(summary, { ...summary, net_sales: summary.net_sales * 2 })
      const net = rows.find(r => r.key === 'net_sales')
      if (summary.net_sales > 0) {expect(net.delta).toBeCloseTo(-0.5, 10)}
      expect(rows.find(r => r.key === 'products_sold').delta).toBe(summary.products_sold ? 0 : Number.NaN)
    }
  })
})
