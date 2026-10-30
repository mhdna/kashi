// Row shaping shared by several reports: categories out of the product
// rows, and "people" -- one employee seen from both sides, their sales and
// their attendance punches.

import { buildTimesheet } from '@/utils/attendanceSessions'

export function categoryRows (products) {
  const byName = new Map()
  for (const p of products) {
    const name = p.product_id > 0 ? (p.category || 'No category') : 'Unresolved items'
    const row = byName.get(name) ?? { category: name, quantity: 0, revenue: 0, products: 0, cost: 0, costed_revenue: 0 }
    row.quantity += p.quantity
    row.revenue += p.revenue
    row.cost += p.cost
    row.costed_revenue += p.costed_revenue
    row.products += p.product_id > 0 ? 1 : 0
    byName.set(name, row)
  }
  return [...byName.values()].toSorted((a, b) => b.revenue - a.revenue)
}

const norm = name => String(name ?? '').trim().toLowerCase()

// People across the period, keyed so the Employees filter, the salesperson
// rows and the attendance rows all agree. kashi's salesperson list is the
// bridge: a salesperson's device user number (or, failing that, the same
// name in the same branch) ties their punches to their sales.
//
// Returns { list: [{ key, name, branchId }], forSales(row), forAttendance(personKey) }.
export function buildPeople ({ salespersons = [], staff = [], timesheet = [] }) {
  const people = new Map()
  const bySalespersonId = new Map()
  const byAttendance = new Map() // `${branch}|${device user}` -> key
  const byName = new Map() // `${branch}|${name}` -> key

  function add (key, name, branchId) {
    if (!people.has(key)) {people.set(key, { key, name: name || 'No salesperson', branchId })}
    if (name && !byName.has(`${branchId}|${norm(name)}`)) {byName.set(`${branchId}|${norm(name)}`, key)}
    return key
  }

  for (const s of staff) {
    const key = add(`sp:${s.id}`, s.name, s.branch_id)
    bySalespersonId.set(s.id, key)
    if (s.attendance_user_id) {byAttendance.set(`${s.branch_id}|${s.attendance_user_id}`, key)}
  }

  // Attendance: timesheet personKey is `${branch}|${device user id or name}`.
  const attendanceKey = new Map()
  for (const row of timesheet) {
    if (attendanceKey.has(row.personKey)) {continue}
    const key = byAttendance.get(row.personKey)
      ?? byName.get(`${row.branchId}|${norm(row.name)}`)
      ?? add(`att:${row.personKey}`, row.name, row.branchId)
    attendanceKey.set(row.personKey, key)
  }

  function forSales (row) {
    if (row.salesperson_id > 0) {
      return bySalespersonId.get(row.salesperson_id) ?? add(`sp:${row.salesperson_id}`, row.salesperson_name, row.branch_id)
    }
    return byName.get(`${row.branch_id}|${norm(row.salesperson_name)}`)
      ?? add(`name:${row.branch_id}|${norm(row.salesperson_name)}`, row.salesperson_name, row.branch_id)
  }
  for (const row of salespersons) {forSales(row)}

  return {
    list: [...people.values()].toSorted((a, b) => a.name.localeCompare(b.name)),
    forSales,
    forAttendance: personKey => attendanceKey.get(personKey),
  }
}

// The timesheet (one row per person per day) with each row's person key.
export function timesheetRows (ctx) {
  return buildTimesheet(ctx.attendance ?? []).map(row => ({ ...row, person: ctx.people.forAttendance(row.personKey) }))
}

// Salesperson rows merged per person (a person who sold in two branches is
// one row only if kashi links them; otherwise one per branch), filtered to
// the Employees filter when asked.
export function salespersonRows (rows, ctx, { filter = false } = {}) {
  const merged = new Map()
  for (const r of rows) {
    const key = ctx.people.forSales(r)
    if (filter && ctx.employees.length > 0 && !ctx.employees.includes(key)) {continue}
    const m = merged.get(key) ?? {
      key,
      // Sales nobody was picked for are kept apart per branch, so say which.
      named: Boolean(r.salesperson_name),
      name: r.salesperson_name || `No salesperson (${ctx.branchNames[r.branch_id] ?? r.branch_name})`,
      branch: ctx.branchNames[r.branch_id] ?? r.branch_name,
      sales_count: 0, sales_total: 0, returns_count: 0, returns_total: 0, net_total: 0,
      discount_total: 0, discounted_count: 0, items_sold: 0, biggest_sale: 0, clients_served: 0, selling_days: 0,
    }
    for (const field of ['sales_count', 'sales_total', 'returns_count', 'returns_total', 'net_total', 'discount_total', 'discounted_count', 'items_sold', 'clients_served', 'selling_days']) {
      m[field] += r[field]
    }
    m.biggest_sale = Math.max(m.biggest_sale, r.biggest_sale)
    merged.set(key, m)
  }
  return [...merged.values()].toSorted((a, b) => b.net_total - a.net_total)
}
