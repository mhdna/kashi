import { API_BASE } from '@/config'
import { authFetch } from './useApi'

// The Reports page's figures (api/reports.go): everything the end-of-period
// reports draw on, for a range of days and a set of branches.
export function useReports () {
  return {
    getReports: async ({ from, to, branchIds = [] }) => {
      const url = new URL(`${API_BASE}/reports`)
      url.searchParams.set('from', from)
      url.searchParams.set('to', to)
      if (branchIds.length > 0) {url.searchParams.set('branch_ids', branchIds.join(','))}
      const res = await authFetch(url)
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return body
    },
  }
}
