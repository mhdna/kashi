import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FR } from './fr'

let store
function stubStorage () {
  store = {}
  vi.stubGlobal('localStorage', {
    getItem: k => store[k] ?? null,
    setItem: (k, v) => { store[k] = String(v) },
  })
}

async function fresh (saved) {
  stubStorage()
  if (saved) {store.locale = saved}
  vi.resetModules()
  return import('./index.js')
}

describe('i18n', () => {
  beforeEach(stubStorage)

  it('defaults to French', async () => {
    const m = await fresh()
    expect(m.locale.value).toBe('fr')
    expect(m.t('Log out')).toBe('Se déconnecter')
    expect(document.documentElement.lang).toBe('fr')
  })

  it('toggles to English, remembers it, and a reload keeps it', async () => {
    const m = await fresh()
    m.setLocale('en')
    expect(m.t('Log out')).toBe('Log out')
    expect(store.locale).toBe('en')
    expect((await fresh('en')).locale.value).toBe('en')
  })

  it('ignores a bad saved value', async () => {
    expect((await fresh('de')).locale.value).toBe('fr')
  })

  it('shows English for untranslated text and fills placeholders', async () => {
    const m = await fresh()
    expect(m.t('Nope {n}', { n: 3 })).toBe('Nope 3')
  })

  it('keeps every placeholder in each French string', () => {
    for (const [en, fr] of Object.entries(FR)) {
      const ph = s => (s.match(/\{\w+\}/g) || []).sort().join()
      expect(ph(fr), en).toBe(ph(en))
    }
  })
})
