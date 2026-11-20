// The salesperson score: how stores judge a salesperson, from the sales,
// returns, hours and targets of a period (pages/salesperson-performance.vue,
// api/salesperson_performance.go).
//
// Credit: a sale belongs to its invoice's salesperson. A return or exchange
// counts for the salesperson of the sale it came back from, exactly as it was
// sold, its new items too (the reports do the same, db/query/reports.sql):
// a return made from kashi carries the till's default salesperson, not
// whoever served the customer. Only when kashi doesn't have the sale does it
// count for the return's own salesperson. Money is the line's total, which
// already has the item discount, the deal and its share of the invoice
// discount off, before tax.
//
// Discounts aren't scored: most come from the branch's discount list, which
// the till applies on its own, so they say what was sold, not how. Selling
// at a discount already shows in net sales against the target and per hour.
//
// The score compares each person with the team: target reached, sales per
// hour, average sale, items per sale, and (lower is better) returns. A part that can't be worked out (no target, no hours) is left out
// and the score is out of the parts that remain, so nobody gets zero for data
// kashi doesn't have.

export const SCORE_RULE = Object.freeze({
  minHours: 40,
  minInvoices: 10,
  // Nobody gets more for being further than this above the team.
  cap: 1.5,
  parts: Object.freeze([
    { key: 'target', label: 'Target reached', weight: 40 },
    { key: 'perHour', label: 'Sales per hour', weight: 25 },
    { key: 'avgSale', label: 'Average sale', weight: 15 },
    { key: 'itemsPerSale', label: 'Items per sale', weight: 15 },
    { key: 'returnRate', label: 'Return rate', weight: 5, lowerIsBetter: true },
  ]),
})

export function grade (score) {
  if (score === null || score === undefined) {return ''}
  if (score >= 85) {return 'A'}
  if (score >= 70) {return 'B'}
  if (score >= 55) {return 'C'}
  return 'D'
}

const num = v => Number(v) || 0

// The person a sale or return is credited to, the way the page's first table
// keys them: kashi's salesperson id, or the name a till sent.
export function salespersonKey (id, name) {
  const n = Number(id) || 0
  if (n > 0) {return `id:${n}`}
  const typed = String(name ?? '').trim()
  return typed ? `name:${typed}` : ''
}

function blank (key, name, branchId) {
  return { key, name, branchId, sold: 0, returned: 0, invoices: 0, units: 0, branches: new Set() }
}

// What a sold line adds: its money and units.
function addSoldLine (row, item) {
  row.sold += num(item.line_total)
  row.units += num(item.quantity)
}

// collectMetrics adds up each person's sales and returns. Unassigned sales
// (no salesperson) come back under key '' so the totals still add up.
export function collectMetrics ({ invoices = [], returns = [] }) {
  const rows = new Map()
  const get = (key, name, branchId) => {
    let row = rows.get(key)
    if (!row) {
      row = blank(key, key ? name : 'No salesperson', branchId)
      rows.set(key, row)
    }
    row.branches.add(branchId)
    return row
  }
  for (const invoice of invoices) {
    if (invoice.kind && invoice.kind !== 'sales' && invoice.kind !== 'sale') {continue}
    const row = get(salespersonKey(invoice.salesperson_id, invoice.salesperson_name), invoice.salesperson_name, invoice.branch_id)
    row.invoices += 1
    for (const item of invoice.items ?? []) {addSoldLine(row, item)}
  }
  for (const ret of returns) {
    // original_invoice_code is set when kashi has the sale it came back from.
    const known = Boolean(ret.original_invoice_code)
    const key = known
      ? salespersonKey(ret.original_salesperson_id, ret.original_salesperson_name)
      : salespersonKey(ret.salesperson_id, ret.salesperson_name)
    const name = known ? ret.original_salesperson_name : ret.salesperson_name
    for (const item of ret.items ?? []) {
      const row = get(key, name, ret.branch_id)
      if (num(item.quantity) < 0) {
        row.returned += -num(item.line_total)
      } else if (num(item.quantity) > 0) {
        addSoldLine(row, item)
      }
    }
  }
  for (const row of rows.values()) {row.net = row.sold - row.returned}
  return rows
}

