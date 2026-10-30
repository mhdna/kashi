// Language of the Reports page's sheets. English text is the key; AR holds
// its Arabic. The page sets the language before (re)drawing a report, so
// components can call t() at setup.

import { AR } from './reportArabic'

let lang = 'en'

export function setReportLang (value) {
  lang = value === 'ar' ? 'ar' : 'en'
}
export const reportLang = () => lang
export const isRtl = () => lang === 'ar'

// t('Sales on {day}', { day }) -- an untranslated string falls back to English.
export function t (text, params) {
  let out = (lang === 'ar' && AR[text]) || text
  if (params) {
    for (const [key, value] of Object.entries(params)) {out = out.replaceAll(`{${key}}`, String(value))}
  }
  return out
}
