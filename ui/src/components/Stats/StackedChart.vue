<template>
  <VChart class="chart" :option="option" />
</template>

<script setup>
  import VChart from '@/components/EChart.vue'
  import { t } from '@/i18n'

  const option = {
    backgroundColor: 'transparent',
    grid: {
      left: 0,
      right: 0,
    },
    tooltip: {
      trigger: 'axis',
      appendTo: 'body',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: 'transparent',
        },
      },
    },
    legend: {
      data: [t('Branch A'), t('Branch B'), t('Branch C'), t('Brand D')],
    },
    toolbox: {
      feature: {
        saveAsImage: {},
      },
    },
    xAxis: [
      {
        type: 'category',
        boundaryGap: false,
        data: [t('Mon'), t('Tue'), t('Wed'), t('Thu'), t('Fri'), t('Sat'), t('Sun')],
      },
    ],
    yAxis: [
      {
        type: 'value',
      },
    ],
    series: [
      {
        name: t('Branch A'),
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        emphasis: {
          focus: 'series',
        },
        data: [120, 132, 101, 134, 90, 230, 210],
      },
      {
        name: t('Branch B'),
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        emphasis: {
          focus: 'series',
        },
        data: [220, 182, 191, 234, 290, 330, 310],
      },
      {
        name: t('Branch C'),
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        emphasis: {
          focus: 'series',
        },
        data: [150, 232, 201, 154, 190, 330, 410],
      },
      {
        name: t('Branch D'),
        type: 'line',
        stack: 'Total',
        areaStyle: {},
        emphasis: {
          focus: 'series',
        },
        data: [320, 332, 301, 334, 390, 330, 320],
      },
      {
        name: t('Search Engine'),
        type: 'line',
        stack: 'Total',
        label: {
          show: true,
          position: 'top',
        },
        areaStyle: {},
        emphasis: {
          focus: 'series',
        },
        data: [820, 932, 901, 934, 1290, 1330, 1320],
      },
    ],
  }
</script>

<style scoped>
.chart {
  height: 400px;
  width: 100%;
}
</style>
