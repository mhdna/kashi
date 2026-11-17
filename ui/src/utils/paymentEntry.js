// The payment calculator kashi shows for a remote return, exchange or
// settlement change, worked like the till's: type an amount, tap a payment
// method, and it becomes an entry in that method's currency; tap a method with
// nothing typed and it takes what's left. Amounts are USD cents, signed:
// positive is paid by the customer, negative is given back.
//
// What it sends is what the till reads (kashi-pos remote_payments.go):
// [{ account_name, amount_cents, amount_units? }], amount_units being the
// count in a cash-rounded currency's own units (LBP), which the till rounds
// and judges itself.

const MICRO = 1_000_000

// The methods a branch offers, in its order, each with its own currency and
// colour. Coupons are entered on the till only, so they're left out.
export function branchMethods (raw) {
  return (raw ?? [])
    .filter(m => m.is_enabled !== false && String(m.name).trim().toLowerCase() !== 'on coupon')
    .map(m => ({
      name: m.name,
      currency: m.override_currency_code || m.currency_code || 'USD',
      color: m.override_color || m.color || '',
      order: m.override_sort_order ?? m.sort_order ?? 0,
    }))
    .toSorted((a, b) => a.order - b.order || a.name.localeCompare(b.name))
}

// A currency's rate and cash step; USD (or one not found) is 1:1 in cents.
export function currencyOf (code, currencies) {
  const c = (currencies ?? []).find(x => x.code === code)
  if (!c || code === 'USD') {
    return { code: code || 'USD', unitsPerUsdMicros: MICRO, roundingUnit: 1, cents: true }
  }
  return {
    code,
    unitsPerUsdMicros: c.units_per_usd_micros || MICRO,
    roundingUnit: Math.max(c.cash_rounding_unit, 1),
    cents: false,
  }
}

function roundHalfUp (x) {
  return Math.sign(x) * Math.round(Math.abs(x))
}

// units of a non-USD currency -> USD cents
export function unitsToCents (units, cur) {
  return roundHalfUp(units * 100 * MICRO / cur.unitsPerUsdMicros)
}

// USD cents -> units of the currency, to its cash step (LL 5,000 and so on)
export function centsToUnits (cents, cur) {
  const units = cents * cur.unitsPerUsdMicros / (100 * MICRO)
  return roundHalfUp(units / cur.roundingUnit) * cur.roundingUnit
}

// The typed text as a number, or null when nothing usable is typed.
export function parseTyped (text) {
  const t = String(text ?? '').replace(/,/g, '').trim()
  if (t === '' || t === '.') {
    return null
  }
  const n = Number(t)
  return Number.isFinite(n) && n >= 0 ? n : null
}

// One entry: the method, its signed cents, and units when counted in a
// cash-rounded currency.
function entryFor (method, cents, units, cur) {
  const entry = { name: method.name, currency: cur.code, cents }
  if (!cur.cents) {
    entry.units = units
  }
  return entry
}

// addEntry is a tap on a method: the typed amount (dollars for USD, units for
// LBP), or what's left when nothing is typed. sign '+' or '-' forces the
// direction; otherwise it goes the way the total does (a refund gives back).
// Answers the new list, or the same one when there's nothing to add.
export function addEntry (entries, { method, typed, sign = null, dueCents, currencies }) {
  const cur = currencyOf(method.currency, currencies)
  const remaining = dueCents - enteredCents(entries)
  const amount = parseTyped(typed)
  let cents
  let units = 0
  if (amount === null) {
    if (remaining === 0) {
      return entries
    }
    if (cur.cents) {
      cents = remaining
    } else {
      units = centsToUnits(Math.abs(remaining), cur) * Math.sign(remaining)
      cents = unitsToCents(units, cur)
    }
  } else {
    if (amount === 0) {
      return entries
    }
    const directions = { '+': 1, '-': -1 }
    const direction = directions[sign] ?? (dueCents < 0 ? -1 : 1)
    if (cur.cents) {
      cents = direction * Math.round(amount * 100)
    } else {
      units = direction * Math.round(amount)
      cents = unitsToCents(units, cur)
    }
  }
  if (cents === 0) {
    return entries
  }
  return [...entries, entryFor(method, cents, Math.abs(units), cur)]
}

export function removeEntry (entries, index) {
  return entries.filter((_, i) => i !== index)
}

export function enteredCents (entries) {
  return (entries ?? []).reduce((sum, e) => sum + e.cents, 0)
}

// Half a cash step of the coarsest counted currency: LBP can't hit a total to
// the cent, and the till allows that much (validatePayments).
export function toleranceCents (entries, currencies) {
  let tolerance = 0
  for (const e of entries ?? []) {
    if (e.units !== undefined) {
      const cur = currencyOf(e.currency, currencies)
      tolerance = Math.max(tolerance, Math.ceil(unitsToCents(cur.roundingUnit, cur) / 2))
    }
  }
  return tolerance
}

// Where the entries stand against the total: due, entered, left, and done
// once what's left is zero (or within the LBP step).
export function paymentStatus (entries, dueCents, currencies) {
  const entered = enteredCents(entries)
  const remaining = dueCents - entered
  const done = (entries?.length ?? 0) > 0 && Math.abs(remaining) <= toleranceCents(entries, currencies)
  return { due: dueCents, entered, remaining, done }
}

// The command's payments: one line per entry, the till nets a method's lines.
export function toPayload (entries) {
  return (entries ?? []).map(e => ({
    account_name: e.name,
    amount_cents: e.cents,
    ...(e.units === undefined ? {} : { amount_units: e.units }),
  }))
}

// A sale's current settlement (kashi's branch_invoice_payments) as entries, to
// start a change of settlement from.
export function entriesFromSettlement (payments, methods, currencies) {
  return (payments ?? [])
    .filter(p => p.amount !== 0)
    .map(p => {
      const method = methods.find(m => m.name.toLowerCase() === String(p.account_name).toLowerCase())
      const cur = currencyOf(method?.currency ?? 'USD', currencies)
      const entry = { name: method?.name ?? p.account_name, currency: cur.code, cents: p.amount }
      if (!cur.cents) {
        entry.units = Math.abs(centsToUnits(Math.abs(p.amount), cur))
      }
      return entry
    })
}

// What a remote return refunds, as kashi can tell: each returned unit at what
// it was sold for on its line (negative). An exchange adds the replacements at
// the branch's prices. The till prices the real one (deals included) and
// refuses payments that don't match it, saying its own figure.
export function estimateCents (returned, replacements = [], prices = {}) {
  let total = 0
  for (const row of returned ?? []) {
    const qty = Math.abs(row.item.quantity) || 1
    total -= Math.round(row.item.line_total * row.quantity / qty)
  }
  for (const row of replacements ?? []) {
    total += (prices[row.variant.id] ?? 0) * row.quantity
  }
  return total
}

// Replacements kashi has no price for at that branch: the estimate leaves them
// out, so the dialog says so.
export function unpricedReplacements (replacements, prices) {
  return (replacements ?? []).filter(row => prices[row.variant.id] === undefined)
}

export function formatCents (cents) {
  const sign = cents < 0 ? '-' : ''
  const abs = Math.abs(cents)
  return `${sign}$${Math.floor(abs / 100).toLocaleString('en-US')}.${String(abs % 100).padStart(2, '0')}`
}

// An entry as typed: "$12.50" or "LBP 900,000", signed.
export function formatEntry (entry) {
  if (entry.units === undefined) {
    return formatCents(entry.cents)
  }
  const sign = entry.cents < 0 ? '-' : ''
  return `${sign}${entry.currency} ${entry.units.toLocaleString('en-US')}`
}
