<template>
  <VChart class="chart" :option="option" :loading="loading" autoresize />
</template>

<script setup>
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import {
  TooltipComponent,
  LegendComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([
  TooltipComponent,
  LegendComponent,
  PieChart,
  CanvasRenderer
])

import { useTheme } from "vuetify";
const theme = useTheme();
const currentTheme = computed(() => theme.name);

import { ref } from 'vue'

const option = ref({
  tooltip: {
    trigger: 'item'
  },
  series: [
    {
      name: 'Access From',
      type: 'pie',
      radius: '50%',
      data: [
        { value: 1048, name: 'Search Engine' },
        { value: 735, name: 'Direct' },
        { value: 580, name: 'Email' },
        { value: 484, name: 'Union Ads' },
        { value: 300, name: 'Video Ads' }
      ],
      emphasis: {
        itemStyle: {
          shadowBlur: 10,
          shadowOffsetX: 0,
          shadowColor: 'rgba(0, 0, 0, 0.5)'
        }
      }
    }
  ]
})
</script>

<style scoped>
.chart {
  height: 260px;
  width: 100%;
}
</style>
