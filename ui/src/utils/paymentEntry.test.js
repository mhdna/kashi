import { describe, expect, it } from 'vitest'
import {
  addEntry,
  branchMethods,
  centsToUnits,
  currencyOf,
  entriesFromSettlement,
  estimateCents,
  formatCents,
  formatEntry,
  parseTyped,
  paymentStatus,
  removeEntry,
  toPayload,
  unitsToCents,
  unpricedReplacements,
} from './paymentEntry'

// LBP at 89,500 to the dollar, counted in LL 5,000 steps.
const currencies = [
  { code: 'USD', units_per_usd_micros: 1_000_000, cash_rounding_unit: 1 },
  { code: 'LBP', units_per_usd_micros: 89_500_000_000, cash_rounding_unit: 5000 },
]
const cash = { name: 'Cash $$', currency: 'USD' }
const lira = { name: 'Cash LL', currency: 'LBP' }
const visa = { name: 'Visa', currency: 'USD' }
const account = { name: 'On Account', currency: 'USD' }

describe('branchMethods', () => {
  it('keeps enabled methods in the branch order, with its currency and colour, and no coupons', () => {
    const got = branchMethods([
      { name: 'Visa', is_enabled: true, currency_code: 'USD', sort_order: 3, color: '#00f' },
      { name: 'Cash LL', is_enabled: true, currency_code: 'USD', override_currency_code: 'LBP', sort_order: 2, override_sort_order: 0, color: '#0a0' },
      { name: 'On Coupon', is_enabled: true, currency_code: 'USD', sort_order: 1 },
      { name: 'Whish', is_enabled: false, currency_code: 'USD', sort_order: 1 },
      { name: 'Cash $$', is_enabled: true, currency_code: 'USD', sort_order: 1, override_color: '#f00', color: '#999' },
    ])
    expect(got.map(m => m.name)).toEqual(['Cash LL', 'Cash $$', 'Visa'])
    expect(got[0].currency).toBe('LBP')
    expect(got[1].color).toBe('#f00')
  })
  it('copes with nothing', () => {
    expect(branchMethods(null)).toEqual([])
  })
})

describe('currency conversion', () => {
  const lbp = currencyOf('LBP', currencies)
  it('USD and unknown codes are cents', () => {
    expect(currencyOf('USD', currencies).cents).toBe(true)
    expect(currencyOf('EUR', currencies).cents).toBe(true)
  })
  it('LBP units to cents and back to its cash step', () => {
    expect(unitsToCents(89_500, lbp)).toBe(100)
    expect(unitsToCents(900_000, lbp)).toBe(1006) // 10.0558...
    expect(centsToUnits(1000, lbp)).toBe(895_000)
    expect(centsToUnits(1001, lbp)).toBe(895_000) // 895,895 -> nearest 5,000
    expect(centsToUnits(1003, lbp)).toBe(900_000) // 897,685 -> up
  })
})

describe('parseTyped', () => {
  it('reads numbers, commas included, and nothing else', () => {
    expect(parseTyped('12.5')).toBe(12.5)
    expect(parseTyped('900,000')).toBe(900_000)
    for (const t of ['', ' ', '.', 'abc', '-5', null, undefined]) {
      expect(parseTyped(t)).toBeNull()
    }
  })
})

describe('addEntry', () => {
  it('a tap with nothing typed takes what is left, in that method', () => {
    let entries = addEntry([], { method: cash, typed: '40', dueCents: 6556, currencies })
    entries = addEntry(entries, { method: visa, typed: '', dueCents: 6556, currencies })
    expect(entries).toEqual([
      { name: 'Cash $$', currency: 'USD', cents: 4000 },
      { name: 'Visa', currency: 'USD', cents: 2556 },
    ])
    expect(paymentStatus(entries, 6556, currencies)).toEqual({ due: 6556, entered: 6556, remaining: 0, done: true })
  })

  it('a refund gives back: typed amounts go negative', () => {
    let entries = addEntry([], { method: cash, typed: '10', dueCents: -1556, currencies })
    entries = addEntry(entries, { method: visa, typed: '', dueCents: -1556, currencies })
    expect(entries.map(e => e.cents)).toEqual([-1000, -556])
    expect(paymentStatus(entries, -1556, currencies).done).toBe(true)
  })

  it('+ and - force the direction, like the till', () => {
    const paid = addEntry([], { method: cash, typed: '20', sign: '+', dueCents: -500, currencies })
    expect(paid[0].cents).toBe(2000)
    const change = addEntry(paid, { method: cash, typed: '25', sign: '-', dueCents: -500, currencies })
    expect(change[1].cents).toBe(-2500)
    expect(paymentStatus(change, -500, currencies).done).toBe(true)
  })

  it('LBP is typed in units and kept with its cents', () => {
    const entries = addEntry([], { method: lira, typed: '900000', dueCents: 1006, currencies })
    expect(entries).toEqual([{ name: 'Cash LL', currency: 'LBP', cents: 1006, units: 900_000 }])
  })

  it('LBP with nothing typed takes what is left to the cash step, close enough to be done', () => {
    const entries = addEntry([], { method: lira, typed: '', dueCents: 1001, currencies })
    expect(entries[0].units).toBe(895_000)
    const status = paymentStatus(entries, 1001, currencies)
    expect(status.remaining).toBe(1)
    expect(status.done).toBe(true)
  })

  it('adds nothing for zero, or a tap when nothing is left', () => {
    const done = [{ name: 'Cash $$', currency: 'USD', cents: 500 }]
    expect(addEntry(done, { method: cash, typed: '', dueCents: 500, currencies })).toBe(done)
    expect(addEntry([], { method: cash, typed: '0', dueCents: 500, currencies })).toEqual([])
    expect(addEntry([], { method: cash, typed: '0.001', dueCents: 500, currencies })).toEqual([])
  })

  it('an over-payment shows as a negative remainder and is not done', () => {
    const entries = addEntry([], { method: cash, typed: '70', dueCents: 6556, currencies })
    const status = paymentStatus(entries, 6556, currencies)
    expect(status.remaining).toBe(-444)
    expect(status.done).toBe(false)
  })

  it('nothing entered is never done, even for a zero total', () => {
    expect(paymentStatus([], 0, currencies).done).toBe(false)
  })
})

