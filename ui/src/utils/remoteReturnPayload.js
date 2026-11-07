// The remote_return_invoice command body kashi sends a till (kashi-pos
// commands.go remoteReturnInvoiceCommandPayload).
//
// printReceipt is the dialog's "Print the receipt on the till": unticked
// sends print_receipt: false and the till never prints it; ticked sends
// nothing, so the till prints by its own "Print return receipts" setting
// (and a till from before the setting prints as it always has).
export function remoteReturnPayload ({ invoice, reason = '', items, replacements, cancel = false, printReceipt = true }) {
  const payload = {
    branch_invoice_id: invoice.id,
    client_ref: invoice.client_ref,
    reason: reason || '',
  }
  if (cancel) {
    payload.cancel = true
  } else {
    payload.items = items ?? []
    payload.replacements = replacements ?? []
  }
  if (printReceipt === false) {
    payload.print_receipt = false
  }
  return payload
}

export const REMOTE_RETURN_RECEIPT_LABEL = 'Print the receipt on the till'
export const REMOTE_RETURN_RECEIPT_HINT = "Unticked, the till doesn't print it. Ticked, it prints if the till prints return receipts. It can be printed later with Print on till."
