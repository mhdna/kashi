import { describe, expect, it } from 'vitest'
import { moveSource, nextSortOrder, sortSources, sourceNameError, totalAnswered, withShares } from './clientSources'
import fixture from './testdata/client_sources.json'

const rows = fixture.client_sources

describe('client sources', () => {
  it('the API answer: sorted, counted and shared out of 100', () => {
    const shown = withShares(rows)
    expect(shown.map(r => r.name)).toEqual(['Social media', 'SMS', 'A friend', 'Walking by', 'Other', 'Radio'])
    expect(totalAnswered(rows)).toBe(85)
    // 41/85, 12/85, 22/85, 7/85, 0, 3/85
    expect(shown.map(r => r.share)).toEqual([48, 14, 26, 8, 0, 4])
    expect(shown.reduce((s, r) => s + r.share, 0)).toBe(100)
  })

  it('nobody answered: every share is 0, none, one', () => {
    expect(withShares([])).toEqual([])
    expect(withShares(null)).toEqual([])
    expect(withShares(rows.map(r => ({ ...r, client_count: 0 }))).every(r => r.share === 0)).toBe(true)
    expect(withShares([{ id: 1, sort_order: 1, client_count: 3 }])[0].share).toBe(100)
  })

  it('shares always add up to 100 and stay within 1 of the exact value, any counts', () => {
    let seed = 11
    const rand = n => { seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648; return seed % n }
    for (let run = 0; run < 500; run++) {
      const list = Array.from({ length: 1 + rand(8) }, (_, i) => ({ id: i + 1, sort_order: rand(4), client_count: rand(3) === 0 ? 0 : rand(1000) }))
      const total = totalAnswered(list)
      const shown = withShares(list)
      expect(shown).toHaveLength(list.length)
      if (total === 0) {
        expect(shown.every(r => r.share === 0)).toBe(true)
        continue
      }
      expect(shown.reduce((s, r) => s + r.share, 0)).toBe(100)
      for (const r of shown) {
        expect(Math.abs(r.share - (r.client_count * 100) / total)).toBeLessThan(1)
      }
    }
  })

  it('sorts by order then id, without changing the input', () => {
    const input = [{ id: 3, sort_order: 1 }, { id: 1, sort_order: 2 }, { id: 2, sort_order: 1 }]
    expect(sortSources(input).map(r => r.id)).toEqual([2, 3, 1])
    expect(input.map(r => r.id)).toEqual([3, 1, 2])
  })

  it('a new option goes last', () => {
    expect(nextSortOrder(rows)).toBe(7)
    expect(nextSortOrder([])).toBe(1)
  })

  it('moving swaps neighbours and renumbers only what changed', () => {
    expect(moveSource(rows, 1, -1)).toEqual([
      { ...rows[1], sort_order: 1 },
      { ...rows[0], sort_order: 2 },
    ])
    expect(moveSource(rows, 0, -1)).toEqual([])
    expect(moveSource(rows, rows.length - 1, 1)).toEqual([])
    // Ties from older rows are settled into 1..n.
    const tied = [{ id: 1, sort_order: 0 }, { id: 2, sort_order: 0 }, { id: 3, sort_order: 0 }]
    expect(moveSource(tied, 2, -1).map(r => [r.id, r.sort_order])).toEqual([[1, 1], [3, 2], [2, 3]])
  })

  it('a name must be given and not taken by another option', () => {
    expect(sourceNameError('  ', rows)).toBe('Enter a name for the option')
    expect(sourceNameError(' sms ', rows)).toBe('There is already an option with this name')
    expect(sourceNameError('SMS', rows, 2)).toBe('')
    expect(sourceNameError('Flyer', rows)).toBe('')
  })
})
