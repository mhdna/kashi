// Opening a sale for exchange past the branch's exchange window: the admin
// picks "only today" or a number of days counting today, and the till lets
// the sale be exchanged until the end of the last one (open_invoice_exchange,
// kashi-pos exchange_opening.go).

export const MAX_OPENING_DAYS = 365

function pad (n) {
  return String(n).padStart(2, '0')
}

// YYYY-MM-DD of a Date in the browser's own zone (the shop's, for an admin
// in the same country). toISOString would give the UTC day instead.
export function localDateString (date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

// The number of days the opening covers, or 0 when the input can't be sent.
export function openingDays (mode, days) {
  if (mode === 'today') {return 1}
  const n = Number(days)
  if (!Number.isInteger(n) || n < 1 || n > MAX_OPENING_DAYS) {return 0}
  return n
}

// The last day the sale stays open, as YYYY-MM-DD, or '' when the input is
// invalid. Five days from Sunday the 22nd ends on Thursday the 26th.
export function openingUntilDate (mode, days, now = new Date()) {
  const n = openingDays(mode, days)
  if (!n) {return ''}
  const last = new Date(now.getFullYear(), now.getMonth(), now.getDate() + n - 1)
  return localDateString(last)
}

// "Thu 26 Nov 2026" for a YYYY-MM-DD.
export function formatOpeningDate (value) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value || '')
  if (!match) {return ''}
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
  return date.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' })
}

export function openingPayload (invoice, mode, days, reason, now = new Date()) {
  return {
    client_ref: invoice.client_ref,
    until_date: openingUntilDate(mode, days, now),
    reason: (reason || '').trim(),
  }
}

// Whether the invoice details can offer the button: a sale a till has.
export function canOpenForExchange (invoice) {
  return Boolean(invoice && invoice.kind === 'sales' && !invoice.historical && invoice.client_ref
    && !(invoice.cancelled_at?.Valid ?? invoice.cancelled_at))
}
