import { API_BASE } from '@/config'
import { authFetch } from './useApi'
import { PHOTO_FETCH } from './usePhotoActions'

// Storage explorer feed for transfer photos -- same shape and blob/object-URL
// approach as useExpenseImages.js. A transfer has at most one photo, so the
// row id is the transfer id.
export function useTransferAttachments () {
  async function fetchTransferAttachments (pageId = 0, pageSize = 24) {
    const res = await authFetch(`${API_BASE}/transfer_attachments?page_id=${pageId}&page_size=${pageSize}`)
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    const data = await res.json().catch(() => null)
    return { images: data?.images ?? [], total: data?.total ?? 0 }
  }

  async function fetchTransferAttachmentObjectUrl (transferId, { width } = {}) {
    const q = width ? `?w=${width}` : ''
    const res = await authFetch(`${API_BASE}/transfers/${transferId}/attachment${q}`, PHOTO_FETCH)
    if (!res.ok) {
      throw new Error(`Failed to load the photo of transfer ${transferId}`)
    }
    return URL.createObjectURL(await res.blob())
  }

  return { fetchTransferAttachments, fetchTransferAttachmentObjectUrl }
}
