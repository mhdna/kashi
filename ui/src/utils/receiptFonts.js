// The till's receipt fonts in the browser: each file is fetched once from
// kashi (/receipt_designer/font) and added to the page under its own family
// name, so a line is shown, and typed over, in the letters the till prints.

const loading = new Map()

// receiptFontFamily is the page's family name for a font and weight.
export function receiptFontFamily (key, bold = false) {
  return `kashi-receipt-${key}${bold ? '-bold' : ''}`
}

export function receiptFontURL (base, key, bold = false) {
  const q = new URLSearchParams({ key })
  if (bold) {q.set('bold', '1')}
  return `${base}/receipt_designer/font?${q}`
}

// loadReceiptFont adds a font to the page (once) and answers with its family
// name; fetchFile answers with the file's bytes. A font that fails to load
// answers '' and is tried again next time.
export function loadReceiptFont (key, bold, fetchFile) {
  if (!key) {return Promise.resolve('')}
  const family = receiptFontFamily(key, bold)
  if (loading.has(family)) {return loading.get(family)}
  const p = (async () => {
    const data = await fetchFile(key, bold)
    if (typeof FontFace !== 'undefined' && typeof document !== 'undefined' && document.fonts) {
      const face = new FontFace(family, data)
      await face.load()
      document.fonts.add(face)
    }
    return family
  })().catch(() => {
    loading.delete(family)
    return ''
  })
  loading.set(family, p)
  return p
}

// fontStyle is the CSS for words in a receipt font, with a fallback while it
// loads.
export function fontStyle (key, bold = false) {
  if (!key) {return {}}
  return { fontFamily: `'${receiptFontFamily(key, bold)}', monospace`, fontWeight: 'normal' }
}

const kindTitles = { sans: 'Sans', serif: 'Serif', mono: 'Typewriter and mono', display: 'Display', script: 'Script', arabic: 'Arabic' }

// fontItems is the catalog's fonts for a list, each with its kind.
export function fontItems (catalog) {
  return (catalog?.fonts ?? []).map(f => ({ value: f.key, title: f.label, kind: kindTitles[f.kind] ?? f.kind }))
}

// Only for tests: forget what was loaded.
export function forgetReceiptFonts () {
  loading.clear()
}
