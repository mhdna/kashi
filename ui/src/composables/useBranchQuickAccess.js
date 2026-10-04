import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Quick Access is the row of product shortcut cards on a till's selling
// screen. A branch has one central list here; each of its registers chooses
// whether to show that list ('kashi') or keep its own ('till', the default).
// Saving either reaches the tills within a second or so -- kashi's database
// pushes the change (see api/branch_quick_access.go).

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

export function useBranchQuickAccess () {
  // { items: [...cards in order], registers: [{ id, code, name, mode, ... }] }
  async function getBranchQuickAccess (branchId) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/quick_access`)
    return { items: data?.items ?? [], registers: data?.registers ?? [] }
  }

  // Replaces the whole list; the order of `items` is the card order.
  async function saveBranchQuickAccess (branchId, items) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/quick_access`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        items: items.map(i => ({
          product_variant_id: i.product_variant_id,
          color: i.color ?? '',
          text_color: i.text_color ?? '',
        })),
      }),
    })
    return { items: data?.items ?? [], registers: data?.registers ?? [] }
  }

  async function setRegisterQuickAccessMode (registerId, mode) {
    return requestJSON(`${API_BASE}/branch_registers/${registerId}/quick_access_mode`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode }),
    })
  }

  return { getBranchQuickAccess, saveBranchQuickAccess, setRegisterQuickAccessMode }
}
