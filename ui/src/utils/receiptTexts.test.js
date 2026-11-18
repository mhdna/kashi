import { describe, expect, it } from 'vitest'
import catalog from '@/data/receiptThemes.json'
import {
  emptyTexts, isHidden, isReworded, paperStrips, parseTexts, pictureRows,
  receiptModes, sectionLabels, shownText, textParts, textsJSON, toggleHidden, wording,
} from './receiptTexts'
import cases from './testdata/receipt_texts_cases.json'

const T = catalog.texts
const limits = { sections: T.sections, maxLength: T.max_length }
const save = texts => textsJSON(texts, limits)

const everyMode = []
for (const theme of catalog.themes) {
  for (const layout of theme.layouts) {
    for (const m of layout.modes) {everyMode.push({ where: `${theme.id}/${layout.id}/${m.mode}`, m })}
  }
}

describe('the catalog the till wrote', () => {
  it('has every look in all five modes, each with its parts', () => {
    expect(everyMode.length).toBe(catalog.themes.reduce((n, t) => n + t.layouts.length, 0) * 5)
    for (const { where, m } of everyMode) {
      expect(receiptModes.map(r => r.mode), where).toContain(m.mode)
      expect(m.height, where).toBeGreaterThan(0)
      expect(m.parts.length, where).toBeGreaterThan(0)
    }
  })
  it('names every section and text part this side knows', () => {
    expect(T.sections).toEqual(Object.keys(sectionLabels))
    for (const p of textParts) {expect(T.defaults[p.part], p.part).toBeTruthy()}
  })
})

describe('textsJSON agrees with kashi\'s API on every case', () => {
  for (const c of cases) {
    it(c.name, () => {
      expect(save(parseTexts(c.in))).toBe(c.out)
    })
  }
})

describe('parseTexts and textsJSON', () => {
  it('reads nothing or junk as every default', () => {
    for (const raw of ['', '  ', null, undefined, '{oops', '[]', '"x"', 'null']) {expect(parseTexts(raw)).toEqual(emptyTexts())}
  })
  it('the reported case: rewording Follow us and leaving out payments saves both', () => {
    const texts = { ...emptyTexts(), socials_title: ' Find us on: ', hide: ['payments'] }
    expect(save(texts)).toBe('{"socials_title":"Find us on:","hide":["payments"]}')
  })
  it('randomized: saving what was read back changes nothing', () => {
    const pick = arr => arr[Math.floor(Math.random() * arr.length)]
    const words = ['', ' ', 'PAID', '  Find us  ', 'شكراً', 'a\r\nb', 'x'.repeat(320), '{percent} off']
    for (let run = 0; run < 500; run++) {
      const texts = emptyTexts()
      for (const p of textParts) {texts[p.part] = pick(words)}
      texts.hide = T.sections.filter(() => Math.random() < 0.4).concat(Math.random() < 0.3 ? ['junk'] : [])
      const once = save(texts)
      expect(save(parseTexts(once))).toBe(once)
      if (once) {
        const parsed = JSON.parse(once)
        for (const v of Object.values(parsed)) {
          if (typeof v === 'string') {expect([...v].length).toBeLessThanOrEqual(T.max_length)}
        }
      }
    }
  })
})

describe('wording on the paper', () => {
  it('shows the default until reworded, and the reward and code in the referral parts', () => {
    expect(wording(emptyTexts(), 'socials_title', T.defaults)).toBe('Follow us on:')
    expect(shownText({ ...emptyTexts(), referral_note: 'Bring a friend, get {percent} off' }, 'referral_note', T)).toBe(`Bring a friend, get ${T.referral_percent}% off`)
    expect(shownText(emptyTexts(), 'referral_title', T)).toBe(`${T.defaults.referral_title} ${T.referral_code}`)
    expect(isReworded(emptyTexts(), 'thanks', T.defaults)).toBe(false)
    expect(isReworded({ thanks: '  ' }, 'thanks', T.defaults)).toBe(false)
    expect(isReworded({ thanks: 'See you' }, 'thanks', T.defaults)).toBe(true)
  })
  it('toggles a section out and back', () => {
    const out = toggleHidden(emptyTexts(), 'policy')
    expect(isHidden(out, 'policy')).toBe(true)
    expect(isHidden(toggleHidden(out, 'policy'), 'policy')).toBe(false)
  })
  it('has no long dashes or dots in anything it says', () => {
    const said = [...textParts.map(p => p.label), ...Object.values(sectionLabels), ...receiptModes.map(m => m.label)].join(' ')
    expect(said).not.toMatch(/[–—·•]/)
  })
})

