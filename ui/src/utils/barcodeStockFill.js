// "Fill from inventory" on the Barcodes page: one label per piece on the shelf.
// Takes the rows of an inventory's stock (GET /inventories/:id/stock) and makes
// the print-queue entries for them. Kept apart from the page so it can be tested.

// Whole pieces on hand; zero, negative and non-numeric stock gets no label.
export function labelQty (quantity) {
  const n = Math.floor(Number(quantity))
  return Number.isFinite(n) && n > 0 ? n : 0
}

// Queue entries (same shape useBarcodeQueue.add makes) for every stocked variant.
export function stockQueueEntries (stock) {
  const entries = []
  for (const row of stock ?? []) {
    const qty = labelQty(row.quantity)
    if (qty === 0 || !row.barcode) {continue}
    entries.push({
      id: row.variant_id,
      name: row.product_name,
      code: row.product_code,
      variant: [row.color_name, row.size_name].filter(Boolean).join(' / '),
      barcode: row.barcode,
      qty,
    })
  }
  return entries
}

// The queue after filling: a variant already queued takes the inventory's
// quantity (it is set, not added, so pressing the button twice changes nothing);
// the rest of the queue is left as it was.
export function applyStockFill (queue, entries) {
  const byId = new Map(entries.map(e => [e.id, e]))
  const merged = queue.map(q => (byId.has(q.id) ? { ...q, qty: byId.get(q.id).qty } : q))
  const queued = new Set(queue.map(q => q.id))
  for (const e of entries) {
    if (!queued.has(e.id)) {merged.push(e)}
  }
  return merged
}

// Every page of an inventory's stock. fetchPage(pageId) -> { stock, total }.
export async function fetchAllStock (fetchPage, pageSize = 500) {
  const all = []
  for (let offset = 0; ; offset += pageSize) {
    const { stock, total } = await fetchPage(offset)
    all.push(...stock)
    if (stock.length < pageSize || all.length >= total) {break}
  }
  return all
}
