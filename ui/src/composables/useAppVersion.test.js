import { describe, expect, it } from 'vitest'
import { freshUrl } from './useAppVersion'

describe('freshUrl', () => {
  it('adds the new build as a query so the cached page is not reused', () => {
    expect(freshUrl('https://x.test/sales?a=1', 'b2')).toBe('https://x.test/sales?a=1&_b=b2')
  })
  it('replaces an earlier build param instead of stacking them', () => {
    expect(freshUrl('https://x.test/?_b=b1', 'b2')).toBe('https://x.test/?_b=b2')
  })
  it('keeps the hash and falls back to a timestamp without a build', () => {
    const u = new URL(freshUrl('https://x.test/p#top', ''))
    expect(u.hash).toBe('#top')
    expect(u.searchParams.get('_b')).toMatch(/^\d+$/)
  })
})
