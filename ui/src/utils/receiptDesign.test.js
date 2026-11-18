import { describe, expect, it } from 'vitest'
import {
  alignOptions, boldState, can, caseOptions, designJSON, elementsOf, emptyDesign, groupInfo, isGroupChanged,
  isSectionHidden, markAt, parseDesign, previewQuery, receiptPreviewURL, resetGroup, setBold, setElement,
  sizeStep, toggleSection,
} from './receiptDesign'
// What kashi's API serves (TestReceiptDesignerCatalog writes it), so a part
// or a line the till adds is checked here too.
import catalog from './testdata/receipt_designer_catalog.json'

describe('the catalog from the till', () => {
  it('has every part with lines, each line able to change something', () => {
    expect(catalog.groups.map(g => g.key)).toEqual(['header', 'meta', 'columns', 'totals', 'payments', 'policy', 'referral', 'socials', 'thanks'])
    for (const g of catalog.groups) {expect(elementsOf(catalog, g.key).length, g.key).toBeGreaterThan(0)}
    for (const e of catalog.elements) {
      const any = ['text', 'case', 'size', 'bold', 'align', 'hide'].some(w => can(catalog, e, w))
      expect(any, e.key).toBe(true)
    }
  })
  it('the reported case: the things asked for can be changed', () => {
    const byKey = Object.fromEntries(catalog.elements.map(e => [e.key, e]))
    expect(can(catalog, byKey.col_item, 'text')).toBe(true)
    expect(can(catalog, byKey.col_item, 'case')).toBe(true)
    expect(can(catalog, byKey.branch_name, 'size')).toBe(true)
    expect(can(catalog, byKey.policy_text, 'text')).toBe(true)
    expect(byKey.policy_text.default).toBe('Exchanges accepted within 3 days.')
    expect(can(catalog, byKey.thanks, 'align')).toBe(true)
    expect(can(catalog, byKey.socials_title, 'text')).toBe(true)
    expect(catalog.date_formats.length).toBeGreaterThan(4)
  })
  it('names things without long dashes or dots', () => {
    const said = [...catalog.groups.map(g => g.label), ...catalog.elements.map(e => e.label),
      ...caseOptions.map(o => o.title), ...alignOptions.map(o => o.title)].join(' ')
    expect(said).not.toMatch(/[–—·•]/)
  })
})

describe('editing a design', () => {
  it('sends nothing for every default', () => {
    expect(designJSON(emptyDesign())).toBe('')
  })
  it('sets one setting of one line and keeps the rest', () => {
    let d = setElement(emptyDesign(), 'col_item', { text: 'Article' })
    d = setElement(d, 'col_item', { case: 'title' })
    d = setElement(d, 'thanks', { align: 'right' })
    expect(JSON.parse(designJSON(d))).toEqual({ elements: { col_item: { text: 'Article', case: 'title' }, thanks: { align: 'right' } } })
  })
  it('drops a line put back to nothing', () => {
    const d = setElement(setElement(emptyDesign(), 'thanks', { text: 'Bye' }), 'thanks', { text: '' })
    expect(d.elements.thanks).toBeUndefined()
    expect(designJSON(d)).toBe('')
  })
  it('steps size within the range', () => {
    let d = emptyDesign()
    for (let i = 0; i < 30; i++) {d = sizeStep(d, catalog, 'store_name', 1)}
    expect(d.elements.store_name.size).toBe(catalog.size_max)
    for (let i = 0; i < 60; i++) {d = sizeStep(d, catalog, 'store_name', -1)}
    expect(d.elements.store_name.size).toBe(catalog.size_min)
    d = sizeStep(sizeStep(emptyDesign(), catalog, 'store_name', 1), catalog, 'store_name', -1)
    expect(d.elements.store_name).toBeUndefined()
  })
  it('bold has three states: the look\'s own, bold, regular', () => {
    let d = setBold(emptyDesign(), 'thanks', 'regular')
    expect(boldState(d, 'thanks')).toBe('regular')
    expect(JSON.parse(designJSON(d)).elements.thanks).toEqual({ bold: false })
    d = setBold(d, 'thanks', 'bold')
    expect(boldState(d, 'thanks')).toBe('bold')
    d = setBold(d, 'thanks', 'look')
    expect(boldState(d, 'thanks')).toBe('look')
    expect(designJSON(d)).toBe('')
  })
  it('leaves a part out and puts it back, and resets a part whole', () => {
    let d = toggleSection(emptyDesign(), 'payments')
    expect(isSectionHidden(d, 'payments')).toBe(true)
    d = setElement(d, 'payments_title', { text: 'Paid with' })
    expect(isGroupChanged(d, catalog, 'payments')).toBe(true)
    expect(isGroupChanged(d, catalog, 'totals')).toBe(false)
    d = setElement(d, 'subtotal', { text: 'Before' })
    d = resetGroup(d, catalog, 'payments')
    expect(isSectionHidden(d, 'payments')).toBe(false)
    expect(d.elements.payments_title).toBeUndefined()
    expect(d.elements.subtotal).toEqual({ text: 'Before' })
    expect(groupInfo(catalog, 'payments').section).toBe('payments')
  })
  it('the date format belongs to the customer, salesperson and date part', () => {
    let d = { ...emptyDesign(), date_format: 'iso' }
    expect(isGroupChanged(d, catalog, 'meta')).toBe(true)
    d = resetGroup(d, catalog, 'meta')
    expect(d.date_format).toBe('')
  })
})

