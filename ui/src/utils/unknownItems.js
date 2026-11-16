import { formatAmount } from '@/utils/money'

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

// What the label reader made of a unit's picture (api/label_reader.go), as
// the Unknown items page shows it. status: 'read', 'failed' or 'none'.
// details joins the code, barcode and label price with commas; warning is
// set when the label's price isn't what the cashier charged (the product
// still takes the cashier's price). fromPhoto: the label reader made the
// product the unit went to.
export function labelReading (item) {
  const status = item?.label_status === 'read' || item?.label_status === 'failed' ? item.label_status : 'none'
  const reading = { status, name: '', details: '', warning: '', fromPhoto: item?.resolved_source === 'label_photo' }
  if (status !== 'read') {
    return reading
  }
  reading.name = String(item.label_name || '').trim()
  const hasPrice = item.label_price !== null && item.label_price !== undefined
  reading.details = [
    String(item.label_code || '').trim(),
    String(item.label_barcode || '').trim(),
    hasPrice ? `label ${formatAmount(item.label_price)}` : '',
  ].filter(Boolean).join(', ')
  if (item.label_price_mismatch && hasPrice) {
    reading.warning = `Label says ${formatAmount(item.label_price)}, sold at ${formatAmount(item.unit_price)}`
  }
  return reading
}
