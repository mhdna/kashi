<template>
  <div class="rpt-chart" :style="{ height: `${height}px` }">
    <VChart
      autoresize
      :init-options="{ devicePixelRatio: 2 }"
      :option="option"
      style="width: 100%; height: 100%"
      theme="kashi-light"
    />
  </div>
</template>

<script setup>
  // One chart on a report sheet: always the light theme (reports print on
  // white), drawn at twice the pixel density so the PDF stays sharp.
  import VChart from '@/components/EChart.vue'

  defineProps({
    option: { type: Object, required: true },
    height: { type: Number, default: 240 },
  })
</script>
