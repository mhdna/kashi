import { ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// Unacknowledged security alerts (api/security.go), shared so the nav badge
// and the Security page agree: the page sets it from each overview it loads,
// so acknowledging there clears the badge at once.
const openSecurityAlerts = ref(0)

export function useSecurityAlerts () {
  async function refreshOpenSecurityAlerts () {
    const res = await authFetch(`${API_BASE}/security/alerts/open_count`)
    if (!res.ok) {throw new Error(`Request failed with status ${res.status}`)}
    const data = await res.json()
    openSecurityAlerts.value = data?.open_alerts ?? 0
  }

  return { openSecurityAlerts, refreshOpenSecurityAlerts }
}
