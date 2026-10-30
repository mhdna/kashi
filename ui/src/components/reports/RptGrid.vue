<template>
  <div class="rpt-grid" :style="{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }">
    <slot />
  </div>
</template>

<script setup>
  // Side-by-side blocks on the sheet (two charts, a chart and a table).
  defineProps({ cols: { type: Number, default: 2 } })
</script>
