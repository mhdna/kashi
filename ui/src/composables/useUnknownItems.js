import { API_BASE } from '@/config'
import { authFetch } from './useApi'
import { PHOTO_FETCH } from './usePhotoActions'

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

// The pictured units tills sold as "Unknown item", waiting for someone to say
// what they were (api/unknown_items.go).
export function useUnknownItems () {
  async function fetchUnknownItems ({ resolved = false, pageId = 0, pageSize = 24 } = {}) {
    const data = await requestJSON(
      `${API_BASE}/unknown_items?resolved=${resolved}&page_id=${pageId}&page_size=${pageSize}`,
    )
    return { items: data?.items ?? [], total: data?.total ?? 0, pending: data?.pending ?? 0 }
  }

  async function fetchWaiting () {
    const data = await requestJSON(`${API_BASE}/unknown_items/waiting`)
    return data?.items ?? []
  }

  async function waiveUnknownItems (slotIds) {
    const data = await requestJSON(`${API_BASE}/unknown_items/waive`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ slot_ids: slotIds }),
    })
    return data?.waived ?? 0
  }

  async function searchVariants (search) {
    const data = await requestJSON(`${API_BASE}/unknown_items/variants?search=${encodeURIComponent(search)}`)
    return data?.variants ?? []
  }

  function resolveUnknownItems (slotIds, variantId) {
    return requestJSON(`${API_BASE}/unknown_items/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ slot_ids: slotIds, variant_id: variantId }),
    })
  }

  // <img> can't carry the Bearer token, so the picture comes back as an
  // object URL (the caller revokes it).
  async function fetchImageObjectUrl (imageId, { width } = {}) {
    const q = width ? `?w=${width}` : ''
    const res = await authFetch(`${API_BASE}/unknown_items/images/${imageId}/file${q}`, PHOTO_FETCH)
    if (!res.ok) throw new Error(`Failed to load picture ${imageId}`)
    return URL.createObjectURL(await res.blob())
  }

  return { fetchUnknownItems, fetchWaiting, waiveUnknownItems, searchVariants, resolveUnknownItems, fetchImageObjectUrl }
}
