import { describe, expect, it } from 'vitest'
import { dollarsToCents, formToRequest, listNames, programSummary, programToForm, referralRow, statusColor } from './referrals'

describe('referral rules form', () => {
  it('round-trips the API rules', () => {
    const api = {
      enabled: true, branch_ids: [3, 7], reward_percent: 15, max_reward_cents: 2550, min_friend_spend: 3000,
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
    expect(dollarsToCents('1,250.10')).toBe(125_010)
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
    expect(req).toMatchObject({ enabled: false, branch_ids: [], reward_percent: 15, unlock_after_days: 0, expires_after_days: 0, max_reward_cents: 0 })
  })

  it('says what the rules do, without long dashes or dots', () => {
    expect(programSummary({ enabled: false })).toMatch(/off at every branch/)
    const names = { 3: 'Centro', 7: 'Hara' }
    const s = programSummary({ enabled: true, branch_ids: [3, 7], reward_percent: 15, max_reward_cents: 2000, min_friend_spend: 3000 }, id => names[id])
    expect(s).toBe('On at Centro and Hara. A new customer\'s first purchase with a client\'s code gives that client 15% off one later visit, up to $20, when the purchase is $30 or more.')
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

describe('referrals per branch', () => {
  it('the reported case: on at one branch only', () => {
    const req = formToRequest({ ...programToForm({ reward_percent: 15 }), branchIds: [7] })
    expect(req).toMatchObject({ enabled: true, branch_ids: [7] })
  })
  it('no branch ticked is off, whatever the program said before', () => {
    expect(formToRequest({ ...programToForm({ enabled: true, reward_percent: 15 }), branchIds: [] })).toMatchObject({ enabled: false, branch_ids: [] })
  })
  it('sends each branch once, in order, and nothing that is not an id', () => {
    expect(formToRequest({ ...programToForm({}), branchIds: [9, 3, 9, '4', 0, -1, null] }).branch_ids).toEqual([3, 4, 9])
  })
  it('on with no branches, or off with some, reads as off', () => {
    expect(programSummary({ enabled: true, branch_ids: [] })).toMatch(/off at every branch/)
    expect(programSummary({ enabled: false, branch_ids: [3] })).toMatch(/off at every branch/)
  })
  it('names the branches as a person would', () => {
    expect(listNames([])).toBe('')
    expect(listNames(['Centro'])).toBe('Centro')
    expect(listNames(['Centro', 'Hara'])).toBe('Centro and Hara')
    expect(listNames(['Centro', 'Hara', 'Dora'])).toBe('Centro, Hara and Dora')
    expect(programSummary({ enabled: true, branch_ids: [5], reward_percent: 10 })).toContain('On at Branch #5.')
  })
})
