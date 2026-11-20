import { describe, expect, it } from 'vitest'
import { collectMetrics, explainPart, grade, hoursFromTimesheet, personalTargets, salespersonKey, SCORE_RULE, scoreSalespersons } from './salespersonScore'
// What GET /salesperson_performance really sends for a month at Centro
// (TestSalespersonPerformanceWithReturnsAndTargets writes it): Rana's sales,
// Ali taking back Rana's belt, Ali exchanging his own shirt for a jacket, a
// sale with no salesperson, and a $250 target.
import api from './testdata/salesperson_performance.json'

const hoursMap = entries => new Map(entries.map(([key, perBranch]) => [key, new Map(Object.entries(perBranch).map(([b, h]) => [Number(b), h]))]))
const RANA = 'id:11'
const ALI = 'id:12'
const loose = { ...SCORE_RULE, minInvoices: 1, minHours: 0 }
function fixture (opts = {}) {
  return scoreSalespersons({
  invoices: api.invoices, returns: api.returns, branchTargets: api.branch_targets,
  hours: hoursMap([[RANA, { 1: 50 }], [ALI, { 1: 30 }]]), rule: loose, ...opts,
})
}
const person = (result, key) => result.people.find(p => p.key === key)
const part = (p, key) => p.parts.find(x => x.key === key)

describe('credit: who each sale and return counts for', () => {
  it('the reported case: the fixture, line by line', () => {
    const rows = collectMetrics({ invoices: api.invoices, returns: api.returns })
    const rana = rows.get(RANA)
    const ali = rows.get(ALI)
    // Rana: $45 jeans (10% off $50) + $45 belt, then a buy 1 get 1 with the free unit at $0.
    expect([rana.sold, rana.invoices, rana.units]).toEqual([13_500, 2, 4])
    // Ali took back her belt: it counts against her, not him.
    expect(rana.returned).toBe(4500)
    expect(rana.net).toBe(9000)
    // Ali: two shirts ($10 student discount spread on them), then he
    // exchanged one ($30 back) for an $80 jacket.
    expect([ali.sold, ali.returned, ali.net, ali.invoices, ali.units]).toEqual([14_000, 3000, 11_000, 1, 3])
    // The sale with no salesperson is kept apart, so the totals still add up.
    expect(rows.get('').net).toBe(2000)
  })

  it('every cent of the period is someone\'s or no one\'s', () => {
    const result = fixture()
    const lines = [...api.invoices.flatMap(i => i.items), ...api.returns.flatMap(r => r.items)]
    const all = lines.reduce((a, l) => a + l.line_total, 0)
    expect(result.totals.net).toBe(all)
    expect(result.people.reduce((a, p) => a + p.net, 0) + result.unassigned.net).toBe(all)
  })

  it('keys people like the first table: by id, else by the typed name', () => {
    expect(salespersonKey(12, 'Ali')).toBe('id:12')
    expect(salespersonKey(0, ' Ali ')).toBe('name:Ali')
    expect(salespersonKey(null, '')).toBe('')
    expect(salespersonKey('7', 'x')).toBe('id:7')
  })

  it('the reported case: a return made from kashi stamped Ahmad still counts for the seller, new items too', () => {
    const rows = collectMetrics({
      invoices: [{ kind: 'sales', branch_id: 1, salesperson_id: 14, salesperson_name: 'Lama D.', items: [{ unit_price: 5000, quantity: 2, line_total: 10_000 }] }],
      returns: [
        { branch_id: 1, salesperson_id: 0, salesperson_name: 'Ahmad Robaie', original_salesperson_id: 14, original_salesperson_name: 'Lama D.', original_invoice_code: 'SA/CT/1',
          items: [{ unit_price: 5000, quantity: -1, line_total: -5000 }, { unit_price: 7000, quantity: 1, line_total: 7000 }] },
      ],
    })
    expect([...rows.keys()]).toEqual(['id:14'])
    expect(rows.get('id:14')).toMatchObject({ sold: 17_000, returned: 5000, net: 12_000, invoices: 1, units: 3 })
  })

  it('a sale with no salesperson: its return counts for no one, not for the return\'s', () => {
    const rows = collectMetrics({ returns: [{ branch_id: 1, salesperson_id: 12, salesperson_name: 'Ali', original_salesperson_id: 0, original_salesperson_name: '', original_invoice_code: 'SA/CT/9', items: [{ quantity: -1, line_total: -900 }] }] })
    expect(rows.get('').returned).toBe(900)
    expect(rows.get(ALI)).toBeUndefined()
  })

  it('a return of a sale kashi doesn\'t have counts for the return\'s salesperson', () => {
    const rows = collectMetrics({ returns: [{ branch_id: 1, salesperson_id: 12, salesperson_name: 'Ali', original_salesperson_id: 0, original_salesperson_name: '', original_invoice_code: '', items: [{ quantity: -1, line_total: -900 }] }] })
    expect(rows.get(ALI).returned).toBe(900)
  })
})

