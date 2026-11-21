import { describe, expect, it } from 'vitest'
import { amountCents, clientTerm, countInvoiceFilters, emptyFilters, hasInvoiceFilters, invoiceFilterError, invoiceFilterParams, invoiceFilterSummary } from './invoiceFilters'

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
      .toMatchObject({ branch_id: 2, register_id: 3, kind: 'sales', status: 'active', date_from: '2026-11-01', date_to: '2026-11-17', client: null })
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

describe('product and total filters', () => {
  it('the reported case: product codes starting with SH12', () => {
    expect(invoiceFilterParams({ product: ' SH12 ', productField: 'code', productMatch: 'starts' }))
      .toMatchObject({ product: 'SH12', product_field: 'code', product_match: 'starts' })
  })
  it('sends no field or mode for the defaults, and none of them without a term', () => {
    expect(invoiceFilterParams({ product: 'shirt', productField: 'any', productMatch: 'contains' }))
      .toMatchObject({ product: 'shirt', product_field: null, product_match: null })
    expect(invoiceFilterParams({ product: '   ', productField: 'name', productMatch: 'exact' }))
      .toMatchObject({ product: null, product_field: null, product_match: null })
  })
  it('turns typed dollars into cents, negatives and blanks included', () => {
    expect(amountCents('12.5')).toBe(1250)
    expect(amountCents(' -15 ')).toBe(-1500)
    expect(amountCents('0')).toBe(0)
    expect(amountCents('0.1')).toBe(10)
    expect(amountCents('19.99')).toBe(1999)
    expect(amountCents('')).toBeNull()
    expect(amountCents(null)).toBeNull()
    expect(amountCents('ten')).toBeNull()
    expect(invoiceFilterParams({ totalMin: '0', totalMax: '' })).toMatchObject({ total_min: 0, total_max: null })
  })
  it('one cent off anywhere is caught: every cent from -50.00 to 50.00 round-trips', () => {
    for (let cents = -5000; cents <= 5000; cents++) {
      const typed = (cents / 100).toFixed(2)
      expect(amountCents(typed), typed).toBe(cents)
    }
  })
})

describe('countInvoiceFilters', () => {
  it('is 0 for the empty dialog, and counts each filter once', () => {
    expect(countInvoiceFilters(emptyFilters())).toBe(0)
    expect(countInvoiceFilters({ ...emptyFilters(), product: 'SH', productField: 'code', productMatch: 'starts' })).toBe(1)
    expect(countInvoiceFilters({ ...emptyFilters(), branchId: 2, kind: 'sales', totalMin: '0', dateFrom: '2026-11-01' })).toBe(4)
    expect(hasInvoiceFilters({ ...emptyFilters(), totalMax: '0' })).toBe(true, 'a zero total is a filter')
  })
  it('randomized: the count is the number of parameters the API gets', () => {
    const keys = { branchId: 3, registerId: 4, kind: 'return', status: 'active', dateFrom: '2026-10-01', dateTo: '2026-10-31', client: 'Maya', salesperson: 'Rana', product: 'SH', totalMin: '-5', totalMax: '20' }
    for (let run = 0; run < 500; run++) {
      const filters = emptyFilters()
      let want = 0
      for (const [key, value] of Object.entries(keys)) {
        if (Math.random() < 0.5) { filters[key] = value; want++ }
      }
      expect(countInvoiceFilters(filters)).toBe(want)
      const sent = Object.entries(invoiceFilterParams(filters)).filter(([, v]) => v !== null && v !== '').map(([k]) => k)
      expect(sent.filter(k => k !== 'product_field' && k !== 'product_match').length).toBe(want)
    }
  })
})

describe('invoiceFilterSummary', () => {
  it('says each filter in words, with names for ids', () => {
    expect(invoiceFilterSummary({
      ...emptyFilters(), branchId: 2, registerId: 9, kind: 'return', product: 'SH12', productField: 'code', productMatch: 'starts',
      totalMin: '10', totalMax: '20', dateFrom: '2026-11-01',
    }, { branch: () => 'Centro', register: () => 'Centro - R1' })).toEqual([
      'Centro', 'Centro - R1', 'Returns', 'From 2026-11-01', 'Product code starts with "SH12"', 'Total 10.00 to 20.00',
    ])
  })
  it('has nothing to say for no filters, and no long dashes or dots ever', () => {
    expect(invoiceFilterSummary(emptyFilters())).toEqual([])
    const all = invoiceFilterSummary({ ...emptyFilters(), status: 'cancelled', dateTo: '2026-11-02', client: 'Omar', product: 'x', totalMax: '-1' })
    expect(all).toEqual(['Cancelled', 'Until 2026-11-02', 'Client "Omar"', 'Code, name or barcode contains "x"', 'Total at most -1.00'])
    expect(all.join(' ')).not.toMatch(/[–—·•]/)
  })
})

describe('invoiceFilterError', () => {
  it('stops a range that is backwards or not a number', () => {
    expect(invoiceFilterError(emptyFilters())).toBe('')
    expect(invoiceFilterError({ totalMin: '20', totalMax: '10' })).toMatch(/above/)
    expect(invoiceFilterError({ totalMin: '10', totalMax: '10' })).toBe('')
    expect(invoiceFilterError({ totalMin: 'abc' })).toMatch(/not a number/)
    expect(invoiceFilterError({ totalMax: '1..2' })).toMatch(/not a number/)
    expect(invoiceFilterError({ dateFrom: '2026-11-02', dateTo: '2026-11-01' })).toMatch(/after/)
  })
})

describe('salesperson filter', () => {
  it('is sent trimmed, wildcards untouched, and nothing when blank', () => {
    expect(invoiceFilterParams({ salesperson: ' Ra* ' }).salesperson).toBe('Ra*')
    expect(invoiceFilterParams({ salesperson: '  ' }).salesperson).toBeNull()
    expect(invoiceFilterParams({}).salesperson).toBeNull()
  })
  it('counts once, shows Clear, and reads in the summary', () => {
    expect(countInvoiceFilters({ ...emptyFilters(), salesperson: 'Rana' })).toBe(1)
    expect(hasInvoiceFilters({ ...emptyFilters(), salesperson: 'Rana' })).toBe(true)
    expect(hasInvoiceFilters({ ...emptyFilters(), salesperson: ' ' })).toBe(false)
    expect(invoiceFilterSummary({ ...emptyFilters(), salesperson: 'Ra*', client: 'Maya' })).toEqual(['Client "Maya"', 'Salesperson "Ra*"'])
  })
})
