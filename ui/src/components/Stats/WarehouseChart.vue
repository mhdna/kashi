<template>
  <v-chart
    autoresize
    class="chart"
    :loading="loading"
    :option="option"
    theme="dark"
  />
</template>

<script setup lang="ts">
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
  const option = ref({})

  const formatMoney = (cents: number) => '$ ' + Math.round(Number(cents || 0) / 100).toLocaleString('en-US')
  const formatCount = (n: number) => Number(n || 0).toLocaleString('en-US') + ' units'

  function tooltipFormatter (info: any) {
    const d = info.data || {}
    const rows = [
      `<div style="font-weight:600;margin-bottom:4px">${info.name}</div>`,
      `Book Value:&nbsp;&nbsp;${formatMoney(info.value)}`,
    ]
    if (d.qty !== undefined) rows.push(`Quantity:&nbsp;&nbsp;${formatCount(d.qty)}`)
    if (d.code) rows.push(`Code:&nbsp;&nbsp;${d.code}`)
    if (d.location) rows.push(`Location:&nbsp;&nbsp;${d.location}`)
    return rows.join('<br>')
  }

  function buildOption () {
    const data = buildTreemapData(props.location)
    option.value = {
      backgroundColor: 'transparent',
      tooltip: { appendTo: 'body' },
      series: [
        {
          type: 'treemap',
          animationDurationUpdate: 180,
          animationEasing: 'cubicOut',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          width: '100%',
          height: '100%',
          roam: false,
          nodeClick: 'zoomToNode',
          data,
          leafDepth: null,
          tooltip: { formatter: tooltipFormatter },
          breadcrumb: {
            show: true,
            top: 4,
            left: 4,
            height: 24,
            emptyItemStyle: { color: 'rgba(0,0,0,0.25)' },
            itemStyle: { color: 'rgba(0,0,0,0.55)', textStyle: { color: '#fff' } },
          },
          // Same rich-text label for every node -- category, color group
          // (Packs only) and leaf product alike -- matching the Assets
          // Treemap's look exactly. Groups render it because their upperLabel
          // is disabled below; leaves render it because they have no children.
          label: {
            position: 'insideTopLeft',
            color: '#fff',
            fontSize: 12,
            lineHeight: 16,
            textBorderColor: 'rgba(0,0,0,0.5)',
            textBorderWidth: 2,
            formatter (params: any) {
              const arr = [`{name|${params.name}}`, `{cost|${formatMoney(params.value)}}`]
              if (params.data && params.data.code) arr.push(`{code|${params.data.code}}`)
              return arr.join('\n')
            },
            rich: {
              name: { fontSize: 12, color: '#fff', lineHeight: 16 },
              cost: {
                fontSize: 20,
                fontWeight: 'bold',
                color: '#FFEB3B',
                lineHeight: 26,
                textBorderColor: 'rgba(0,0,0,0.55)',
                textBorderWidth: 2,
              },
              code: { fontSize: 11, color: 'rgba(255,255,255,0.8)', lineHeight: 15 },
            },
          },
          upperLabel: { show: false },
          itemStyle: { borderColor: 'rgba(0,0,0,0.4)', borderWidth: 1, gapWidth: 2 },
          levels: [
            {
              // categories (Packs, Baby, Kids, Men, Lady, Unidentified)
              colorMappingBy: 'id',
              itemStyle: { borderWidth: 3, gapWidth: 3, borderColor: 'rgba(0,0,0,0.4)' },
              upperLabel: { show: false },
            },
            {
              // color groups inside Packs (e.g. "Black/Black"), or the leaf
              // product directly when a category has no color data
              colorSaturation: [0.3, 0.5],
              itemStyle: { gapWidth: 1, borderColorSaturation: 0.6 },
              upperLabel: { show: false },
            },
            {
              // leaf product inside a color group
              colorSaturation: [0.45, 0.65],
              itemStyle: { gapWidth: 1, borderColorSaturation: 0.7 },
            },
          ],
          emphasis: {
            itemStyle: {
              borderColor: '#FFD700',
              borderWidth: 3,
            },
          },
        },
      ],
    }
  }

  onMounted(async () => {
    try {
      await fetchRows()
    } catch (error: any) {
      notify.error(error.message)
    }
    buildOption()
  })

  watch(() => props.location, buildOption)
</script>

<style scoped>
.chart {
  height: 100%;
  width: 100%;
  cursor: pointer;
}
</style>
