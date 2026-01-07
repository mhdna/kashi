<template>
    <VChart class="chart" :option="option" />
</template>

<script setup>
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { PieChart } from "echarts/charts";
import {
    TitleComponent,
    TooltipComponent,
    LegendComponent,
} from "echarts/components";
import VChart, { THEME_KEY } from "vue-echarts";
import { ref, provide } from "vue";

use([
    CanvasRenderer,
    PieChart,
    TitleComponent,
    TooltipComponent,
    LegendComponent,
]);

provide(THEME_KEY, "dark");

const colorList = [
    '#4f81bd',
    '#c0504d',
    '#9bbb59',
    '#604a7b',
    '#948a54',
    '#e46c0b'
];
const data = [
    [10, 16, 3, 'A'],
    [16, 18, 15, 'B'],
    [18, 26, 12, 'C'],
    [26, 32, 22, 'D'],
    [32, 56, 7, 'E'],
    [56, 62, 17, 'F']
].map(function (item, index) {
    return {
        value: item,
        itemStyle: {
            color: colorList[index]
        }
    };
});

const option = ref({
    title: {
        text: 'Profit',
        left: 'center'
    },
    tooltip: {},
    xAxis: {
        scale: true
    },
    yAxis: {},
    series: [
        {
            type: 'custom',
            renderItem: function (params, api) {
                var yValue = api.value(2);
                var start = api.coord([api.value(0), yValue]);
                var size = api.size([api.value(1) - api.value(0), yValue]);
                var style = api.style();
                return {
                    type: 'rect',
                    shape: {
                        x: start[0],
                        y: start[1],
                        width: size[0],
                        height: size[1]
                    },
                    style: style
                };
            },
            label: {
                show: true,
                position: 'top'
            },
            dimensions: ['from', 'to', 'profit'],
            encode: {
                x: [0, 1],
                y: 2,
                tooltip: [0, 1, 2],
                itemName: 3
            },
            data: data
        }
    ]
});
</script>

<style scoped>
.chart {
    height: 400px;
}
</style>