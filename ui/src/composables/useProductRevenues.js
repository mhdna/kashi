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

function buildParams ({ from, to, categoryValueId, subcategoryValueId, brandValueId } = {}) {
  const url = new URL(API_BASE, window.location.origin)
  url.searchParams.set('from', from)
  url.searchParams.set('to', to)
  if (categoryValueId) { url.searchParams.set('category_value_id', String(categoryValueId)) }
  if (subcategoryValueId) { url.searchParams.set('subcategory_value_id', String(subcategoryValueId)) }
  if (brandValueId) { url.searchParams.set('brand_value_id', String(brandValueId)) }
  return url.searchParams
}

// Data source for the Product Revenues page: revenue/quantity per product
// (listProductRevenue) plus a daily trend rollup (listProductRevenueDaily),
// both scoped to kashi's own admin sales invoices and filterable by the
// category/sub-category/brand product attributes -- see the comment on
// ListProductRevenue in db/query/product_revenue.sql for why branch (POS)
// sales aren't included yet.
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

  return { listProductRevenue, listProductRevenueDaily }
}
