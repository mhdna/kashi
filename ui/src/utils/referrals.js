// The Referrals page's logic (api/referrals.go): the rules form, in dollars
// and days, to and from the API's cents, and how a referral reads in the
// list. Plain functions, so the page stays a view.
import { t } from '../i18n'

// programToForm is the API's rules as the form edits them.
export function programToForm (p = {}) {
  return {
    enabled: Boolean(p.enabled),
    // The branches where referrals are on; none is off everywhere.
    branchIds: Array.isArray(p.branch_ids) ? [...p.branch_ids] : [],
    rewardPercent: Number(p.reward_percent ?? 15),
    maxRewardDollars: centsToDollars(p.max_reward_cents),
    minFriendSpendDollars: centsToDollars(p.min_friend_spend),
    unlockAfterDays: Number(p.unlock_after_days ?? 0),
    expiresAfterDays: Number(p.expires_after_days ?? 0),
    cancelOnReturn: p.cancel_on_return ?? true,
    maxRewardsPerMonth: Number(p.max_rewards_per_month ?? 0),
    // Who a code rewards: the client who shared it, the new customer, or both.
    rewardTo: p.reward_to || 'owner',
    friendPercent: Number(p.friend_percent || 10),
  }
}

// rewardToOptions are who a referral can reward. A function, so the labels
// follow the EN/FR toggle.
export function rewardToOptions () {
  return [
    { value: 'owner', title: t('The client who shared the code') },
    { value: 'friend', title: t('The new customer') },
    { value: 'both', title: t('Both') },
  ]
}

export function rewardsOwner (to) { return to !== 'friend' }
export function rewardsFriend (to) { return to === 'friend' || to === 'both' }

// formToRequest is the form as the API takes it; it throws with what to fix.
export function formToRequest (f) {
  const rewardTo = ['owner', 'friend', 'both'].includes(f.rewardTo) ? f.rewardTo : 'owner'
  // The owner's percent is only asked for when the owner is rewarded.
  const percent = rewardsOwner(rewardTo) ? wholeNumber(f.rewardPercent, 'The reward') : Number(f.rewardPercent) || 15
  if (percent < 1 || percent > 100) {throw new Error('The reward must be from 1% to 100%')}
  let friendPercent = 0
  if (rewardsFriend(rewardTo)) {
    friendPercent = wholeNumber(f.friendPercent, "The new customer's percent")
    if (friendPercent < 1 || friendPercent > 100) {throw new Error("The new customer's percent must be from 1% to 100%")}
  }
  const branchIds = [...new Set((f.branchIds ?? []).map(Number).filter(id => Number.isInteger(id) && id > 0))].toSorted((a, b) => a - b)
  return {
    enabled: branchIds.length > 0,
    branch_ids: branchIds,
    reward_percent: percent,
    max_reward_cents: dollarsToCents(f.maxRewardDollars, 'The cap'),
    min_friend_spend: dollarsToCents(f.minFriendSpendDollars, 'The minimum spend'),
    unlock_after_days: wholeNumber(f.unlockAfterDays, 'Days before it can be used'),
    expires_after_days: wholeNumber(f.expiresAfterDays, 'Days it lasts'),
    cancel_on_return: Boolean(f.cancelOnReturn),
    max_rewards_per_month: wholeNumber(f.maxRewardsPerMonth, 'Rewards a month'),
    reward_to: rewardTo,
    friend_percent: friendPercent,
  }
}

function centsToDollars (cents) {
  const n = Number(cents ?? 0)
  return n ? String(n / 100) : ''
}

// dollarsToCents reads "12.5", "" (none) or "$30" as cents, rounding to the
// nearest cent.
export function dollarsToCents (value, label = 'The amount') {
  const text = String(value ?? '').trim().replace(/^\$/, '').replace(/,/g, '')
  if (text === '') {return 0}
  if (!/^\d+(\.\d+)?$/.test(text)) {throw new Error(`${label} must be an amount in dollars`)}
  return Math.round(Number(text) * 100)
}