describe('paperStrips over every look and mode', () => {
  it('with nothing changed, the strips are the whole picture, in order, without a gap', () => {
    for (const { where, m } of everyMode) {
      const strips = paperStrips(m, emptyTexts(), T)
      let y = 0
      for (const s of strips) {
        expect(s.kind, where).not.toBe('hidden')
        expect(s.top, where).toBe(y)
        expect(s.bottom, where).toBeGreaterThan(s.top)
        y = s.bottom
      }
      expect(y, where).toBe(m.height)
      expect(strips.some(s => s.reworded), where).toBe(false)
    }
  })

  it('every text part the mode prints is a strip, the referral ones only with a referral code', () => {
    for (const { where, m } of everyMode) {
      const parts = paperStrips(m, emptyTexts(), T).filter(s => s.kind !== 'image').map(s => s.part)
      for (const p of ['payments_title', 'policy_title', 'socials_title', 'thanks']) {expect(parts, where).toContain(p)}
      expect(parts.includes('referral_note'), where).toBe(m.mode === 'referral')
    }
  })

  it('the socials title keeps its icons beside it', () => {
    for (const { where, m } of everyMode) {
      const socials = paperStrips(m, emptyTexts(), T).find(s => s.kind === 'socials')
      expect(socials?.icons, where).toBeTruthy()
      expect(socials.icons.right, where).toBeGreaterThan(socials.icons.left)
    }
  })

  it('randomized: leaving parts out shows exactly the picture less their bands, as the till prints', () => {
    for (let run = 0; run < 400; run++) {
      const { where, m } = everyMode[Math.floor(Math.random() * everyMode.length)]
      const hide = T.sections.filter(() => Math.random() < 0.5)
      const strips = paperStrips(m, { ...emptyTexts(), hide }, T)
      const cut = m.parts.filter(p => p.kind === 'section' && hide.includes(p.part)).reduce((n, p) => n + p.bottom - p.top, 0)
      expect(pictureRows(strips), `${where} ${hide}`).toBe(m.height - cut)
      for (const section of hide) {
        const printed = m.parts.some(p => p.kind === 'section' && p.part === section)
        expect(strips.some(s => s.kind === 'hidden' && s.section === section), `${where} ${section}`).toBe(printed)
        expect(strips.some(s => s.section === section && s.kind !== 'hidden'), `${where} ${section}`).toBe(false)
      }
    }
  })

  it('one row off anywhere is caught: a band moved by one changes the rows shown', () => {
    const { m } = everyMode[0]
    const hide = ['payments']
    const good = pictureRows(paperStrips(m, { ...emptyTexts(), hide }, T))
    const moved = { ...m, parts: m.parts.map(p => p.kind === 'section' && p.part === 'payments' ? { ...p, bottom: p.bottom + 1 } : p) }
    expect(pictureRows(paperStrips(moved, { ...emptyTexts(), hide }, T))).toBe(good - 1)
  })

  it('a reworded part is flagged and carries the new words, the rest stays picture', () => {
    const { m } = everyMode.find(e => e.m.mode === 'referral')
    const strips = paperStrips(m, { ...emptyTexts(), thanks: 'See you soon', socials_title: 'Find us:' }, T)
    expect(strips.find(s => s.part === 'thanks')).toMatchObject({ reworded: true, text: 'See you soon' })
    expect(strips.find(s => s.part === 'socials_title')).toMatchObject({ reworded: true, text: 'Find us:' })
    expect(strips.find(s => s.part === 'payments_title')).toMatchObject({ reworded: false, text: 'PAYMENT DETAILS' })
  })
})
