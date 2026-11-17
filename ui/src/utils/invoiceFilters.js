// The Retail invoices page's server-side filters, as the query the API reads.

// What the client box sends: its text without the spaces around it, or null so
// an empty box adds no parameter at all.
export function clientTerm (text) {
  const term = String(text ?? '').trim()
  return term || null
}

export function invoiceFilterParams ({ branchId, registerId, kind, status, dateFrom, dateTo, client }) {
  return {
    branch_id: branchId || null,
    register_id: registerId ?? null,
    kind: kind ?? null,
    status: status ?? null,
    date_from: dateFrom || '',
    date_to: dateTo || '',
    client: clientTerm(client),
  }
}

// True when any filter beyond the branch and search boxes is set (shows Clear).
export function hasInvoiceFilters ({ registerId, kind, status, dateFrom, dateTo, client }) {
  return Boolean(registerId || kind || status || dateFrom || dateTo || clientTerm(client))
}
