import { describe, expect, it } from 'vitest'
// Written by api/report_product_group_test.go from the real handler's answer
// (UPDATE_FIXTURES=1 rewrites it), so the two sides can't drift apart.
import fixture from './__fixtures__/productGroup.json'
import { branchRows, clientRows, codesLabel, compareRows, dayRows, groupKpis, grossUnits, parseCodes, returnRate, salespersonRows, totalOf } from './productGroup'

const kpi = (items, label) => items.find(k => k.label === label)

describe('parseCodes', () => {
  // The same cases as the API's TestParseCodePrefixes: what the box shows is
  // what the API reads.
  it.each([
    ['TT', ['TT']],
    ['tt', ['TT']],
    [' TT , tx ', ['TT', 'TX']],
    ['TT,TT,tt', ['TT']],
    ['TT،TX;AB', ['TT', 'TX', 'AB']],
    ['TT,,  ,TX', ['TT', 'TX']],
    ['TT 1', ['TT 1']],
    ['', []],
    [' , ', []],
    [null, []],
  ])('%j', (raw, want) => {
    expect(parseCodes(raw)).toEqual(want)
  })
  it('reads back what it shows', () => {
    expect(parseCodes(codesLabel(parseCodes(' tt, tx ,TT')))).toEqual(['TT', 'TX'])
    expect(codesLabel(['TT', 'TX'], '، ')).toBe('TT، TX')
  })
})

describe('the group from the API', () => {
  it('headline figures', () => {
    const items = groupKpis(fixture)
    expect(kpi(items, 'Net sales')).toMatchObject({ value: '$105', hint: '21.0% of all sales' })
    expect(kpi(items, 'Units sold (net)')).toMatchObject({ value: '6', hint: '1 came back (14.3%)' })
    expect(kpi(items, 'Sales')).toMatchObject({ value: '3', hint: '1.0 a day' })
    expect(kpi(items, 'Gross profit')).toMatchObject({ value: '$24', hint: '53.3% margin, on 42.9% of sales with a known cost' })
    expect(kpi(items, 'Average unit price').value).toBe('$17.50')
    expect(kpi(items, 'Products sold').value).toBe('3')
    expect(kpi(items, 'Customers')).toMatchObject({ value: '2', hint: '66.7% of sales named' })
    // Profit sits right after net sales.
    expect(items.map(k => k.label).slice(0, 2)).toEqual(['Net sales', 'Gross profit'])
  })

  it('nothing came back: 0, not -0', () => {
    const items = groupKpis({ ...fixture, summary: { ...fixture.summary, returned: 0 } })
    expect(kpi(items, 'Units sold (net)').hint).toBe('0 came back (0.0%)')
    expect(Object.is(compareRows(fixture.previous.summary, fixture.previous.summary).find(r => r.measure === 'Units back').now, 0)).toBe(true)
  })

  it('no known cost: no profit figure', () => {
    const items = groupKpis({ ...fixture, summary: { ...fixture.summary, cost: 0, costed_revenue: 0 } })
    expect(kpi(items, 'Gross profit')).toBeUndefined()
  })

  it('compared with August', () => {
    const rows = compareRows(fixture.summary, fixture.previous.summary)
    const row = name => rows.find(r => r.measure === name)
    expect(row('Net sales')).toMatchObject({ now: 10500, before: 2000, deltaText: '+425.0%' })
    expect(row('Gross profit')).toMatchObject({ now: 2400, before: 800 })
    expect(row('Units back')).toMatchObject({ now: 1, before: 0, lowerIsBetter: true })
    expect(Number.isNaN(row('Units back').delta)).toBe(true) // nothing came back before: no percent
    expect(row('Customers')).toMatchObject({ now: 2, before: 1, deltaText: '+100.0%' })
  })

  it('every day of the period, adding up to the total', () => {
    const days = dayRows(fixture.daily, '2026-09-01', '2026-09-30')
    expect(days).toHaveLength(30)
    expect(days.reduce((a, d) => a + d.revenue, 0)).toBe(fixture.summary.revenue)
    expect(days.reduce((a, d) => a + d.quantity, 0)).toBe(fixture.summary.quantity)
    expect(days.reduce((a, d) => a + d.sales_count, 0)).toBe(fixture.summary.sales_count)
    expect(days[0]).toMatchObject({ day: '2026-09-01', revenue: 0, sales_count: 0 })
    expect(days[1]).toMatchObject({ day: '2026-09-02', revenue: 5500 })
  })

  it('branches and salespersons add up to the summary', () => {
    for (const [rows, key] of [[branchRows(fixture), 'branch_name'], [salespersonRows(fixture), 'name']]) {
      const total = totalOf(rows, 'Total', key)
      expect(total).toMatchObject({ revenue: fixture.summary.revenue, quantity: fixture.summary.quantity, sales_count: fixture.summary.sales_count, cost: fixture.summary.cost, profit: 2400 })
    }
  })

  it('kashi\'s branch names win, and an unpicked salesperson says where', () => {
    expect(branchRows(fixture, { 1: 'Centro' }).find(b => b.branch_id === 1).branch_name).toBe('Centro')
    const sp = salespersonRows(fixture, { 2: 'Hara' })
    expect(sp.find(s => s.salesperson_name === '').name).toBe('No salesperson (Hara)')
    expect(sp.find(s => s.salesperson_name === 'Rami').name).toBe('Rami')
  })

  it('customers, biggest first, falling back to the phone', () => {
    expect(clientRows(fixture).map(c => c.client_id)).toEqual([5, 6])
    expect(clientRows({ clients: [{ client_id: 1, name: '', phone: '+961 70' }] })[0].name).toBe('+961 70')
  })

  it('profit columns stay empty where no cost is known', () => {
    const rows = branchRows({ branches: [{ branch_id: 1, branch_name: 'A', revenue: 100, cost: 0, costed_revenue: 0 }] })
    expect(rows[0]).toMatchObject({ profit: null, margin: null })
  })
})

