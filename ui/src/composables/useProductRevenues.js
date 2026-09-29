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

function buildParams ({ from, to, categoryValueId, subcategoryValueId, brandValueId, branchId } = {}) {
  const url = new URL(API_BASE, window.location.origin)
  url.searchParams.set('from', from)
  url.searchParams.set('to', to)
  if (categoryValueId) { url.searchParams.set('category_value_id', String(categoryValueId)) }
  if (subcategoryValueId) { url.searchParams.set('subcategory_value_id', String(subcategoryValueId)) }
  if (brandValueId) { url.searchParams.set('brand_value_id', String(brandValueId)) }
  if (branchId) { url.searchParams.set('branch_id', String(branchId)) }
  return url.searchParams
}

// Data source for the Product Revenues page, which has two tabs backed by
// two independent sources -- see the long comment on ListBranchProductRevenue
// in db/query/product_revenue.sql for why they can't be merged into one
// figure:
//   - Kashi sales: real revenue from kashi's own admin sales invoices
//     (listProductRevenue/listProductRevenueDaily).
//   - Kashi-POS sales: units sold + an ESTIMATED revenue (current catalog
//     price x quantity) from branch sales, the only branch-sales source
//     with a real per-product link (listBranchProductRevenue/
//     listBranchProductRevenueDaily/listBranchRevenueByBranch). Filterable
//     by branch, unlike the admin tab.
export function useProductRevenues () {
  async function listProductRevenue (filters) {
    const params = buildParams(filters)
    const data = await requestJSON(`${API_BASE}/product_revenue?${params.toString()}`)
    return data?.products ?? []
  }

  async function listProductRevenueDaily (filters) {
    const params = buildParams(filters)
    const data = await requestJSON(`${API_BASE}/product_revenue/daily?${params.toString()}`)
    return data?.days ?? []
  }

  async function listBranchProductRevenue (filters) {
    const params = buildParams(filters)
    const data = await requestJSON(`${API_BASE}/branch_product_revenue?${params.toString()}`)
    return data?.products ?? []
  }

  async function listBranchProductRevenueDaily (filters) {
    const params = buildParams(filters)
    const data = await requestJSON(`${API_BASE}/branch_product_revenue/daily?${params.toString()}`)
    return data?.days ?? []
  }

  async function listBranchRevenueByBranch (filters) {
    const params = buildParams(filters)
    const data = await requestJSON(`${API_BASE}/branch_product_revenue/by_branch?${params.toString()}`)
    return data?.branches ?? []
  }

  return {
    listProductRevenue,
    listProductRevenueDaily,
    listBranchProductRevenue,
    listBranchProductRevenueDaily,
    listBranchRevenueByBranch,
  }
}
