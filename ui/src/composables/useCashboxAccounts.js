import { API_BASE } from '@/config'
import { createResource } from './useApiResource'

const resource = createResource({ path: '/cashbox_accounts', rootKey: 'cashbox_accounts' })

export function useCashboxAccounts () {
  return {
    cashboxAccounts: resource.items,
    fetchCashboxAccounts: resource.fetchAll,
    createCashboxAccount: resource.create,
    updateCashboxAccount: resource.update,
    deleteCashboxAccount: resource.remove,
    async setCashboxAccountActive (id, isActive) {
      const item = await resource.requestJSON(`${API_BASE}/cashbox_accounts/${encodeURIComponent(id)}/active`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: isActive }),
      })
      const idx = resource.items.value.findIndex(i => i.id === id)
      if (idx !== -1) {
        resource.items.value[idx] = item
      }
      return item
    },
  }
}
