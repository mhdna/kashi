import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

vi.mock('@/composables/useAdminSocket', () => ({
  useAdminSocket: () => ({ status: { value: 'open' }, ensureConnected: () => {}, onMessage: () => () => {} }),
}))
vi.mock('@/composables/useApi', () => ({ NO_INTERNET_MESSAGE: 'x', SERVER_DOWN_MESSAGE: 'y' }))
vi.mock('@/composables/useNetworkStatus', () => ({ useNetworkStatus: () => ({ online: { value: true } }) }))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({
    branches: { value: [{ id: 1, name: 'Centro' }] },
    fetchBranches: async () => {},
    fetchConnectedBranches: async () => [1],
    fetchConnectedRegisters: async () => [10],
    listRegisters: async () => [{ id: 10, code: 'CT1' }, { id: 11, code: 'CT2' }],
  }),
}))
vi.mock('@/components/WatchTillButton.vue', () => ({
  default: { props: ['registerId', 'name', 'icon'], template: '<i data-test="eye" :data-id="registerId" :data-name="name" />' },
}))

globalThis.visualViewport ??= { width: 1024, height: 768, offsetLeft: 0, offsetTop: 0, scale: 1, addEventListener () {}, removeEventListener () {} }
globalThis.ResizeObserver ??= class { observe () {} unobserve () {} disconnect () {} }

describe('ConnectedBranchesCard', () => {
  it('shows a watch eye only for the online tills of a connected branch', async () => {
    setActivePinia(createPinia())
    const { default: Card } = await import('./ConnectedBranchesCard.vue')
    const wrapper = mount(Card, { attachTo: document.body, global: { plugins: [vuetify()] } })
    await vi.waitFor(() => expect(wrapper.find('.v-card').exists()).toBe(true))
    await wrapper.find('.v-card').trigger('click')
    await vi.waitFor(() => expect(document.body.querySelectorAll('[data-test="eye"]').length).toBe(1))
    const eye = document.body.querySelector('[data-test="eye"]')
    expect(eye.dataset.id).toBe('10')
    expect(eye.dataset.name).toBe('Centro CT1')
    wrapper.unmount()
  })
})
