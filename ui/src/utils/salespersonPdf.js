// The Salesperson performance page as a PDF: its two tables (the ranking
// against the best, and the Performance score) on a report sheet that
// utils/reportExport printReport prints. performanceTables is the plain data
// (tested cell by cell); performanceSheet turns it into the sheet, with
// text only (names come from tills, so never as HTML).

import { SCORE_RULE } from './salespersonScore'

const percent = ratio => `${((Number(ratio) || 0) * 100).toFixed(1)}%`

// ranking: the page's rankedRows. score: the score table's people.
// money(cents) formats money as the page does.
export function performanceTables ({ ranking = [], score = [], money }) {
  const weights = SCORE_RULE.parts.map(p => `${p.label} ${p.weight}`).join(', ')
  return [
    {
      title: 'Performance ranking',
      note: 'Score = 50% × (sold amount ÷ highest amount) + 25% × (invoices ÷ highest invoices) + 25% × (goods sold ÷ highest goods)',
      columns: [
        { title: '#', num: true },
        { title: 'Employee' },
        { title: 'Branch' },
        { title: 'Score', right: true },
        { title: 'Sold', right: true },
        { title: 'Invoices', right: true },
        { title: 'Goods', right: true },
        { title: 'Discounts', right: true },
        { title: 'Price deductions', right: true },
      ],
      rows: ranking.map((r, i) => [
        String(i + 1), r.name, r.branchName ?? '', r.score.toFixed(1), money(r.totalAmount),
        r.totalInvoices.toLocaleString('en-US'), r.totalGoods.toLocaleString('en-US'), money(r.totalDiscounts), money(r.totalPriceOverrides),
      ]),
      total: ranking.length > 0
        ? ['', 'Total', '', '', money(sum(ranking, r => r.totalAmount)), sum(ranking, r => r.totalInvoices).toLocaleString('en-US'),
            sum(ranking, r => r.totalGoods).toLocaleString('en-US'), money(sum(ranking, r => r.totalDiscounts)), money(sum(ranking, r => r.totalPriceOverrides))]
        : null,
      empty: 'No salesperson activity.',
    },
    {
      title: 'Performance score',
      note: `Score out of 100: ${weights}. Ranked from ${SCORE_RULE.minInvoices} invoices and ${SCORE_RULE.minHours} hours worked. Grades: A from 85, B from 70, C from 55, D under 55.`,
      columns: [
        { title: '#', num: true },
        { title: 'Employee' },
        { title: 'Branch' },
        { title: 'Score' },
        { title: 'Net sales', right: true },
        { title: 'Target', right: true },
        { title: 'Reached', right: true },
        { title: 'Hours', right: true },
        { title: 'Per hour', right: true },
        { title: 'Average sale', right: true },
        { title: 'Items per sale', right: true },
        { title: 'Return rate', right: true },
      ],
      rows: score.map(p => [
        p.rank === null || p.rank === undefined ? '' : String(p.rank),
        p.name,
        p.branchName ?? '',
        p.score === null ? p.notRankedWhy : `${p.score.toFixed(1)} ${p.grade}`,
        money(p.net),
        p.target > 0 ? money(p.target) : 'None',
        p.target > 0 ? percent(p.net / p.target) : '',
        p.hours > 0 ? p.hours.toFixed(1) : '',
        p.perHour === null ? '' : money(p.perHour),
        money(p.avgSale),
        p.itemsPerSale.toFixed(2),
        percent(p.returnRate),
      ]),
      total: null,
      empty: 'No salesperson activity.',
    },
  ]
}

function sum (rows, f) {
  return rows.reduce((a, r) => a + (Number(f(r)) || 0), 0)
}

function el (doc, tag, className, text) {
  const node = doc.createElement(tag)
  if (className) {node.className = className}
  if (text !== undefined) {node.textContent = text}
  return node
}

// The sheet printReport copies: a heading with the period and branch, then
// each table.
export function performanceSheet ({ title, period, branch, generated, tables }, doc = document) {
  const sheet = el(doc, 'div', 'rpt-sheet')
  const head = el(doc, 'header', 'rpt-head')
  const left = el(doc, 'div')
  left.append(el(doc, 'h1', 'rpt-title', title), el(doc, 'p', 'rpt-subtitle', period))
  const meta = el(doc, 'div', 'rpt-meta')
  const strong = el(doc, 'strong', '', branch)
  const branchLine = el(doc, 'div')
  branchLine.append(strong)
  meta.append(branchLine, el(doc, 'div', '', `Generated ${generated}`))
  head.append(left, meta)
  sheet.append(head)

  for (const table of tables) {
    const section = el(doc, 'section', 'rpt-section')
    section.append(el(doc, 'h2', 'rpt-h2', table.title))
    if (table.note) {section.append(el(doc, 'p', 'rpt-note', table.note))}
    if (table.rows.length === 0) {
      section.append(el(doc, 'p', 'rpt-empty', table.empty))
    } else {
      const cls = col => (col.num ? 'rpt-num' : (col.right ? 'rpt-right' : ''))
      const t = el(doc, 'table', 'rpt-table')
      const tr = el(doc, 'tr')
      for (const col of table.columns) {tr.append(el(doc, 'th', cls(col), col.title))}
            const thead = el(doc, 'thead')
      thead.append(tr)
      t.append(thead)
      const body = el(doc, 'tbody')
      const addRow = (parent, cells) => {
        const row = el(doc, 'tr')
        for (const [i, cell] of cells.entries()) {row.append(el(doc, 'td', cls(table.columns[i]), cell))}
        parent.append(row)
      }
      for (const cells of table.rows) {addRow(body, cells)}
      t.append(body)
      if (table.total) {
        const foot = el(doc, 'tfoot')
        addRow(foot, table.total)
        t.append(foot)
      }
      section.append(t)
    }
    sheet.append(section)
  }
  return sheet
}
