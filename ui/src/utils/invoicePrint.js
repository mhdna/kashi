// Printing a branch invoice's receipt on a till, from kashi: a reprint_invoice
// command (api/branch_command.go) that kashi routes to the register that rang
// the invoice up; the till prints it with kashi-pos's ReprintInvoice.

// canPrintOnTill says whether some till holds this invoice and can print it.
// An invoice imported from the old POS (historical) lives on no till, and one
// without a client_ref can't be named to the till. Any kind prints: a sale, a
// return or an exchange, cancelled or not.
export function canPrintOnTill (invoice) {
  if (!invoice || typeof invoice !== 'object') {return false}
  if (invoice.historical === true) {return false}
  return typeof invoice.client_ref === 'string' && invoice.client_ref.trim() !== ''
}

// printOnTillPayload is the command body: the till looks the invoice up by
// its client_ref, its own id for it.
export function printOnTillPayload (invoice) {
  return { client_ref: invoice.client_ref.trim() }
}

export function printOnTillSentMessage (invoice, branchName) {
  const code = invoice?.branch_invoice_code || 'The invoice'
  return `${code} sent to ${branchName || 'the till'} — it prints within a few seconds if the till is online.`
}

export const PRINT_ON_TILL_TITLE = 'Print this receipt on the till that rang it up'
