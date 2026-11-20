import { describe, expect, it } from 'vitest'
import { performanceSheet, performanceTables } from './salespersonPdf'
import { SCORE_RULE, scoreSalespersons } from './salespersonScore'
// What GET /salesperson_performance really sends (the Go test writes it).
import api from './testdata/salesperson_performance.json'

const money = cents => `$${(cents / 100).toFixed(2)}`
const loose = { ...SCORE_RULE, minInvoices: 1, minHours: 0 }
const hours = new Map([['id:11', new Map([[1, 50]])], ['id:12', new Map([[1, 30]])]])
const scored = scoreSalespersons({ invoices: api.invoices, returns: api.returns, branchTargets: api.branch_targets, hours, rule: loose })
  .people
.map(p => ({ ...p, branchName: 'Centro' }))
const ranking = [
  { name: 'Ali', branchName: 'Centro', score: 100, totalAmount: 14_000, totalInvoices: 1, totalGoods: 3, totalDiscounts: 0, totalPriceOverrides: 0 },
  { name: 'Rana', branchName: 'Centro', score: 87.5, totalAmount: 13_500, totalInvoices: 2, totalGoods: 4, totalDiscounts: 500, totalPriceOverrides: 250 },
]

describe('performanceTables', () => {
  it('the ranking table, row by row, with a total that adds up', () => {
    const [table] = performanceTables({ ranking, score: [], money })
    expect(table.title).toBe('Performance ranking')
    expect(table.rows).toEqual([
      ['1', 'Ali', 'Centro', '100.0', '$140.00', '1', '3', '$0.00', '$0.00'],
      ['2', 'Rana', 'Centro', '87.5', '$135.00', '2', '4', '$5.00', '$2.50'],
    ])
    expect(table.total).toEqual(['', 'Total', '', '', '$275.00', '3', '7', '$5.00', '$2.50'])
    for (const row of [...table.rows, table.total]) {expect(row).toHaveLength(table.columns.length)}
  })

  it('the score table: the fixture as the page scores it', () => {
    const [, table] = performanceTables({ ranking: [], score: scored, money })
    expect(table.title).toBe('Performance score')
    const ali = scored.find(p => p.key === 'id:12')
    const rana = scored.find(p => p.key === 'id:11')
    expect(table.rows).toEqual([
      ['1', 'Ali Saad', 'Centro', `${ali.score.toFixed(1)} ${ali.grade}`, '$110.00', '$93.75', '117.3%', '30.0', '$3.67', '$110.00', '3.00', '21.4%'],
      ['2', 'Rana Haddad', 'Centro', `${rana.score.toFixed(1)} ${rana.grade}`, '$90.00', '$156.25', '57.6%', '50.0', '$1.80', '$45.00', '2.00', '33.3%'],
    ])
    // The note gives the current rule, discounts not in it.
    expect(table.note).toContain('Target reached 40, Sales per hour 25, Average sale 15, Items per sale 15, Return rate 5')
    expect(table.note).not.toMatch(/Discount/)
  })

  it('someone not ranked, with no target or hours, shows why and blanks', () => {
    const p = { ...scored[0], rank: null, score: null, grade: '', notRankedWhy: 'Under 10 invoices', target: 0, hours: 0, perHour: null }
    const [, table] = performanceTables({ ranking: [], score: [p], money })
    expect(table.rows[0].slice(0, 9)).toEqual(['', p.name, 'Centro', 'Under 10 invoices', money(p.net), 'None', '', '', ''])
  })

  it('nobody: no rows and no total', () => {
    for (const t of performanceTables({ money })) {
      expect(t.rows).toEqual([])
      expect(t.total).toBeNull()
    }
  })

  it('no dashes, middle dots or NaN anywhere', () => {
    const text = JSON.stringify(performanceTables({ ranking, score: scored, money }))
    expect(text).not.toMatch(/[—–·]|NaN|undefined/)
  })
})

describe('performanceSheet', () => {
  it('heading, both tables and their cells', () => {
    const tables = performanceTables({ ranking, score: scored, money })
    const sheet = performanceSheet({ title: 'Salesperson performance', period: 'November 2026', branch: 'Centro', generated: 'Nov 20', tables })
    expect(sheet.classList.contains('rpt-sheet')).toBe(true)
    expect(sheet.querySelector('.rpt-title').textContent).toBe('Salesperson performance')
    expect(sheet.querySelector('.rpt-subtitle').textContent).toBe('November 2026')
    expect(sheet.querySelector('.rpt-meta').textContent).toContain('Centro')
    expect([...sheet.querySelectorAll('.rpt-h2')].map(h => h.textContent)).toEqual(['Performance ranking', 'Performance score'])
    const [first, second] = sheet.querySelectorAll('table')
    expect([...first.querySelectorAll('tbody tr')].map(tr => [...tr.cells].map(td => td.textContent))).toEqual(tables[0].rows)
    expect([...first.querySelectorAll('tfoot td')].map(td => td.textContent)).toEqual(tables[0].total)
    expect([...second.querySelectorAll('tbody tr')].map(tr => [...tr.cells].map(td => td.textContent))).toEqual(tables[1].rows)
    expect(second.querySelector('tfoot')).toBeNull()
    expect([...second.querySelectorAll('thead th')].map(th => th.textContent)).toEqual(tables[1].columns.map(c => c.title))
  })

  it('a name typed at a till stays text, never markup', () => {
    const tables = performanceTables({ ranking: [{ ...ranking[0], name: '<img src=x onerror=alert(1)>' }], money })
    const sheet = performanceSheet({ title: 't', period: 'p', branch: 'b', generated: 'g', tables })
    expect(sheet.querySelector('img')).toBeNull()
    expect(sheet.querySelector('tbody td:nth-child(2)').textContent).toBe('<img src=x onerror=alert(1)>')
  })

  it('an empty table says so', () => {
    const sheet = performanceSheet({ title: 't', period: 'p', branch: 'b', generated: 'g', tables: performanceTables({ money }) })
    expect(sheet.querySelectorAll('table')).toHaveLength(0)
    expect(sheet.querySelectorAll('.rpt-empty')).toHaveLength(2)
  })
})
