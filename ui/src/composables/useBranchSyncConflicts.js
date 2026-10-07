import { computed, ref } from 'vue'
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

// Shared across the nav drawer and the Conflicts page (see commandsStatus in
// useBranchCommands for the same idea).
const conflictSummary = ref([])

// Red when a sale took stock below zero somewhere (oversold: stock needs a
// fix), yellow for any other open conflict, green when none are open.
const conflictsStatus = computed(() => {
  if (conflictSummary.value.some(b => b.open_oversold > 0)) {return 'critical'}
  if (conflictSummary.value.some(b => b.open > 0)) {return 'warning'}
  return 'healthy'
})

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

  async function refreshConflictSummary () {
    const data = await requestJSON(`${API_BASE}/branch_sync_conflicts/summary`)
    conflictSummary.value = data?.branches ?? []
    return conflictSummary.value
  }

  return {
    conflictSummary,
    conflictsStatus,
    refreshConflictSummary,
    openConflictCount,
    resolveConflict,
    dismissConflict,
    bulkResolve,
  }
}