describe('targets', () => {
  it('split by hours worked: $250 over 50 and 30 hours', () => {
    const r = fixture()
    expect(person(r, RANA).target).toBeCloseTo(15_625)
    expect(person(r, ALI).target).toBeCloseTo(9375)
  })

  it('split evenly when the branch has no hours, and none when there is no target', () => {
    const r = fixture({ hours: new Map() })
    expect(person(r, RANA).target).toBe(12_500)
    expect(person(r, ALI).target).toBe(12_500)
    const none = fixture({ branchTargets: [] })
    expect(person(none, RANA).target).toBe(0)
    expect(part(person(none, RANA), 'target').available).toBe(false)
  })

  it('a person with hours at another branch shares that branch\'s target too', () => {
    const rows = collectMetrics({ invoices: api.invoices, returns: api.returns })
    const t = personalTargets(rows, [{ branch_id: 1, amount: 8000 }, { branch_id: 2, amount: 6000 }], hoursMap([[RANA, { 1: 10, 2: 10 }], [ALI, { 1: 30 }]]))
    expect(t.get(RANA)).toBeCloseTo(2000 + 6000)
    expect(t.get(ALI)).toBeCloseTo(6000)
  })
})

describe('the score, worked out by hand for the fixture', () => {
  it('each part as the rule says', () => {
    const r = fixture()
    const rana = person(r, RANA)
    // Team averages: the two of them together.
    expect(r.averages.perHour).toBeCloseTo(20_000 / 80)
    expect(r.averages.avgSale).toBeCloseTo(20_000 / 3)
    expect(r.averages.itemsPerSale).toBeCloseTo(7 / 3)
    expect(r.averages.returnRate).toBeCloseTo(7500 / 27_500)

    const want = {
      target: Math.min(9000 / 15_625, 1.5) / 1.5 * 40,
      perHour: Math.min((9000 / 50) / (20_000 / 80), 1.5) / 1.5 * 25,
      avgSale: Math.min(4500 / (20_000 / 3), 1.5) / 1.5 * 15,
      itemsPerSale: Math.min(2 / (7 / 3), 1.5) / 1.5 * 15,
      returnRate: Math.max(0, 1 - (4500 / 13_500) / (2 * 7500 / 27_500)) * 5,
    }
    for (const [key, points] of Object.entries(want)) {
      expect(part(rana, key).points, key).toBeCloseTo(points, 6)
    }
    const total = Object.values(want).reduce((a, b) => a + b, 0)
    expect(rana.score).toBeCloseTo(total, 6)
    expect(rana.score).toBeCloseTo(44.63, 1)
    expect(rana.grade).toBe('D')
    expect(rana.outOf).toBe(100)
    // Ali sold more per hour.
    const ali = person(r, ALI)
    expect(ali.score).toBeGreaterThan(rana.score)
    expect([ali.rank, rana.rank]).toEqual([1, 2])
  })

  it('a part kashi can\'t work out is left out, and the score is out of the rest', () => {
    const r = fixture({ hours: new Map(), branchTargets: [] })
    const rana = person(r, RANA)
    expect(rana.outOf).toBe(35)
    expect(part(rana, 'perHour').available).toBe(false)
    expect(part(rana, 'target').available).toBe(false)
    expect(rana.score).toBeCloseTo(rana.points / 35 * 100, 9)
  })
})

describe('discounts are not scored', () => {
  it('the rule: target 40, sales per hour 25, average sale 15, items per sale 15, return rate 5, out of 100', () => {
    expect(SCORE_RULE.parts.map(p => [p.key, p.weight])).toEqual([['target', 40], ['perHour', 25], ['avgSale', 15], ['itemsPerSale', 15], ['returnRate', 5]])
    expect(SCORE_RULE.parts.reduce((a, p) => a + p.weight, 0)).toBe(100)
  })

  it('the reported case: a discount-list price off the same takings changes no score', () => {
    // The same line totals, once at full price and once as a 50% list discount on a doubled price.
    const asSold = discounted => api.invoices.map(inv => ({
      ...inv,
      items: inv.items.map(it => (discounted ? { ...it, unit_price: it.unit_price * 2, discount: 50 } : { ...it, discount: 0 })),
    }))
    const plain = fixture({ invoices: asSold(false) })
    const listed = fixture({ invoices: asSold(true) })
    for (const p of plain.people) {
      expect(person(listed, p.key).score).toBeCloseTo(p.score, 9)
      expect(p.parts.map(x => x.key)).not.toContain('discountRate')
    }
  })
})

