<template>
  <VChart autoresize class="chart" :option="option" />
</template>

<script setup>
  import VChart from '@/components/EChart.vue'
  import { t } from '@/i18n'

  const option = ref({
    tooltip: {
      trigger: 'item',
      appendTo: 'body',
    },
    series: [
      {
        name: t('Access From'),
        type: 'pie',
        radius: '50%',
        data: [
          { value: 1048, name: t('Search Engine') },
          { value: 735, name: t('Direct') },
          { value: 580, name: t('Email') },
          { value: 484, name: t('Union Ads') },
          { value: 300, name: t('Video Ads') },
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  })
</script>

<style scoped>
.chart {
  height: 260px;
  width: 100%;
}
</style>
