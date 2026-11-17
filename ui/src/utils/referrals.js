// The Referrals page's logic (api/referrals.go): the rules form, in dollars
// and days, to and from the API's cents, and how a referral reads in the
// list. Plain functions, so the page stays a view.

// programToForm is the API's rules as the form edits them.
export function programToForm (p = {}) {
  return {
    enabled: Boolean(p.enabled),
    rewardPercent: Number(p.reward_percent ?? 15),
    maxRewardDollars: centsToDollars(p.max_reward_cents),
    minFriendSpendDollars: centsToDollars(p.min_friend_spend),
    unlockAfterDays: Number(p.unlock_after_days ?? 0),
    expiresAfterDays: Number(p.expires_after_days ?? 0),
    cancelOnReturn: p.cancel_on_return ?? true,
    maxRewardsPerMonth: Number(p.max_rewards_per_month ?? 0),
  }
}

// formToRequest is the form as the API takes it; it throws with what to fix.
export function formToRequest (f) {
  const percent = wholeNumber(f.rewardPercent, 'The reward')
  if (percent < 1 || percent > 100) {throw new Error('The reward must be from 1% to 100%')}
  return {
    enabled: Boolean(f.enabled),
    reward_percent: percent,
    max_reward_cents: dollarsToCents(f.maxRewardDollars, 'The cap'),
    min_friend_spend: dollarsToCents(f.minFriendSpendDollars, 'The minimum spend'),
    unlock_after_days: wholeNumber(f.unlockAfterDays, 'Days before it can be used'),
    expires_after_days: wholeNumber(f.expiresAfterDays, 'Days it lasts'),
    cancel_on_return: Boolean(f.cancelOnReturn),
    max_rewards_per_month: wholeNumber(f.maxRewardsPerMonth, 'Rewards a month'),
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

// programSummary is one line saying what the rules do now.
export function programSummary (p) {
  if (!p?.enabled) {return 'Referrals are off. Tills don\'t take codes or print them.'}
  const parts = [`A new customer's first purchase with a client's code gives that client ${p.reward_percent}% off one later visit`]
  if (p.max_reward_cents > 0) {parts.push(`up to $${p.max_reward_cents / 100}`)}
  if (p.min_friend_spend > 0) {parts.push(`when the purchase is $${p.min_friend_spend / 100} or more`)}
  return parts.join(', ') + '.'
}

export const statusLabels = {
  available: 'Ready to use',
  used: 'Used',
  cancelled: 'Taken back',
  refused: 'No reward',
}

// statusColor is plain text colour, never a fill.
export function statusColor (status) {
  return { available: 'success', used: '', cancelled: 'error', refused: 'medium-emphasis' }[status] ?? ''
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
    statusLabel: statusLabels[r.status] ?? r.status,
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
        ? `${r.percent}% used on ${r.used_code}, and again on ${r.used_again_code}`
        : `${r.percent}% used on ${r.used_code}`
    }
    case 'available': {
      return r.max_cents > 0 ? `${r.percent}% off, up to $${r.max_cents / 100}` : `${r.percent}% off`
    }
    case 'cancelled': {
      return 'The friend\'s purchase was cancelled or returned'
    }
  }
  return ''
}
