import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const getBranchSettings = vi.fn()
const updateBranchSettings = vi.fn()
vi.mock('@/composables/useBranchSettings', () => ({ useBranchSettings: () => ({ getBranchSettings, updateBranchSettings }) }))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({ branches: ref([{ id: 7, name: 'Centro' }, { id: 8, name: 'Hara' }]), fetchBranches: async () => {} }),
}))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success: vi.fn(), error: vi.fn() }) }))
const canEdit = ref(true)
vi.mock('@/composables/usePageAccess', () => ({ usePageAccess: () => ({ canEdit }) }))

const { default: ReceiptDesigner } = await import('./receipt-designer.vue')

let wrapper
beforeEach(() => { setActivePinia(createPinia()); canEdit.value = true })
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks(); document.body.innerHTML = '' })

async function open () {
  wrapper = mount(ReceiptDesigner, { global: { plugins: [vuetify()] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}
const state = () => wrapper.vm.$.setupState
const button = label => wrapper.findAll('button').find(b => b.text().trim() === label)

describe('Receipt Designer', () => {
  it('shows the paper in every mode, and the referral note only in the referral one', async () => {
    await open()
    for (const label of ['Sale', 'Sale with referral code', 'Sale with a client', 'Exchange', 'Return']) {
      await button(label).trigger('click')
      await flushPromises()
      expect(wrapper.find('.rp').exists(), label).toBe(true)
      const referral = label === 'Sale with referral code'
      expect(wrapper.find('[data-test="part-referral_title"]').exists(), label).toBe(referral)
      expect(wrapper.text().includes('The referral code prints only while referrals are on.'), label).toBe(referral)
    }
  })

  it('a branch opens with its own wording, and Save sends it cleaned with the look', async () => {
    getBranchSettings.mockResolvedValue({ receipt_theme: 'modern', receipt_layout: 'B', receipt_texts: '{"thanks":"See you","hide":["payments"]}' })
    await open()
    state().target = 7
    await flushPromises()
    expect(state().texts).toMatchObject({ thanks: 'See you', hide: ['payments'] })
    expect(wrapper.find('[data-test="hidden-payments"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="part-thanks"]').text()).toContain('See you')

    state().texts = { ...state().texts, socials_title: '  Find us on:  ' }
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings).toHaveBeenCalledWith(7, {
      receipt_theme: 'modern',
      receipt_layout: 'B',
      receipt_texts: '{"socials_title":"Find us on:","thanks":"See you","hide":["payments"]}',
    })
  })

  it('all branches: the same wording goes to each, and every default is sent as empty', async () => {
    await open()
    await button('Save').trigger('click')
    await flushPromises()
    expect(updateBranchSettings).toHaveBeenCalledTimes(2)
    expect(updateBranchSettings.mock.calls.map(c => c[1].receipt_texts)).toEqual(['', ''])
  })

  it('read-only roles see the paper but cannot change it', async () => {
    canEdit.value = false
    await open()
    expect(wrapper.find('.rp').exists()).toBe(true)
    expect(wrapper.find('.rp-tools').exists()).toBe(false)
    expect(button('Save')).toBeUndefined()
  })
})
