import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (path, options) {
  const res = await authFetch(`${API_BASE}${path}`, options)
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
  return body
}

function sendJSON (method, path, payload) {
  return requestJSON(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

// The WhatsApp page: post-sale message rules, the message log and its stats
// -- see api/whatsapp_rules.go. The Cloud API credentials and the default
// template stay on Settings > WhatsApp (useWhatsAppSettings).
export function useWhatsApp () {
  return {
    getRules: async () => (await requestJSON('/whatsapp/rules')).rules ?? [],
    saveRules: async rules => (await sendJSON('PUT', '/whatsapp/rules', { rules })).rules ?? [],
    testRules: payload => sendJSON('POST', '/whatsapp/rules/test', payload),
    setMessageCost: costMicros => sendJSON('PUT', '/whatsapp/cost', { cost_micros: costMicros }),
    getStats: params => requestJSON(`/whatsapp/stats?${new URLSearchParams(params)}`),
    listMessages: params => requestJSON(`/whatsapp/messages?${new URLSearchParams(params)}`),
    searchClients: async search =>
      (await requestJSON(`/whatsapp/clients?${new URLSearchParams({ search })}`)).clients ?? [],
    clientsByPhones: async phones =>
      phones.length === 0
        ? []
        : (await requestJSON(`/whatsapp/clients?${new URLSearchParams({ phones: phones.join(',') })}`)).clients ?? [],
  }
}
