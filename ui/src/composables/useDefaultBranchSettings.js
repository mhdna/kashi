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

export function useDefaultBranchSettings () {
  async function getDefaultBranchSettings () {
    const data = await requestJSON(`${API_BASE}/default_branch_settings`)
    return data?.settings ?? null
  }

  async function updateDefaultBranchSettings (payload) {
    const data = await requestJSON(`${API_BASE}/default_branch_settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    return data?.settings ?? null
  }

  // Fans the saved template out to every branch as an update_settings
  // command, same mechanism as editing one branch's settings by hand (see
  // useBranchSettings.updateBranchSettings) -- each branch applies it on its
  // own next sync tick or live push. Returns how many branches were queued.
  async function applyDefaultBranchSettingsToAll () {
    const data = await requestJSON(`${API_BASE}/default_branch_settings/apply_to_all`, {
      method: 'POST',
    })
    return data?.branches_queued ?? 0
  }

  return { getDefaultBranchSettings, updateDefaultBranchSettings, applyDefaultBranchSettingsToAll }
}
