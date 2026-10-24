import { computed, ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// The WhatsApp failure watch (api/whatsapp_health.go getWhatsAppFailures):
// sale messages still not sent after the till's retries. Shared by the nav
// dot and the WhatsApp overview's banner.
const failures = ref(null)

// Red when several failed within the hour (a number or the bridge is likely
// down), yellow for any failure in the last day, green otherwise; 'loading'
// until the first answer, rather than a guessed green.
const whatsAppFailureStatus = computed(() => failures.value?.status ?? 'loading')

async function refreshWhatsAppFailures () {
  const res = await authFetch(`${API_BASE}/whatsapp/failures`)
  if (!res.ok) {throw new Error(`Request failed with status ${res.status}`)}
  failures.value = await res.json()
  return failures.value
}

export function useWhatsAppFailures () {
  return { failures, whatsAppFailureStatus, refreshWhatsAppFailures }
}
