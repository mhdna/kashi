import { ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Branches don't fit the generic createResource shape (see useApiResource.js):
// create returns { branch, register, api_key } instead of the bare row,
// there's no update/delete, and activate/deactivate are bespoke POST
// actions. So this composable is hand-written rather than built on
// createResource. A branch's tills ("registers") each log in with their own
// code and key; those calls live here too.

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

const branches = ref([])
const loaded = ref(false)

export function useBranches () {
  async function fetchBranches (force = false) {
    if (loaded.value && !force) {
      return branches.value
    }
    const data = await requestJSON(`${API_BASE}/branches`)
    branches.value = data?.branches ?? []
    loaded.value = true
    return branches.value
  }

  async function createBranch ({ name, code }) {
    const data = await requestJSON(`${API_BASE}/branches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, code }),
    })
    branches.value.push(data.branch)
    return data // { branch, api_key } -- api_key is shown once, never again
  }

  async function setBranchActive (id, active) {
    await requestJSON(`${API_BASE}/branches/${id}/${active ? 'activate' : 'deactivate'}`, { method: 'POST' })
    const branch = branches.value.find(b => b.id === id)
    if (branch) {
      branch.is_active = active
    }
  }

  async function listRegisters (branchId) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/registers`)
    return data?.registers ?? []
  }

  async function createRegister (branchId, { code, name }) {
    return requestJSON(`${API_BASE}/branches/${branchId}/registers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, name }),
    }) // { register, api_key } -- api_key is shown once, never again
  }

  async function updateRegister (id, { name }) {
    const data = await requestJSON(`${API_BASE}/branch_registers/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    return data?.register
  }

  async function setRegisterActive (id, active) {
    await requestJSON(`${API_BASE}/branch_registers/${id}/${active ? 'activate' : 'deactivate'}`, { method: 'POST' })
  }

  async function rotateRegisterKey (id) {
    return requestJSON(`${API_BASE}/branch_registers/${id}/rotate_key`, { method: 'POST' }) // { id, api_key }
  }

  async function fetchConnectedRegisters () {
    const data = await requestJSON(`${API_BASE}/branch_registers/connected`)
    return data?.register_ids ?? []
  }

  // Initial value for "which branches are live right now" -- kept fresh
  // after this by the "branch_connection_changed" push over the admin
  // WebSocket (see ConnectedBranchesCard.vue), same poll-then-push pattern
  // used for recent branch invoices.
  async function fetchConnectedBranches () {
    const data = await requestJSON(`${API_BASE}/branches/connected`)
    return data?.branch_ids ?? []
  }

  // Initial value for "which branches are mid a full sync right now" --
  // kept fresh after this by the "branch_sync_status_changed" push over the
  // admin WebSocket (see SyncingBranchesCard.vue), same poll-then-push
  // pairing as fetchConnectedBranches above.
  async function fetchSyncingBranches () {
    const data = await requestJSON(`${API_BASE}/branches/syncing`)
    return data?.branch_ids ?? []
  }

  return {
    branches,
    fetchBranches,
    createBranch,
    setBranchActive,
    fetchConnectedBranches,
    fetchSyncingBranches,
    listRegisters,
    createRegister,
    updateRegister,
    setRegisterActive,
    rotateRegisterKey,
    fetchConnectedRegisters,
  }
}
