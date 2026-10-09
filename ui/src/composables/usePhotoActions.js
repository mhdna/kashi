import { computed } from 'vue'
import { API_BASE } from '@/config'
import { useAuthStore } from '@/stores/auth'
import { authFetch } from './useApi'

// What the admin photo viewer (ZoomableImage) can do to an uploaded photo
// besides showing it: turn it (api/photo_rotation.go) or swap it for another
// file (api/photo_replace.go). Either way the server changes what it serves
// for the photo, so afterwards the caller only re-fetches the photo (and
// its thumbnail).
// Fetch options for a stored photo: always revalidate. The server answers
// with an ETag that names the photo's rotation, so a turned or replaced
// photo comes back new -- but only if the browser asks; a copy cached under
// the old day-long max-age would otherwise be reused without asking.
export const PHOTO_FETCH = { cache: 'no-cache' }

const PHOTO_PATHS = {
  expense_image: id => `/branch_expense_images/${id}`,
  shift_receipt: id => `/branch_shift_receipt_images/${id}`,
  transfer: id => `/transfers/${id}/attachment`,
}

async function send (url, init) {
  const res = await authFetch(url, { method: 'PUT', ...init })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Couldn't change the photo (status ${res.status})`)
  }
}

export function usePhotoActions () {
  const auth = useAuthStore()

  // "Rotate Photos" / "Replace Photos" in Settings > Roles; admins always
  // can.
  const canRotate = computed(() => auth.canEdit('photo_rotation'))
  const canReplace = computed(() => auth.canEdit('photo_replace'))

  // delta is -90 (left), 90 (right) or 180, added to the saved turn.
  function rotatePhoto (kind, id, delta) {
    return send(`${API_BASE}${PHOTO_PATHS[kind](id)}/rotation`, {
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ delta }),
    })
  }

  // file is the picked File; the old photo is deleted on the server.
  function replacePhoto (kind, id, file) {
    const body = new FormData()
    body.append('image', file)
    return send(`${API_BASE}${PHOTO_PATHS[kind](id)}/replace`, { body })
  }

  return { canRotate, canReplace, rotatePhoto, replacePhoto }
}
