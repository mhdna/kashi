<template>
  <!-- Stand-in for an app-bar pill while its first fetch is in flight: same
       height and rounding as the real v-card pill, so the bar doesn't jump
       when it swaps in. On mobile the pills are 34px icon squares. -->
  <v-skeleton-loader
    class="pill-skeleton"
    color="transparent"
    type="chip"
    :width="mobile ? 34 : width"
  />
</template>

<script setup>
  import { useDisplay } from 'vuetify'

  defineProps({
    // Rough desktop width of the pill it stands in for.
    width: { type: Number, default: 96 },
  })

  const { mobile } = useDisplay()
</script>

<style scoped>
.pill-skeleton :deep(.v-skeleton-loader__chip) {
  margin: 0;
  max-width: none;
  width: 100%;
  height: 34px;
  border-radius: 24px;
}
</style>