describe('who is ranked', () => {
  it('needs 10 invoices and 40 hours by default', () => {
    const r = scoreSalespersons({ invoices: api.invoices, returns: api.returns, branchTargets: api.branch_targets, hours: hoursMap([[RANA, { 1: 50 }], [ALI, { 1: 30 }]]) })
    for (const p of r.people) {
      expect(p.ranked).toBe(false)
      expect(p.score).toBeNull()
      expect(p.rank).toBeNull()
      expect(p.notRankedWhy).toBe('Under 10 invoices')
    }
    const hoursOnly = fixture({ rule: { ...SCORE_RULE, minInvoices: 1 } })
    expect(person(hoursOnly, ALI).notRankedWhy).toBe('Under 40 hours worked')
    expect(person(hoursOnly, RANA).ranked).toBe(true)
  })

  it('without any attendance, hours don\'t decide who is ranked', () => {
    const r = fixture({ hours: new Map(), rule: { ...SCORE_RULE, minInvoices: 1 } })
    expect(r.people.every(p => p.ranked)).toBe(true)
  })
})

describe('grades', () => {
  it.each([[100, 'A'], [85, 'A'], [84.99, 'B'], [70, 'B'], [69.9, 'C'], [55, 'C'], [54.9, 'D'], [0, 'D'], [null, '']])('%s is %s', (s, g) => {
    expect(grade(s)).toBe(g)
  })
})

// Random months: money adds up, every score and part stays in range, the
// ranking is 1, 2, 3 ..., and doubling every amount changes no score.
describe('randomized', () => {
  let seed = 3
  const rand = () => ((seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648) / 2_147_483_648)
  const pick = n => Math.floor(rand() * n)
  function month (scale = 1) {
    const s0 = seed
    const invoices = []
    const returns = []
    const people = [[11, 'Rana'], [12, 'Ali'], [13, 'Maya'], [0, 'Old Name'], [0, '']]
    for (let i = 0; i < 40 + pick(80); i++) {
      const [id, name] = people[pick(people.length)]
      const items = Array.from({ length: 1 + pick(4) }, () => {
        const base = 500 + pick(9000)
        const unit = base * scale
        const qty = 1 + pick(3)
        // Rounded before scaling, so twice the money is exactly twice.
        const disc = pick(3) === 0 ? Math.round(base * qty * 0.1) * scale : 0
        const inv = pick(4) === 0 ? Math.round(base * qty * 0.05) * scale : 0
        return { unit_price: unit, quantity: qty, line_total: unit * qty - disc - inv, deal_discount: 0, invoice_discount: inv }
      })
      invoices.push({ id: i, kind: 'sales', branch_id: 1 + pick(2), salesperson_id: id, salesperson_name: name, items })
    }
    for (let i = 0; i < pick(15); i++) {
      const [oid, oname] = people[pick(people.length)]
      const [id, name] = people[pick(people.length)]
      const items = [{ unit_price: 2000 * scale, quantity: -1, line_total: -1800 * scale }]
      if (pick(2)) {items.push({ unit_price: 3000 * scale, quantity: 1, line_total: 3000 * scale })}
      returns.push({ branch_id: 1, salesperson_id: id, salesperson_name: name, original_salesperson_id: oid, original_salesperson_name: oname, original_invoice_code: pick(4) ? 'SA/1' : '', items })
    }
    const hours = hoursMap([[RANA, { 1: 20 + pick(120) }], [ALI, { 1: pick(160), 2: pick(30) }], ['id:13', { 2: 20 + pick(100) }]])
    const branchTargets = [{ branch_id: 1, amount: (50_000 + pick(500_000)) * scale }, { branch_id: 2, amount: (50_000 + pick(300_000)) * scale }]
    const out = { invoices, returns, hours, branchTargets, seedAfter: seed }
    seed = s0
    return out
  }

  it('over 300 months', () => {
    for (let i = 0; i < 300; i++) {
      const data = month()
      const rule = { ...SCORE_RULE, minInvoices: 3, minHours: 10 }
      const r = scoreSalespersons({ ...data, rule })
      const lines = [...data.invoices.flatMap(x => x.items), ...data.returns.flatMap(x => x.items)]
      const all = lines.reduce((a, l) => a + l.line_total, 0)
      expect(r.totals.net).toBe(all)
      expect(r.people.reduce((a, p) => a + p.net, 0) + (r.unassigned?.net ?? 0)).toBe(all)
      expect(r.totals.sold - r.totals.returned).toBe(all)
      // Each branch's target is shared out in full.
      const shared = r.people.reduce((a, p) => a + p.target, 0)
      expect(shared).toBeCloseTo(data.branchTargets.reduce((a, t) => a + t.amount, 0), 3)
      let rank = 0
      for (const p of r.people) {
        for (const x of p.parts) {
          expect(x.points, `${p.key} ${x.key}`).toBeGreaterThanOrEqual(0)
          expect(x.points, `${p.key} ${x.key}`).toBeLessThanOrEqual(x.weight + 1e-9)
        }
        if (p.score === null) {
          expect(p.ranked && p.outOf > 0).toBe(false)
        } else {
          expect(p.score).toBeGreaterThanOrEqual(0)
          expect(p.score).toBeLessThanOrEqual(100 + 1e-9)
          expect(p.rank).toBe(++rank)
        }
      }
      // The same month in twice the money scores the same.
      const doubled = scoreSalespersons({ ...month(2), rule })
      for (const p of r.people) {
        const q = person(doubled, p.key)
        if (p.score === null) {expect(q.score).toBeNull()} else {expect(q.score).toBeCloseTo(p.score, 6)}
      }
      seed = data.seedAfter
    }
  })

  it('selling far above the team never scores more than the part', () => {
    const invoices = [
      { kind: 'sales', branch_id: 1, salesperson_id: 11, salesperson_name: 'Rana', items: [{ unit_price: 1_000_000, quantity: 50, line_total: 50_000_000 }] },
      { kind: 'sales', branch_id: 1, salesperson_id: 12, salesperson_name: 'Ali', items: [{ unit_price: 100, quantity: 1, line_total: 100 }] },
    ]
    const r = scoreSalespersons({ invoices, branchTargets: [{ branch_id: 1, amount: 1000 }], rule: loose })
    const rana = person(r, RANA)
    expect(part(rana, 'target').points).toBe(40)
    expect(part(rana, 'avgSale').points).toBe(15)
    expect(part(rana, 'itemsPerSale').points).toBe(15)
  })
})

