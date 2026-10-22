import { polyfillCountryFlagEmojis } from 'country-flag-emoji-polyfill'
import { createApp } from 'vue'

import { installClientLogger } from '@/composables/useClientLogs'
import { registerPlugins } from '@/plugins'

import App from './App.vue'

import '@/styles/global.scss'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'

installClientLogger()

// Windows browsers draw flag emoji as bare letters ("LB"); this loads a flag
// font only when the platform can't render them.
polyfillCountryFlagEmojis()

// After a deploy, an open tab's index.html still points at the previous
// build's hashed chunks, which are gone from the server, so the next lazy
// route import fails. Reload to pick up the new index.html. The timestamp
// guard stops a reload loop if the chunk is missing for some other reason.
window.addEventListener('vite:preloadError', event => {
  let last = 0
  try {
    last = Number(sessionStorage.getItem('chunk-reload-at')) || 0
    sessionStorage.setItem('chunk-reload-at', String(Date.now()))
  } catch {
    // Storage blocked: no loop guard, but still worth one reload.
  }
  if (Date.now() - last > 10_000) {
    event.preventDefault()
    window.location.reload()
  }
})

const app = createApp(App)
app.config.errorHandler = (err, _i, info) => console.error(`Vue error (${info})`, err)
registerPlugins(app)
app.mount('#app')
