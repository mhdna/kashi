<template>
  <!-- ltr even on an Arabic sheet: the canvas would otherwise move the
       punctuation of Latin labels ("Omar H." -> ".Omar H"); Arabic text
       still reads right-to-left, and the axes do the mirroring. -->
  <div class="rpt-chart" dir="ltr" :style="{ height: `${height}px` }">
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
