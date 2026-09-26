import { formatMoney } from '@/utils/money'

// Which admin-socket message types surface as branch activity (a toast or
// the app-bar ticker). Mirrors the message types documented on
// adminWSMessage in api/admin_ws.go.
export const BRANCH_ACTIVITY_TYPES = [
  'branch_invoice_created',
  'branch_expense_created',
  'branch_loan_created',
  'branch_shift_closed',
  'branch_settlement_changed',
  'branch_attendance_event',
  'branch_attendance_events',
  'branch_attendance_changed',
  'branch_visitor_event',
  'branch_sync_conflict',
]

export function isBranchActivityMessage (message) {
  return BRANCH_ACTIVITY_TYPES.includes(message?.type)
}

function amountColor (amount) {
  if (amount > 0) {
    return 'success'
  }
  if (amount < 0) {
    return 'error'
  }
  return 'warning'
}

function invoiceKindWord (message) {
  if (message.type === 'branch_expense_created') {
    return 'expense'
  }
  if (message.type === 'branch_loan_created') {
    return 'loan'
  }
  if (message.kind === 'exchange') {
    return 'exchange'
  }
  if (message.kind === 'return') {
    return 'return'
  }
  if (message.kind) {
    return 'revenue'
  }
  return message.amount < 0 ? 'return' : 'revenue'
}

function invoiceIcon (message) {
  if (message.type === 'branch_expense_created') {
    return 'mdi-cash-minus'
  }
  if (message.type === 'branch_loan_created') {
    // Borrowed money coming in -- a hand passing a coin, not a refund.
    return 'mdi-hand-coin'
  }
  if (message.kind === 'exchange') {
    return 'mdi-swap-horizontal'
  }
  if (message.kind === 'return') {
    // Money refunded to the customer.
    return 'mdi-cash-refund'
  }
  return 'mdi-sale'
}

// The manager-approval workflow has three outcomes that read very
// differently -- a filed request, an approved edit, a rejected one. The
// backend sends `kind` as "complaint" | "approved" | "rejected" (see
// attendanceChangeKind in api/branch_attendance.go).
const ATTENDANCE_CHANGE_ICON = {
  complaint: 'mdi-clock-edit-outline',
  approved: 'mdi-clock-check-outline',
  rejected: 'mdi-clock-remove-outline',
}

function attendanceChangeIcon (kind) {
  return ATTENDANCE_CHANGE_ICON[kind] ?? 'mdi-clock-edit-outline'
}

const SYNC_CONFLICT_WORDING = {
  phone_name_mismatch: 'a client name disagreement',
  invalid_phone: 'a client with no usable phone',
  phone_taken: 'a client phone already used by another client',
}

// invoiceTone picks the toast background tint for an invoice-family
// message. Extracted from describeBranchActivity to keep that function's
// complexity under the lint ceiling (and out of a nested ternary).
function invoiceTone (type, kind) {
  if (type === 'branch_expense_created') {
    return 'red-darken-4'
  }
  if (type === 'branch_loan_created') {
    return 'deep-purple-darken-4'
  }
  if (kind === 'exchange') {
    return 'orange-darken-4'
  }
  if (kind === 'return') {
    return 'red-darken-4'
  }
  return 'green-darken-4'
}

// describeSyncConflict is split out of describeBranchActivity's switch so
// that function's cyclomatic complexity doesn't creep further.
function describeSyncConflict (message, branch) {
  const what = SYNC_CONFLICT_WORDING[message.kind] ?? 'a sync conflict'
  return {
    text: `${branch}: ${what} needs resolving`,
    shortText: `${branch}: sync conflict`,
    color: 'warning',
    tone: 'amber-darken-4',
    icon: 'mdi-alert-circle-outline',
    money: false,
  }
}

