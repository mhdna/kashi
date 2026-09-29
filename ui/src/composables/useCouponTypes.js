import { API_BASE } from '@/config'
import { createResource } from './useApiResource'

const resource = createResource({
  path: '/coupon_types',
  listPath: '/coupon_types',
  rootKey: 'coupon_types',
})

async function updateCouponType (id, body) {
  const item = await resource.requestJSON(`${API_BASE}/coupon_types/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const idx = resource.items.value.findIndex(t => t.id === id)
  if (idx !== -1) {resource.items.value[idx] = item ?? { ...resource.items.value[idx], ...body }}
  return item
}

async function deleteCouponType (id) {
  await resource.requestJSON(`${API_BASE}/coupon_types/${id}`, { method: 'DELETE' })
  resource.items.value = resource.items.value.filter(t => t.id !== id)
}

export function useCouponTypes () {
  return {
    couponTypes: resource.items,
    fetchCouponTypes: resource.fetchAll,
    createCouponType: resource.create,
    updateCouponType,
    deleteCouponType,
  }
}
