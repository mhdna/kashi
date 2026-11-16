import { itemQuery } from '@/components/reports/itemReport'
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

    // The Items report's figures (api/item_report.go): one group of goods,
    // picked by code, category and supplier.
    getItemReport: async ({ from, to, branchIds = [], filter }) => {
      const url = new URL(`${API_BASE}/reports/items`)
      for (const [key, value] of Object.entries(itemQuery({ from, to, branchIds, filter }))) {url.searchParams.set(key, value)}
      const res = await authFetch(url)
      const body = await res.json().catch(() => ({}))
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return body
    },

    // Product categories, for the Items report's Category box.
    getProductCategories: async () => {
      const res = await authFetch(`${API_BASE}/product_categories`)
      const body = await res.json().catch(() => [])
      if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
      return Array.isArray(body) ? body : (body.product_categories ?? [])
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