// describeBranchActivity turns one admin-socket message into the shape both
// the toast (BranchActivityToast) and the app-bar reel (SyncCard) render:
//
//   text      full one-line summary, branch name + detail -- for the toast
//   shortText trimmed version for the app-bar reel, where up to three sit
//             side by side: branch + the essential figure, nothing more
//   color     'success' | 'error' | 'warning' | 'info'
//   trendIcon triangle up/down, for the money types
//   icon      leading mdi glyph
//   amount    signed cents when the message carries money, else undefined
//   money     true when the toast should use its amount+currency+kind layout
//
// Returns null for a type that isn't branch activity.
export function describeBranchActivity (message, branchName) {
  const branch = branchName(message.branch_id)

  switch (message.type) {
    case 'branch_invoice_created':
    case 'branch_expense_created':
    case 'branch_loan_created': {
      const kind = invoiceKindWord(message)
      const money = `${formatMoney(message.amount)} ${message.currency_code ?? ''}`.trim()
      return {
        text: `${branch}: ${money} ${kind}`.replace(/\s+/g, ' ').trim(),
        // Kind is already carried by the icon + colour, so the reel drops it.
        shortText: `${branch}: ${money}`,
        color: amountColor(message.amount),
        // Toast background tint keys off what happened, not the amount sign:
        // green revenue, red return/expense, orange exchange, purple loan.
        tone: invoiceTone(message.type, kind),
        trendIcon: message.amount < 0 ? 'mdi-triangle-down' : 'mdi-triangle',
        icon: invoiceIcon(message),
        amount: message.amount,
        currencyCode: message.currency_code,
        kindWord: kind,
        money: true,
      }
    }

    case 'branch_shift_closed': {
      const balanced = !message.amount
      const detail = balanced
        ? 'balanced'
        : `${formatMoney(message.amount)} ${message.currency_code ?? ''}`.trim()
      const who = message.label ? ` by ${message.label}` : ''
      return {
        text: `${branch}: shift closed${who} (${detail})`,
        shortText: `${branch}: shift ${detail}`,
        color: balanced ? 'success' : amountColor(message.amount),
        trendIcon: message.amount < 0 ? 'mdi-triangle-down' : 'mdi-triangle',
        icon: 'mdi-cash-register',
        amount: message.amount,
        money: false,
      }
    }

    case 'branch_settlement_changed': {
      return {
        text: `${branch}: settlement changed${message.label ? ` on ${message.label}` : ''}`,
        shortText: `${branch}: settlement`,
        color: 'info',
        icon: 'mdi-cash-edit',
        money: false,
      }
    }

    case 'branch_attendance_event': {
      return {
        text: `${branch}: attendance${message.label ? ` — ${message.label}` : ''}`,
        shortText: `${branch}: attendance`,
        color: 'info',
        icon: 'mdi-fingerprint',
        money: false,
      }
    }

    case 'branch_attendance_events': {
      const n = message.amount ?? 0
      return {
        text: `${branch}: ${n} attendance event${n === 1 ? '' : 's'}`,
        shortText: `${branch}: ${n} attendance`,
        color: 'info',
        icon: 'mdi-fingerprint',
        money: false,
      }
    }

    case 'branch_attendance_changed': {
      const verb = {
        complaint: 'time-change requested',
        approved: 'time change approved',
        rejected: 'time change rejected',
      }[message.kind] ?? 'attendance change'
      const shortVerb = {
        complaint: 'time change asked',
        approved: 'time change ok',
        rejected: 'time change no',
      }[message.kind] ?? 'attendance change'
      return {
        text: `${branch}: ${verb}${message.label ? ` — ${message.label}` : ''}`,
        shortText: `${branch}: ${shortVerb}`,
        color: message.kind === 'rejected' ? 'warning' : 'info',
        icon: attendanceChangeIcon(message.kind),
        money: false,
      }
    }

    case 'branch_visitor_event': {
      // kind is the counter direction: "in" (▲) or "out" (▼); amount is the
      // running gross tally for that direction on the current local day.
      const isOut = message.kind === 'out'
      const word = isOut ? 'out' : 'in'
      const soFar = message.amount ?? 0
      return {
        text: `${branch}: customer ${word} — ${soFar} ${word} today`,
        shortText: `${branch}: ${soFar} ${word}`,
        color: 'info',
        tone: 'blue-darken-4',
        icon: isOut ? 'mdi-account-arrow-left' : 'mdi-account-arrow-right',
        money: false,
      }
    }

    case 'branch_sync_conflict': {
      return describeSyncConflict(message, branch)
    }

    default: {
      return null
    }
  }
}
