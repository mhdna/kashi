// The Arabic face for the report sheets: IBM Plex Sans Arabic (SIL Open
// Font License, see assets/fonts/ibm-plex-sans-arabic/LICENSE.txt), Arabic
// subset only -- Latin text keeps Roboto. The same @font-face rules go into
// the page and into the print frame the PDF is made from.

import regular from '@/assets/fonts/ibm-plex-sans-arabic/arabic-400.woff2?url'
import medium from '@/assets/fonts/ibm-plex-sans-arabic/arabic-500.woff2?url'
import bold from '@/assets/fonts/ibm-plex-sans-arabic/arabic-700.woff2?url'

const ARABIC_RANGE = 'U+0600-06FF, U+0750-077F, U+0870-088E, U+0890-0891, U+0897-08E1, U+08E3-08FF, U+200C-200E, U+2010-2011, U+204F, U+2E41, U+FB50-FDFF, U+FE70-FE74, U+FE76-FEFC, U+102E0-102FB, U+10E60-10E7E, U+10EC2-10EC4, U+10EFC-10EFF, U+1EE00-1EEF1'

export const ARABIC_FONT = 'IBM Plex Sans Arabic'

export const reportFontCss = [[regular, 400], [medium, 500], [bold, 700]]
  .map(([url, weight]) => `@font-face { font-family: "${ARABIC_FONT}"; font-style: normal; font-display: block; font-weight: ${weight}; src: url("${new URL(url, location.href).href}") format("woff2"); unicode-range: ${ARABIC_RANGE}; }`)
  .join('\n')

// Adds the faces to the page once.
export function installReportFonts () {
  if (document.querySelector('style[data-report-fonts]')) {return}
  const style = document.createElement('style')
  style.dataset.reportFonts = ''
  style.textContent = reportFontCss
  document.head.append(style)
}
