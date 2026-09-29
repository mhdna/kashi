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

// Branch sync conflicts: rows a till pushed that kashi couldn't apply on its
// own -- a client whose phone matches a different name, a client with no
// usable phone, or a product barcode the catalog has never seen. The list
// is polled (30s) by the Conflicts tab; the branch_sync_conflict admin-WS
// message is only a nudge to refresh sooner.
export function useBranchSyncConflicts () {
  async function openConflictCount (branchId) {
    const data = await requestJSON(`${API_BASE}/branch_sync_conflicts?branch_id=${branchId}&status=open&page_size=5`)
    return data?.total ?? 0
  }

  async function resolveConflict (id, body) {
    return requestJSON(`${API_BASE}/branch_sync_conflicts/${id}/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  }

  async function dismissConflict (id) {
    return requestJSON(`${API_BASE}/branch_sync_conflicts/${id}/dismiss`, { method: 'POST' })
  }

  async function bulkResolve (ids, choice) {
    return requestJSON(`${API_BASE}/branch_sync_conflicts/bulk_resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids, choice }),
    })
  }

  return { openConflictCount, resolveConflict, dismissConflict, bulkResolve }
}