// personalTargets: each branch's target split between the people of that
// branch by the hours they worked there; a branch without any hours splits it
// evenly between the people who sold there. hours: Map key -> Map branch ->
// hours.
export function personalTargets (rows, branchTargets, hours) {
  const out = new Map()
  for (const target of branchTargets ?? []) {
    const branch = target.branch_id
    const amount = num(target.amount)
    if (amount <= 0) {continue}
    const people = [...rows.values()].filter(r => r.key && (r.branches.has(branch) || hours?.get(r.key)?.has(branch)))
    const worked = people.map(r => num(hours?.get(r.key)?.get(branch)))
    const total = worked.reduce((a, b) => a + b, 0)
    for (const [i, r] of people.entries()) {
      const share = total > 0 ? amount * worked[i] / total : amount / people.length
      if (share > 0) {out.set(r.key, (out.get(r.key) ?? 0) + share)}
    }
  }
  return out
}

const clampRatio = (ratio, cap) => Math.min(Math.max(ratio, 0), cap)

// scoreSalespersons is the whole table: each person's numbers, the parts of
// their score with how each was worked out, the team averages they were
// compared with, and the ranking.
export function scoreSalespersons ({ invoices = [], returns = [], branchTargets = [], hours = new Map(), rule = SCORE_RULE } = {}) {
  const rows = collectMetrics({ invoices, returns })
  const targets = personalTargets(rows, branchTargets, hours)
  const people = [...rows.values()].filter(r => r.key).map(r => {
    const worked = [...(hours.get(r.key)?.values() ?? [])].reduce((a, b) => a + num(b), 0)
    return {
      key: r.key,
      name: r.name,
      branchId: r.branchId,
      sold: r.sold,
      returned: r.returned,
      net: r.net,
      invoices: r.invoices,
      units: r.units,
      hours: worked,
      target: targets.get(r.key) ?? 0,
      perHour: worked > 0 ? r.net / worked : null,
      avgSale: r.invoices > 0 ? r.net / r.invoices : 0,
      itemsPerSale: r.invoices > 0 ? r.units / r.invoices : 0,
      returnRate: r.sold > 0 ? r.returned / r.sold : 0,
    }
  })
  const anyHours = people.some(p => p.hours > 0)
  for (const p of people) {
    p.ranked = p.invoices >= rule.minInvoices && (!anyHours || p.hours >= rule.minHours)
    p.notRankedWhy = p.ranked
      ? ''
      : (p.invoices < rule.minInvoices ? `Under ${rule.minInvoices} invoices` : `Under ${rule.minHours} hours worked`)
  }

  // The team: the ranked people taken together.
  const team = people.filter(p => p.ranked)
  const sum = f => team.reduce((a, p) => a + f(p), 0)
  const withHours = team.filter(p => p.hours > 0)
  const averages = {
    perHour: withHours.length > 0 ? withHours.reduce((a, p) => a + p.net, 0) / withHours.reduce((a, p) => a + p.hours, 0) : null,
    avgSale: sum(p => p.invoices) > 0 ? sum(p => p.net) / sum(p => p.invoices) : null,
    itemsPerSale: sum(p => p.invoices) > 0 ? sum(p => p.units) / sum(p => p.invoices) : null,
    returnRate: sum(p => p.sold) > 0 ? sum(p => p.returned) / sum(p => p.sold) : null,
  }

  for (const p of people) {
    p.parts = rule.parts.map(part => scorePart(part, p, averages, rule.cap))
    const available = p.parts.filter(x => x.available)
    const weights = available.reduce((a, x) => a + x.weight, 0)
    const points = available.reduce((a, x) => a + x.points, 0)
    p.points = points
    p.outOf = weights
    p.score = p.ranked && weights > 0 ? points / weights * 100 : null
    p.grade = grade(p.score)
  }
  people.sort((a, b) => (b.score ?? -1) - (a.score ?? -1) || b.net - a.net || a.name.localeCompare(b.name))
  let rank = 0
  for (const p of people) {p.rank = p.score === null ? null : ++rank}

  const unassigned = rows.get('')
  return {
    people,
    averages,
    unassigned: unassigned ? { sold: unassigned.sold, returned: unassigned.returned, net: unassigned.net, invoices: unassigned.invoices } : null,
    totals: {
      net: [...rows.values()].reduce((a, r) => a + r.net, 0),
      sold: [...rows.values()].reduce((a, r) => a + r.sold, 0),
      returned: [...rows.values()].reduce((a, r) => a + r.returned, 0),
    },
  }
}

