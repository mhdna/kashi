import { API_BASE } from '@/config'
import { createResource } from './useApiResource'

const resource = createResource({ path: '/clients', rootKey: 'clients' })

export function useClients () {
  return {
    clients: resource.items,
    fetchClients: resource.fetchAll,
    createClient: resource.create,
    updateClient: resource.update,
    // Resolves to "deleted" or "deactivated": a client with history is
    // deactivated instead of deleted (see deleteClient in api/client.go).
    deleteClient: async id => {
      const res = await resource.requestJSON(`${API_BASE}/clients/${encodeURIComponent(id)}`, { method: 'DELETE' })
      if (res?.status !== 'deactivated') {
        resource.items.value = resource.items.value.filter(c => c.id !== id)
      }
      return res?.status ?? 'deleted'
    },
    // Brings a deactivated client back.
    setClientActive: (id, isActive) =>
      resource.requestJSON(`${API_BASE}/clients/${encodeURIComponent(id)}/active`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: isActive }),
      }),
  }
}
