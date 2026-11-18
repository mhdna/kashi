// The Receipt Designer's design: how each line of the printed receipt is
// changed (receipt_texts JSON). The till's own code draws and cleans it (kashi
// runs a copy, /receipt_designer/preview); this module only edits it.
//
// The catalog (/receipt_designer/catalog) lists the parts of the receipt
// (groups), each line in them (elements) with what it can change (`can`, a
// bit set: text, case, size, bold, align, hide) and the date formats.

export const caseOptions = [
  { value: '', title: 'As written' },
  { value: 'upper', title: 'ALL CAPITALS' },
  { value: 'title', title: 'First Letters' },
  { value: 'lower', title: 'all small' },
]

export const alignOptions = [
  { value: '', title: 'As the look has it' },
  { value: 'left', title: 'Left' },
  { value: 'center', title: 'Centre' },
  { value: 'right', title: 'Right' },
]

// emptyDesign is every default: nothing changed, nothing left out.
export function emptyDesign () {
  return { elements: {}, hide: [], date_format: '' }
}

// parseDesign reads a stored receipt_texts value. The older per-part fields
// become the elements they now are, so a design saved before keeps its words.
export function parseDesign (raw) {
  const out = emptyDesign()
  if (!raw || !String(raw).trim()) {return out}
  let value
  try {
    value = JSON.parse(raw)
  } catch {
    return out
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {return out}
  if (value.elements && typeof value.elements === 'object' && !Array.isArray(value.elements)) {
    for (const [key, e] of Object.entries(value.elements)) {
      if (e && typeof e === 'object') {out.elements[key] = { ...e }}
    }
  }
  const legacy = ['payments_title', 'policy_title', 'referral_title', 'referral_note', 'socials_title', 'thanks']
  for (const key of legacy) {
    if (typeof value[key] === 'string' && value[key].trim() && !out.elements[key]?.text) {
      out.elements[key] = { ...out.elements[key], text: value[key] }
    }
  }
  if (Array.isArray(value.hide)) {out.hide = value.hide.filter(h => typeof h === 'string')}
  if (typeof value.date_format === 'string') {out.date_format = value.date_format}
  return out
}

function isEmptyElement (e) {
  return !e || Object.entries(e).every(([k, v]) => {
    if (k === 'text' || k === 'case' || k === 'align') {return !String(v ?? '').trim()}
    if (k === 'size') {return !Number(v)}
    if (k === 'hide') {return !v}
    if (k === 'bold') {return v === undefined || v === null}
    return true
  })
}

// designJSON is the design to send: elements with something set, only
// sections left out, '' for every default. kashi and the till clean it the
// rest of the way (the preview answers with what will be kept).
export function designJSON (design) {
  const out = {}
  const elements = {}
  for (const [key, e] of Object.entries(design?.elements ?? {})) {
    if (isEmptyElement(e)) {continue}
    const clean = {}
    if (String(e.text ?? '').trim()) {clean.text = e.text}
    if (e.case) {clean.case = e.case}
    if (Number(e.size)) {clean.size = Number(e.size)}
    if (e.bold === true || e.bold === false) {clean.bold = e.bold}
    if (e.align) {clean.align = e.align}
    if (e.hide) {clean.hide = true}
    elements[key] = clean
  }
  if (Object.keys(elements).length > 0) {out.elements = elements}
  const hide = [...new Set(design?.hide)]
  if (hide.length > 0) {out.hide = hide}
  if (design?.date_format) {out.date_format = design.date_format}
  return Object.keys(out).length > 0 ? JSON.stringify(out) : ''
}

// can says whether an element can change something ('text', 'case', ...).
export function can (catalog, info, what) {
  return Boolean((info?.can ?? 0) & (catalog?.can?.[what] ?? 0))
}

export function elementsOf (catalog, group) {
  return (catalog?.elements ?? []).filter(e => e.group === group)
}

export function groupInfo (catalog, group) {
  return (catalog?.groups ?? []).find(g => g.key === group) ?? null
}

// setElement changes one setting of one line, keeping the design's others.
export function setElement (design, key, patch) {
  const next = { ...design.elements[key], ...patch }
  const elements = { ...design.elements }
  if (isEmptyElement(next)) {delete elements[key]} else {elements[key] = next}
  return { ...design, elements }
}

// resetGroup puts every line of a part back to its default.
export function resetGroup (design, catalog, group) {
  const elements = { ...design.elements }
  for (const e of elementsOf(catalog, group)) {delete elements[e.key]}
  const section = groupInfo(catalog, group)?.section
  const hide = section ? design.hide.filter(h => h !== section) : design.hide
  return { ...design, elements, hide, date_format: group === 'meta' ? '' : design.date_format }
}

export function isGroupChanged (design, catalog, group) {
  const section = groupInfo(catalog, group)?.section
  return elementsOf(catalog, group).some(e => !isEmptyElement(design.elements[e.key]))
    || Boolean(section && design.hide.includes(section))
    || (group === 'meta' && Boolean(design.date_format))
}

export function isSectionHidden (design, section) {
  return Boolean(section) && design.hide.includes(section)
}

export function toggleSection (design, section) {
  const hide = design.hide.includes(section) ? design.hide.filter(h => h !== section) : [...design.hide, section]
  return { ...design, hide }
}

// sizeStep moves a line's size by step points, within the catalog's range.
export function sizeStep (design, catalog, key, step) {
  const now = Number(design.elements[key]?.size ?? 0)
  const next = Math.min(catalog.size_max, Math.max(catalog.size_min, now + step))
  return setElement(design, key, { size: next })
}

// boldState is the bold toggle's value: 'look' (as the look has it), 'bold'
// or 'regular'; setBold takes the same.
export function boldState (design, key) {
  const b = design.elements[key]?.bold
  if (b === true) {return 'bold'}
  if (b === false) {return 'regular'}
  return 'look'
}

export function setBold (design, key, state) {
  return setElement(design, key, { bold: state === 'bold' ? true : (state === 'regular' ? false : undefined) })
}

// previewQuery is the preview request for a look, a mode, a branch and the
// design.
export function previewQuery ({ theme, layout, mode, branchId, design }) {
  const q = new URLSearchParams()
  if (theme) {q.set('theme', theme)}
  if (layout) {q.set('layout', layout)}
  if (mode) {q.set('mode', mode)}
  if (branchId) {q.set('branch_id', String(branchId))}
  const texts = designJSON(design)
  if (texts) {q.set('texts', texts)}
  return q.toString()
}

// markAt is the part under a point of the paper (in the picture's pixels).
export function markAt (marks, y) {
  let best = null
  for (const m of marks ?? []) {
    if (y >= m.top && y < m.bottom && (!best || m.bottom - m.top < best.bottom - best.top)) {best = m}
  }
  return best
}

// receiptPreviewURL is where a look's picture is served (the theme list's
// thumbnails), with its version so a new picture is never answered from the
// browser's copy of the old one.
export function receiptPreviewURL (modeInfo, base = '/receipt-previews/') {
  const url = `${base}${modeInfo?.file ?? ''}`
  return modeInfo?.version ? `${url}?v=${modeInfo.version}` : url
}
