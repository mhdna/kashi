// The Live cart dialog's line logic (components/LiveCartButton.vue): which
// lines can be changed, what is typed for each, and the override sent to the
// till (api/live_cart.go, kashi-pos live_cart.go).
//
// A product's line goes by its barcode. Every unknown item shares one
// barcode, so one goes by its line ref (the till's unknownRef, reported as
// lineRef), and also takes a deal choice. A till too old to report lineRef
// can't have its unknown items changed.

// The key a line's draft is kept under.
export function lineKey (line) {
  return line.unknown ? `ref:${line.lineRef}` : line.barcode
}

// Whether kashi can change this line.
export function canSetLine (line) {
  if (!line.barcode || !(line.qty > 0)) {return false}
  return !line.unknown || Boolean(line.lineRef)
}

// What the dialog says instead of fields on a line it can't change.
export function lockedLineText (line) {
  if (line.unknown && !line.lineRef && line.qty > 0) {return 'Update this till to set unknown items'}
  return '—'
}

// The deal choices for an unknown item, named for the deal it would join
// ("B1G1"); '' keeps the salesperson's.
export function unknownDealItems (label) {
  return [
    { title: 'As rung', value: '' },
    { title: `${label || 'Deal'} · else 30%`, value: 'b2g1_30' },
    { title: `${label || 'Deal'} · else 20%`, value: 'b2g1_20' },
    { title: 'No deal', value: 'none' },
  ]
}

export function emptyLineDraft () {
  return { price: '', discount: '', noDeal: false, unknownDeal: '' }
}

// The draft as the till's overrides in force have it.
export function draftLinesFromOverrides (overrides) {
  const lines = {}
  for (const l of overrides?.lines || []) {
    const key = l.line_ref ? `ref:${l.line_ref}` : l.barcode
    lines[key] = {
      price: l.price_cents == null ? '' : (l.price_cents / 100).toFixed(2),
      discount: l.discount == null ? '' : String(l.discount),
      noDeal: Boolean(l.no_deal),
      unknownDeal: l.unknown_deal || '',
    }
  }
  return lines
}

// The override lines to send for the cart's lines and what is typed for
// them. Throws with a message for the user when something typed can't go.
export function buildOverrideLines (cartLines, draftLines, allDealsOff) {
  const out = []
  const seen = new Set()
  for (const line of cartLines || []) {
    if (!canSetLine(line)) {continue}
    const key = lineKey(line)
    if (seen.has(key)) {continue}
    seen.add(key)
    const l = draftLines[key]
    if (!l) {continue}
    const entry = { barcode: line.barcode }
    if (line.unknown) {entry.line_ref = line.lineRef}
    if (String(l.price ?? '').trim() !== '') {
      const cents = Math.round(Number(l.price) * 100)
      if (!(cents > 0)) {throw new Error(`${line.name}: enter a price above 0.`)}
      entry.price_cents = cents
    }
    if (String(l.discount ?? '').trim() !== '') {
      const discount = Number(l.discount)
      if (!Number.isInteger(discount) || discount < 0 || discount > 100) {
        throw new Error(`${line.name}: enter a discount from 0 to 100.`)
      }
      entry.discount = discount
    }
    if (line.unknown) {
      if (l.unknownDeal && !allDealsOff) {
        // A deal never stacks on a discount: the till would drop the deal.
        if (l.unknownDeal !== 'none' && entry.discount > 0) {
          throw new Error(`${line.name}: a deal doesn't go with a discount; clear one of them.`)
        }
        entry.unknown_deal = l.unknownDeal
      }
    } else if (l.noDeal && !allDealsOff) {
      entry.no_deal = true
    }
    if (entry.price_cents != null || entry.discount != null || entry.no_deal || entry.unknown_deal) {out.push(entry)}
  }
  return out
}
