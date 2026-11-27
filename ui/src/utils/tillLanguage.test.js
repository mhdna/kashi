import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { FR } from '@/i18n/fr'
import fixture from './testdata/till_languages.json'
import { TILL_LANGUAGES } from './tillLanguage'

const page = name => readFileSync(`${process.cwd()}/src/pages/${name}.vue`, 'utf8')

describe('till language setting', () => {
  it('offers exactly the languages the server accepts', () => {
    expect(TILL_LANGUAGES.map(l => l.value).sort()).toEqual(fixture)
    expect(TILL_LANGUAGES.map(l => l.title)).toEqual(['Français', 'English'])
  })

  it('is on a branch\'s settings, French when unset', () => {
    const source = page('branch-settings')
    expect(source).toContain('v-model="settingsForm.till_language"')
    expect(source).toContain(':items="TILL_LANGUAGES"')
    expect(source).toContain('till_language: d.till_language || \'fr\'')
  })

  it('is on the default settings, left as is unless picked', () => {
    const source = page('default-branch-settings')
    expect(source).toContain('v-model="form.till_language"')
    expect(source).toContain('till_language: \'\',')
    expect(source).toContain('{ title: t(\'Leave as is\'), value: \'\' }, ...TILL_LANGUAGES')
  })

  it('has French for its labels', () => {
    for (const key of ['Till language', 'Language on the tills', 'Leave as is']) {
      expect(FR[key]).toBeTruthy()
    }
  })
})
