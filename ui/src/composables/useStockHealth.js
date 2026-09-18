import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url) {
  const res = await authFetch(url)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

// Overall inventory health: days-of-inventory, value, and the same
// breakdown per inventory location -- see api/stock_health.go. Backs both
// the Stock Health page and the sidebar status dot.
export function useStockHealth () {
  async function fetchStockHealth () {
    return requestJSON(`${API_BASE}/stock_health`)
  }

  // Per-location (warehouse or branch store) breakdown -- same metrics as
  // fetchStockHealth, one row per inventory. page_size caps at 100
  // server-side; this app's location count stays well under that, so one
  // unpaginated call is enough for the Inventory Analysis page's charts and
  // location table.
  async function fetchStockHealthByInventory ({ search = '', pageSize = 100, pageId = 0 } = {}) {
    const params = new URLSearchParams({ search, page_size: pageSize, page_id: pageId })
    const data = await requestJSON(`${API_BASE}/stock_health/by_inventory?${params}`)
    return { inventories: data?.inventories ?? [], total: data?.total ?? 0 }
  }

  return { fetchStockHealth, fetchStockHealthByInventory }
}
