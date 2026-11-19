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

// Picked by toggles: none picked is the look's own (each line's Default
// button gets back there).
export const alignOptions = [
  { value: 'left', title: 'Left' },
  { value: 'center', title: 'Centre' },
  { value: 'right', title: 'Right' },
]

export const lineStyleOptions = [
  { value: 'dotted', title: 'Dotted' },
  { value: 'dashed', title: 'Dashed' },
  { value: 'solid', title: 'Solid' },
  { value: 'double', title: 'Double' },
]

// The social platforms the till has icons and handles for.
export const socialPlatforms = [
  { value: 'instagram', title: 'Instagram' },
  { value: 'facebook', title: 'Facebook' },
  { value: 'tiktok', title: 'TikTok' },
  { value: 'threads', title: 'Threads' },
  { value: 'youtube', title: 'YouTube' },
]

// emptyDesign is every default: nothing changed, nothing left out.
export function emptyDesign () {
  return { elements: {}, hide: [], date_format: '', groups: {}, decor: {}, order: [], banners: [] }
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
  for (const field of ['groups', 'decor']) {
    if (value[field] && typeof value[field] === 'object' && !Array.isArray(value[field])) {
      for (const [key, v] of Object.entries(value[field])) {
        if (v && typeof v === 'object') {out[field][key] = { ...v }}
      }
    }
  }
  if (Array.isArray(value.order)) {out.order = value.order.filter(o => typeof o === 'string')}
  if (Array.isArray(value.banners)) {out.banners = value.banners.filter(b => b && typeof b === 'object').map(b => ({ ...b }))}
  return out
}

function isEmptyDecor (d) {
  return !d || (!d.above && !d.below && !d.box && !d.rounded && !Number(d.thickness))
}

function isEmptyElement (e) {
  return !e || Object.entries(e).every(([k, v]) => {
    if (k === 'text' || k === 'case' || k === 'align' || k === 'font' || k === 'lang' || k === 'text_ar') {return !String(v ?? '').trim()}
    if (k === 'size') {return !Number(v)}
    if (k === 'hide' || k === 'show') {return !v}
    if (k === 'bold') {return v === undefined || v === null}
    return true
  })
}

function cleanElement (e) {
  const clean = {}
  if (String(e.text ?? '').trim()) {clean.text = e.text}
  if (e.case) {clean.case = e.case}
  if (Number(e.size)) {clean.size = Number(e.size)}
  if (e.bold === true || e.bold === false) {clean.bold = e.bold}
  if (e.align) {clean.align = e.align}
  if (e.font) {clean.font = e.font}
  if (e.lang) {clean.lang = e.lang}
  if (String(e.text_ar ?? '').trim()) {clean.text_ar = e.text_ar}
  if (e.hide) {clean.hide = true}
  if (e.show) {clean.show = true}
  return clean
}

function cleanDecor (d) {
  const clean = {}
  for (const f of ['above', 'below', 'box']) {if (d[f]) {clean[f] = d[f]}}
  if (d.rounded && d.box) {clean.rounded = true}
  if (Number(d.thickness)) {clean.thickness = Number(d.thickness)}
  return clean
}

// cleanMap keeps the entries of a map with something set, each cleaned.
function cleanMap (map, isEmpty, clean) {
  const out = {}
  for (const [key, v] of Object.entries(map ?? {})) {
    if (!isEmpty(v)) {out[key] = clean(v)}
  }
  return Object.keys(out).length > 0 ? out : null
}

