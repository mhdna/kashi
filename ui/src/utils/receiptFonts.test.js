import { afterEach, describe, expect, it, vi } from 'vitest'
import { fontItems, fontStyle, forgetReceiptFonts, loadReceiptFont, receiptFontFamily, receiptFontURL } from './receiptFonts'
import catalog from './testdata/receipt_designer_catalog.json'

afterEach(() => forgetReceiptFonts())

describe('receipt fonts', () => {
  it('every font the till prints is offered, grouped by kind', () => {
    const items = fontItems(catalog)
    expect(items.length).toBeGreaterThanOrEqual(27)
    expect(items.map(i => i.value)).toEqual(expect.arrayContaining(['pacifico', 'inter', 'bebas-neue', 'go-mono', 'belleza']))
    expect(new Set(items.map(i => i.kind))).toEqual(new Set(['Sans', 'Serif', 'Typewriter and mono', 'Display', 'Script', 'Arabic']))
    for (const i of items) {expect(i.title, i.value).not.toMatch(/[–—·•]/)}
  })

  it('each font file is fetched once per weight; a failure is tried again', async () => {
    const fetchFile = vi.fn(async () => new ArrayBuffer(8))
    expect(await loadReceiptFont('inter', false, fetchFile)).toBe('kashi-receipt-inter')
    await loadReceiptFont('inter', false, fetchFile)
    expect(await loadReceiptFont('inter', true, fetchFile)).toBe('kashi-receipt-inter-bold')
    expect(fetchFile.mock.calls).toEqual([['inter', false], ['inter', true]])
    const failing = vi.fn(async () => { throw new Error('offline') })
    expect(await loadReceiptFont('lobster', false, failing)).toBe('')
    expect(await loadReceiptFont('lobster', false, fetchFile)).toBe('kashi-receipt-lobster')
    expect(await loadReceiptFont('', false, fetchFile)).toBe('')
  })

  it('words are shown in the font\'s family, its bold cut a family of its own', () => {
    expect(fontStyle('pacifico', true)).toEqual({ fontFamily: `'${receiptFontFamily('pacifico', true)}', monospace`, fontWeight: 'normal' })
    expect(fontStyle('')).toEqual({})
    expect(receiptFontURL('/api', 'inter', true)).toBe('/api/receipt_designer/font?key=inter&bold=1')
  })
})
