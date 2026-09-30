import { ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url) {
  const res = await authFetch(url)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json()
}

function groupRows (rows, inventoryId) {
  const source = inventoryId === 'all'
    ? rows
    : rows.filter(r => String(r.inventory_id) === String(inventoryId))

  const groups = new Map()
  for (const r of source) {
    if (!groups.has(r.category_id)) {
      groups.set(r.category_id, { name: r.category_name, color: r.category_color, code: `C${r.category_id}`, items: new Map() })
    }
    const group = groups.get(r.category_id)
    if (!group.items.has(r.variant_id)) {
      group.items.set(r.variant_id, { name: r.product_name, code: r.product_code, qty: 0, locations: [] })
    }
    const item = group.items.get(r.variant_id)
    item.qty += Number(r.quantity)
    item.locations.push(r.inventory_name)
  }
  return groups
}

// Same shape/responsibility as useAssetsTreemap, for the "stuff inventory"
// (products/product_variants/inventory_stock) side that feeds the
// Warehouses Treemap.
export function useWarehouseTreemap () {
  const rows = ref([])
  const loaded = ref(false)
  const loading = ref(false)

  async function fetchRows (force = false) {
    if (loaded.value && !force) {return rows.value}
    loading.value = true
    try {
      const data = await requestJSON(`${API_BASE}/inventories/treemap`)
      rows.value = data.rows ?? []
      loaded.value = true
      return rows.value
    } finally {
      loading.value = false
    }
  }

  function buildTreemapData (inventoryId = 'all') {
    const groups = groupRows(rows.value, inventoryId)
    return [...groups.values()].map(g => {
      const children = [...g.items.values()].map(item => ({
        name: item.name,
        value: item.qty,
        code: item.code,
        location: [...new Set(item.locations)].join(', '),
      }))
      return { name: g.name, code: g.code, itemStyle: { color: g.color || '#546E7A' }, children }
    })
  }

  function totalQty (inventoryId = 'all') {
    const source = inventoryId === 'all'
      ? rows.value
      : rows.value.filter(r => String(r.inventory_id) === String(inventoryId))
    return source.reduce((sum, r) => sum + Number(r.quantity), 0)
  }

  function locationOptions () {
    const seen = new Map()
    for (const r of rows.value) {
      if (!seen.has(r.inventory_id)) {seen.set(r.inventory_id, r.inventory_name)}
    }
    return [...seen.entries()].map(([value, title]) => ({ title, value: String(value) }))
  }

  return { rows, loading, fetchRows, buildTreemapData, totalQty, locationOptions }
}
