import { afterEach, describe, expect, it, vi } from 'vitest'
import { BLANK_FIRST_PAGE_KEY, loadBlankFirstPage, printRequestBody, saveBlankFirstPage } from './barcodePrint'

function memoryStorage () {
  const data = new Map()
  return {
    getItem: k => (data.has(k) ? data.get(k) : null),
    setItem: (k, v) => data.set(k, String(v)),
    removeItem: k => data.delete(k),
    data,
  }
}

const throwing = {
  getItem () { throw new Error('blocked') },
  setItem () { throw new Error('blocked') },
  removeItem () { throw new Error('blocked') },
}

describe('blank first page preference', () => {
  it('is off until this browser turns it on, and stays as last saved', () => {
    const s = memoryStorage()
    expect(loadBlankFirstPage(s)).toBe(false)
    saveBlankFirstPage(true, s)
    expect(s.data.get(BLANK_FIRST_PAGE_KEY)).toBe('1')
    expect(loadBlankFirstPage(s)).toBe(true)
    saveBlankFirstPage(false, s)
    expect(s.data.has(BLANK_FIRST_PAGE_KEY)).toBe(false)
    expect(loadBlankFirstPage(s)).toBe(false)
  })

  it('reads only "1" as on', () => {
    for (const v of ['0', 'true', '', 'yes', ' 1']) {
      const s = memoryStorage()
      s.setItem(BLANK_FIRST_PAGE_KEY, v)
      expect(loadBlankFirstPage(s), v).toBe(false)
    }
  })

  it('is off and never throws when storage is missing or blocked', () => {
    expect(loadBlankFirstPage(throwing)).toBe(false)
    expect(loadBlankFirstPage(null)).toBe(false)
    expect(() => saveBlankFirstPage(true, throwing)).not.toThrow()
    expect(() => saveBlankFirstPage(false, null)).not.toThrow()
  })

  it('uses the browser localStorage by default', () => {
    const s = memoryStorage()
    vi.stubGlobal('localStorage', s)
    try {
      expect(loadBlankFirstPage()).toBe(false)
      saveBlankFirstPage(true)
      expect(s.getItem(BLANK_FIRST_PAGE_KEY)).toBe('1')
      expect(loadBlankFirstPage()).toBe(true)
      saveBlankFirstPage(false)
      expect(s.getItem(BLANK_FIRST_PAGE_KEY)).toBe(null)
    } finally {
      vi.unstubAllGlobals()
    }
  })
})

describe('printRequestBody', () => {
  const items = [{ variant_id: 7, qty: 3 }, { variant_id: 9, qty: 1 }]

  it('sends the blank page only when it is on', () => {
    const base = { items, separatorEnabled: false, separatorFields: ['name', 'qty'], labelWidth: 0, labelHeight: 0 }
    expect(printRequestBody({ ...base, blankFirstPage: true }).blank_first_page).toBe(true)
    expect(printRequestBody({ ...base, blankFirstPage: false }).blank_first_page).toBe(false)
    expect(printRequestBody(base).blank_first_page).toBe(false)
  })

  it('keeps the items, separator and label size exactly as the page had them', () => {
    expect(printRequestBody({
      items, separatorEnabled: true, separatorFields: ['name'], labelWidth: '216', labelHeight: 108, blankFirstPage: true,
    })).toEqual({
      items,
      separator: { enabled: true, fields: ['name'] },
      label: { width: 216, height: 108 },
      blank_first_page: true,
    })
  })

  it('a label size that is not a number means the saved size (0)', () => {
    for (const v of ['', null, undefined, 'abc', Number.NaN]) {
      const body = printRequestBody({ items, labelWidth: v, labelHeight: v })
      expect(body.label, String(v)).toEqual({ width: 0, height: 0 })
    }
  })
})

describe('useBarcodes().printLabels', () => {
  afterEach(() => {
    vi.resetModules()
    vi.doUnmock('@/composables/useApi')
  })

  it('posts the body as given, blank_first_page included', async () => {
    const authFetch = vi.fn(async () => ({ ok: true, blob: async () => 'PDF' }))
    vi.doMock('@/composables/useApi', () => ({ authFetch }))
    const { useBarcodes } = await import('@/composables/useBarcodes')
    const body = printRequestBody({ items: [{ variant_id: 7, qty: 2 }], blankFirstPage: true })
    await expect(useBarcodes().printLabels(body)).resolves.toBe('PDF')
    expect(authFetch).toHaveBeenCalledTimes(1)
    const [url, opts] = authFetch.mock.calls[0]
    expect(url).toMatch(/\/barcodes\/print$/)
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body)).toEqual(body)
    expect(JSON.parse(opts.body).blank_first_page).toBe(true)
  })
})
