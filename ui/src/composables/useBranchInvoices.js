import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

export function useBranchInvoices () {
  async function listBranchInvoiceItems (invoiceId) {
    const data = await requestJSON(`${API_BASE}/branch_invoices/${invoiceId}/items`)
    return data?.items ?? []
  }

  // Most recently received branch invoices -- ordered by id DESC server-side,
  // which tracks received_at since ids are assigned in receipt order. Used by
  // SyncCard to show real "last synced" data instead of a placeholder.
  async function listRecentBranchInvoices (pageSize = 5) {
    const data = await requestJSON(`${API_BASE}/branch_invoices?page_size=${pageSize}&page_id=0`)
    return data?.branch_invoices ?? []
  }

  // Enqueues a remote "remote_return_invoice" command (see
  // api/branch_command.go) asking the branch to process a return itself,
  // using its own real local IDs -- kashi never fabricates a branch invoice
  // row directly. Returns as soon as kashi has queued it; the branch
  // executes asynchronously and the resulting real return invoice arrives
  // later through the normal /branch/return_invoices sync path, same as
  // any other branch-originated return.
  //
  // `items` is optional -- a list of { branch_product_id, quantity } for
  // the specific lines/quantities to return (branch_product_id comes
  // straight from listBranchInvoiceItems, which is the branch's own local
  // product id -- kashi never reinterprets it). Omitted/empty returns
  // every line on the invoice at full quantity, same as before item-level
  // selection existed.
  //
  // `replacements` makes it an exchange: { variant_id, quantity } lines from
  // kashi's catalogue, which kashi rewrites into the till's own product keys
  // before queueing (see withTillReplacements in api/branch_command.go).
  async function requestRemoteReturn (branchId, invoice, reason, items = [], replacements = []) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        type: 'remote_return_invoice',
        payload: {
          branch_invoice_id: invoice.id,
          client_ref: invoice.client_ref,
          reason: reason || '',
          items,
          replacements,
        },
      }),
    })
    return data?.command ?? null
  }

  // Same command with cancel: true -- the till returns the whole sale and
  // flags it as a cancellation, so the sale and its return both come back
  // marked cancelled_at and drop out of sales statistics. For a sale rung up
  // by mistake; a real customer return stays requestRemoteReturn.
  async function requestRemoteCancel (branchId, invoice, reason) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        type: 'remote_return_invoice',
        payload: {
          branch_invoice_id: invoice.id,
          client_ref: invoice.client_ref,
          reason: reason || '',
          cancel: true,
        },
      }),
    })
    return data?.command ?? null
  }

  // listSalesStats returns { branch_id, day, invoice_count, revenue } rollup
  // rows (sales only, revenue in cents) over an inclusive YYYY-MM-DD range,
  // unpaginated -- the sales-side mirror of useBranchVisitors' listVisitorStats,
  // so the Footfall page can line walk-ins up against sales for one window.
  async function listSalesStats ({ from, to, branchId } = {}) {
    const url = new URL(`${API_BASE}/branch_sales_stats`, window.location.origin)
    url.searchParams.set('from', from)
    url.searchParams.set('to', to)
    if (branchId) {
      url.searchParams.set('branch_id', String(branchId))
    }
    const data = await requestJSON(url.toString())
    return data?.sales_days ?? []
  }

  // The list under the same filters as an Excel file (a Blob).
  async function exportBranchInvoices (params) {
    const url = new URL(`${API_BASE}/branch_invoices/export`)
    for (const [key, value] of Object.entries(params)) {
      if (value !== null && value !== undefined && value !== '') {url.searchParams.set(key, value)}
    }
    const res = await authFetch(url.toString())
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    return res.blob()
  }

  return { exportBranchInvoices, listBranchInvoiceItems, listRecentBranchInvoices, requestRemoteCancel, requestRemoteReturn, listSalesStats }
}
