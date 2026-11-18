import { describe, expect, it } from 'vitest'
import { navItems, navPages, navSections, sectionForPath } from './navItems'

// Every page the old long drawer linked to.
const oldPages = [
  '/', '/daily-income', '/invoices', '/unknown-items', '/shifts', '/whatsapp', '/clients', '/loyalty', '/referrals', '/emails',
  '/alerts', '/stock-health', '/branch-commands', '/branch-conflicts', '/dashboard', '/targets', '/footfall', '/busy-hours',
  '/product-revenues', '/conversion', '/sales-analytics', '/sales-analysis', '/financials', '/sales-dashboard', '/employees',
  '/attendance', '/salesperson-performance', '/assets-treemap', '/warehouses-treemap', '/branches', '/branch-settings',
  '/default-branch-settings', '/price-lists', '/discount-lists', '/pricing', '/products', '/colors-sizes', '/attributes',
  '/autofill-rules', '/expenses', '/loans', '/currencies', '/inventory', '/inventory-analysis', '/batches', '/transfers',
  '/stock-counts', '/stock-movements', '/barcodes', '/assets', '/suppliers', '/purchase-invoice', '/calendar', '/reports',
  '/storage', '/logs', '/audit-log', '/app-updates', '/security', '/user-guide', '/receipt-designer', '/barcode-designer', '/settings',
]

describe('nav sections', () => {
  it('every page is still there, in exactly one section', () => {
    const all = navSections.flatMap(s => navPages(s).map(p => p.to))
    expect(new Set(all)).toEqual(new Set(oldPages))
    expect(all).toHaveLength(new Set(all).size)
  })

  it('the reported case: the designers sit together under Design', () => {
    const design = navSections.find(s => s.key === 'design')
    expect(navPages(design).map(p => p.title)).toEqual(['Receipt Designer', 'Barcode Designer'])
    expect(sectionForPath(navSections, '/receipt-designer')).toBe('design')
  })

  it('each section has a key, a title and an icon, and nothing in long dashes or dots', () => {
    const keys = navSections.map(s => s.key)
    expect(keys).toHaveLength(new Set(keys).size)
    for (const s of navSections) {
      expect(s.title && s.icon && s.items.length > 0, s.key).toBeTruthy()
      for (const p of navPages(s)) {expect(p.title).not.toMatch(/[–—·•]/)}
    }
  })

  it('a page\'s section is found from its path, deeper paths too', () => {
    expect(sectionForPath(navSections, '/')).toBe('overview')
    expect(sectionForPath(navSections, '/invoices')).toBe('sales')
    expect(sectionForPath(navSections, '/invoices/42')).toBe('sales')
    expect(sectionForPath(navSections, '/pricing')).toBe('sales')
    expect(sectionForPath(navSections, '/stock-counts/7')).toBe('stock')
    expect(sectionForPath(navSections, '/settings')).toBe('system')
    expect(sectionForPath(navSections, '/branch-settings')).toBe('branches')
    // '/' is Home only, not every page.
    expect(sectionForPath(navSections, '/nowhere')).toBeNull()
    // A path that only starts with a page's letters isn't that page.
    expect(sectionForPath(navSections, '/assetsx')).toBeNull()
  })

  it('the command palette still gets every page, grouped by section', () => {
    expect(navItems.map(i => i.title)).toEqual(navSections.map(s => s.title))
    expect(navItems.flatMap(i => i.children).length).toBe(oldPages.length)
  })
})
