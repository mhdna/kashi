// The Product group report's figures (api/report_product_group.go): the
// sales of the products whose code starts with the typed beginnings ("TT",
// or "TT, TX"). Plain functions over the API's answer, so the sheet and its
// tests share them. Amounts are cents; units and money are net of returns,
// and `returned` is the units that came back (negative).

import { t } from './reportI18n'
import { change, daysBetween, int, money, num1, pct, ratio, day as ymdOf } from './reportKit'

// The beginnings as the API reads them: split on commas (an Arabic comma or
// a semicolon too), trimmed, upper-cased, repeats dropped.
export function parseCodes (text) {
  const out = []
  for (const part of String(text ?? '').split(/[,،;]/)) {
    const code = part.trim().toUpperCase()
    if (code && !out.includes(code)) {out.push(code)}
  }
  return out
}

// How they read on the sheet: "TT, TX".
export const codesLabel = (codes, sep = ', ') => codes.join(sep)

// Units that came back (positive; never -0, which prints as "-0"), units
// sold before returns, and the share that came back.
export const unitsBack = s => Math.abs(s.returned)
export const grossUnits = s => s.quantity + unitsBack(s)
export const returnRate = s => ratio(unitsBack(s), grossUnits(s))

const profitOf = r => (r.costed_revenue > 0 ? r.costed_revenue - r.cost : null)
const marginOf = r => (r.costed_revenue > 0 ? ratio(r.costed_revenue - r.cost, r.costed_revenue) : null)

export function groupKpis (data) {
  const s = data.summary
  const items = [
    { label: t('Net sales'), value: money(s.revenue), hint: t('{pct} of all sales', { pct: pct(ratio(s.revenue, data.all_revenue)) }) },
    { label: t('Units sold (net)'), value: int(s.quantity), hint: t('{n} came back ({pct})', { n: int(unitsBack(s)), pct: pct(returnRate(s)) }) },
    { label: t('Sales'), value: int(s.sales_count), hint: t('{n} a day', { n: num1(ratio(s.sales_count, s.selling_days)) }) },
    { label: t('Average unit price'), value: money(ratio(s.revenue, s.quantity)) },
    { label: t('Products sold'), value: int(s.products) },
    { label: t('Customers'), value: int(s.clients), hint: t('{pct} of sales named', { pct: pct(ratio(s.named_sales_count, s.sales_count)) }) },
  ]
  // Profit only where the goods' cost is known; the hint says how much of
  // the group's sales that covers.
  if (s.costed_revenue > 0) {
    items.splice(1, 0, {
      label: t('Gross profit'),
      value: money(profitOf(s)),
      hint: t('{pct} margin, on {share} of sales with a known cost', { pct: pct(marginOf(s)), share: pct(ratio(s.costed_revenue, s.revenue)) }),
    })
  }
  return items
}

// This period against the one before, like the overview's. For returns a
// drop is the good direction.
export function compareRows (now, before) {
  const measures = [
    { measure: t('Net sales'), type: 'money', of: x => x.revenue },
    { measure: t('Units sold (net)'), type: 'int', of: x => x.quantity },
    { measure: t('Sales'), type: 'int', of: x => x.sales_count },
    { measure: t('Average unit price'), type: 'money', of: x => ratio(x.revenue, x.quantity) },
    { measure: t('Units back'), type: 'int', of: unitsBack, lowerIsBetter: true },
    { measure: t('Customers'), type: 'int', of: x => x.clients },
  ]
  if (now.costed_revenue > 0 || before.costed_revenue > 0) {
    measures.splice(1, 0, { measure: t('Gross profit'), type: 'money', of: x => x.costed_revenue - x.cost })
  }
  return measures.map(m => {
    const a = m.of(now)
    const b = m.of(before)
    // A figure that flipped sign (more came back than sold) has no percent.
    const delta = b && Math.sign(b) === Math.sign(a) ? (a - b) / Math.abs(b) : Number.NaN
    return { ...m, now: a, before: b, delta, deltaText: change(delta) }
  })
}

// Every day of the period, the days nothing of the group sold at zero.
export function dayRows (daily, from, to) {
  const byDay = new Map(daily.map(d => [ymdOf(d.day), d]))
  return daysBetween(from, to).map(ymd => ({ quantity: 0, revenue: 0, cost: 0, costed_revenue: 0, returned: 0, sales_count: 0, ...byDay.get(ymd), day: ymd }))
}

const withProfit = r => ({ ...r, profit: profitOf(r), margin: marginOf(r) })

export function branchRows (data, branchNames = {}) {
  return data.branches.map(b => withProfit({ ...b, branch_name: branchNames[b.branch_id] ?? b.branch_name }))
}

// Sales nobody was picked for stay apart per branch, so say which.
export function salespersonRows (data, branchNames = {}) {
  return data.salespersons.map(s => {
    const branch = branchNames[s.branch_id] ?? s.branch_name
    return { ...s, branch, name: s.salesperson_name || t('No salesperson ({branch})', { branch }) }
  })
}

export function clientRows (data) {
  return data.clients.map(c => ({ ...c, name: c.name || c.phone }))
}

// The footer row of a table: its rows added up.
export function totalOf (rows, label, key) {
  const sum = field => rows.reduce((a, r) => a + (r[field] ?? 0), 0)
  const total = { [key]: label, sales_count: sum('sales_count'), quantity: sum('quantity'), revenue: sum('revenue'), cost: sum('cost'), costed_revenue: sum('costed_revenue') }
  return { ...total, profit: profitOf(total), margin: marginOf(total), share: 1 }
}
