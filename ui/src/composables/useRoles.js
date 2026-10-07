import { ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Roles and the sections (each with its pages) they grant access to, from
// GET /roles (admin only). Module-scope so Settings > Users (role picker) and Settings > Roles
// (editor) share one copy. `sections` comes from the server so the editor
// always lists exactly what the API enforces (api/access.go).
const roles = ref([])
const sections = ref([])

async function requestJSON (path, options) {
  const res = await authFetch(`${API_BASE}${path}`, options)
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(body.error || `Request failed with status ${res.status}`)
  }
  return body
}

function json (method, body) {
  return {
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
}
}

export function useRoles () {
  async function fetchRoles () {
    const data = await requestJSON('/roles')
    roles.value = data.roles ?? []
    sections.value = data.sections ?? []
    return roles.value
  }

  // `permissions` is { sections: { [sectionKey]: on }, pages: { [pageKey]: { view, edit } } }.
  async function createRole (name, permissions) {
    await requestJSON('/roles', json('POST', { name, permissions }))
    await fetchRoles()
  }

  async function updateRole (id, name, permissions) {
    await requestJSON(`/roles/${id}`, json('PUT', { name, permissions }))
    await fetchRoles()
  }

  async function deleteRole (id) {
    await requestJSON(`/roles/${id}`, { method: 'DELETE' })
    await fetchRoles()
  }

  return { roles, sections, fetchRoles, createRole, updateRole, deleteRole }
}
