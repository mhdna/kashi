// The Receipt Designer's paper: the branch's receipt wording (receipt_texts)
// and how the till's preview picture is cut into strips so each part can be
// reworded or left out in place.
//
// The till draws every look in every mode and says where each part landed
// (make receipt-previews in kashi-pos, catalog in data/receiptThemes.json).
// Leaving a part out prints exactly the picture with that part's strip cut
// (TestReceiptMarksHidingCutsTheBand there), so the paper here is the real
// picture, strip by strip; only a reworded part is drawn by kashi.

// Text parts the designer can reword, top to bottom, with what each is called
// and the section it belongs to (left out with it).
export const textParts = [
  { part: 'payments_title', section: 'payments', label: 'Payment title' },
  { part: 'policy_title', section: 'policy', label: 'Exchange policy title' },
  { part: 'referral_title', section: 'referral', label: 'Referral code title' },
  { part: 'referral_note', section: 'referral', label: 'Referral line' },
  { part: 'socials_title', section: 'socials', label: 'Socials title' },
  { part: 'thanks', section: 'thanks', label: 'Thank-you line' },
]

export const sectionLabels = {
  payments: 'Payment details',
  policy: 'Exchange policy',
  referral: 'Referral code',
  socials: 'Socials',
  thanks: 'Thank-you line',
}

export const receiptModes = [
  { mode: 'sale', label: 'Sale' },
  { mode: 'referral', label: 'Sale with referral code' },
  { mode: 'client', label: 'Sale with a client' },
  { mode: 'exchange', label: 'Exchange' },
  { mode: 'return', label: 'Return' },
]

const TEXT_KEYS = textParts.map(p => p.part)

// emptyTexts is every default: nothing reworded, nothing left out.
export function emptyTexts () {
  return { payments_title: '', policy_title: '', referral_title: '', referral_note: '', socials_title: '', thanks: '', hide: [] }
}

function cleanText (value, maxLength) {
  const s = String(value ?? '').replace(/\r\n/g, '\n').trim()
  const chars = [...s]
  return chars.length > maxLength ? chars.slice(0, maxLength).join('') : s
}

// parseTexts reads a stored receipt_texts value into the designer's form.
// Anything unreadable is every default, as on the till.
export function parseTexts (raw) {
  const out = emptyTexts()
  if (!raw || !String(raw).trim()) {return out}
  let value
  try {
    value = JSON.parse(raw)
  } catch {
    return out
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {return out}
  for (const key of TEXT_KEYS) {
    if (typeof value[key] === 'string') {out[key] = value[key]}
  }
  if (Array.isArray(value.hide)) {out.hide = value.hide.filter(h => typeof h === 'string')}
  return out
}

// textsJSON is what the designer saves: each text trimmed and cut to
// maxLength characters, a blank one left out (the default), known sections
// only, once each and in print order, and '' for every default. It is the
// same clean-up kashi's API and the till do (receipt_texts_cases.json).
export function textsJSON (texts, { sections, maxLength }) {
  const out = {}
  for (const key of TEXT_KEYS) {
    const value = cleanText(texts?.[key], maxLength)
    if (value) {out[key] = value}
  }
  const hide = new Set((texts?.hide ?? []).map(h => String(h).trim()))
  const kept = sections.filter(s => hide.has(s))
  if (kept.length > 0) {out.hide = kept}
  return Object.keys(out).length > 0 ? JSON.stringify(out) : ''
}

// wording is what a text part prints: the branch's own, else the default.
export function wording (texts, part, defaults) {
  const own = String(texts?.[part] ?? '').trim()
  return own || defaults[part] || ''
}

// shown is a text part as it reads on the paper: the referral line with the
// reward put in, the referral title with the code after it.
export function shownText (texts, part, catalogTexts) {
  const text = wording(texts, part, catalogTexts.defaults)
  if (part === 'referral_note') {return text.replaceAll('{percent}', `${catalogTexts.referral_percent}%`)}
  if (part === 'referral_title') {return `${text} ${catalogTexts.referral_code}`}
  return text
}

// isReworded: the part prints something other than the picture shows.
export function isReworded (texts, part, defaults) {
  return wording(texts, part, defaults) !== wording({}, part, defaults)
}

export function isHidden (texts, section) {
  return (texts?.hide ?? []).includes(section)
}

// toggleHidden leaves a section out, or puts it back.
export function toggleHidden (texts, section) {
  const hide = isHidden(texts, section)
    ? texts.hide.filter(h => h !== section)
    : [...(texts.hide ?? []), section]
  return { ...texts, hide }
}

// paperStrips cuts one mode's picture into what the paper shows, top to
// bottom:
//   { kind: 'image', top, bottom }                  a strip of the picture
//   { kind: 'text', part, section, top, bottom, text, sizePx, bold, reworded }
//     a part that can be reworded; drawn from the picture until it is
//   { kind: 'socials', ...text, icons: { top, bottom, left, right } }
//     the socials title, which shares its row with the icons
//   { kind: 'hidden', section }                     a part left out
// Image strips together with text strips cover the picture exactly, less the
// sections left out.
export function paperStrips (modeInfo, texts, catalogTexts) {
  const parts = modeInfo?.parts ?? []
  const height = modeInfo?.height ?? 0
  const sections = parts.filter(p => p.kind === 'section')
  const textMarks = parts.filter(p => p.kind === 'text')
  const icons = parts.find(p => p.kind === 'icons')

  // Cuts: where hidden sections start and end, and where text parts are.
  const pieces = []
  for (const section of sections) {
    if (isHidden(texts, section.part)) {pieces.push({ kind: 'hidden', section: section.part, top: section.top, bottom: section.bottom })}
  }
  for (const mark of textMarks) {
    const info = textParts.find(p => p.part === mark.part)
    if (!info) {continue}
    if (isHidden(texts, info.section)) {continue}
    const strip = {
      kind: mark.part === 'socials_title' ? 'socials' : 'text',
      part: mark.part,
      section: info.section,
      top: mark.top,
      bottom: mark.bottom,
      text: shownText(texts, mark.part, catalogTexts),
      sizePx: mark.size_px,
      bold: Boolean(mark.bold),
      reworded: isReworded(texts, mark.part, catalogTexts.defaults),
    }
    if (strip.kind === 'socials' && icons) {strip.icons = { top: icons.top, bottom: icons.bottom, left: icons.left, right: icons.right }}
    pieces.push(strip)
  }
  pieces.sort((a, b) => a.top - b.top || (a.kind === 'hidden' ? -1 : 1))

  const strips = []
  let y = 0
  for (const piece of pieces) {
    if (piece.top < y) {continue} // inside a hidden section already cut
    if (piece.top > y) {strips.push({ kind: 'image', top: y, bottom: piece.top })}
    if (piece.kind === 'hidden') {
      strips.push({ kind: 'hidden', section: piece.section })
    } else {
      strips.push(piece)
    }
    y = piece.bottom
  }
  if (y < height) {strips.push({ kind: 'image', top: y, bottom: height })}
  return strips
}

// pictureRows: how many rows of the picture the paper still shows (all of
// it, less the sections left out). The till's receipt is exactly as long
// when nothing is reworded.
export function pictureRows (strips) {
  return strips.reduce((sum, s) => sum + (s.kind === 'hidden' ? 0 : s.bottom - s.top), 0)
}
