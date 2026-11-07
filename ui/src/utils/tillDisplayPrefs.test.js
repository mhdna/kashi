import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { DISPLAY_DEFAULTS, displayFormFrom } from './tillDisplayPrefs'

describe('displayFormFrom', () => {
  it('prints return and exchange receipts unless kashi turned it off', () => {
    expect(displayFormFrom(null).print_return_receipts).toBe(true)
    expect(displayFormFrom({}).print_return_receipts).toBe(true)
    expect(displayFormFrom({ theme: 'dark' }).print_return_receipts).toBe(true)
    expect(displayFormFrom({ print_return_receipts: false }).print_return_receipts).toBe(false)
    expect(displayFormFrom({ print_return_receipts: true }).print_return_receipts).toBe(true)
  })

  it('keeps every saved pref over the defaults, and the defaults for the rest', () => {
    const form = displayFormFrom({ quick_access_rows: 0, theme: 'dark', print_return_receipts: false })
    expect(form).toEqual({ ...DISPLAY_DEFAULTS, quick_access_rows: 0, theme: 'dark', print_return_receipts: false })
  })

  it('ignores what isn\'t a set of prefs', () => {
    for (const saved of [undefined, null, 'x', 3, []]) {
      expect(displayFormFrom(saved)).toEqual({ ...DISPLAY_DEFAULTS })
    }
  })

  it('never changes the defaults', () => {
    const form = displayFormFrom(null)
    form.print_return_receipts = false
    expect(DISPLAY_DEFAULTS.print_return_receipts).toBe(true)
    expect(() => { DISPLAY_DEFAULTS.print_return_receipts = false }).toThrow()
  })
})

// The till only takes the prefs it knows (kashi-pos display_prefs.go), so
// the key kashi sends must be the one the till reads.
describe('the till pref kashi sends', () => {
  it('is the key kashi-pos reads', () => {
    expect(Object.keys(DISPLAY_DEFAULTS)).toContain('print_return_receipts')
  })

  it('has a switch in branch settings and a row in the default branch settings', () => {
    const page = (p) => readFileSync(resolve(process.cwd(), p), 'utf8')
    expect(page('src/pages/branch-settings.vue')).toMatch(/v-model="displayForm\.print_return_receipts"/)
    expect(page('src/pages/default-branch-settings.vue')).toMatch(/key: 'print_return_receipts'/)
  })
})
