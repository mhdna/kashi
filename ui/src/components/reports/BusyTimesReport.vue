<template>
  <RptKpis :items="kpis" />

  <RptSection note="Average sales per day at each weekday and hour, by the branch's own clock." title="Sales by weekday and hour">
    <RptChart :height="240" :option="heatChart" />
  </RptSection>

  <RptSection title="Through the day and the week">
    <RptGrid>
      <RptCard title="Sales per hour (average per day)">
        <RptChart :height="200" :option="hourCountChart" />
      </RptCard>
      <RptCard title="Net sales per hour (average per day)">
        <RptChart :height="200" :option="hourMoneyChart" />
      </RptCard>
      <RptCard title="Sales per weekday (average per day)">
        <RptChart :height="200" :option="weekdayCountChart" />
      </RptCard>
      <RptCard title="Net sales per weekday (average per day)">
        <RptChart :height="200" :option="weekdayMoneyChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="visitors > 0" note="Customers in from the people counters, and the share of them who bought." title="Footfall and conversion">
    <RptChart :height="200" :option="visitorsChart" />
    <RptChart :height="180" :option="conversionChart" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, csvSection, day, dayLabel, daysBetween, heatmapOption, hourLabel, int, lineOption, num1, pct, ratio, weekdayCounts, WEEKDAYS } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'

  // When the stores are busy: sales by weekday and hour, and footfall with
  // conversion where the tills count people.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)
  const counts = computed(() => weekdayCounts(props.ctx.from, props.ctx.to))
  const nDays = computed(() => counts.value.reduce((a, b) => a + b, 0) || 1)

  const cells = computed(() => data.value.weekday_hour.map(c => ({ ...c, perDay: c.sales_count / (counts.value[c.weekday] || 1) })))
  const hours = computed(() => {
    const used = cells.value.map(c => c.hour)
    if (used.length === 0) {return []}
    const lo = Math.min(...used)
    return Array.from({ length: Math.max(...used) - lo + 1 }, (_, i) => lo + i)
  })
  const byHour = computed(() => hours.value.map(h => {
    const list = cells.value.filter(c => c.hour === h)
    return { hour: hourLabel(h), sales: list.reduce((s, c) => s + c.sales_count, 0) / nDays.value, money: list.reduce((s, c) => s + c.sales_total, 0) / nDays.value }
  }))
  const byWeekday = computed(() => WEEKDAYS.map((name, i) => {
    const list = cells.value.filter(c => c.weekday === i + 1)
    const n = counts.value[i + 1] || 1
    return { weekday: name, sales: list.reduce((s, c) => s + c.sales_count, 0) / n, money: list.reduce((s, c) => s + c.sales_total, 0) / n }
  }))

  const visitors = computed(() => data.value.daily.reduce((sum, d) => sum + d.visitors, 0))
  const kpis = computed(() => {
    const busiestHour = byHour.value.toSorted((a, b) => b.sales - a.sales)[0]
    const busiestDay = byWeekday.value.toSorted((a, b) => b.money - a.money)[0]
    const quietDay = byWeekday.value.toSorted((a, b) => a.money - b.money)[0]
    const items = [
      { label: 'Busiest hour', value: busiestHour?.hour ?? '—', hint: busiestHour ? `${num1(busiestHour.sales)} sales a day` : '' },
      { label: 'Busiest weekday', value: busiestDay?.weekday ?? '—' },
      { label: 'Quietest weekday', value: quietDay?.weekday ?? '—' },
      { label: 'Sales per day', value: num1(data.value.summary.sales_count / nDays.value) },
    ]
    if (visitors.value > 0) {
      items.push(
        { label: 'Visitors', value: int(visitors.value), hint: `${num1(visitors.value / nDays.value)} a day` },
        { label: 'Conversion', value: pct(ratio(data.value.summary.sales_count, visitors.value)) },
      )
    }
    return items
  })

  const heatChart = computed(() => heatmapOption({ cells: cells.value.map(c => ({ weekday: c.weekday, hour: c.hour, value: Math.round(c.perDay * 10) / 10 })), kind: 'int' }))
  const hourCountChart = computed(() => barOption({ categories: byHour.value.map(h => h.hour), series: [{ name: 'Sales', data: byHour.value.map(h => Math.round(h.sales * 10) / 10) }] }))
  const hourMoneyChart = computed(() => barOption({ categories: byHour.value.map(h => h.hour), series: [{ name: 'Net sales', data: byHour.value.map(h => h.money / 100) }], kind: 'money' }))
  const weekdayCountChart = computed(() => barOption({ categories: WEEKDAYS, series: [{ name: 'Sales', data: byWeekday.value.map(w => Math.round(w.sales * 10) / 10) }], labels: true }))
  const weekdayMoneyChart = computed(() => barOption({ categories: WEEKDAYS, series: [{ name: 'Net sales', data: byWeekday.value.map(w => w.money / 100) }], kind: 'money', labels: true }))

  const days = computed(() => {
    const byDay = new Map(data.value.daily.map(d => [day(d.day), d]))
    return daysBetween(props.ctx.from, props.ctx.to).map(d => ({ day: d, visitors: byDay.get(d)?.visitors ?? 0, sales: byDay.get(d)?.sales_count ?? 0 }))
  })
  const labels = computed(() => days.value.map(d => dayLabel(d.day, { day: 'numeric', month: 'short' })))
  const visitorsChart = computed(() => barOption({ categories: labels.value, series: [{ name: 'Visitors', data: days.value.map(d => d.visitors) }, { name: 'Sales', data: days.value.map(d => d.sales) }] }))
  const conversionChart = computed(() => lineOption({ categories: labels.value, series: [{ name: 'Conversion %', data: days.value.map(d => (d.visitors > 0 ? Math.round(d.sales / d.visitors * 1000) / 10 : null)) }] }))

  defineExpose({
    csv: () => [
      csvSection('Weekday x hour', [
        { key: 'weekday', title: 'Weekday', value: r => WEEKDAYS[r.weekday - 1] },
        { key: 'hour', title: 'Hour', value: r => hourLabel(r.hour) },
        { key: 'sales_count', title: 'Sales', type: 'int' },
        { key: 'sales_total', title: 'Sales total', type: 'money' },
        { key: 'perDay', title: 'Sales per day', type: 'num1' },
      ], cells.value),
      csvSection('Days', [{ key: 'day', title: 'Day', type: 'date' }, { key: 'visitors', title: 'Visitors', type: 'int' }, { key: 'sales', title: 'Sales', type: 'int' }], days.value),
    ],
  })
</script>
