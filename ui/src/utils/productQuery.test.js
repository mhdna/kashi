import { describe, expect, it } from 'vitest'
import { isFromPhoto, productQueryParams } from './productQuery'

describe('productQueryParams', () => {
  it('sends only the filters that are set', () => {
    expect(productQueryParams({})).toEqual({})
    expect(productQueryParams(null)).toEqual({})
    expect(productQueryParams({ is_active: null, has_variants: null, source: null, attribute_value_ids: [] })).toEqual({})
  })

  it('From photo sends source=label_photo, alone or with the rest', () => {
    expect(productQueryParams({ source: 'label_photo' })).toEqual({ source: 'label_photo' })
    expect(productQueryParams({
      is_active: false, has_variants: true, source: 'label_photo',
      created_from: '2026-11-01', created_to: '2026-11-16', attribute_value_ids: [3, 4],
    })).toEqual({
      is_active: 'false', has_variants: 'true', source: 'label_photo',
      created_from: '2026-11-01', created_to: '2026-11-16', attribute_value_ids: [3, 4],
    })
  })
})

describe('isFromPhoto', () => {
  it('only the label reader source is tagged', () => {
    expect(isFromPhoto({ source: 'label_photo' })).toBe(true)
    expect(isFromPhoto({ source: '' })).toBe(false)
    expect(isFromPhoto({})).toBe(false)
    expect(isFromPhoto(undefined)).toBe(false)
  })
})
