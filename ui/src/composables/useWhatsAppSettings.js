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

// Org-wide WhatsApp Business Platform (Cloud API) config, backed by the
// single-row app_settings table's whatsapp_* columns -- see
// db/migrations/000091_whatsapp_settings and api/whatsapp.go. Branches never
// see this: they POST /branch/whatsapp/thank_you and kashi does the send
// with these credentials.
export function useWhatsAppSettings () {
  async function getWhatsAppSettings () {
    const data = await requestJSON(`${API_BASE}/whatsapp_settings`)
    return data?.settings ?? null
  }

  async function updateWhatsAppSettings (payload) {
    const data = await requestJSON(`${API_BASE}/whatsapp_settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    return data?.settings ?? null
  }

  return { getWhatsAppSettings, updateWhatsAppSettings }
}
