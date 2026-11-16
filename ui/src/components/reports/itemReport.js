// The Items report (pages/reports.vue, ItemsReport.vue): a report on one
// group of goods, picked by code ("starts with X"), category and supplier.
// The figures come from GET /reports/items (api/item_report.go); this is the
// screen logic around them, kept plain so it can be tested to the cent.

import { t } from './reportI18n'
import { day, daysBetween, ratio } from './reportKit'

// ---- the filter -------------------------------------------------------------
// { codes: ['X', 'BZ'], skipCodes: ['XY'], categoryIds: [3], supplierIds: [] }

export const emptyItemFilter = () => ({ codes: [], skipCodes: [], categoryIds: [], supplierIds: [] })

// What the cashier typed into a code box, as the list the API takes: each
// entry may hold several codes ("X, BZ" or "X BZ"); trimmed, upper-cased,
// each once, in the order typed.
export function parseCodes (entries) {
  const list = Array.isArray(entries) ? entries : [entries]
  const out = []
  for (const entry of list) {
    for (const part of String(entry ?? '').split(/[\s,،;]+/)) {
      const code = part.trim().toUpperCase()
      if (code && !out.includes(code)) {out.push(code)}
    }
  }
  return out
}

const ids = list => [...new Set((list ?? []).map(Number).filter(id => Number.isInteger(id) && id > 0))]

export function normalizeItemFilter (filter) {
  return {
    codes: parseCodes(filter?.codes ?? []),
    skipCodes: parseCodes(filter?.skipCodes ?? []),
    categoryIds: ids(filter?.categoryIds),
    supplierIds: ids(filter?.supplierIds),
  }
}

// Leaving codes out alone picks nothing: the report needs a code, a
// category or a supplier (the API refuses otherwise).
export function hasItemFilter (filter) {
  const f = normalizeItemFilter(filter)
  return f.codes.length + f.categoryIds.length + f.supplierIds.length > 0
}

// The query string for /reports/items, and a key that changes when it does.
export function itemQuery ({ from, to, branchIds = [], filter }) {
  const f = normalizeItemFilter(filter)
  const params = { from, to }
  if (branchIds.length > 0) {params.branch_ids = branchIds.join(',')}
  if (f.codes.length > 0) {params.codes = f.codes.join(',')}
  if (f.skipCodes.length > 0) {params.skip_codes = f.skipCodes.join(',')}
  if (f.categoryIds.length > 0) {params.category_ids = f.categoryIds.join(',')}
  if (f.supplierIds.length > 0) {params.supplier_ids = f.supplierIds.join(',')}
  return params
}
export const itemFilterKey = filter => JSON.stringify(normalizeItemFilter(filter))

// The filter in the URL (?codes=X,BZ&skip=XY&cats=3&sups=4) and back, so a
// report on one group can be linked to.
export function itemFilterToQuery (filter) {
  const f = normalizeItemFilter(filter)
  const q = {}
  if (f.codes.length > 0) {q.codes = f.codes.join(',')}
  if (f.skipCodes.length > 0) {q.skip = f.skipCodes.join(',')}
  if (f.categoryIds.length > 0) {q.cats = f.categoryIds.join(',')}
  if (f.supplierIds.length > 0) {q.sups = f.supplierIds.join(',')}
  return q
}
export function itemFilterFromQuery (query) {
  const list = v => (typeof v === 'string' ? v.split(',') : [])
  return normalizeItemFilter({ codes: list(query.codes), skipCodes: list(query.skip), categoryIds: list(query.cats), supplierIds: list(query.sups) })
}

// What the sheet's header says the report is on: "Codes starting with X or
// BZ, not XY. Category: Shoes. Supplier: Acme". Names come from the lists
// the page loaded; an id it doesn't know shows as #id.
export function itemFilterLabel (filter, { categories = {}, suppliers = {} } = {}) {
  const f = normalizeItemFilter(filter)
  const or = list => list.join(t(' or '))
  const parts = []
  if (f.codes.length > 0) {
    parts.push(f.skipCodes.length > 0
      ? t('Codes starting with {codes}, not {skip}', { codes: or(f.codes), skip: or(f.skipCodes) })
      : t('Codes starting with {codes}', { codes: or(f.codes) }))
  } else if (f.skipCodes.length > 0) {
    parts.push(t('Codes not starting with {skip}', { skip: or(f.skipCodes) }))
  }
  if (f.categoryIds.length > 0) {
    parts.push(t('Category: {names}', { names: or(f.categoryIds.map(id => categories[id] ?? `#${id}`)) }))
  }
  if (f.supplierIds.length > 0) {
    parts.push(t('Supplier: {names}', { names: or(f.supplierIds.map(id => suppliers[id] ?? `#${id}`)) }))
  }
  return parts.join('. ')
}

// ---- the figures ------------------------------------------------------------