function wholeNumber (value, label) {
  const text = String(value ?? '').trim()
  if (text === '') {return 0}
  if (!/^\d+$/.test(text)) {throw new Error(`${label} must be a whole number`)}
  return Number(text)
}

// listNames reads ["Centro", "Hara", "Dora"] as "Centro, Hara and Dora".
export function listNames (names) {
  if (names.length <= 1) {return names.join('')}
  return t('{head} and {last}', { head: names.slice(0, -1).join(', '), last: names.at(-1) })
}

// programSummary is one line saying what the rules do now, and where.
// branchName turns a branch id into its name.
export function programSummary (p, branchName = id => t('Branch #{id}', { id })) {
  const ids = p?.enabled ? (p.branch_ids ?? []) : []
  if (ids.length === 0) {return t('Referrals are off at every branch. Tills don\'t take codes or print them.')}
  const to = p.reward_to || 'owner'
  const gives = []
  if (rewardsFriend(to)) {gives.push(t('the new customer {percent}% off that purchase', { percent: p.friend_percent }))}
  if (rewardsOwner(to)) {gives.push(t('the client who shared it {percent}% off one later visit', { percent: p.reward_percent }))}
  // The owner only (as it always was) keeps its own sentence.
  const parts = to === 'owner'
    ? [t('A new customer\'s first purchase with a client\'s code gives that client {percent}% off one later visit', { percent: p.reward_percent })]
    : [t('A new customer\'s first purchase with a client\'s code gives {gives}', { gives: gives.join(t(', and ')) })]
  if (p.max_reward_cents > 0) {parts.push(t('up to ${amount}', { amount: p.max_reward_cents / 100 }))}
  if (p.min_friend_spend > 0) {parts.push(t('when the purchase is ${amount} or more', { amount: p.min_friend_spend / 100 }))}
  return t('On at {branches}. {details}.', { branches: listNames(ids.map(id => branchName(id))), details: parts.join(', ') })
}

// statusLabels are the words for a referral's status. A function, so the
// labels follow the EN/FR toggle.
export function statusLabels () {
  return {
    available: t('Ready to use'),
    used: t('Used'),
    cancelled: t('Taken back'),
    refused: t('No reward'),
    friend: t('New customer rewarded'),
  }
}

// statusColor is plain text colour, never a fill.
export function statusColor (status) {
  return { available: 'success', used: '', cancelled: 'error', refused: 'medium-emphasis', friend: 'success' }[status] ?? ''
}

// referralRow is a referral as the list shows it.
export function referralRow (r) {
  return {
    id: r.id,
    when: r.created_at,
    owner: r.owner_name || (r.owner_phone ? r.owner_phone : 'Nobody'),
    ownerPhone: r.owner_phone,
    friend: r.friend_name || r.friend_phone || 'Walk-in',
    friendPhone: r.friend_phone,
    code: r.code,
    branch: r.branch_name,
    sale: r.sale_code,
    saleTotal: Number(r.sale_total ?? 0),
    status: r.status,
    statusLabel: statusLabels()[r.status] ?? r.status,
    detail: referralDetail(r),
  }
}

function referralDetail (r) {
  switch (r.status) {
    case 'refused': {
      return r.refuse_reason
    }
    case 'used': {
      return r.used_again_code
        ? t('{percent}% used on {code}, and again on {again}', { percent: r.percent, code: r.used_code, again: r.used_again_code })
        : t('{percent}% used on {code}', { percent: r.percent, code: r.used_code })
    }
    case 'available': {
      return r.max_cents > 0
        ? t('{percent}% off, up to ${amount}', { percent: r.percent, amount: r.max_cents / 100 })
        : t('{percent}% off', { percent: r.percent })
    }
    case 'cancelled': {
      return t('The friend\'s purchase was cancelled or returned')
    }
  }
  return ''
}
