<template>
  <v-chart autoresize class="chart" :loading="loading" :option="option" />
</template>

<script setup lang="ts">
  import { onMounted, ref, watch } from 'vue'
  import VChart from 'vue-echarts'
  import { useAssetsTreemap } from '@/composables/useAssetsTreemap'
  import { useNotify } from '@/composables/useNotify'

  const props = defineProps({
    // inventory id, or 'all' for every location
    location: {
      type: String,
      default: 'all',
    },
  })

  const notify = useNotify()
  const { loading, fetchRows, buildTreemapData } = useAssetsTreemap()
  const option = ref({})

  const formatMoney = cents => '$ ' + Math.round(Number(cents || 0) / 100).toLocaleString('en-US')
  const formatCount = n => Number(n || 0).toLocaleString('en-US') + ' units'

  function tooltipFormatter (info) {
    const d = info.data || {}
    const rows = [
      `<div style="font-weight:600;margin-bottom:4px">${info.name}</div>`,
      `Book Value:&nbsp;&nbsp;${formatMoney(info.value)}`,
    ]
    if (d.qty !== undefined) rows.push(`Count:&nbsp;&nbsp;${formatCount(d.qty)}`)
    if (d.code) rows.push(`Asset Tag:&nbsp;&nbsp;${d.code}`)
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
          // shorten the zoom transition -- the default easing feels laggy on a
          // full-viewport treemap
          animationDurationUpdate: 180,
          animationEasing: 'cubicOut',
          // fill the whole component, no margins
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          width: '100%',
          height: '100%',
          roam: false,
          nodeClick: 'zoomToNode',
          data,
          // keep the whole tree reachable so zooming back out always works
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
          // Leaf labels sit on the saturated group color; a thin stroke keeps
          // white text readable without the cost of a blurred text shadow
          // (blur repaints on every zoom frame and is the main lag source).
          label: {
            position: 'insideTopLeft',
            color: '#fff',
            fontSize: 12,
            lineHeight: 16,
            textBorderColor: 'rgba(0,0,0,0.5)',
            textBorderWidth: 2,
            formatter (params) {
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
          // Group header bar: dark strip so the white title stays readable
          // whatever the page theme is. Needs its own formatter, otherwise it
          // inherits label.formatter.
          upperLabel: {
            show: true,
            height: 24,
            color: '#fff',
            fontSize: 13,
            fontWeight: 'bold',
            backgroundColor: 'rgba(0,0,0,0.55)',
            formatter (params) {
              // the invisible root node has no name/value -- skip its header
              if (!params.name) return ''
              return `  ${params.name}  ${formatMoney(params.value)}`
            },
          },
          itemStyle: { borderColor: 'rgba(0,0,0,0.4)', borderWidth: 1, gapWidth: 2 },
          levels: [
            {
              colorMappingBy: 'id',
              itemStyle: { borderWidth: 3, gapWidth: 3, borderColor: 'rgba(0,0,0,0.4)' },
              upperLabel: { show: false },
            },
            {
              colorSaturation: [0.35, 0.5],
              itemStyle: { gapWidth: 1, borderColorSaturation: 0.6 },
            },
          ],
          emphasis: {
            // no shadowBlur here -- blurred shadows on large rects are a major
            // repaint cost during the zoom animation
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
    } catch (error) {
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
