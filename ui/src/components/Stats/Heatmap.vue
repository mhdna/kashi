<template>
  <div class="heatmap-wrap">
    <v-heatmap
      :cell-size="15"
      :columns="weeks"
      gap="3"
      hide-column-headers
      item-column="week"
      item-row="weekday"
      item-value="value"
      :items="items"
      :rows="weekdays"
      :thresholds="thresholds"
    />
  </div>
</template>

<script setup>
  import { computed } from 'vue'
  import { ymd } from '@/utils/dates'

  const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

  // Linear ramp from a faint primary tint to the solid primary, matching the
  // old echarts calendar heatmap's single-hue intensity scale.
  const thresholds = {
    from: { min: 0, color: '#C5CAE9' },
    to: { min: 10_000, color: '#3F51B5' },
  }

  // One row per day of the given year, pivoted as weekday x ISO-ish week so
  // v-heatmap renders the familiar contribution-calendar grid.
  function buildYear (year) {
    const start = new Date(`${year}-01-01T00:00:00`)
    const end = new Date(`${Number(year) + 1}-01-01T00:00:00`)
    const rows = []
    // Monday-based weekday index (0 = Mon ... 6 = Sun).
    const mondayIndex = d => (d.getDay() + 6) % 7
    let week = 1
    for (const t = new Date(start); t < end; t.setDate(t.getDate() + 1)) {
      if (t > start && mondayIndex(t) === 0) week++
      rows.push({
        date: ymd(t),
        weekday: weekdays[mondayIndex(t)],
        week,
        value: Math.floor(Math.random() * 10_000),
      })
    }
    return rows
  }

  const items = computed(() => buildYear(new Date().getFullYear()))
  const weeks = computed(() => [...new Set(items.value.map(r => r.week))])
</script>

<style scoped>
.heatmap-wrap {
  width: 100%;
  overflow-x: auto;
}
</style>
