// The Retail invoices page's server-side filters (set in the Filters dialog),
// as the query the API reads and as the short summary shown under the search.
import { t } from '../i18n'

// The names boxes (client, salesperson) take a * wildcard: "Maya*" starts with
// Maya, "*sen" ends with sen, "Ma*ya" has anything between. Without one the
// text matches anywhere in the name. The server reads it the same way.

// What the client box sends: its text without the spaces around it, or null so
// an empty box adds no parameter at all.
export function clientTerm (text) {
  const term = String(text ?? '').trim()
  return term || null
}

// The option lists are functions, so their labels follow the EN/FR toggle.
export function productFieldOptions () {
  return [
    { title: t('Code, name or barcode'), value: 'any' },
    { title: t('Product code'), value: 'code' },
    { title: t('Product name'), value: 'name' },
    { title: t('Barcode'), value: 'barcode' },
  ]
}

export function productMatchOptions () {
  return [
    { title: t('Contains'), value: 'contains' },
    { title: t('Starts with'), value: 'starts' },
    { title: t('Ends with'), value: 'ends' },
    { title: t('Is exactly'), value: 'exact' },
  ]
}

export function kindOptions () {
  return [
    { title: t('Sales'), value: 'sales' },
    { title: t('Returns'), value: 'return' },
    { title: t('Exchanges'), value: 'exchange' },
  ]
}

export function statusOptions () {
  return [
    { title: t('Active'), value: 'active' },
    { title: t('Cancelled'), value: 'cancelled' },
  ]
}

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
    salesperson: '',
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
  branchId, registerId, kind, status, dateFrom, dateTo, client, salesperson,
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
    salesperson: clientTerm(salesperson),
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
  return [p.branch_id, p.register_id, p.kind, p.status, p.date_from, p.date_to, p.client, p.salesperson, p.product, p.total_min, p.total_max]
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
  if (p.branch_id) {out.push(names.branch?.(p.branch_id) ?? t('Branch #{id}', { id: p.branch_id }))}
  if (p.register_id) {out.push(names.register?.(p.register_id) ?? t('Register #{id}', { id: p.register_id }))}
  if (p.kind) {out.push(titleOf(kindOptions(), p.kind))}
  if (p.status) {out.push(titleOf(statusOptions(), p.status))}
  if (p.date_from && p.date_to) {out.push(t('{from} to {to}', { from: p.date_from, to: p.date_to }))}
  else if (p.date_from) {out.push(t('From {date}', { date: p.date_from }))}
  else if (p.date_to) {out.push(t('Until {date}', { date: p.date_to }))}
  if (p.client) {out.push(t('Client "{client}"', { client: p.client }))}
  if (p.salesperson) {out.push(t('Salesperson "{name}"', { name: p.salesperson }))}
  if (p.product) {
    const field = titleOf(productFieldOptions(), filters.productField || 'any')
    const match = titleOf(productMatchOptions(), filters.productMatch || 'contains').toLowerCase()
    out.push(t('{field} {match} "{product}"', { field, match, product: p.product }))
  }
  if (p.total_min !== null && p.total_max !== null) {out.push(t('Total {min} to {max}', { min: money(p.total_min), max: money(p.total_max) }))}
  else if (p.total_min !== null) {out.push(t('Total at least {amount}', { amount: money(p.total_min) }))}
  else if (p.total_max !== null) {out.push(t('Total at most {amount}', { amount: money(p.total_max) }))}
  return out
}

// What's wrong with the filters as typed, or '' when they can be applied.
export function invoiceFilterError (filters = {}) {
  const min = String(filters.totalMin ?? '').trim()
  const max = String(filters.totalMax ?? '').trim()
  if (min !== '' && amountCents(min) === null) {return t('The lowest total is not a number.')}
  if (max !== '' && amountCents(max) === null) {return t('The highest total is not a number.')}
  const lo = amountCents(min)
  const hi = amountCents(max)
  if (lo !== null && hi !== null && lo > hi) {return t('The lowest total is above the highest.')}
  if (filters.dateFrom && filters.dateTo && filters.dateFrom > filters.dateTo) {return t('The start date is after the end date.')}
  return ''
}
