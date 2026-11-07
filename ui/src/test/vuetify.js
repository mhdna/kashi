// The real Vuetify kit and the app's own icon set, for mounting components in
// tests (global: { plugins: [vuetify()] }).
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { aliases, mdi } from '@/plugins/icons'

export function vuetify () {
  return createVuetify({ components, directives, icons: { defaultSet: 'mdi', aliases, sets: { mdi } } })
}