describe('reading a saved design', () => {
  it('reads nothing or junk as every default', () => {
    for (const raw of ['', ' ', null, undefined, '{oops', '[]', '"x"', 'null']) {expect(parseDesign(raw)).toEqual(emptyDesign())}
  })
  it('keeps the wording saved before lines had their own settings', () => {
    const d = parseDesign('{"payments_title":"PAID WITH","thanks":"See you","hide":["policy"]}')
    expect(d.elements.payments_title.text).toBe('PAID WITH')
    expect(d.elements.thanks.text).toBe('See you')
    expect(d.hide).toEqual(['policy'])
  })
  it('a line\'s own words come before the old field', () => {
    const d = parseDesign('{"thanks":"Old","elements":{"thanks":{"text":"New","case":"upper"}}}')
    expect(d.elements.thanks).toEqual({ text: 'New', case: 'upper' })
  })
  it('randomized: what is saved reads back the same', () => {
    const pick = arr => arr[Math.floor(Math.random() * arr.length)]
    for (let run = 0; run < 400; run++) {
      let d = emptyDesign()
      for (const e of catalog.elements) {
        if (Math.random() < 0.6) {continue}
        if (can(catalog, e, 'text')) {d = setElement(d, e.key, { text: pick(['', 'Hi', 'شكراً', 'x'.repeat(40)]) })}
        if (can(catalog, e, 'case')) {d = setElement(d, e.key, { case: pick(caseOptions).value })}
        if (can(catalog, e, 'size')) {d = sizeStep(d, catalog, e.key, pick([-2, -1, 0, 1, 3]))}
        if (can(catalog, e, 'bold')) {d = setBold(d, e.key, pick(['look', 'bold', 'regular']))}
        if (can(catalog, e, 'align')) {d = setElement(d, e.key, { align: pick(alignOptions).value })}
        if (can(catalog, e, 'hide')) {d = setElement(d, e.key, { hide: Math.random() < 0.3 })}
      }
      for (const g of catalog.groups) {if (g.section && Math.random() < 0.2) {d = toggleSection(d, g.section)}}
      if (Math.random() < 0.3) {d = { ...d, date_format: pick(catalog.date_formats).key }}
      const once = designJSON(d)
      expect(designJSON(parseDesign(once))).toBe(once)
    }
  })
})

describe('the preview request and the paper', () => {
  it('asks for the look, mode, branch and the design', () => {
    const q = new URLSearchParams(previewQuery({ theme: 'modern', layout: 'B', mode: 'exchange', branchId: 7, design: setElement(emptyDesign(), 'thanks', { text: 'Bye' }) }))
    expect(Object.fromEntries(q)).toEqual({ theme: 'modern', layout: 'B', mode: 'exchange', branch_id: '7', texts: '{"elements":{"thanks":{"text":"Bye"}}}' })
    expect(previewQuery({ mode: 'sale', design: emptyDesign() })).toBe('mode=sale')
  })
  it('finds the part under a point, the smallest when parts nest', () => {
    const marks = [{ group: 'header', top: 0, bottom: 100 }, { group: 'totals', top: 300, bottom: 420 }, { group: 'payments', top: 330, bottom: 380 }]
    expect(markAt(marks, 50).group).toBe('header')
    expect(markAt(marks, 350).group).toBe('payments')
    expect(markAt(marks, 300).group).toBe('totals')
    expect(markAt(marks, 420)).toBeNull()
    expect(markAt(marks, 200)).toBeNull()
  })
  it('asks for a thumbnail by its version', () => {
    expect(receiptPreviewURL({ file: 'classic-A.png', version: 'abc' })).toBe('/receipt-previews/classic-A.png?v=abc')
    expect(receiptPreviewURL({ file: 'a.png' })).toBe('/receipt-previews/a.png')
  })
})
