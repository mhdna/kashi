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

// Category -> (color group, only when the item actually has a color, e.g.
// Packs) -> product. Categories whose items have no color (Baby/Kids/Men/
// Lady/Unidentified today) skip the middle level entirely, same 2-level
// shape as before -- this only adds structure where real color data exists,
// so packs never get flattened together with plain clothing.
function buildTree (rows, inventoryId) {
  const source = inventoryId === 'all'
    ? rows
    : rows.filter(r => String(r.inventory_id) === String(inventoryId))

  const categories = new Map()
  for (const r of source) {
    if (!categories.has(r.category_id)) {
      categories.set(r.category_id, { name: r.category_name, color: r.category_color, colorGroups: new Map(), items: new Map() })
    }
    const cat = categories.get(r.category_id)
    const bucket = r.color_name
      ? (cat.colorGroups.has(r.color_name) ? cat.colorGroups.get(r.color_name) : (() => {
        const g = { name: r.color_name, hex: r.color_hex, items: new Map() }
        cat.colorGroups.set(r.color_name, g)
        return g
      })())
      : cat

    if (!bucket.items.has(r.variant_id)) {
      bucket.items.set(r.variant_id, { name: r.product_name, code: r.product_code, qty: 0, costCents: Number(r.variant_cost), locations: [] })
    }
    const item = bucket.items.get(r.variant_id)
    item.qty += Number(r.quantity)
    item.locations.push(r.inventory_name)
  }
  return categories
}

function leafNode (item) {
  return {
    name: item.name,
    value: item.qty * item.costCents,
    qty: item.qty,
    code: item.code,
    location: [...new Set(item.locations)].join(', '),
  }
}

// Same shape/responsibility as useAssetsTreemap, for the "stuff inventory"
// (products/product_variants/inventory_stock) side that feeds the
// Warehouses Treemap. Sized by total book value (qty * avg_cost, both from
// the DB) now that real costs exist for these products; falls back to 0 for
// any variant whose avg_cost is still unset.
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
    const categories = buildTree(rows.value, inventoryId)
    return [...categories.values()].map(cat => {
      const directLeaves = [...cat.items.values()].map(item => leafNode(item))
      const colorGroups = [...cat.colorGroups.values()].map(g => {
        const children = [...g.items.values()].map(item => leafNode(item))
        const value = children.reduce((sum, i) => sum + i.value, 0)
        return { name: g.name, value, itemStyle: { color: g.hex || '#546E7A' }, children }
      })
      const children = [...colorGroups, ...directLeaves]
      const value = children.reduce((sum, i) => sum + i.value, 0)
      return { name: cat.name, value, itemStyle: { color: cat.color || '#546E7A' }, children }
    })
  }

  function totals (inventoryId = 'all') {
    const source = inventoryId === 'all'
      ? rows.value
      : rows.value.filter(r => String(r.inventory_id) === String(inventoryId))
    let qty = 0
    let valueCents = 0
    for (const r of source) {
      qty += Number(r.quantity)
      valueCents += Number(r.quantity) * Number(r.variant_cost)
    }
    return { qty, valueCents }
  }

  function locationOptions () {
    const seen = new Map()
    for (const r of rows.value) {
      if (!seen.has(r.inventory_id)) {seen.set(r.inventory_id, r.inventory_name)}
    }
    return [...seen.entries()].map(([value, title]) => ({ title, value: String(value) }))
  }

  return { rows, loading, fetchRows, buildTreemapData, totals, locationOptions }
}
