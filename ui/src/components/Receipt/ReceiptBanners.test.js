import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { bannerKinds, designJSON, emptyDesign, setGlobalBannerStyle, setKindBanner } from '@/utils/receiptDesign'
import catalog from '@/utils/testdata/receipt_designer_catalog.json'
import ReceiptBanners from './ReceiptBanners.vue'

let wrapper
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = '' })

const draw = vi.fn(async b => `data:image/png;base64,${btoa(JSON.parse(b).title)}`)
async function open (design = emptyDesign(), props = {}) {
  wrapper = mount(ReceiptBanners, {
    props: { catalog, design, draw, deals: [{ name: 'Students 15%' }], 'onUpdate:design': d => wrapper.setProps({ design: d }), ...props },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const kind = when => ({ when, place: '' })

describe('ReceiptBanners', () => {
  it('the reported case: a list of invoice kinds incl. each deal, words typed per kind', async () => {
    await open()
    for (const k of ['always', 'deal', 'item_deal', 'item_discount', 'referral', 'deal:Students 15%']) {expect(find(`kind-${k}`).exists(), k).toBe(true)}
    await find('title-deal:Students 15%').find('input').setValue('STUDENT DEAL')
    expect(wrapper.props('design').banners).toEqual([{ style: 'ribbon', title: 'STUDENT DEAL', when: 'deal:Students 15%' }])
    await flushPromises()
    expect(find('kind-deal:Students 15%').find('img').attributes('src')).toContain(btoa('STUDENT DEAL'))
  })

  it('one design for all, changed once; a kind can keep its own', async () => {
    let d = setKindBanner(emptyDesign(), kind(''), { title: 'SALE' })
    d = setKindBanner(d, kind('deal'), { title: 'DEAL' })
    d = setKindBanner(d, kind('deal'), { style: 'stars' })
    await open(d)
    wrapper.vm.$.setupState.update(setGlobalBannerStyle(wrapper.props('design'), 'burst'))
    await flushPromises()
    const b = wrapper.props('design').banners
    expect(b.find(x => !x.when).style).toBe('burst')
    expect(b.find(x => x.when === 'deal')).toMatchObject({ style: 'stars', own_style: true })
    expect(JSON.parse(designJSON(wrapper.props('design'))).banner_style).toBe('burst')
  })

  it('an own design set back to "same as all" follows the global one again', () => {
    let d = setGlobalBannerStyle(emptyDesign(), 'frame')
    d = setKindBanner(d, kind('deal'), { title: 'X', style: 'stars' })
    d = setKindBanner(d, kind('deal'), { style: '' })
    expect(d.banners[0].style).toBe('frame')
    expect(d.banners[0].own_style).toBeUndefined()
    expect(setGlobalBannerStyle(d, 'burst').banners[0].style).toBe('burst')
  })

  it('a kind left with no words is not sent', () => {
    const d = setKindBanner(emptyDesign(), bannerKinds([]).find(k => k.when === 'referral'), { title: '  ' })
    expect(d.banners[0].place).toBe('bottom')
    expect(designJSON(d)).toBe('')
  })

  it('a saved banner for a deal that is gone still shows, to clear', async () => {
    await open(setKindBanner(emptyDesign(), kind('deal:Old deal'), { title: 'OLD' }))
    expect(find('kind-deal:Old deal').exists()).toBe(true)
  })

  it('a kind that prints with a deal shows a way to see it there', async () => {
    await open(setKindBanner(emptyDesign(), kind('item_deal'), { title: 'BOGO' }))
    await find('see-item_deal').trigger('click')
    expect(wrapper.emitted('see')).toEqual([['item_deal']])
  })

  it('nothing in long dashes', async () => {
    await open()
    expect(wrapper.text()).not.toMatch(/[–—·•]/)
  })
})
