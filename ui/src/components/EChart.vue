<template>
  <VChart ref="chart" v-bind="$attrs" />
</template>

<script setup>
  // The one place the chart library gets pulled in: every chart imports this
  // instead of 'vue-echarts', so echarts (~840KB) only loads with a page that
  // actually draws a chart, not with the app shell.
  import { computed, onBeforeUnmount, onMounted, provide, ref } from 'vue'
  import VChart, { THEME_KEY } from 'vue-echarts'
  import { useTheme } from 'vuetify'
  import '@/plugins/echarts'

  defineOptions({ inheritAttrs: false })

  // ECharts theme for every chart under here: flips with Vuetify's
  // light/dark mode (vue-echarts calls setTheme(), no re-mount). A chart
  // that sets its own `theme` prop still wins. Both names are registered in
  // plugins/echarts.js.
  const vuetifyTheme = useTheme()
  provide(THEME_KEY, computed(() => (vuetifyTheme.global.current.value.dark ? 'kashi-dark' : 'kashi-light')))

  // On a touch screen a tap (or a scroll that starts on the chart) shows the
  // tooltip, and with no mouse to leave the chart nothing ever hides it. Hide
  // it once the page scrolls or the user touches somewhere else; tapping a
  // point to read it still works.
  const chart = ref(null)
  const touchOnly = typeof window !== 'undefined' && window.matchMedia('(hover: none)').matches

  function hideTip () {
    chart.value?.dispatchAction({ type: 'hideTip' })
  }

  function onTouchStart (event) {
    if (!chart.value?.$el?.contains(event.target)) hideTip()
  }

  onMounted(() => {
    if (!touchOnly) return
    // Capture: the page scrolls inside v-main, not the window.
    window.addEventListener('scroll', hideTip, { capture: true, passive: true })
    document.addEventListener('touchstart', onTouchStart, { passive: true })
  })

  onBeforeUnmount(() => {
    window.removeEventListener('scroll', hideTip, { capture: true })
    document.removeEventListener('touchstart', onTouchStart)
  })
</script>
