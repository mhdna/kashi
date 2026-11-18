import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import catalog from '@/data/receiptThemes.json'
import { vuetify } from '@/test/vuetify'
import { emptyTexts } from '@/utils/receiptTexts'
import ReceiptPaper from './ReceiptPaper.vue'

const layout = catalog.themes.find(t => t.id === 'classic').layouts.find(l => l.id === 'A')
const modeInfo = mode => layout.modes.find(m => m.mode === mode)

let wrapper
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

function open (mode = 'sale', texts = emptyTexts(), props = {}) {
  wrapper = mount(ReceiptPaper, {
    props: { modeInfo: modeInfo(mode), texts, catalogTexts: catalog.texts, ...props },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  return wrapper
}
const part = name => wrapper.find(`[data-test="part-${name}"]`)
const lastTexts = () => wrapper.emitted('update:texts')?.at(-1)?.[0]

describe('ReceiptPaper', () => {
  it('the reported case: the sale shows the socials row with its icons, and every part to click', () => {
    open('sale')
    for (const p of ['payments_title', 'policy_title', 'socials_title', 'thanks']) {expect(part(p).exists(), p).toBe(true)}
    expect(part('referral_title').exists()).toBe(false)
    // Untouched, each part is the till's own picture.
    expect(part('socials_title').find('.rp-image').exists()).toBe(true)
  })

  it('the referral mode shows the referral code and its line', () => {
    open('referral')
    expect(part('referral_title').exists()).toBe(true)
    expect(part('referral_note').exists()).toBe(true)
  })

  it('draws the picture at the paper width, each strip cut from its place', () => {
    open('sale', emptyTexts(), { paperWidth: 288 })
    const scale = 288 / modeInfo('sale').width
    const heights = wrapper.findAll('.rp-image').map(el => Number.parseFloat(el.element.style.height))
    const total = heights.reduce((a, b) => a + b, 0)
    expect(total).toBeCloseTo(modeInfo('sale').height * scale, 3)
  })

  it('click a part, type, Enter: the new words are sent up', async () => {
    open('sale')
    await part('socials_title').find('button.rp-face').trigger('click')
    await flushPromises()
    const editor = wrapper.find('[data-test="edit-socials_title"]')
    expect(editor.element.value).toBe('Follow us on:')
    await editor.setValue('  Find us on:  ')
    await editor.trigger('keydown', { key: 'Enter' })
    expect(lastTexts()).toMatchObject({ socials_title: 'Find us on:' })
  })

  it('typing the default back, or Esc, rewords nothing', async () => {
    open('sale')
    await part('thanks').find('button.rp-face').trigger('click')
    const editor = wrapper.find('[data-test="edit-thanks"]')
    await editor.setValue('Thank you for shopping with us!')
    await editor.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('update:texts')).toBeUndefined()
    await part('thanks').find('button.rp-face').trigger('click')
    await wrapper.find('[data-test="edit-thanks"]').setValue('Bye')
    await wrapper.find('[data-test="edit-thanks"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.emitted('update:texts')).toBeUndefined()
    expect(wrapper.find('[data-test="edit-thanks"]').exists()).toBe(false)
  })

  it('a reworded part shows its words, with a way back to the default', async () => {
    open('sale', { ...emptyTexts(), thanks: 'See you soon', socials_title: 'Find us:' })
    expect(part('thanks').text()).toContain('See you soon')
    expect(part('socials_title').text()).toContain('Find us:')
    expect(part('socials_title').find('.rp-icons').exists()).toBe(true)
    await wrapper.find('[data-test="reset-thanks"]').trigger('click')
    expect(lastTexts().thanks).toBe('')
  })

  it('the eye leaves a part out, and Show puts it back', async () => {
    open('sale')
    await wrapper.find('[data-test="hide-socials"]').trigger('click')
    expect(lastTexts().hide).toEqual(['socials'])
    wrapper.unmount()
    open('sale', { ...emptyTexts(), hide: ['socials'] })
    expect(part('socials_title').exists()).toBe(false)
    expect(wrapper.find('[data-test="hidden-socials"]').text()).toContain('Socials left out')
    await wrapper.find('[data-test="show-socials"]').trigger('click')
    expect(lastTexts().hide).toEqual([])
  })

  it('read-only: nothing to click, no tools', async () => {
    open('sale', emptyTexts(), { editable: false })
    expect(wrapper.find('.rp-tools').exists()).toBe(false)
    await part('thanks').find('button.rp-face').trigger('click')
    expect(wrapper.find('[data-test="edit-thanks"]').exists()).toBe(false)
  })

  it('the referral line explains {percent} while it is edited', async () => {
    open('referral')
    await part('referral_note').find('button.rp-face').trigger('click')
    expect(part('referral_note').text()).toContain('{percent} prints as the reward')
  })
})
