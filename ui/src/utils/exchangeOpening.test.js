import { describe, expect, it } from 'vitest'
import {
  canOpenForExchange, formatOpeningDate, localDateString, MAX_OPENING_DAYS, openingDays, openingPayload, openingUntilDate,
} from './exchangeOpening'

// Sunday 22 Nov 2026, late evening: the UTC day may already be the 23rd.
const now = new Date(2026, 10, 22, 23, 30)

describe('openingUntilDate', () => {
  it('only today ends today', () => {
    expect(openingUntilDate('today', 99, now)).toBe('2026-11-22')
  })
  it('counts today among the days', () => {
    expect(openingUntilDate('days', 1, now)).toBe('2026-11-22')
    expect(openingUntilDate('days', 5, now)).toBe('2026-11-26')
    expect(openingUntilDate('days', '10', now)).toBe('2026-12-01')
  })
  it('crosses months, years and leap days', () => {
    expect(openingUntilDate('days', 40, now)).toBe('2026-12-31')
    expect(openingUntilDate('days', 41, now)).toBe('2027-01-01')
    expect(openingUntilDate('days', 2, new Date(2028, 1, 28, 9))).toBe('2028-02-29')
  })
  it('refuses what the till could not take', () => {
    for (const bad of [0, -1, 1.5, '', null, undefined, 'abc', MAX_OPENING_DAYS + 1]) {
      expect(openingUntilDate('days', bad, now)).toBe('')
      expect(openingDays('days', bad)).toBe(0)
    }
    expect(openingUntilDate('days', MAX_OPENING_DAYS, now)).not.toBe('')
  })
  it('over every day count and many start times, the last day is exactly days-1 after today', () => {
    let seed = 5
    const rand = n => { seed = (seed * 16_807) % 2_147_483_647; return seed % n }
    for (let i = 0; i < 500; i++) {
      const start = new Date(2026, rand(12), 1 + rand(28), rand(24), rand(60))
      const n = 1 + rand(MAX_OPENING_DAYS)
      const until = openingUntilDate('days', n, start)
      const [y, m, d] = until.split('-').map(Number)
      const startDay = new Date(start.getFullYear(), start.getMonth(), start.getDate())
      const diff = Math.round((new Date(y, m - 1, d) - startDay) / 86_400_000)
      expect(diff).toBe(n - 1)
    }
  })
})

describe('helpers', () => {
  it('uses the local day, not the UTC one', () => {
    expect(localDateString(now)).toBe('2026-11-22')
  })
  it('formats the last day for the admin', () => {
    expect(formatOpeningDate('2026-11-26')).toContain('26 Nov 2026')
    expect(formatOpeningDate('')).toBe('')
    expect(formatOpeningDate('26/11/2026')).toBe('')
  })
  it('builds the command payload', () => {
    expect(openingPayload({ client_ref: 'sale-1' }, 'days', 3, '  travelling ', now))
      .toEqual({ client_ref: 'sale-1', until_date: '2026-11-24', reason: 'travelling' })
  })
  it('offers the button only for a sale a till has', () => {
    const sale = { kind: 'sales', client_ref: 'sale-1', historical: false, cancelled_at: { Time: '0001-01-01T00:00:00Z', Valid: false } }
    expect(canOpenForExchange(sale)).toBe(true)
    expect(canOpenForExchange({ ...sale, cancelled_at: null })).toBe(true)
    expect(canOpenForExchange({ ...sale, cancelled_at: { Time: '2026-11-20T10:00:00Z', Valid: true } })).toBe(false)
    expect(canOpenForExchange({ ...sale, historical: true })).toBe(false)
    expect(canOpenForExchange({ ...sale, kind: 'return' })).toBe(false)
    expect(canOpenForExchange({ ...sale, kind: 'exchange' })).toBe(false)
    expect(canOpenForExchange({ ...sale, client_ref: '' })).toBe(false)
    expect(canOpenForExchange(null)).toBe(false)
  })
})
