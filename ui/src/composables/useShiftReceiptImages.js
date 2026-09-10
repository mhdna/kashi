import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Mirrors useExpenseImages.js -- the close-shift receipt photo a cashier
// uploads from the POS (see api/branch_shift_receipt_image.go) is fetched
// the same way: a Storage-explorer feed plus a per-image blob route, both
// Bearer-gated like the rest of the admin API.
async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

export function useShiftReceiptImages () {
  async function fetchShiftReceiptImages (pageId = 0, pageSize = 24) {
    const data = await requestJSON(`${API_BASE}/branch_shift_receipt_images?page_id=${pageId}&page_size=${pageSize}`)
    return { images: data?.images ?? [], total: data?.total ?? 0 }
  }

  // Metadata for one branch_shifts row's photo, or null if none was
  // uploaded -- the Shifts page uses this after clicking a row whose
  // has_receipt_image flag is set.
  async function fetchShiftReceiptImageForShift (branchShiftId) {
    const res = await authFetch(`${API_BASE}/branch_shifts/${branchShiftId}/receipt_image`)
    if (res.status === 404) {
      return null
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    const data = await res.json().catch(() => null)
    return data?.image ?? null
  }

  // <img src> can't carry an Authorization header, so the file is fetched
  // as a blob and handed back as an object URL the caller can bind
  // directly.
  async function fetchShiftReceiptImageObjectUrl (imageId) {
    const res = await authFetch(`${API_BASE}/branch_shift_receipt_images/${imageId}/file`)
    if (!res.ok) {
      throw new Error(`Failed to load image ${imageId}`)
    }
    const blob = await res.blob()
    return URL.createObjectURL(blob)
  }

  return { fetchShiftReceiptImages, fetchShiftReceiptImageForShift, fetchShiftReceiptImageObjectUrl }
}
