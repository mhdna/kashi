import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Payment methods themselves live centrally (useCashboxAccounts.js). This
// composable edits one branch's exceptions to that set: which methods it
// offers, and where a method's position, color or currency differs there.
//
// A PUT carrying only is_enabled:true and no overrides clears the row --
// the server treats "enabled, nothing pinned" as inheriting central rather
// than storing a row of nulls.

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  if (res.status === 204) {
    return null
  }
  return res.json().catch(() => null)
}

export function useBranchPaymentMethods () {
  async function listBranchPaymentMethods (branchId) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/payment_methods`)
    return data?.payment_methods ?? []
  }

  // Fields left out of `overrides` are cleared back to central's value.
  async function setBranchPaymentMethod (branchId, accountId, { isEnabled = true, ...overrides }) {
    return requestJSON(`${API_BASE}/branches/${branchId}/payment_methods/${accountId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        is_enabled: isEnabled,
        currency_code: overrides.currencyCode ?? null,
        sort_order: overrides.sortOrder ?? null,
        color: overrides.color ?? null,
      }),
    })
  }

  return { listBranchPaymentMethods, setBranchPaymentMethod }
}
