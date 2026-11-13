// Tills name an unknown unit "Unknown item $25.00", and the pages show the
// unit's price next to it, so a label ending in the unit's own price drops
// it (else "Unknown item $25.00 $25.00"). A label with another price, or a
// name the cashier typed, stays as it is. The phone upload page has the same
// function (api/unknown_upload_assets/upload.html, unknownLabel).
export function unknownLabel (label, cents) {
  let text = ` ${String(label || '').trim()}`
  const price = ` $${(Number(cents || 0) / 100).toFixed(2)}`
  if (text.endsWith(price)) {
    text = text.slice(0, -price.length)
  }
  return text.trim() || 'Unknown item'
}
