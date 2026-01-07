<template>
  <VChart class="chart" :option="option" />
</template>

<script setup>
import Vchart from "vue-echarts";
import { use } from "echarts/core";
import { HeatmapChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  VisualMapComponent,
  CalendarComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { time } from "echarts/core";

use([
  TitleComponent,
  TooltipComponent,
  VisualMapComponent,
  CalendarComponent,
  HeatmapChart,
  CanvasRenderer,
]);
function getVirtualData(year) {
  const date = +time.parse(year + "-01-01");
  const end = +time.parse(+year + 1 + "-01-01");
  const dayTime = 3600 * 24 * 1000;
  const data = [];
  for (let t = date; t < end; t += dayTime) {
    data.push([
      time.format(t, "{yyyy}-{MM}-{dd}", false),
      Math.floor(Math.random() * 10000),
    ]);
  }
  return data;
}

const option = {
  title: {
    top: 30,
    left: "center",
    text: "Daily Step Count",
  },
  tooltip: {},
  visualMap: {
    min: 0,
    max: 10000,
    type: "piecewise",
    orient: "horizontal",
    left: "center",
    top: 65,
  },
  calendar: {
    top: 120,
    left: 30,
    right: 30,
    cellSize: ["auto", 13],
    range: "2016",
    itemStyle: {
      borderWidth: 0.5,
    },
    yearLabel: { show: false },
  },
  series: {
    type: "heatmap",
    coordinateSystem: "calendar",
    data: getVirtualData("2016"),
  },
};
</script>