// designJSON is the design to send: lines and parts with something set, only
// sections left out, '' for every default. kashi and the till clean it the
// rest of the way (the preview answers with what will be kept).
export function designJSON (design) {
  const out = {}
  const elements = cleanMap(design?.elements, isEmptyElement, cleanElement)
  if (elements) {out.elements = elements}
  const hide = [...new Set(design?.hide)]
  if (hide.length > 0) {out.hide = hide}
  if (design?.date_format) {out.date_format = design.date_format}
  // A part takes capitals, size, weight, alignment and font, not words or hiding.
  const groups = cleanMap(design?.groups, isEmptyElement, g => cleanElement({ ...g, text: '', hide: false }))
  if (groups) {out.groups = groups}
  const decor = cleanMap(design?.decor, isEmptyDecor, cleanDecor)
  if (decor) {out.decor = decor}
  if ((design?.order ?? []).length > 0) {out.order = [...design.order]}
  const banners = (design?.banners ?? []).filter(b => b.style && String(b.title ?? '').trim()).map(b => cleanBanner(b))
  if (banners.length > 0) {out.banners = banners}
  return Object.keys(out).length > 0 ? JSON.stringify(out) : ''
}

// can says whether an element can change something ('text', 'case', ...).
export function can (catalog, info, what) {
  return Boolean((info?.can ?? 0) & (catalog?.can?.[what] ?? 0))
}

export function elementsOf (catalog, group) {
  return (catalog?.elements ?? []).filter(e => e.group === group)
}

// partsOf is a part's switches: elements that can only be printed or left
// out, like what each item row shows (name, colour and size, code, before and
// after price). One that can 'show' is off until switched on.
export function partsOf (catalog, group) {
  const onOff = (catalog?.can?.hide ?? 0) | (catalog?.can?.show ?? 0)
  return elementsOf(catalog, group).filter(e => e.can && (e.can & ~onOff) === 0)
}

export function isPartPrinted (design, catalog, info) {
  const e = design?.elements?.[info.key] ?? {}
  return can(catalog, info, 'show') ? Boolean(e.show) : !e.hide
}

