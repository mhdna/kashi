import { API_BASE } from '@/config'
import { createResource } from './useApiResource'

const resource = createResource({
  path: '/coupon_lists',
  listPath: '/coupon_lists',
  rootKey: 'coupon_lists',
})

// createCouponList's response ({list, codes, requested, generated}) isn't a
// bare list row -- resource.create would push the whole envelope onto
// items, so this bypasses it and appends just the `list` field, same shape
// every other row already has.
async function createCouponList (body) {
  const result = await resource.requestJSON(`${API_BASE}/coupon_lists`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  resource.items.value.unshift({
    ...result.list,
    coupon_type_name: result.coupon_type_name,
    coupon_count: result.generated,
    redeemed_count: 0,
  })
  return result
}

async function deactivateCouponList (id) {
  await resource.requestJSON(`${API_BASE}/coupon_lists/${id}/deactivate`, { method: 'POST' })
  const idx = resource.items.value.findIndex(l => l.id === id)
  if (idx !== -1) {resource.items.value[idx] = { ...resource.items.value[idx], is_active: false }}
}

export function useCouponLists () {
  return {
    couponLists: resource.items,
    fetchCouponLists: resource.fetchAll,
    createCouponList,
    deactivateCouponList,
  }
}