// One part of a person's score, with the numbers it came from.
function scorePart (part, p, averages, cap) {
  const out = { key: part.key, label: part.label, weight: part.weight, lowerIsBetter: Boolean(part.lowerIsBetter), available: false, value: null, compare: null, ratio: null, points: 0 }
  if (part.key === 'target') {
    if (p.target > 0) {
      out.available = true
      out.value = p.net
      out.compare = p.target
      out.ratio = p.net / p.target
      out.points = clampRatio(out.ratio, cap) / cap * part.weight
    }
    return out
  }
  const value = p[part.key]
  const average = averages[part.key]
  if (value === null || value === undefined || average === null || average === undefined) {return out}
  out.available = true
  out.value = value
  out.compare = average
  if (part.lowerIsBetter) {
    // Twice the team's rate or more scores nothing; none at all scores in full.
    out.ratio = average > 0 ? value / average : (value > 0 ? Infinity : 0)
    out.points = average > 0 ? Math.max(0, 1 - value / (2 * average)) * part.weight : (value > 0 ? 0 : part.weight)
  } else {
    out.ratio = average > 0 ? value / average : 0
    out.points = average > 0 ? clampRatio(out.ratio, cap) / cap * part.weight : 0
  }
  return out
}

// hoursFromTimesheet: worked hours per person per branch, from the
// attendance timesheet (utils/attendanceSessions buildTimesheet) and the
// people its rows were matched to (components/reports/reportRows
// buildPeople: 'sp:<id>' for a salesperson kashi knows). Hours of a device
// user kashi can't match to a salesperson count for no one.
export function hoursFromTimesheet (rows, personFor) {
  const out = new Map()
  for (const row of rows ?? []) {
    const person = personFor(row.personKey)
    if (!person || !String(person).startsWith('sp:')) {continue}
    const key = `id:${String(person).slice(3)}`
    const perBranch = out.get(key) ?? new Map()
    perBranch.set(row.branchId, (perBranch.get(row.branchId) ?? 0) + num(row.worked) / 60)
    out.set(key, perBranch)
  }
  return out
}

const two = n => (Math.round(n * 100) / 100).toLocaleString('en-US', { maximumFractionDigits: 2 })

// explainPart says how one part of a score was worked out, in words and the
// numbers, for the calculation dialog. show(key, value) formats a value of
// that part (money, items, a rate).
export function explainPart (part, show, cap = SCORE_RULE.cap) {
  if (!part.available) {
    if (part.key === 'target') {return 'Not counted: no target for this period.'}
    if (part.key === 'perHour') {return 'Not counted: no hours worked on record.'}
    return 'Not counted: nobody ranked to compare with.'
  }
  const capPct = `${two(cap * 100)}%`
  if (part.key === 'target') {
    const pct = `${two(part.ratio * 100)}%`
    return `${show('net', part.value)} ÷ ${show('net', part.compare)} = ${pct}, then min(${pct}, ${capPct}) ÷ ${capPct} × ${part.weight} = ${two(part.points)}`
  }
  if (part.lowerIsBetter) {
    return `${show(part.key, part.value)} against the team's ${show(part.key, part.compare)}: (1 − ${show(part.key, part.value)} ÷ (2 × ${show(part.key, part.compare)})) × ${part.weight} = ${two(part.points)}`
  }
  return `${show(part.key, part.value)} ÷ the team's ${show(part.key, part.compare)} = ${two(part.ratio)}, then min(${two(part.ratio)}, ${two(cap)}) ÷ ${two(cap)} × ${part.weight} = ${two(part.points)}`
}