// setPartPrinted switches one of them, keeping the design's others.
export function setPartPrinted (design, catalog, info, printed) {
  return can(catalog, info, 'show')
    ? setElement(design, info.key, { show: printed, hide: undefined })
    : setElement(design, info.key, { hide: !printed, show: undefined })
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

// resetElement puts one line back to the look's own.
export function resetElement (design, key) {
  const elements = { ...design.elements }
  delete elements[key]
  return { ...design, elements }
}

export function isElementChanged (design, key) {
  return !isEmptyElement(design.elements[key])
}

// resetGroup puts a whole part back: its lines, its own settings, its lines
// and box, and printing it.
export function resetGroup (design, catalog, group) {
  const elements = { ...design.elements }
  for (const e of elementsOf(catalog, group)) {delete elements[e.key]}
  const groups = { ...design.groups }
  delete groups[group]
  const decor = { ...design.decor }
  delete decor[group]
  const section = groupInfo(catalog, group)?.section
  const hide = section ? design.hide.filter(h => h !== section) : design.hide
  return { ...design, elements, groups, decor, hide, date_format: group === 'meta' ? '' : design.date_format }
}

export function isGroupChanged (design, catalog, group) {
  const section = groupInfo(catalog, group)?.section
  return elementsOf(catalog, group).some(e => !isEmptyElement(design.elements[e.key]))
    || Boolean(section && design.hide.includes(section))
    || (group === 'meta' && Boolean(design.date_format))
    || !isEmptyElement(design.groups?.[group])
    || !isEmptyDecor(design.decor?.[group])
}

// setGroup changes a setting for every line of a part at once.
export function setGroup (design, group, patch) {
  const next = { ...design.groups?.[group], ...patch }
  const groups = { ...design.groups }
  if (isEmptyElement(next)) {delete groups[group]} else {groups[group] = next}
  return { ...design, groups }
}

export function groupSizeStep (design, catalog, group, step) {
  const now = Number(design.groups?.[group]?.size ?? 0)
  return setGroup(design, group, { size: Math.min(catalog.size_max, Math.max(catalog.size_min, now + step)) })
}

// setDecor changes a part's lines and box.
export function setDecor (design, group, patch) {
  const next = { ...design.decor?.[group], ...patch }
  const decor = { ...design.decor }
  if (isEmptyDecor(next)) {delete decor[group]} else {decor[group] = next}
  return { ...design, decor }
}

// The parts under the totals that can move, in print order.
export function footerOrder (design, catalog) {
  const all = catalog?.footer_order ?? ['payments', 'policy', 'referral', 'socials', 'thanks']
  const named = (design.order ?? []).filter(k => all.includes(k))
  return [...new Set([...named, ...all])]
}

// moveSection moves a part under the totals up (-1) or down (1); a part that
// can't move further, or one that doesn't move, leaves the design as it was.
export function moveSection (design, catalog, section, step) {
  const order = footerOrder(design, catalog)
  const i = order.indexOf(section)
  const j = i + step
  if (i < 0 || j < 0 || j >= order.length) {return design}
  ;[order[i], order[j]] = [order[j], order[i]]
  const all = catalog?.footer_order ?? []
  return { ...design, order: order.join(',') === all.join(',') ? [] : order }
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

// boldState is the bold toggle's value: 'bold', 'regular', or undefined for
// the look's own; setBold takes the same.
export function boldState (design, key) {
  const b = design.elements[key]?.bold
  if (b === true) {return 'bold'}
  if (b === false) {return 'regular'}
  return undefined
}

export function setBold (design, key, state) {
  return setElement(design, key, { bold: state === 'bold' ? true : (state === 'regular' ? false : undefined) })
}

export function groupBoldState (design, group) {
  const b = design.groups?.[group]?.bold
  if (b === true) {return 'bold'}
  if (b === false) {return 'regular'}
  return undefined
}

export function setGroupBold (design, group, state) {
  return setGroup(design, group, { bold: state === 'bold' ? true : (state === 'regular' ? false : undefined) })
}

// emptyShop is a branch's details as the Shop details tab edits them.
export function emptyShop () {
  return { store_name: '', phone: '', website: '', return_policy: '', social_platforms: [], social_handles: {} }
}

// shopFromSettings reads a branch's saved settings into the tab.
export function shopFromSettings (s) {
  const shop = emptyShop()
  if (!s) {return shop}
  shop.store_name = s.market_name ?? ''
  shop.phone = s.branch_phone || s.market_phone || ''
  shop.website = s.website ?? ''
  shop.return_policy = s.return_policy ?? ''
  const platforms = Array.isArray(s.social_platforms) ? s.social_platforms : []
  shop.social_platforms = platforms.filter(p => socialPlatforms.some(o => o.value === p))
  shop.social_handles = s.social_handles && typeof s.social_handles === 'object' ? { ...s.social_handles } : {}
  return shop
}

// shopPayload is what the tab saves to a branch (update_settings). The
// phone is each branch's own, so saving for every branch leaves it out.
export function shopPayload (shop, { allBranches = false } = {}) {
  const handles = {}
  for (const p of shop.social_platforms) {
    const h = String(shop.social_handles?.[p] ?? '').trim()
    if (h) {handles[p] = h}
  }
  const out = {
    market_name: shop.store_name.trim(),
    website: shop.website.trim(),
    return_policy: shop.return_policy.trim(),
    social_platforms: shop.social_platforms.filter(p => handles[p]),
    social_handles: handles,
  }
  if (!allBranches) {out.branch_phone = shop.phone.trim()}
  return out
}

// shopPreview is the shop as the preview takes it, so unsaved details show.
export function shopPreview (shop) {
  const p = shopPayload(shop)
  return JSON.stringify({
    store_name: p.market_name, phone: p.branch_phone, website: p.website, return_policy: p.return_policy,
    social_platforms: p.social_platforms, social_handles: p.social_handles,
  })
}

// previewQuery is the preview request for a look, a mode, a branch and the
// design.
export function previewQuery ({ theme, layout, mode, branchId, design, shop, deal }) {
  const q = new URLSearchParams()
  if (theme) {q.set('theme', theme)}
  if (layout) {q.set('layout', layout)}
  if (mode) {q.set('mode', mode)}
  if (branchId) {q.set('branch_id', String(branchId))}
  const texts = designJSON(design)
  if (texts) {q.set('texts', texts)}
  if (shop) {q.set('shop', shopPreview(shop))}
  if (deal) {q.set('deal', JSON.stringify({ name: deal.name, kind: deal.kind, percent: deal.percent ?? 0, amount: deal.amount ?? 0 }))}
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

// dealModes are the designer's modes for kashi's deals (invoice discounts):
// one for each in use, shown as a sale to a client with the deal applied,
// so a deal added later gets its own view too.
export function dealModes (rules) {
  // One view per deal name: two branches' "10% above $50" look the same.
  const seen = new Set()
  return (rules ?? [])
    .filter(r => r && r.is_active !== false && !r.deleted_at?.Valid)
    .filter(r => !seen.has(r.name) && seen.add(r.name))
    .map(r => ({ mode: `deal:${r.id}`, label: r.name, deal: { name: r.name, kind: r.kind, percent: r.percent, amount: r.amount } }))
}

// Banners: display lettering the till draws (receipt_banner.go). A branch
// keeps up to 12 (one per deal, say); each receipt prints one, the best fit.
export const bannerMax = 12

function cleanBanner (b) {
  const out = { style: b.style, title: String(b.title).trim() }
  if (String(b.subtitle ?? '').trim()) {out.subtitle = String(b.subtitle).trim()}
  if (b.when) {out.when = b.when}
  if (b.place) {out.place = b.place}
  if (b.font) {out.font = b.font}
  return out
}

// addBanner puts a banner (a preset, say) at the end, while there is room.
export function addBanner (design, banner) {
  const banners = design.banners ?? []
  if (banners.length >= bannerMax) {return design}
  return { ...design, banners: [...banners, { ...banner }] }
}

export function updateBanner (design, i, patch) {
  const banners = (design.banners ?? []).map((b, j) => (j === i ? { ...b, ...patch } : b))
  return { ...design, banners }
}

export function removeBanner (design, i) {
  return { ...design, banners: (design.banners ?? []).filter((_, j) => j !== i) }
}

export function moveBanner (design, i, step) {
  const banners = [...(design.banners ?? [])]
  const j = i + step
  if (j < 0 || j >= banners.length) {return design}
  ;[banners[i], banners[j]] = [banners[j], banners[i]]
  return { ...design, banners }
}

// bannerWhenOptions are when a banner can print: always, on a buy X get Y
// sale, with any deal on the invoice, or with one of kashi's deals.
export function bannerWhenOptions (deals) {
  return [
    { value: '', title: 'Always' },
    { value: 'item_deal', title: 'On a buy X get Y sale' },
    { value: 'deal', title: 'With any deal on the invoice' },
    ...(deals ?? []).filter(d => d?.name).map(d => ({ value: `deal:${d.name}`, title: `With ${d.name}` })),
  ]
}

export const bannerPlaceOptions = [
  { value: '', title: 'Under the shop\'s name' },
  { value: 'bottom', title: 'At the end' },
]

// bannerMode is the view where a banner shows: the one its condition needs.
export function bannerMode (banner, deals) {
  const when = banner?.when ?? ''
  if (when === 'item_deal') {return 'item_deal'}
  if (when === 'deal') {return dealModes(deals)[0]?.mode ?? 'sale'}
  if (when.startsWith('deal:')) {
    const name = when.slice(5)
    return dealModes(deals).find(m => m.label === name)?.mode ?? 'item_deal'
  }
  return ''
}

// A line's language: English (''), Arabic ('ar') or both, English then
// Arabic under it.
export const langOptions = [
  { value: '', title: 'EN' },
  { value: 'ar', title: 'عربي' },
  { value: 'both', title: 'Both' },
]
