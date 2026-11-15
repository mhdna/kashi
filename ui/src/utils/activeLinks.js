import { formatAmount } from '@/utils/money'

// The phone links tills hand out (as a QR code) that still work, as the home
// page's "Active links" list shows them. Rows come from GET /home/active_links.

const TITLES = {
  unknown: 'Unknown items',
  expense: 'Expense photo',
  loan: 'Loan photo',
  shift: 'Shift receipt',
  transfer: 'Transfer photo',
}

export function linkTitle (link) {
  return TITLES[link.kind] ?? 'Upload link'
}

// What the link is for, in a few words: the branch, then what it covers.
export function linkDetail (link) {
  const parts = []
  if (link.branch_name) {parts.push(link.branch_name)}
  switch (link.kind) {
    case 'unknown': {
      parts.push(`${link.done} of ${link.units} pictured`)
      break
    }
    case 'expense':
    case 'loan': {
      const amount = link.amount ? formatAmount(link.amount, link.currency_code) : ''
      const what = [link.label?.trim(), amount].filter(Boolean).join(', ')
      if (what) {parts.push(what)}
      break
    }
    case 'shift': {
      parts.push(`Shift ${link.ref_id}`)
      break
    }
    case 'transfer': {
      parts.push(link.label?.trim() || `Transfer ${link.ref_id}`)
      break
    }
  }
  return parts.join(', ')
}

// "Expires in 12 min", "in 1 h 5 min", or "Expired" once it's past.
export function timeLeft (expiresAt, now = Date.now()) {
  const minutes = Math.floor((new Date(expiresAt).getTime() - now) / 60_000)
  if (minutes < 0) {return 'Expired'}
  if (minutes < 1) {return 'Expires in under a minute'}
  if (minutes < 60) {return `Expires in ${minutes} min`}
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest ? `Expires in ${hours} h ${rest} min` : `Expires in ${hours} h`
}

// The full link, the same one the till shows: the API's address plus the
// short path (/u/<token>, /e/<token>...).
export function linkUrl (link, apiBase) {
  return `${apiBase.replace(/\/+$/, '')}${link.path}`
}