describe('removeEntry', () => {
  it('drops one line', () => {
    const entries = [{ cents: 1 }, { cents: 2 }, { cents: 3 }]
    expect(removeEntry(entries, 1)).toEqual([{ cents: 1 }, { cents: 3 }])
  })
})

describe('toPayload', () => {
  it('is what the till reads, units only for counted currencies', () => {
    expect(toPayload([
      { name: 'Cash $$', currency: 'USD', cents: -1000 },
      { name: 'Cash LL', currency: 'LBP', cents: -1006, units: 900_000 },
    ])).toEqual([
      { account_name: 'Cash $$', amount_cents: -1000 },
      { account_name: 'Cash LL', amount_cents: -1006, amount_units: 900_000 },
    ])
  })
})

describe('entriesFromSettlement', () => {
  it("starts a change of settlement from the sale's split", () => {
    const methods = [cash, lira, account]
    expect(entriesFromSettlement([
      { account_name: 'on account', amount: 6556 },
      { account_name: 'Cash LL', amount: 1000 },
      { account_name: 'Gone method', amount: 5 },
      { account_name: 'Cash $$', amount: 0 },
    ], methods, currencies)).toEqual([
      { name: 'On Account', currency: 'USD', cents: 6556 },
      { name: 'Cash LL', currency: 'LBP', cents: 1000, units: 895_000 },
      { name: 'Gone method', currency: 'USD', cents: 5 },
    ])
  })
})

describe('estimateCents', () => {
  const line = (lineTotal, quantity) => ({ item: { line_total: lineTotal, quantity } })
  it('a return refunds each unit at what its line sold it for', () => {
    expect(estimateCents([{ ...line(6556, 1), quantity: 1 }])).toBe(-6556)
    expect(estimateCents([{ ...line(3000, 3), quantity: 2 }])).toBe(-2000)
    expect(estimateCents([{ ...line(1000, 3), quantity: 1 }])).toBe(-333)
  })
  it('an exchange adds the replacements at the branch price', () => {
    const replacements = [{ variant: { id: 7 }, quantity: 2 }, { variant: { id: 9 }, quantity: 1 }]
    expect(estimateCents([{ ...line(1000, 1), quantity: 1 }], replacements, { 7: 750 })).toBe(500)
    expect(unpricedReplacements(replacements, { 7: 750 }).map(r => r.variant.id)).toEqual([9])
  })
})

describe('format', () => {
  it('money and entries', () => {
    expect(formatCents(6556)).toBe('$65.56')
    expect(formatCents(-5)).toBe('-$0.05')
    expect(formatCents(12_345_678)).toBe('$123,456.78')
    expect(formatEntry({ currency: 'LBP', cents: -1006, units: 900_000 })).toBe('-LBP 900,000')
    expect(formatEntry({ currency: 'USD', cents: 1250 })).toBe('$12.50')
  })
})

// Whatever is tapped, the figures shown add up: entered is the entries'
// sum, remaining is due less entered, and the payload carries the same cents.
describe('randomized identities', () => {
  let seed = 7
  const rand = n => {
    seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648
    return seed % n
  }
  it('hold over 500 random sessions', () => {
    const methods = [cash, lira, visa, account]
    for (let run = 0; run < 500; run++) {
      const due = (rand(2) ? 1 : -1) * (1 + rand(50_000))
      let entries = []
      const taps = 1 + rand(5)
      for (let i = 0; i < taps; i++) {
        const method = methods[rand(methods.length)]
        const typed = rand(3) === 0 ? '' : String(method.currency === 'LBP' ? rand(400) * 5000 : rand(30_000) / 100)
        const sign = [null, null, '+', '-'][rand(4)]
        entries = addEntry(entries, { method, typed, sign, dueCents: due, currencies })
      }
      if (rand(4) === 0 && entries.length > 0) {
        entries = removeEntry(entries, rand(entries.length))
      }
      const status = paymentStatus(entries, due, currencies)
      const sum = entries.reduce((s, e) => s + e.cents, 0)
      expect(status.entered).toBe(sum)
      expect(status.remaining).toBe(due - sum)
      expect(toPayload(entries).reduce((s, p) => s + p.amount_cents, 0)).toBe(sum)
      for (const e of entries) {
        expect(e.cents).not.toBe(0)
        if (e.units !== undefined) {
          expect(e.units % 5000).toBe(0)
          expect(e.cents).toBe(Math.sign(e.cents) * unitsToCents(e.units, currencyOf('LBP', currencies)))
        }
      }
      // A final tap in USD with nothing typed always settles it exactly.
      const settled = addEntry(entries, { method: cash, typed: '', dueCents: due, currencies })
      expect(paymentStatus(settled, due, currencies).remaining).toBe(0)
    }
  })
})
