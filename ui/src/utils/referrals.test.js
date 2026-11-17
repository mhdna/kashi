import { describe, expect, it } from 'vitest'
import { dollarsToCents, formToRequest, programSummary, programToForm, referralRow, statusColor } from './referrals'

describe('referral rules form', () => {
  it('round-trips the API rules', () => {
    const api = {
      enabled: true, reward_percent: 15, max_reward_cents: 2550, min_friend_spend: 3000,
      unlock_after_days: 0, expires_after_days: 60, cancel_on_return: true, max_rewards_per_month: 3,
    }
    const form = programToForm(api)
    expect(form.maxRewardDollars).toBe('25.5')
    expect(form.minFriendSpendDollars).toBe('30')
    expect(formToRequest(form)).toEqual(api)
  })

  it('reads empty as none and dollars to the cent', () => {
    expect(dollarsToCents('')).toBe(0)
    expect(dollarsToCents('$30')).toBe(3000)
    expect(dollarsToCents('1,250.10')).toBe(125010)
    expect(dollarsToCents('0.29')).toBe(29)
    expect(() => dollarsToCents('-5')).toThrow(/dollars/)
    expect(() => dollarsToCents('abc', 'The cap')).toThrow(/The cap/)
  })

  it('refuses what the API would', () => {
    const ok = programToForm({ reward_percent: 15 })
    expect(() => formToRequest({ ...ok, rewardPercent: 0 })).toThrow(/1% to 100%/)
    expect(() => formToRequest({ ...ok, rewardPercent: 101 })).toThrow(/1% to 100%/)
    expect(() => formToRequest({ ...ok, rewardPercent: 12.5 })).toThrow(/whole/)
    expect(() => formToRequest({ ...ok, unlockAfterDays: -1 })).toThrow(/whole/)
  })

  it('defaults to the user\'s 15%, off, right away, forever', () => {
    const req = formToRequest(programToForm({}))
    expect(req).toMatchObject({ enabled: false, reward_percent: 15, unlock_after_days: 0, expires_after_days: 0, max_reward_cents: 0 })
  })

  it('says what the rules do, without long dashes or dots', () => {
    expect(programSummary({ enabled: false })).toMatch(/off/)
    const s = programSummary({ enabled: true, reward_percent: 15, max_reward_cents: 2000, min_friend_spend: 3000 })
    expect(s).toBe('A new customer\'s first purchase with a client\'s code gives that client 15% off one later visit, up to $20, when the purchase is $30 or more.')
    expect(s).not.toMatch(/[–—·•]/)
  })
})

describe('referral rows', () => {
  const base = { id: 1, code: 'K7M4QH', owner_name: 'Rami', friend_name: 'Sara', branch_name: 'Hara', sale_code: 'SA/1', sale_total: 4000, percent: 15, max_cents: 0 }
  it('reads each status', () => {
    expect(referralRow({ ...base, status: 'available' })).toMatchObject({ statusLabel: 'Ready to use', detail: '15% off', owner: 'Rami', friend: 'Sara' })
    expect(referralRow({ ...base, status: 'used', used_code: 'SA/2' }).detail).toBe('15% used on SA/2')
    expect(referralRow({ ...base, status: 'used', used_code: 'SA/2', used_again_code: 'SA/3' }).detail).toBe('15% used on SA/2, and again on SA/3')
    expect(referralRow({ ...base, status: 'refused', refuse_reason: 'The customer had bought before' }).detail).toBe('The customer had bought before')
    expect(referralRow({ ...base, status: 'cancelled' }).statusLabel).toBe('Taken back')
    expect(referralRow({ ...base, status: 'refused', owner_name: '', friend_name: '' })).toMatchObject({ owner: 'Nobody', friend: 'Walk-in' })
  })
  it('colours text, never a fill', () => {
    expect(statusColor('available')).toBe('success')
    expect(statusColor('cancelled')).toBe('error')
    expect(statusColor('nope')).toBe('')
  })
})
