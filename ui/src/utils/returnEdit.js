// Editing a till return or exchange's refund from kashi: an
// edit_return_invoice command (api/branch_command.go) that the till applies
// (kashi-pos return_edit.go) and syncs back. Amounts are cents; a returned
// line's credit is shown positive ("15.00") and sent signed like the line
// (-1500).

// canEditRefund says whether this invoice is a till return or exchange a till
// holds: not a sale, not imported from the old POS, named by its client_ref.
export function canEditRefund (invoice) {
  if (!invoice || typeof invoice !== 'object') {return false}
  if ((invoice.kind !== 'return' && invoice.kind !== 'exchange') || invoice.historical === true) {return false}
  return typeof invoice.client_ref === 'string' && invoice.client_ref.trim() !== ''
}

// The returned lines, with their position among all the invoice's lines (in
// the order kashi lists them, which is the till's): only these can change.
export function refundRows (items) {
  return (items ?? [])
    .map((item, position) => ({ item, position }))
    .filter(({ item }) => Number(item.quantity) < 0)
    .map(({ item, position }) => {
      const max = Math.abs(Number(item.unit_price) * Number(item.quantity))
      const credit = -Number(item.line_total) || 0
      return {
        position,
        barcode: item.barcode || '',
        name: item.product_name || item.product_code || 'Item',
        code: item.product_code || '',
        quantity: Number(item.quantity),
        credit, // cents, positive
        max, // cents: never more back than the price
      }
    })
}

// "15", "15.5", "15.00" -> 1500; null for anything that isn't an amount.
export function parseCredit (text) {
  const s = String(text ?? '').trim()
  if (!/^\d+(\.\d{1,2})?$/.test(s)) {return null}
  return Math.round(Number(s) * 100)
}

// Why an entered credit can't be sent, or '' when it can.
export function creditProblem (row, cents) {
  if (cents === null || cents === undefined || Number.isNaN(cents)) {return 'Enter an amount like 15 or 15.00'}
  if (cents < 0) {return 'Can\'t be negative'}
  if (cents > row.max) {return `At most ${(row.max / 100).toFixed(2)}, its price`}
  return ''
}

// The edit as entered: the changed lines, and what the return comes to.
// credits maps a row's position to its entered credit in cents.
export function refundEdit (invoice, rows, credits) {
  const lines = []
  let moved = 0
  for (const row of rows) {
    const cents = credits[row.position]
    if (cents === undefined || cents === row.credit) {continue}
    lines.push({ position: row.position, barcode: row.barcode, line_total: -cents })
    moved += row.credit - cents // more credit -> a lower (more negative) total
  }
  const oldTotal = Number(invoice?.grand_total ?? 0)
  return { lines, oldTotal, newTotal: oldTotal + moved, moved }
}

export function refundEditPayload (invoice, lines, reason) {
  return { client_ref: invoice.client_ref.trim(), lines, reason: String(reason ?? '').trim() }
}

export function refundEditSentMessage (invoice, branchName) {
  const code = invoice?.branch_invoice_code || 'The return'
  return `${code}: the change was sent to ${branchName || 'the till'}. It shows here once the till applies it (a few seconds if online).`
}
