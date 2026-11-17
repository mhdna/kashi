import { describe, expect, it } from 'vitest'
import { clientTerm, hasInvoiceFilters, invoiceFilterParams } from './invoiceFilters'

describe('clientTerm', () => {
  it('trims, and gives null for nothing', () => {
    expect(clientTerm('  Maya ')).toBe('Maya')
    expect(clientTerm('')).toBeNull()
    expect(clientTerm('   ')).toBeNull()
    expect(clientTerm(null)).toBeNull()
    expect(clientTerm(undefined)).toBeNull()
  })
  it('keeps a phone as typed, spaces inside and all', () => {
    expect(clientTerm(' +961 70 123 456 ')).toBe('+961 70 123 456')
  })
})

describe('invoiceFilterParams', () => {
  it('has no client when the box is empty, and the others as they were', () => {
    expect(invoiceFilterParams({ branchId: 2, registerId: 3, kind: 'sales', status: 'active', dateFrom: '2026-11-01', dateTo: '2026-11-17', client: '' }))
      .toEqual({ branch_id: 2, register_id: 3, kind: 'sales', status: 'active', date_from: '2026-11-01', date_to: '2026-11-17', client: null })
  })
  it('sends the trimmed client next to the other filters', () => {
    expect(invoiceFilterParams({ client: ' Maya ', kind: 'return' }))
      .toMatchObject({ client: 'Maya', kind: 'return', branch_id: null, register_id: null })
  })
})

describe('hasInvoiceFilters', () => {
  it('is false for nothing, or a blank client box', () => {
    expect(hasInvoiceFilters({})).toBe(false)
    expect(hasInvoiceFilters({ client: '  ' })).toBe(false)
  })
  it('is true when the client box alone is filled, or any other filter', () => {
    expect(hasInvoiceFilters({ client: 'Maya' })).toBe(true)
    for (const key of ['registerId', 'kind', 'status', 'dateFrom', 'dateTo']) {
      expect(hasInvoiceFilters({ [key]: key === 'registerId' ? 5 : 'x' })).toBe(true)
    }
  })
})
