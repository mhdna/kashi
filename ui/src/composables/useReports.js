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

    // The Product group report (api/report_product_group.go): the sales of
    // the products whose code starts with one of codes.
    getProductGroupReport: async ({ from, to, branchIds = [], codes }) => {
      const url = new URL(`${API_BASE}/reports/product_group`)
      url.searchParams.set('from', from)
      url.searchParams.set('to', to)
      url.searchParams.set('codes', codes.join(','))
      if (branchIds.length > 0) {url.searchParams.set('branch_ids', branchIds.join(','))}
      const res = await authFetch(url)
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return body
    },

    // The page's saved defaults and the products and customers left out of
    // every report (api/reports.go). Read and written whole.
    getReportSettings: async () => {
      const res = await authFetch(`${API_BASE}/reports/settings`)
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return body.settings ?? {}
    },
    saveReportSettings: async settings => {
      const res = await authFetch(`${API_BASE}/reports/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ settings }),
      })
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return body.settings
    },
  }
}
