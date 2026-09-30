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
      groups.set(r.category_id, { name: r.category_name, color: r.category_color, items: new Map() })
    }
    const group = groups.get(r.category_id)
    if (!group.items.has(r.asset_id)) {
      group.items.set(r.asset_id, { name: r.asset_name, code: r.asset_code, qty: 0, costCents: Number(r.asset_cost), locations: [] })
    }
    const item = group.items.get(r.asset_id)
    item.qty += Number(r.quantity)
    item.locations.push(r.inventory_name)
  }
  return groups
}

// Fetches the flat (inventory, category, asset) rows once and reshapes them
// on demand into the echarts treemap tree AssetTreemap.vue renders, and into
// the location dropdown options assets-treemap.vue needs. Sized by total
// book value (qty * per-unit cost, both from the DB) -- falls back to
// quantity only if an asset's cost is still 0 (not backfilled yet).
export function useAssetsTreemap () {
  const rows = ref([])
  const loaded = ref(false)
  const loading = ref(false)

  async function fetchRows (force = false) {
    if (loaded.value && !force) {return rows.value}
    loading.value = true
    try {
      const data = await requestJSON(`${API_BASE}/assets/treemap`)
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
      const items = [...g.items.values()].map(item => ({
        name: item.name,
        value: item.qty * item.costCents,
        qty: item.qty,
        code: item.code,
        location: [...new Set(item.locations)].join(', '),
      }))
      const value = items.reduce((sum, i) => sum + i.value, 0)
      return {
        name: g.name,
        value,
        itemStyle: { color: g.color || '#546E7A' },
        children: items,
      }
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
      valueCents += Number(r.quantity) * Number(r.asset_cost)
    }
    return { qty, valueCents }
  }

  // Distinct locations actually present in the data, so the dropdown never
  // lists an inventory (e.g. an unrelated branch's auto-created store) that
  // has no assets in it.
  function locationOptions () {
    const seen = new Map()
    for (const r of rows.value) {
      if (!seen.has(r.inventory_id)) {seen.set(r.inventory_id, r.inventory_name)}
    }
    return [...seen.entries()].map(([value, title]) => ({ title, value: String(value) }))
  }

  return { rows, loading, fetchRows, buildTreemapData, totals, locationOptions }
}
