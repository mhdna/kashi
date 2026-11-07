import { fileURLToPath, URL } from 'node:url'
import Vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

// Tests get only the Vue plugin: the app's router, layout and auto-import
// plugins (vite.config.mjs) are for the running app, not a mounted component.
export default defineConfig({
  plugins: [Vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('src', import.meta.url)) },
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.test.js'],
    css: false,
    server: { deps: { inline: ['vuetify'] } },
  },
})
