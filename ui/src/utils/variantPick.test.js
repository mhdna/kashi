import { describe, expect, it } from 'vitest'
import { parseEntries, savedMessage, VARIANT_PICK_MODES } from './variantPick'

describe('parseEntries', () => {
  it('splits pasted lists on spaces, newlines, commas and semicolons', () => {
    expect(parseEntries('A1, B2;C3\nD4\tE5  F6')).toEqual(['A1', 'B2', 'C3', 'D4', 'E5', 'F6'])
  })
  it('drops blanks and repeats in any case, keeping the first spelling and the order', () => {
    expect(parseEntries(' a1 ,, A1\n\nb2 , B2 ,c3')).toEqual(['a1', 'b2', 'c3'])
  })
  it('is empty for nothing', () => {
    for (const v of ['', '  \n ', ',;', null, undefined]) expect(parseEntries(v)).toEqual([])
  })
  it('keeps codes with dashes and slashes whole', () => {
    expect(parseEntries('T110-80026-02 DR1880-PINK-140')).toEqual(['T110-80026-02', 'DR1880-PINK-140'])
  })
})

describe('savedMessage', () => {
  it('says what the till will do, for one and for many', () => {
    expect(savedMessage('always', 1)).toBe('1 item will ask for the colour/size at the till.')
    expect(savedMessage('always', 18)).toBe('18 items will ask for the colour/size at the till.')
    expect(savedMessage('never', 2)).toBe('2 items will never ask for the colour/size.')
    expect(savedMessage('', 3)).toBe('3 items will follow the branch setting.')
  })
  it('says so when nothing matched, whatever the mode', () => {
    for (const m of ['always', 'never', '']) expect(savedMessage(m, 0)).toBe('No item matched those codes.')
  })
})

describe('modes', () => {
  it('are exactly the values the server accepts', () => {
    expect(VARIANT_PICK_MODES.map(m => m.value).sort()).toEqual(['', 'always', 'never'])
  })
})
