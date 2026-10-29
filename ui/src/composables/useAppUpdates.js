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

function jsonBody (method, body) {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}

// Till app updates: published releases, which release each register is set
// to run, and what every till reports back. See api/app_update.go.
export function useAppUpdates () {
  return {
    // keep: how many old build files stay (0 none, N newest per flavor, -1 all).
    listReleases: async () => {
      const body = await requestJSON(`${API_BASE}/app_releases`)
      return { releases: body?.releases ?? [], keep: body?.keep ?? 0 }
    },
    listRegisters: async () => (await requestJSON(`${API_BASE}/app_updates/registers`))?.registers ?? [],
    listEvents: async () => (await requestJSON(`${API_BASE}/app_updates/events?limit=200`))?.events ?? [],
    // force: install as soon as it's downloaded, even with a shift open.
    setTarget: (releaseId, registerIds, force = false) =>
      requestJSON(`${API_BASE}/app_updates/target`, jsonBody('PUT', { release_id: releaseId, register_ids: registerIds, force })),
    updateNotes: (id, notes) => requestJSON(`${API_BASE}/app_releases/${id}`, jsonBody('PUT', { notes })),
    deleteRelease: id => requestJSON(`${API_BASE}/app_releases/${id}`, { method: 'DELETE' }),
    setKeep: keep => requestJSON(`${API_BASE}/app_updates/keep`, jsonBody('PUT', { keep })),
  }
}
