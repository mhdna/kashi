// The barcode print page's request to POST /barcodes/print, and the one print
// option this browser remembers: some label printers feed the first page of
// every job without printing on it, so the PDF can start with an empty page.

export const BLANK_FIRST_PAGE_KEY = 'kashi.barcodes.blankFirstPage'

// Whether this browser's printer needs the empty first page. Storage can be
// missing or throw (private window, blocked site data); then it's off.
export function loadBlankFirstPage (storage = globalThis.localStorage) {
  try {
    return storage?.getItem(BLANK_FIRST_PAGE_KEY) === '1'
  } catch {
    return false
  }
}

export function saveBlankFirstPage (on, storage = globalThis.localStorage) {
  try {
    if (on) {
      storage?.setItem(BLANK_FIRST_PAGE_KEY, '1')
    } else {
      storage?.removeItem(BLANK_FIRST_PAGE_KEY)
    }
  } catch {
    // Not remembered; this print still uses the choice.
  }
}

// The JSON body of one print. A label size that isn't a number is 0, which
// means the saved size.
export function printRequestBody ({ items, separatorEnabled, separatorFields, labelWidth, labelHeight, blankFirstPage }) {
  return {
    items,
    separator: { enabled: !!separatorEnabled, fields: separatorFields || [] },
    label: { width: Number(labelWidth) || 0, height: Number(labelHeight) || 0 },
    blank_first_page: !!blankFirstPage,
  }
}
