import { describe, expect, it } from 'vitest'
import { activityPill, pillTint, splitActivityText } from './activityPill'

const item = (id, text, extra = {}) => ({ id, text, color: 'success', icon: 'mdi-sale', ...extra })

describe('activityPill', () => {
  it('the reported case: three shift messages show as one, the newest, with +2', () => {
    const p = activityPill([
      item(1, 'Test Branch: shift balanced', { trendIcon: 'mdi-triangle' }),
      item(2, 'Test Branch: shift -5.00 USD', { color: 'error', trendIcon: 'mdi-triangle-down' }),
      item(3, 'Test Branch: shift 2.00 USD', { trendIcon: 'mdi-triangle' }),
    ])
    expect(p).toEqual({ id: 3, key: 'activity-3', branch: 'Test Branch', detail: 'shift 2.00 USD', icon: 'mdi-triangle', color: 'success', tint: 'success', more: 2 })
  })

  it('the reported case: a growing group keeps its key, so it only changes text', () => {
    const a = activityPill([item(1, 'Centro: 5 to fix', { groupKey: 'sync_conflict:1' })])
    const b = activityPill([item(2, 'Centro: 6 to fix', { groupKey: 'sync_conflict:1' })])
    expect(a.key).toBe('sync_conflict:1')
    expect(b.key).toBe(a.key)
    expect(b.detail).toBe('6 to fix')
    expect(activityPill([item(3, 'Centro: 2.00 USD')]).key).not.toBe(a.key)
  })

  it('nothing showing is null', () => {
    expect(activityPill([])).toBeNull()
    expect(activityPill(undefined)).toBeNull()
  })

  it('one message has no +N', () => {
    expect(activityPill([item(7, 'Main: 2 in', { color: 'orange' })])).toMatchObject({ more: 0, branch: 'Main', detail: '2 in', tint: 'warning', icon: 'mdi-sale' })
  })

  it('falls back to the leading icon, then a sync cloud', () => {
    expect(activityPill([item(1, 'A: b')]).icon).toBe('mdi-sale')
    expect(activityPill([{ id: 1, text: 'A: b', color: 'info' }]).icon).toBe('mdi-cloud')
  })
})

describe('splitActivityText', () => {
  it('splits on the first ": " only', () => {
    expect(splitActivityText('Centro: settlement: x')).toEqual({ branch: 'Centro', detail: 'settlement: x' })
  })
  it('no prefix is all detail', () => {
    expect(splitActivityText('hello')).toEqual({ branch: '', detail: 'hello' })
    expect(splitActivityText(': x')).toEqual({ branch: '', detail: ': x' })
    expect(splitActivityText(null)).toEqual({ branch: '', detail: '' })
  })
  it('round-trips any "branch: detail" text', () => {
    for (let n = 0; n < 300; n++) {
      const branch = `B${n}${'x'.repeat(n % 7)}`
      const detail = `d ${n}${n % 3 ? ': more' : ''}`
      expect(splitActivityText(`${branch}: ${detail}`)).toEqual({ branch, detail })
    }
  })
})

describe('pillTint', () => {
  it('keeps theme colours, anything else is warning', () => {
    for (const c of ['success', 'error', 'warning', 'info']) {expect(pillTint(c)).toBe(c)}
    for (const c of ['orange', undefined, 'purple']) {expect(pillTint(c)).toBe('warning')}
  })
})
