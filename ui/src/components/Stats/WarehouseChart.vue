<template>
  <div ref="chartWrap" class="chart-wrap">
    <v-chart
      autoresize
      class="chart"
      :loading="loading"
      :option="option"
      theme="dark"
    />
  </div>
</template>

<script lang="ts" setup>
  import { onMounted, ref, watch } from 'vue'
  import VChart from 'vue-echarts'
  import { useNotify } from '@/composables/useNotify'
  import { useWarehouseTreemap } from '@/composables/useWarehouseTreemap'

  const props = defineProps({
    // inventory id, or 'all' for every location
    location: {
      type: String,
      default: 'all',
    },
  })

  const notify = useNotify()
  const { loading, fetchRows, buildTreemapData } = useWarehouseTreemap()

  const chartWrap = ref<HTMLDivElement | null>(null)
  const option = ref({})

  function createChartOption (data: any[]) {
    return {
      tooltip: {
        appendTo: 'body',
        formatter: (info: any) => {
          const { name, value, data: itemData } = info
          let content = `<strong>${name}</strong>`
          if (itemData?.code) {
            content += `<br/>Code: ${itemData.code}`
          }
          if (value) {
            content += `<br/>Quantity: ${value}`
          }
          if (itemData?.location) {
            content += `<br/>Location: ${itemData.location}`
          }
          return content
        },
      },
      series: [
        {
          name: 'Inventory',
          type: 'treemap',
          visibleMin: 1,
          data: data,
          leafDepth: null,
          width: '100%',
          height: '100%',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          roam: false,
          nodeClick: 'link',
          label: {
            show: true,
            formatter: (params: any) => {
              if (params.value) {
                return `${params.name}\n${params.value}`
              }
              return params.name
            },
            fontSize: 14,
            fontWeight: 'bold',
          },
          upperLabel: {
            backgroundColor: '#000',
            show: true,
            height: 18,
            color: '#fff',
            fontSize: 13,
            fontWeight: 'bold',
          },
          itemStyle: {
            borderColor: '#fff',
            borderWidth: 0,
            gapWidth: 2,
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
              borderColor: '#FFD700',
              borderWidth: 3,
            },
            label: {
              fontSize: 16,
              fontWeight: 'bold',
            },
          },
          levels: [
            {
              itemStyle: {
                borderColor: '#000',
                borderWidth: 0,
                gapWidth: 2,
              },
              upperLabel: {
                show: true,
              },
            },
            {
              itemStyle: {
                borderColor: '#000',
                borderWidth: 0,
                gapWidth: 2,
              },
              label: {
                fontSize: 12,
              },
            },
          ],
        },
      ],
    }
  }

  async function refresh () {
    try {
      await fetchRows()
    } catch (error: any) {
      notify.error(error.message)
    }
    option.value = createChartOption(buildTreemapData(props.location))
  }

  onMounted(refresh)
  watch(() => props.location, () => {
    option.value = createChartOption(buildTreemapData(props.location))
  })
</script>

<style scoped>
.chart-wrap {
  height: 100%;
  width: 100%;
}

.chart {
  width: 100%;
  height: 100%;
  cursor: pointer;
}
</style>
