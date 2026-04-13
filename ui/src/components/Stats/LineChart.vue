<template>
  <VChart class="chart" :option="option" theme="light" :loading="loading" autoresize />
</template>

<script setup>
import VChart, { THEME_KEY } from "vue-echarts";
import { use } from "echarts/core";
import { LineChart } from "echarts/charts";
import {
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  GridComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

use([
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  GridComponent,
  LineChart,
  CanvasRenderer,
]);
import { useTheme } from "vuetify";
const theme = useTheme();
const currentTheme = computed(() => theme.name);

import { ref } from 'vue'

const option = ref({
  tooltip: {
    trigger: 'axis'
  },
  grid: {
    top: '4%',
    left: '0%',
    right: '0%',
    bottom: '10%',
    containLabel: true
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: []
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      name: 'Income',
      type: 'line',
      smooth: true,
      data: []
    }
  ]
})

async function fetchIncome() {
  const res = await fetch('http://localhost:4123/income')
  const data = await res.json()

  const branches = ['Branch A', 'Branch B', 'Branch C', 'Branch D']

  option.value.xAxis.data = data.income.map(i => i.date)

  option.value.series = branches.map(b => ({
    name: b,
    type: 'line',
    smooth: true,
    data: data.income.map(i => i.amounts[b])
  }))
}

// initial load
fetchIncome()

// live updates every 5s
// setInterval(fetchIncome, 5000)
</script>

<style scoped>
.chart {
  height: 220px;
  width: 100%;
}
</style>
