import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// The Busy Hours page's numbers (api/busy_hours.go): customers in and sales by
// weekday (1 Monday .. 7 Sunday) and hour, for a range of days.
export function useBusyHours () {
  return {
    getBusyHours: async ({ from, to, branchIds = [] }) => {
      const url = new URL(`${API_BASE}/busy_hours`)
      url.searchParams.set('from', from)
      url.searchParams.set('to', to)
      if (branchIds.length > 0) {url.searchParams.set('branch_ids', branchIds.join(','))}
      const res = await authFetch(url)
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return { visitors: body.visitors ?? [], sales: body.sales ?? [] }
    },
  }
}