describe('hours and explanations', () => {
  it('adds a person\'s worked minutes per branch, only for salespersons kashi knows', () => {
    const rows = [
      { personKey: '1|7', branchId: 1, worked: 480 },
      { personKey: '1|7', branchId: 1, worked: 300 },
      { personKey: '2|7', branchId: 2, worked: 60 },
      { personKey: '1|99', branchId: 1, worked: 600 },
    ]
    const people = { '1|7': 'sp:11', '2|7': 'sp:11', '1|99': 'att:1|99' }
    const h = hoursFromTimesheet(rows, k => people[k])
    expect([...h.keys()]).toEqual(['id:11'])
    expect(h.get('id:11').get(1)).toBe(13)
    expect(h.get('id:11').get(2)).toBe(1)
  })

  it('says each part of Rana\'s score in numbers', () => {
    const rana = person(fixture(), RANA)
    const show = (key, v) => (key === 'itemsPerSale' ? v.toFixed(2) : (key.endsWith('Rate') ? `${(v * 100).toFixed(1)}%` : `$${(v / 100).toFixed(2)}`))
    expect(explainPart(part(rana, 'target'), show)).toBe('$90.00 ÷ $156.25 = 57.6%, then min(57.6%, 150%) ÷ 150% × 40 = 15.36')
    expect(explainPart(part(rana, 'perHour'), show)).toBe('$1.80 ÷ the team\'s $2.50 = 0.72, then min(0.72, 1.5) ÷ 1.5 × 25 = 12')
    expect(explainPart(part(rana, 'returnRate'), show)).toBe('33.3% against the team\'s 27.3%: (1 − 33.3% ÷ (2 × 27.3%)) × 5 = 1.94')
    const bare = person(fixture({ hours: new Map(), branchTargets: [] }), RANA)
    expect(explainPart(part(bare, 'target'), show)).toBe('Not counted: no target for this period.')
    expect(explainPart(part(bare, 'perHour'), show)).toBe('Not counted: no hours worked on record.')
    for (const p of rana.parts) {expect(explainPart(p, show)).not.toMatch(/[—–·•]|NaN|undefined/)}
  })
})
