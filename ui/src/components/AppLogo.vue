<template>
  <div
    aria-label="K"
    class="app-logo"
    role="img"
    :style="{ '--app-logo-image': `url(${logoImage})` }"
  />
</template>

<script setup lang="ts">
  import logoImage from '../../GB.png'
</script>

<style scoped>
  .app-logo {
    aspect-ratio: 1;
    width: clamp(100px, 13vw, 180px);
    background-color: rgba(var(--v-theme-on-surface), 0.38);
    mask: var(--app-logo-image) center / contain no-repeat;
    -webkit-mask: var(--app-logo-image) center / contain no-repeat;
  }
</style>
