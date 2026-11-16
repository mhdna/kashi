<template>
  <img alt="Kashi" class="app-logo" :src="logoImage">
</template>

<script setup lang="ts">
  // The Ribbon K: a grey stroke on maroon, the same in light and dark mode.
  import logoImage from '@/assets/kashi-logo.svg'
</script>

<style scoped>
  .app-logo {
    display: block;
    aspect-ratio: 1;
    width: clamp(72px, 9vw, 112px);
  }
</style>
