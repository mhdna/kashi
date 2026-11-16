<template>
  <div aria-label="Kashi" class="app-logo" role="img" :style="logoStyle" />
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useTheme } from 'vuetify'
  import logoImage from '@/assets/kashi-k.svg'

  // The Ribbon K's stroke on its own, no maroon square, tinted to the page:
  // full-strength on-surface in dark mode, softened to 70% in light mode.
  // The URL is quoted: Vite inlines a small SVG as a data URI with quotes in
  // it, which an unquoted url() can't hold.
  const theme = useTheme()
  const logoStyle = computed(() => ({
    '--app-logo-image': `url("${logoImage}")`,
    '--app-logo-color': theme.global.current.value.dark
      ? 'rgb(var(--v-theme-on-surface))'
      : 'rgba(var(--v-theme-on-surface), 0.7)',
  }))
</script>

<style scoped>
  .app-logo {
    aspect-ratio: 1;
    width: clamp(72px, 9vw, 112px);
    background-color: var(--app-logo-color);
    mask: var(--app-logo-image) center / contain no-repeat;
    -webkit-mask: var(--app-logo-image) center / contain no-repeat;
  }
</style>
