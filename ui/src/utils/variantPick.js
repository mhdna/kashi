// "Pick colour/size on scan", per item: the Products page's dialog sends
// the pasted codes with one of these modes (kashi's products.variant_pick).
export const VARIANT_PICK_MODES = [
  { value: 'always', title: 'Always ask which colour/size' },
  { value: 'never', title: 'Never ask (sell the scanned item)' },
  { value: '', title: 'Follow the branch setting' },
]

// Codes and/or barcodes, pasted or typed: split on whitespace, commas and
// semicolons; blanks and repeats (any case, like the server) are dropped.
export function parseEntries (text) {
  const seen = new Set()
  const entries = []
  for (const raw of String(text ?? '').split(/[\s,;]+/)) {
    const entry = raw.trim()
    if (!entry || seen.has(entry.toLowerCase())) continue
    seen.add(entry.toLowerCase())
    entries.push(entry)
  }
  return entries
}

// What the cashier will see now, in the toast after saving.
export function savedMessage (mode, count) {
  const items = `${count} item${count === 1 ? '' : 's'}`
  if (count === 0) return 'No item matched those codes.'
  if (mode === 'always') return `${items} will ask for the colour/size at the till.`
  if (mode === 'never') return `${items} will never ask for the colour/size.`
  return `${items} will follow the branch setting.`
}
