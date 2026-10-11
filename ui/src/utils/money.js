export function formatMoney (cents) {
  return (Number(cents) / 100).toFixed(2)
}

// The default currency is stored in its smallest unit (USC, US cents -- see
// db/migrations/000013), but formatMoney has already turned the amount into
// dollars, so label it USD rather than USC.
export function displayCurrencyCode (code) {
  return code === 'USC' ? 'USD' : (code ?? '')
}

// Compact money for at-a-glance panels: $840, $12.4K, $1.2M. Negative
// amounts keep the sign outside the symbol: -$1.2K.
export function formatCompactMoney (cents) {
  const n = Number(cents || 0) / 100
  const abs = Math.abs(n)
  const sign = n < 0 ? '-' : ''
  if (abs >= 1e6) {
    return `${sign}$${(abs / 1e6).toFixed(1)}M`
  }
  if (abs >= 1e3) {
    return `${sign}$${(abs / 1e3).toFixed(1)}K`
  }
  return `${sign}$${abs.toFixed(0)}`
}

// An amount for reading, with its sign outside the symbol: "$28", "−$5".
// Invoices carry no currency code (their totals are USD), so no code means
// dollars; any other currency keeps its code after the number:
// "200,000 LBP".
export function formatAmount (amount, currency = '') {
  const value = Number(amount || 0) / 100
  const formatted = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(Math.abs(value))
  const sign = amount < 0 ? '−' : ''
  if (!currency || currency === 'USD') {return `${sign}$${formatted}`}
  return `${sign}${formatted} ${currency}`
}