// The headline numbers, as numbers (cents, units, ratios); the component
// formats them. Discounts are what came off the till's price on the units
// sold (line, deal and invoice discounts); profit only covers sales of goods
// with a known cost.
export function itemFigures (summary, stock = []) {
  const s = summary
  const onHand = stock.reduce((sum, r) => sum + Math.max(0, r.on_hand), 0)
  const discount = s.list_value - s.gross_sales
  const profit = s.costed_revenue - s.cost
  return {
    netSales: s.net_sales,
    grossSales: s.gross_sales,
    share: ratio(s.net_sales, s.all_net_sales),
    netUnits: s.net_units,
    unitsOut: s.units_out,
    unitsBack: s.units_back,
    returnRate: ratio(s.units_back, s.units_out),
    returnedValue: s.returned_value,
    salesCount: s.sales_count,
    avgUnitPrice: ratio(s.gross_sales, s.units_out),
    discount,
    discountRate: ratio(discount, s.list_value),
    discountedLines: s.discounted_lines,
    productsSold: s.products_sold,
    productsInGroup: stock.length,
    sellingDays: s.selling_days,
    unitsPerDay: ratio(s.net_units, s.selling_days),
    customers: s.customers,
    costed: s.costed_revenue > 0,
    profit,
    margin: ratio(profit, s.costed_revenue),
    costedShare: ratio(s.costed_revenue, s.net_sales),
    onHand,
    // Of what there was to sell (sold since the period began plus what is
    // left now), how much went.
    sellThrough: ratio(Math.max(0, s.net_units), Math.max(0, s.net_units) + onHand),
  }
}

// This period against the one before, as SummaryReport's comparison does:
// money and counts change by a percent of themselves, shares by points, and
// a figure that flipped sign (or was zero) has no percent.
export function itemCompareRows (summary, previous) {
  const measures = [
    { key: 'net_sales', measure: t('Net sales'), type: 'money', of: x => x.net_sales },
    { key: 'net_units', measure: t('Units sold (net)'), type: 'int', of: x => x.net_units },
    { key: 'sales_count', measure: t('Sales with these items'), type: 'int', of: x => x.sales_count },
    { key: 'avg_price', measure: t('Average unit price'), type: 'money', of: x => ratio(x.gross_sales, x.units_out) },
    { key: 'share', measure: t('Share of all goods sold'), type: 'pct', of: x => ratio(x.net_sales, x.all_net_sales) },
    { key: 'units_back', measure: t('Units returned'), type: 'int', of: x => x.units_back, lowerIsBetter: true },
    { key: 'discount', measure: t('Discounts given'), type: 'money', of: x => x.list_value - x.gross_sales, lowerIsBetter: true },
    { key: 'products_sold', measure: t('Products sold'), type: 'int', of: x => x.products_sold },
  ]
  return measures.map(m => {
    const now = m.of(summary)
    const before = m.of(previous)
    const delta = m.type === 'pct'
      ? now - before
      : (before && Math.sign(before) === Math.sign(now) ? (now - before) / Math.abs(before) : Number.NaN)
    return { ...m, now, before, delta }
  })
}

// Every day of the period, days with nothing sold at zero.
export function itemDays (daily, from, to) {
  const byDay = new Map(daily.map(d => [day(d.day), d]))
  return daysBetween(from, to).map(ymd => ({ net_units: 0, net_sales: 0, sales_count: 0, ...byDay.get(ymd), day: ymd }))
}

// The group's products with what is left of each: units sold in the period
// (net), on hand now, and the sell-through. stockLeft lists every product
// of the group that sold or has stock; unsold, the ones with stock that
// didn't sell at all, most stock first.
export function itemStock (stock, products) {
  const sold = new Map(products.map(p => [p.product_id, p]))
  const rows = stock.map(s => {
    const p = sold.get(s.product_id)
    const units = p ? p.quantity : 0
    const left = Math.max(0, s.on_hand)
    return {
      product_id: s.product_id,
      product_code: s.product_code,
      product_name: s.product_name,
      category: s.category,
      sold: units,
      revenue: p ? p.revenue : 0,
      on_hand: s.on_hand,
      sell_through: ratio(Math.max(0, units), Math.max(0, units) + left),
      unsold: !p && s.on_hand > 0,
    }
  })
  return {
    stockLeft: rows.filter(r => r.sold !== 0 || r.on_hand !== 0).toSorted((a, b) => (b.sold - a.sold) || (b.on_hand - a.on_hand) || a.product_code.localeCompare(b.product_code)),
    unsold: rows.filter(r => r.unsold).toSorted((a, b) => (b.on_hand - a.on_hand) || a.product_code.localeCompare(b.product_code)),
  }
}

// Who sold the group, the name the till typed or "No salesperson (branch)".
export function itemSalespersons (rows, branchNames = {}) {
  return rows.map(r => ({
    ...r,
    name: r.salesperson_name || t('No salesperson ({branch})', { branch: branchNames[r.branch_id] ?? r.branch_name }),
    branch: branchNames[r.branch_id] ?? r.branch_name,
    avg_price: ratio(r.net_sales, r.net_units),
  })).toSorted((a, b) => b.net_sales - a.net_sales)
}
