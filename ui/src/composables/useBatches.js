import { API_BASE } from '@/config'
import { createResource } from './useApiResource'

const resource = createResource({ path: '/batches', listPath: '/batches', rootKey: 'batches' })

async function createBatch ({ code = '', purchaseId, note = '' }) {
  return resource.requestJSON(`${API_BASE}/batches`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code, purchase_id: purchaseId, note }),
  })
}

async function fetchBatch (id) {
  return resource.requestJSON(`${API_BASE}/batches/${id}`)
}

async function deleteBatch (id) {
  return resource.requestJSON(`${API_BASE}/batches/${id}`, { method: 'DELETE' })
}

// searchBatches lists batches (id/code/purchase/supplier), used for the
// left-side picker list -- unlike fetchBatches/resource.items this isn't
// cached, since it's driven by a live search box.
async function searchBatches (query = '') {
  const params = new URLSearchParams({ search: query || '', page_size: '100' })
  return resource.requestJSON(`${API_BASE}/batches?${params}`)
}

// searchPurchases backs the "point this batch at a purchase invoice" picker
// in the add-batch dialog.
async function searchPurchases (query = '') {
  const params = new URLSearchParams({ search: query || '', page_size: '20' })
  return resource.requestJSON(`${API_BASE}/purchases?${params}`)
}

export function useBatches () {
  return {
    batches: resource.items,
    fetchBatches: resource.fetchAll,
    searchBatches,
    searchPurchases,
    createBatch,
    fetchBatch,
    deleteBatch,
  }
}
