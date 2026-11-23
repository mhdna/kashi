/**
 * plugins/index.js
 *
 * Automatically included in `./src/main.js`
 */

import i18n from '@/i18n'
import router from '@/router'
import pinia from '@/stores'
// Plugins
import vuetify from './vuetify'

export function registerPlugins (app) {
  app
    .use(vuetify)
    .use(pinia)
    .use(router)
    .use(i18n)
}
