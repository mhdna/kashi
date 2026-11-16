import { readFileSync } from 'node:fs'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { vuetify as testVuetify } from '@/test/vuetify'
import AppLogo from './AppLogo.vue'

const file = path => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('Kashi brand', () => {
  it('the logo is the K stroke only, named Kashi for screen readers', () => {
    const logo = mount(AppLogo, { global: { plugins: [testVuetify()] } }).find('[role="img"]')
    expect(logo.attributes('aria-label')).toBe('Kashi')
    expect(decodeURIComponent(logo.attributes('style'))).not.toContain('#660000')
  })

  it('the in-app K has no maroon square; the favicon keeps it', () => {
    const k = file('../assets/kashi-k.svg')
    expect(k).not.toContain('#660000')
    expect(k).toContain('stroke=')
    const favicon = file('../../public/favicon.svg')
    expect(favicon).toContain('fill="#660000"')
    expect(favicon).toContain('stroke="#dddddd"')
  })

  it('the page is called Kashi Cloud and uses the new favicon', () => {
    const html = file('../../index.html')
    expect(html).toContain('<title>Kashi Cloud</title>')
    expect(html).toContain('href="/favicon.svg"')
    expect(html).toContain('href="/apple-touch-icon.png"')
  })

  it('the theme is maroon on the default white page, with a lighter maroon in dark mode', async () => {
    vi.stubGlobal('localStorage', { getItem: () => null })
    const { default: vuetify } = await import('@/plugins/vuetify')
    vi.unstubAllGlobals()
    const { light, dark } = vuetify.theme.themes.value
    expect(light.colors.primary.toLowerCase()).toBe('#660000')
    expect(light.colors.background.toLowerCase()).toBe('#ffffff')
    expect(dark.colors.primary.toLowerCase()).toBe('#e0928a')
  })
})
