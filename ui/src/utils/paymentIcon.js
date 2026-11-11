// The icon beside a payment method, read from its name (same rules as the till,
// kashi-pos `paymentIcon`): a card for Visa and the like, a phone for Whish, a
// bank for transfers, notes for Lebanese lira cash, stacked cash for other cash.
// Every icon here is registered in plugins/icons.js.
export function paymentIcon (name) {
  const value = String(name ?? '').toLowerCase()
  if (/visa|master ?card|credit|debit|\bcard\b/.test(value)) {
    return 'mdi-credit-card-outline'
  }
  if (value.includes('whish')) {
    return 'mdi-cellphone'
  }
  if (/bank|transfer|wire|cheque|check/.test(value)) {
    return 'mdi-bank'
  }
  if (/\bll\b|lbp|lebanese|lira/.test(value)) {
    return 'mdi-cash'
  }
  return 'mdi-cash-multiple'
}
