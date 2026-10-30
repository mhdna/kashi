<template>
  <div class="rpt-grid" :style="{ gridTemplateColumns: template || `repeat(${cols}, minmax(0, 1fr))` }">
    <slot />
  </div>
</template>

<script setup>
  // Side-by-side blocks on the sheet (two charts, a chart and a table).
  // template: a grid-template-columns value for uneven columns ("3fr 2fr").
  defineProps({
    cols: { type: Number, default: 2 },
    template: { type: String, default: '' },
  })
</script>
