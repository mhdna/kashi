import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { addBanner, emptyDesign } from '@/utils/receiptDesign'
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

describe('ReceiptBanners', () => {
  it('the reported case: a gallery of ready-made banners, drawn as they print', async () => {
    await open()
    expect(find('banners-none').exists()).toBe(true)
    for (const [i, p] of catalog.banner_presets.entries()) {
      expect(find(`preset-${i}`).find('img').attributes('src'), p.title).toContain(btoa(p.title))
    }
  })

  it('a click adds one; its words, style, when and where change', async () => {
    await open()
    await find('preset-0').trigger('click')
    expect(wrapper.props('design').banners).toEqual([catalog.banner_presets[0]])
    await find('banner-title-0').find('input').setValue('BOGO')
    expect(wrapper.props('design').banners[0].title).toBe('BOGO')
    wrapper.vm.$.setupState.set(0, { when: 'deal:Students 15%', place: 'bottom', style: 'stars' })
    await flushPromises()
    expect(wrapper.props('design').banners[0]).toMatchObject({ when: 'deal:Students 15%', place: 'bottom', style: 'stars' })
    await flushPromises()
    expect(find('banner-0').find('img').attributes('src')).toContain(btoa('BOGO'))
  })

  it('up to twelve; then the gallery waits', async () => {
    let d = emptyDesign()
    for (const p of [...catalog.banner_presets, ...catalog.banner_presets].slice(0, 12)) {d = addBanner(d, p)}
    await open(d)
    expect(find('preset-5').attributes('disabled')).toBeDefined()
    await find('banner-remove-1').trigger('click')
    expect(wrapper.props('design').banners).toHaveLength(11)
    await find('banner-down-0').trigger('click')
    expect(wrapper.props('design').banners[1].title).toBe(catalog.banner_presets[0].title)
  })

  it('a banner that prints only with a deal shows a way to see it there', async () => {
    await open(addBanner(emptyDesign(), catalog.banner_presets[0]))
    await find('banner-see-0').trigger('click')
    expect(wrapper.emitted('see')).toEqual([['item_deal']])
  })

  it('nothing in long dashes', async () => {
    await open(addBanner(emptyDesign(), catalog.banner_presets[0]))
    expect(wrapper.text()).not.toMatch(/[–—·•]/)
  })
})
