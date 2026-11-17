<template>
  <div
    aria-label="K"
    class="app-logo"
    role="img"
    :style="{
      '--app-logo-image': `url(${logoImage})`,
      '--app-logo-color': logoColor,
    }"
  />
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useTheme } from 'vuetify'
  import logoImage from '@/assets/app-logo.png'

  // Match the home page's hero logo tint: full-strength on-surface in dark
  // mode, softened to 70% in light mode.
  const theme = useTheme()
  const logoColor = computed(() => theme.global.current.value.dark
    ? 'rgb(var(--v-theme-on-surface))'
    : 'rgba(var(--v-theme-on-surface), 0.7)')
</script>

<style scoped>
  .app-logo {
    aspect-ratio: 1;
    width: clamp(100px, 13vw, 180px);
    background-color: var(--app-logo-color);
    mask: var(--app-logo-image) center / contain no-repeat;
    -webkit-mask: var(--app-logo-image) center / contain no-repeat;
  }
</style>
