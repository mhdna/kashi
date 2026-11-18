// The Retail invoices page's server-side filters (set in the Filters dialog),
// as the query the API reads and as the short summary shown under the search.

// What the client box sends: its text without the spaces around it, or null so
// an empty box adds no parameter at all.
export function clientTerm (text) {
  const term = String(text ?? '').trim()
  return term || null
}

export const productFieldOptions = [
  { title: 'Code, name or barcode', value: 'any' },
  { title: 'Product code', value: 'code' },
  { title: 'Product name', value: 'name' },
  { title: 'Barcode', value: 'barcode' },
]

export const productMatchOptions = [
  { title: 'Contains', value: 'contains' },
  { title: 'Starts with', value: 'starts' },
  { title: 'Ends with', value: 'ends' },
  { title: 'Is exactly', value: 'exact' },
]

export const kindOptions = [
  { title: 'Sales', value: 'sales' },
  { title: 'Returns', value: 'return' },
  { title: 'Exchanges', value: 'exchange' },
]

export const statusOptions = [
  { title: 'Active', value: 'active' },
  { title: 'Cancelled', value: 'cancelled' },
]

// Every filter the dialog sets, cleared.
export function emptyFilters () {
  return {
    branchId: null,
    registerId: null,
    kind: null,
    status: null,
    dateFrom: '',
    dateTo: '',
    client: '',
    product: '',
    productField: 'any',
    productMatch: 'contains',
    totalMin: '',
    totalMax: '',
  }
}

// A dollar amount typed in a total box, in cents, or null when the box is
// empty or not a number. Rounded to the cent so 12.345 is 1235, not 1234.5.
export function amountCents (text) {
  const raw = String(text ?? '').trim()
  if (raw === '') {return null}
  const value = Number(raw)
  if (!Number.isFinite(value)) {return null}
  return Math.round(value * 100)
}

export function invoiceFilterParams ({
  branchId, registerId, kind, status, dateFrom, dateTo, client,
  product, productField, productMatch, totalMin, totalMax,
} = {}) {
  const term = String(product ?? '').trim()
  return {
    branch_id: branchId || null,
    register_id: registerId ?? null,
    kind: kind ?? null,
    status: status ?? null,
    date_from: dateFrom || '',
    date_to: dateTo || '',
    client: clientTerm(client),
    // Field and mode only go with a term, and only when they aren't the default.
    product: term || null,
    product_field: term && productField && productField !== 'any' ? productField : null,
    product_match: term && productMatch && productMatch !== 'contains' ? productMatch : null,
    total_min: amountCents(totalMin),
    total_max: amountCents(totalMax),
  }
}

// How many filters are set (the number on the Filters button). The product
// box counts once, whatever its field and mode.
export function countInvoiceFilters (filters = {}) {
  const p = invoiceFilterParams(filters)
  return [p.branch_id, p.register_id, p.kind, p.status, p.date_from, p.date_to, p.client, p.product, p.total_min, p.total_max]
    .filter(v => v !== null && v !== '')
.length
}

// True when any filter is set (shows Clear).
export function hasInvoiceFilters (filters = {}) {
  return countInvoiceFilters(filters) > 0
}

function titleOf (options, value) {
  return options.find(o => o.value === value)?.title ?? value
}

function money (cents) {
  return (cents / 100).toFixed(2)
}

// The filters in words, one phrase each, for the line under the search box.
// names: { branch(id), register(id) } turn ids into what the user picked.
export function invoiceFilterSummary (filters = {}, names = {}) {
  const p = invoiceFilterParams(filters)
  const out = []
  if (p.branch_id) {out.push(names.branch?.(p.branch_id) ?? `Branch #${p.branch_id}`)}
  if (p.register_id) {out.push(names.register?.(p.register_id) ?? `Register #${p.register_id}`)}
  if (p.kind) {out.push(titleOf(kindOptions, p.kind))}
  if (p.status) {out.push(titleOf(statusOptions, p.status))}
  if (p.date_from && p.date_to) {out.push(`${p.date_from} to ${p.date_to}`)}
  else if (p.date_from) {out.push(`From ${p.date_from}`)}
  else if (p.date_to) {out.push(`Until ${p.date_to}`)}
  if (p.client) {out.push(`Client "${p.client}"`)}
  if (p.product) {
    const field = titleOf(productFieldOptions, filters.productField || 'any')
    const match = titleOf(productMatchOptions, filters.productMatch || 'contains').toLowerCase()
    out.push(`${field} ${match} "${p.product}"`)
  }
  if (p.total_min !== null && p.total_max !== null) {out.push(`Total ${money(p.total_min)} to ${money(p.total_max)}`)}
  else if (p.total_min !== null) {out.push(`Total at least ${money(p.total_min)}`)}
  else if (p.total_max !== null) {out.push(`Total at most ${money(p.total_max)}`)}
  return out
}

// What's wrong with the filters as typed, or '' when they can be applied.
export function invoiceFilterError (filters = {}) {
  const min = String(filters.totalMin ?? '').trim()
  const max = String(filters.totalMax ?? '').trim()
  if (min !== '' && amountCents(min) === null) {return 'The lowest total is not a number.'}
  if (max !== '' && amountCents(max) === null) {return 'The highest total is not a number.'}
  const lo = amountCents(min)
  const hi = amountCents(max)
  if (lo !== null && hi !== null && lo > hi) {return 'The lowest total is above the highest.'}
  if (filters.dateFrom && filters.dateTo && filters.dateFrom > filters.dateTo) {return 'The start date is after the end date.'}
  return ''
}
