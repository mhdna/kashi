import { describe, expect, it } from 'vitest'
import {
  addBanner, alignOptions, bannerMax, bannerMode, bannerWhenOptions, boldState, can, caseOptions, dealModes, designJSON, elementsOf, emptyDesign, footerOrder, groupBoldState, groupInfo,
  groupSizeStep, isElementChanged, isGroupChanged, isPartPrinted, isSectionHidden, lineStyleOptions, markAt, moveBanner, moveSection,
  parseDesign, partsOf, previewQuery, receiptPreviewURL, removeBanner, resetElement, resetGroup, setBold, setDecor,
  setElement, setGroup, setGroupBold, setPartPrinted, shopFromSettings, shopPayload, shopPreview, sizeStep,
  toggleSection, updateBanner,
} from './receiptDesign'
// What kashi's API serves (TestReceiptDesignerCatalog writes it), so a part
// or a line the till adds is checked here too.
import catalog from './testdata/receipt_designer_catalog.json'

describe('the catalog from the till', () => {
  it('has every part with lines, each line able to change something', () => {
    expect(catalog.groups.map(g => g.key)).toEqual(['header', 'meta', 'columns', 'totals', 'payments', 'policy', 'referral', 'socials', 'thanks'])
    for (const g of catalog.groups) {expect(elementsOf(catalog, g.key).length, g.key).toBeGreaterThan(0)}
    for (const e of catalog.elements) {
      const any = ['text', 'case', 'size', 'bold', 'align', 'hide', 'show'].some(w => can(catalog, e, w))
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
  it('bold is bold, regular, or nothing picked for the look\'s own', () => {
    let d = setBold(emptyDesign(), 'thanks', 'regular')
    expect(boldState(d, 'thanks')).toBe('regular')
    expect(JSON.parse(designJSON(d)).elements.thanks).toEqual({ bold: false })
    d = setBold(d, 'thanks', 'bold')
    expect(boldState(d, 'thanks')).toBe('bold')
    d = setBold(d, 'thanks', undefined)
    expect(boldState(d, 'thanks')).toBeUndefined()
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

describe('a whole part, its lines and box, and its place', () => {
  it('the reported case: a part centred and regular at once, and back to default', () => {
    let d = setGroup(emptyDesign(), 'payments', { align: 'center' })
    d = setGroupBold(d, 'payments', 'regular')
    d = groupSizeStep(d, catalog, 'payments', 2)
    expect(JSON.parse(designJSON(d))).toEqual({ groups: { payments: { align: 'center', bold: false, size: 2 } } })
    expect(groupBoldState(d, 'payments')).toBe('regular')
    expect(isGroupChanged(d, catalog, 'payments')).toBe(true)
    expect(designJSON(resetGroup(d, catalog, 'payments'))).toBe('')
  })
  it('a line resets on its own, its part untouched', () => {
    let d = setElement(emptyDesign(), 'received', { bold: true, size: 2 })
    d = setGroup(d, 'payments', { align: 'right' })
    expect(isElementChanged(d, 'received')).toBe(true)
    d = resetElement(d, 'received')
    expect(isElementChanged(d, 'received')).toBe(false)
    expect(d.groups.payments).toEqual({ align: 'right' })
  })
  it('lines and a box: kept only with something set', () => {
    let d = setDecor(emptyDesign(), 'policy', { box: 'dashed', rounded: true, thickness: 3 })
    d = setDecor(d, 'thanks', { above: 'double' })
    d = setDecor(d, 'payments', { below: 'none' })
    expect(JSON.parse(designJSON(d)).decor).toEqual({ policy: { box: 'dashed', rounded: true, thickness: 3 }, thanks: { above: 'double' }, payments: { below: 'none' } })
    d = setDecor(d, 'thanks', { above: '' })
    expect(JSON.parse(designJSON(d)).decor.thanks).toBeUndefined()
    // Round corners without a box mean nothing.
    expect(JSON.parse(designJSON(setDecor(emptyDesign(), 'x', { rounded: true })) || '{}').decor?.x?.rounded).toBeUndefined()
    expect(lineStyleOptions.map(o => o.value)).toEqual(catalog.line_styles)
  })
  it('moves a part up and down, and back to the default order sends nothing', () => {
    expect(footerOrder(emptyDesign(), catalog)).toEqual(catalog.footer_order)
    let d = moveSection(emptyDesign(), catalog, 'thanks', -1)
    expect(footerOrder(d, catalog)).toEqual(['payments', 'policy', 'referral', 'thanks', 'socials'])
    expect(JSON.parse(designJSON(d)).order).toEqual(['payments', 'policy', 'referral', 'thanks', 'socials'])
    d = moveSection(d, catalog, 'thanks', 1)
    expect(designJSON(d)).toBe('')
    expect(moveSection(emptyDesign(), catalog, 'payments', -1)).toEqual(emptyDesign())
    expect(moveSection(emptyDesign(), catalog, 'thanks', 1)).toEqual(emptyDesign())
  })
  it('randomized: any moves keep every part once', () => {
    let d = emptyDesign()
    for (let i = 0; i < 300; i++) {
      const s = catalog.footer_order[Math.floor(Math.random() * catalog.footer_order.length)]
      d = moveSection(d, catalog, s, Math.random() < 0.5 ? -1 : 1)
      expect(footerOrder(d, catalog).toSorted()).toEqual(catalog.footer_order.toSorted())
      expect(designJSON(parseDesign(designJSON(d)))).toBe(designJSON(d))
    }
  })
})

describe('shop details', () => {
  it('reads a branch\'s settings and saves them back, the phone only for one branch', () => {
    const shop = shopFromSettings({
      market_name: 'Bazaar', branch_phone: '03 111', website: 'bz.com', return_policy: 'Swaps in 7 days',
      social_platforms: ['instagram', 'myspace', 'tiktok'], social_handles: { instagram: '@bz', tiktok: ' ' },
    })
    expect(shop.social_platforms).toEqual(['instagram', 'tiktok'])
    expect(shopPayload(shop)).toEqual({
      market_name: 'Bazaar', branch_phone: '03 111', website: 'bz.com', return_policy: 'Swaps in 7 days',
      social_platforms: ['instagram'], social_handles: { instagram: '@bz' },
    })
    expect(shopPayload(shop, { allBranches: true })).not.toHaveProperty('branch_phone')
    expect(JSON.parse(shopPreview(shop))).toMatchObject({ store_name: 'Bazaar', phone: '03 111', social_handles: { instagram: '@bz' } })
    expect(shopFromSettings(null)).toEqual(shopFromSettings({}))
  })
})

describe('deal modes', () => {
  it('the reported case: one view for each deal in use, more as they are added', () => {
    const modes = dealModes([
      { id: 1, name: 'Students 15%', kind: 'percent', percent: 15, amount: 0, is_active: true },
      { id: 2, name: '15% above $100', kind: 'percent', percent: 15, amount: 0, min_total: 10_000, is_active: true },
      { id: 3, name: 'Off', kind: 'amount', amount: 500, is_active: false },
      { id: 4, name: 'Gone', kind: 'amount', amount: 500, is_active: true, deleted_at: { Valid: true } },
    ])
    expect(modes.map(m => m.label)).toEqual(['Students 15%', '15% above $100'])
    expect(modes[0]).toEqual({ mode: 'deal:1', label: 'Students 15%', deal: { name: 'Students 15%', kind: 'percent', percent: 15, amount: 0 } })
    expect(dealModes(null)).toEqual([])
  })
  it('a deal goes with the preview', () => {
    const q = new URLSearchParams(previewQuery({ mode: 'client', design: emptyDesign(), deal: { name: 'Staff', kind: 'amount', amount: 500 } }))
    expect(JSON.parse(q.get('deal'))).toEqual({ name: 'Staff', kind: 'amount', percent: 0, amount: 500 })
  })
})

describe('banners', () => {
  it('the reported case: add a ready-made banner, change its words, and it is sent', () => {
    let d = addBanner(emptyDesign(), catalog.banner_presets[0])
    expect(d.banners).toHaveLength(1)
    d = updateBanner(d, 0, { title: '  BUY ONE GET ONE ', subtitle: '' })
    expect(JSON.parse(designJSON(d)).banners).toEqual([{ style: 'ribbon', title: 'BUY ONE GET ONE', when: 'item_deal' }])
    expect(designJSON(parseDesign(designJSON(d)))).toBe(designJSON(d))
  })
  it('takes up to twelve, moves and removes them', () => {
    let d = emptyDesign()
    for (const [i, p] of [...catalog.banner_presets, ...catalog.banner_presets].entries()) {d = addBanner(d, { ...p, title: `${p.title} ${i}` })}
    expect(d.banners).toHaveLength(bannerMax)
    const first = d.banners[0].title
    d = moveBanner(d, 0, 1)
    expect(d.banners[1].title).toBe(first)
    expect(moveBanner(d, 0, -1)).toBe(d)
    d = removeBanner(d, 1)
    expect(d.banners.map(b => b.title)).not.toContain(first)
  })
  it('a banner with no words isn\'t sent', () => {
    expect(designJSON(updateBanner(addBanner(emptyDesign(), catalog.banner_presets[3]), 0, { title: '  ' }))).toBe('')
  })
  it('prints always, on buy X get Y, with any deal, or with one of kashi\'s deals', () => {
    const deals = [{ id: 1, name: 'Students 15%', kind: 'percent', percent: 15, is_active: true }]
    expect(bannerWhenOptions(deals).map(o => o.value)).toEqual(['', 'item_deal', 'deal', 'deal:Students 15%'])
    expect(bannerMode({ when: 'item_deal' }, deals)).toBe('item_deal')
    expect(bannerMode({ when: 'deal:Students 15%' }, dealModes(deals).length > 0 ? deals : [])).toBe('deal:1')
    expect(bannerMode({ when: 'deal' }, deals)).toBe('deal:1')
    expect(bannerMode({}, deals)).toBe('')
  })
})

describe('fonts in the design', () => {
  it('a line\'s, a part\'s and a banner\'s font are sent, and come back', () => {
    let d = setElement(emptyDesign(), 'store_name', { font: 'pacifico' })
    d = setGroup(d, 'totals', { font: 'inter' })
    d = updateBanner(addBanner(d, catalog.banner_presets[0]), 0, { font: 'lobster' })
    const sent = JSON.parse(designJSON(d))
    expect(sent.elements.store_name).toEqual({ font: 'pacifico' })
    expect(sent.groups.totals).toEqual({ font: 'inter' })
    expect(sent.banners[0].font).toBe('lobster')
    expect(designJSON(parseDesign(designJSON(d)))).toBe(designJSON(d))
    expect(isElementChanged(d, 'store_name')).toBe(true)
    expect(designJSON(setElement(d, 'store_name', { font: '' }))).not.toContain('pacifico')
  })
})

describe('item switches (what each item row prints)', () => {
  it('are the on/off-only lines of the items part, from the real catalog', () => {
    expect(partsOf(catalog, 'columns').map(e => e.key)).toEqual(['item_name', 'item_variant', 'item_code', 'item_price_note'])
    for (const g of ['header', 'meta', 'totals', 'payments', 'policy', 'referral', 'socials', 'thanks']) {
      expect(partsOf(catalog, g), g).toEqual([])
    }
    expect(partsOf(null, 'columns')).toEqual([])
  })

  it('print by default except the code; each switches and switches back to nothing saved', () => {
    const parts = partsOf(catalog, 'columns')
    for (const p of parts) {
      expect(isPartPrinted(emptyDesign(), catalog, p), p.key).toBe(p.key !== 'item_code')
      const off = setPartPrinted(emptyDesign(), catalog, p, !isPartPrinted(emptyDesign(), catalog, p))
      expect(isPartPrinted(off, catalog, p), p.key).toBe(p.key === 'item_code')
      expect(isElementChanged(off, p.key)).toBe(true)
      const back = setPartPrinted(off, catalog, p, p.key !== 'item_code')
      expect(back.elements, p.key).toEqual({})
    }
  })

  it('keeps a line\'s other settings and survives a save and load', () => {
    const p = partsOf(catalog, 'columns').find(e => e.key === 'item_code')
    let d = setPartPrinted(emptyDesign(), catalog, p, true)
    d = setElement(d, 'col_item', { text: 'Article' })
    const saved = designJSON(d)
    expect(JSON.parse(saved).elements).toEqual({ item_code: { show: true }, col_item: { text: 'Article' } })
    const loaded = parseDesign(saved)
    expect(isPartPrinted(loaded, catalog, p)).toBe(true)
    expect(isGroupChanged(loaded, catalog, 'columns')).toBe(true)
    expect(resetGroup(loaded, catalog, 'columns').elements).toEqual({})
  })
})

