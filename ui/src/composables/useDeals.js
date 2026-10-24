import { ref } from 'vue'
import { API_BASE } from '@/config'
import { formatMoney } from '@/utils/money'
import { authFetch } from './useApi'

// Deals (api/deals.go, 000201): "Buy 2 get 1 free", "3 for $50", "Buy 3 or
// more, 20% off all". Module-cached like the lists: the discount list page,
// its rules and the deals dialog all read the same few rows.
const deals = ref([])
const loaded = ref(false)

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {throw new Error(data.error || `Request failed with status ${res.status}`)}
  return data
}

async function fetchDeals (force = false) {
  if (loaded.value && !force) {return deals.value}
  const data = await requestJSON(`${API_BASE}/deals`)
  deals.value = data.deals ?? []
  loaded.value = true
  return deals.value
}

async function saveDeal (deal) {
  const body = {
    name: deal.name, kind: deal.kind, buy_qty: deal.buy_qty, get_qty: deal.get_qty,
    percent: deal.percent, bundle_price: deal.bundle_price,
    fallback_percent: deal.fallback_percent ?? 0,
    fallback_only_without_set: deal.fallback_only_without_set ?? false,
  }
  await requestJSON(deal.id ? `${API_BASE}/deals/${deal.id}` : `${API_BASE}/deals`, {
    method: deal.id ? 'PUT' : 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  await fetchDeals(true)
}

async function deleteDeal (id) {
  await requestJSON(`${API_BASE}/deals/${id}`, { method: 'DELETE' })
  await fetchDeals(true)
}

// What a deal does, in the words a cashier would use.
function dealTerms (deal) {
  if (!deal) {return ''}
  switch (deal.kind) {
    case 'buy_get': {
      const reward = deal.percent >= 100 ? 'free' : `${deal.percent}% off`
      const otherwise = deal.fallback_percent
        ? (deal.fallback_only_without_set
          ? `; otherwise ${deal.fallback_percent}% off, but only when no unit is free`
          : `; units outside a set get ${deal.fallback_percent}% off`)
        : ''
      return `Buy ${deal.buy_qty}, get ${deal.get_qty} ${reward} (cheapest)${otherwise}`
    }
    case 'bundle_price': {
      return `${deal.buy_qty} for $${formatMoney(deal.bundle_price)}`
    }
    case 'volume_percent': {
      return `Buy ${deal.buy_qty} or more, ${deal.percent}% off all`
    }
    default: {
      return deal.kind
    }
  }
}

function dealName (id) {
  return deals.value.find(d => d.id === id)?.name ?? 'Deal'
}

export function useDeals () {
  return { deals, fetchDeals, saveDeal, deleteDeal, dealTerms, dealName }
}
