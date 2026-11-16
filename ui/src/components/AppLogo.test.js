import { readFileSync } from 'node:fs'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppLogo from './AppLogo.vue'

const file = path => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('Kashi brand', () => {
  it('the logo is the Ribbon K image, named Kashi for screen readers', () => {
    const img = mount(AppLogo).find('img')
    expect(img.attributes('alt')).toBe('Kashi')
    expect(decodeURIComponent(img.attributes('src'))).toContain('#660000')
  })

  it('the logo and favicon are the grey K on maroon', () => {
    for (const svg of [file('../assets/kashi-logo.svg'), file('../../public/favicon.svg')]) {
      expect(svg).toContain('fill="#660000"')
      expect(svg).toContain('stroke="#dddddd"')
    }
  })

  it('the page is called Kashi Cloud and uses the new favicon', () => {
    const html = file('../../index.html')
    expect(html).toContain('<title>Kashi Cloud</title>')
    expect(html).toContain('href="/favicon.svg"')
    expect(html).toContain('href="/apple-touch-icon.png"')
  })

  it('the theme is maroon on paper, with a lighter maroon in dark mode', async () => {
    vi.stubGlobal('localStorage', { getItem: () => null })
    const { default: vuetify } = await import('@/plugins/vuetify')
    vi.unstubAllGlobals()
    const { light, dark } = vuetify.theme.themes.value
    expect(light.colors.primary.toLowerCase()).toBe('#660000')
    expect(light.colors.background.toLowerCase()).toBe('#f7f2ea')
    expect(dark.colors.primary.toLowerCase()).toBe('#e0928a')
  })
})