// Random groups: the tables' totals equal the summary, whatever was sold.
describe('random groups add up', () => {
  let seed = 7
  const rand = n => {
    seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648
    return seed % n
  }
  it('over 300 groups', () => {
    for (let round = 0; round < 300; round++) {
      const branches = Array.from({ length: 1 + rand(4) }, (_, i) => {
        const quantity = rand(40) - 5
        const costed = rand(2) ? rand(50_000) : 0
        return { branch_id: i + 1, branch_name: `B${i}`, quantity, returned: -rand(3), revenue: rand(100_000) - 5000, cost: costed ? rand(costed) : 0, costed_revenue: costed, sales_count: rand(20) }
      })
      const summary = totalOf(branches, 'Total', 'branch_name')
      const returned = branches.reduce((a, b) => a + b.returned, 0)
      const daily = branches.map((b, i) => ({ ...b, day: `2026-09-${String(1 + ((i * 7 + rand(3)) % 28)).padStart(2, '0')}` }))
      // Several branches can land on one day: merge them like the API.
      const merged = new Map()
      for (const d of daily) {
        const m = merged.get(d.day) ?? { day: d.day, quantity: 0, revenue: 0, sales_count: 0 }
        m.quantity += d.quantity
        m.revenue += d.revenue
        m.sales_count += d.sales_count
        merged.set(d.day, m)
      }
      const days = dayRows([...merged.values()], '2026-09-01', '2026-09-30')
      expect(days).toHaveLength(30)
      expect(days.reduce((a, d) => a + d.revenue, 0)).toBe(summary.revenue)
      expect(days.reduce((a, d) => a + d.quantity, 0)).toBe(summary.quantity)
      expect(totalOf(branchRows({ branches }), 'T', 'branch_name').profit).toBe(summary.costed_revenue > 0 ? summary.costed_revenue - summary.cost : null)

      const s = { quantity: summary.quantity, returned }
      expect(grossUnits(s)).toBe(summary.quantity - returned)
      const rate = returnRate(s)
      expect(rate).toBe(grossUnits(s) ? Math.abs(returned) / grossUnits(s) : 0)
    }
  })
})
