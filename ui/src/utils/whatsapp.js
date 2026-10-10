// Shared bits of the WhatsApp page (pages/whatsapp.vue, components/WhatsApp).

export function ymd (d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export function addDays (d, n) {
  const out = new Date(d)
  out.setDate(out.getDate() + n)
  return out
}

// Every day from..to (YYYY-MM-DD, inclusive), so a chart shows quiet days.
export function daysBetween (from, to) {
  const out = []
  const end = new Date(`${to}T00:00:00`)
  for (let d = new Date(`${from}T00:00:00`); d <= end; d = addDays(d, 1)) {
    out.push(ymd(d))
  }
  return out
}

// Message costs are stored in millionths of a USD; a single message is a
// few cents, so small amounts keep more decimals.
export function formatCost (micros) {
  const usd = Number(micros || 0) / 1e6
  if (usd === 0) {return '$0'}
  if (Math.abs(usd) < 1) {return `$${usd.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')}`}
  return `$${usd.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export function formatDollars (cents) {
  return `$${(Number(cents || 0) / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export const STATUS = {
  sent: { label: 'Sent', color: 'success', icon: 'mdi-check-circle-outline' },
  skipped: { label: 'Skipped', color: 'grey', icon: 'mdi-cancel' },
  failed: { label: 'Failed', color: 'error', icon: 'mdi-alert-circle-outline' },
}

// Placeholders a template's {{1}}, {{2}}, … can be filled with
// (whatsAppTemplateParams in api/whatsapp.go).
export const TEMPLATE_PARAMS = ['name', 'market', 'branch', 'invoice', 'total']

// International numbers get their "+"; a local one ("70123456") reads as is.
export function formatPhone (digits) {
  if (!digits) {return ''}
  return digits.length >= 10 && !digits.startsWith('0') ? `+${digits}` : digits
}

// What happened after Meta accepted a sent message, from its webhook
// (whatsapp_messages.delivery_status). '' = no word from Meta yet.
export const DELIVERY = {
  '': { label: 'Sent', color: 'success', icon: 'mdi-check' },
  sent: { label: 'Sent', color: 'success', icon: 'mdi-check' },
  delivered: { label: 'Delivered', color: 'success', icon: 'mdi-check-all' },
  read: { label: 'Read', color: 'info', icon: 'mdi-check-all' },
  failed: { label: 'Not delivered', color: 'error', icon: 'mdi-alert-circle-outline' },
}

// sqlc serves nullable timestamps as { Time, Valid }.
export function nullTime (v) {
  return v?.Valid ? v.Time : null
}
