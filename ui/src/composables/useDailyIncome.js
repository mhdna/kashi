import { API_BASE } from '@/config'
import { authFetch } from './useApi'

export function useDailyIncome () {
  // branch is one branch id, or an array of them (empty = every branch).
  // registerIds narrows each register's own branch to just those tills.
  async function getDailyIncome (year, branch, registerIds = []) {
    const url = new URL(`${API_BASE}/branch_invoices/daily_income`, window.location.origin)
    url.searchParams.set('year', year)
    if (Array.isArray(branch)) {
      if (branch.length > 0) {url.searchParams.set('branch_ids', branch.join(','))}
    } else if (branch) {
      url.searchParams.set('branch_id', branch)
    }

    if (registerIds.length > 0) {url.searchParams.set('register_ids', registerIds.join(','))}

    const res = await authFetch(url.toString())
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    const data = await res.json().catch(() => null)
    return data?.days ?? []
  }

  // getDaySummary backs the per-day page a heatmap cell opens: sales split by
  // branch / hour / salesperson / payment account, best sellers, footfall,
  // expenses, closed shifts and the invoice list for one local YYYY-MM-DD
  // day (see api/branch_day_summary.go). Money is in cents.
  async function getDaySummary (day, branchId) {
    const url = new URL(`${API_BASE}/branch_invoices/day_summary`, window.location.origin)
    url.searchParams.set('day', day)
    if (branchId) {
      url.searchParams.set('branch_id', branchId)
    }

    const res = await authFetch(url.toString())
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    return res.json()
  }

  return { getDailyIncome, getDaySummary }
}
