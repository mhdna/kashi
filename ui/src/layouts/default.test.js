import { shallowMount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import DefaultLayout from './default.vue'

const screen = vi.hoisted(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {}, removeItem: () => {} })
  return { phone: false }
})
vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/', meta: {}, query: {} }),
  useRouter: () => ({ push: vi.fn(), beforeEach: vi.fn(), afterEach: vi.fn() }),
}))
vi.mock('vuetify', async original => ({
  ...(await original()),
  useDisplay: () => ({ xs: ref(screen.phone), mobile: ref(screen.phone) }),
}))
vi.mock('@/stores/settings', () => ({
  useSettingsStore: () => ({ loaded: false, tablePageSize: 10, init: vi.fn() }),
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ refreshAccess: vi.fn(), logout: vi.fn() }),
}))
vi.mock('@/router', () => ({ default: { push: vi.fn() } }))

describe('default layout', () => {
  it.each([false, true])('offers no language switch (phone: %s)', phone => {
    screen.phone = phone
    const slots = { template: '<div><slot /><slot name="append" /></div>' }
    const wrapper = shallowMount(DefaultLayout, {
      global: {
        plugins: [createPinia()],
        renderStubDefaultSlot: true,
        stubs: { 'v-main': slots, 'v-app-bar': slots, 'v-menu': slots, 'v-list': slots },
      },
    })
    // Comments hold the disabled markup, so they don't count.
    const html = wrapper.html().replace(/<!--[\s\S]*?-->/g, '')
    // The theme control next to it proves the right bar was rendered.
    expect(html).toContain(phone ? 'mdi-logout' : 'toggle-theme-stub')
    expect(html).not.toMatch(/toggle-language-stub|mdi-translate"/)
  })
})
