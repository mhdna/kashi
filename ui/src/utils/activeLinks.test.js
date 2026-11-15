import { describe, expect, it } from 'vitest'
import { linkDetail, linkTitle, linkUrl, timeLeft } from './activeLinks'

const base = { branch_name: 'Centro', ref_id: 7, label: '', amount: 0, currency_code: '', units: 0, done: 0 }

describe('active links', () => {
  it('the reported case: an unknown item link reads as the URL the till showed', () => {
    const link = { ...base, kind: 'unknown', path: '/u/9bZAgBdrUOh31zcm', units: 3, done: 1 }
    expect(linkUrl(link, 'https://cloud.grandbz.com/api')).toBe('https://cloud.grandbz.com/api/u/9bZAgBdrUOh31zcm')
    expect(linkUrl(link, 'https://cloud.grandbz.com/api/')).toBe('https://cloud.grandbz.com/api/u/9bZAgBdrUOh31zcm')
    expect(linkTitle(link)).toBe('Unknown items')
    expect(linkDetail(link)).toBe('Centro, 1 of 3 pictured')
  })

  it('names each kind and says what it covers', () => {
    expect(linkTitle({ kind: 'expense' })).toBe('Expense photo')
    expect(linkTitle({ kind: 'loan' })).toBe('Loan photo')
    expect(linkTitle({ kind: 'shift' })).toBe('Shift receipt')
    expect(linkTitle({ kind: 'transfer' })).toBe('Transfer photo')
    expect(linkTitle({ kind: 'something-new' })).toBe('Upload link')

    expect(linkDetail({ ...base, kind: 'expense', label: 'Electricity', amount: 4550, currency_code: 'USD' })).toBe('Centro, Electricity, $45.5')
    expect(linkDetail({ ...base, kind: 'expense', label: '  ', amount: 0 })).toBe('Centro')
    expect(linkDetail({ ...base, kind: 'loan', label: 'Rana', amount: 10_000, currency_code: 'USD' })).toBe('Centro, Rana, $100')
    expect(linkDetail({ ...base, kind: 'shift' })).toBe('Centro, Shift 7')
    expect(linkDetail({ ...base, kind: 'transfer', label: 'Winter coats' })).toBe('Centro, Winter coats')
    expect(linkDetail({ ...base, kind: 'transfer' })).toBe('Centro, Transfer 7')
    // No branch (a loan made from central): just what it covers.
    expect(linkDetail({ ...base, branch_name: '', kind: 'loan', label: 'Rana' })).toBe('Rana')
  })

  it('says how long the link still works', () => {
    const now = Date.parse('2026-11-15T10:00:00Z')
    const at = minutes => new Date(now + minutes * 60_000).toISOString()
    expect(timeLeft(at(-1), now)).toBe('Expired')
    expect(timeLeft(at(0.5), now)).toBe('Expires in under a minute')
    expect(timeLeft(at(12), now)).toBe('Expires in 12 min')
    expect(timeLeft(at(60), now)).toBe('Expires in 1 h')
    expect(timeLeft(at(65), now)).toBe('Expires in 1 h 5 min')
  })
})
