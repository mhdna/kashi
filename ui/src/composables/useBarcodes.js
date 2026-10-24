import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  if (res.status === 204) {
    return null
  }
  return res.json().catch(() => null)
}

export function useBarcodes () {
  // Assigns fresh unique EAN-13 barcodes to selected variants that don't
  // already have a valid one. Returns { updated, skipped }.
  async function assignBarcodes (variantIds) {
    return requestJSON(`${API_BASE}/barcodes/assign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ variant_ids: variantIds }),
    })
  }

  // Returns a Blob of the generated label PDF.
  async function printLabels ({ items, separator, label }) {
    const res = await authFetch(`${API_BASE}/barcodes/print`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ items, separator, label }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Request failed with status ${res.status}`)
    }
    return res.blob()
  }

  // The saved label design, the themes (each with a preview), and the label size.
  async function getDesign () {
    return requestJSON(`${API_BASE}/barcodes/design`)
  }

  async function saveDesign ({ theme, design }) {
    return requestJSON(`${API_BASE}/barcodes/design`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ theme, design }),
    })
  }

  // The label drawn as SVG by the same code that makes the PDF.
  async function previewDesign ({ design, sample = 0, borders = false }) {
    return requestJSON(`${API_BASE}/barcodes/design/preview`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ design, sample, borders }),
    })
  }

  return { assignBarcodes, printLabels, getDesign, saveDesign, previewDesign }
}
