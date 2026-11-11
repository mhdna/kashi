import { describe, expect, it } from 'vitest'
import { applyStockFill, fetchAllStock, labelQty, stockQueueEntries } from './barcodeStockFill'

function row (id, quantity, extra = {}) {
  return {
  variant_id: id, quantity, barcode: `800${id}`, product_code: `C${id}`, product_name: `P${id}`,
  color_name: '', size_name: '', ...extra,
}
}

describe('labelQty', () => {
  it('is the whole pieces on hand, never below zero', () => {
    expect(labelQty(6)).toBe(6)
    expect(labelQty(2.9)).toBe(2)
    expect(labelQty(0)).toBe(0)
    expect(labelQty(-3)).toBe(0)
    expect(labelQty('x')).toBe(0)
    expect(labelQty(null)).toBe(0)
  })
})

describe('stockQueueEntries', () => {
  it('one entry per stocked variant, quantity = on hand', () => {
    expect(stockQueueEntries([row(1, 6), row(2, 15, { color_name: 'Red', size_name: 'M' })])).toEqual([
      { id: 1, name: 'P1', code: 'C1', variant: '', barcode: '8001', qty: 6 },
      { id: 2, name: 'P2', code: 'C2', variant: 'Red / M', barcode: '8002', qty: 15 },
    ])
  })
  it('skips empty, negative and barcode-less stock', () => {
    expect(stockQueueEntries([row(1, 0), row(2, -4), row(3, 5, { barcode: '' }), row(4, 1)]).map(e => e.id)).toEqual([4])
  })
  it('handles no stock at all', () => {
    expect(stockQueueEntries([])).toEqual([])
    expect(stockQueueEntries(undefined)).toEqual([])
  })
})

describe('applyStockFill', () => {
  const entries = stockQueueEntries([row(1, 6), row(2, 3)])
  it('fills an empty queue', () => {
    expect(applyStockFill([], entries)).toEqual(entries)
  })
  it('sets (not adds to) a quantity already queued, keeps other entries', () => {
    const queue = [{ id: 1, name: 'P1', code: 'C1', variant: '', barcode: '8001', qty: 40 }, { id: 9, name: 'X', qty: 2 }]
    const out = applyStockFill(queue, entries)
    expect(out.map(e => [e.id, e.qty])).toEqual([[1, 6], [9, 2], [2, 3]])
  })
  it('is the same when pressed twice, and does not touch its input', () => {
    const once = applyStockFill([], entries)
    const snapshot = JSON.stringify(once)
    expect(applyStockFill(once, entries)).toEqual(once)
    expect(JSON.stringify(once)).toBe(snapshot)
  })
})

describe('fetchAllStock', () => {
  const make = n => Array.from({ length: n }, (_, i) => row(i + 1, 1))
  it('reads every page', async () => {
    const all = make(1203)
    const offsets = []
    const got = await fetchAllStock(async off => { offsets.push(off); return { stock: all.slice(off, off + 500), total: all.length } })
    expect(got).toHaveLength(1203)
    expect(offsets).toEqual([0, 500, 1000])
  })
  it('stops after one page when it all fits, and on an empty inventory', async () => {
    expect(await fetchAllStock(async () => ({ stock: make(3), total: 3 }))).toHaveLength(3)
    expect(await fetchAllStock(async () => ({ stock: [], total: 0 }))).toEqual([])
  })
  it('stops at a full last page', async () => {
    let calls = 0
    await fetchAllStock(async () => { calls++; return { stock: make(500).slice(0, 500), total: 500 } })
    expect(calls).toBe(1)
  })
})
