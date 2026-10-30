<template>
  <RptKpis :items="kpis" />

  <RptSection v-if="has('best')" :title="t('Best customers (top {n})', { n: topN })">
    <RptTable :columns="topColumns" :limit="topN" numbered :rows="clients" />
  </RptSection>

  <RptSection v-if="has('frequent')" :title="t('Most frequent visitors')">
    <RptTable
      :columns="frequentColumns"
      compact
      :limit="10"
      numbered
      :rows="frequent"
    />
  </RptSection>

  <RptSection v-if="has('who')" :break-before="has('best')" :title="t('Who they are')">
    <RptGrid>
      <RptCard :note="t('Customers by how many times they bought in the period.')" :title="t('Visits per customer')">
        <RptChart :height="200" :option="visitsChart" />
      </RptCard>
      <RptCard :note="t('Customers by what they spent in the period, net of returns.')" :title="t('Spend per customer')">
        <RptChart :height="200" :option="spendChart" />
      </RptCard>
      <RptCard :note="t('New = first purchase ever was in this period.')" :title="t('New and returning customers')">
        <RptChart :height="200" :option="newChart" />
      </RptCard>
      <RptCard :title="t('Customers by home branch')">
        <RptChart :height="200" :option="homeChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection
    v-if="has('when')"
    :note="t('Sales with a named customer, by the time on the till. Averages are per day of that weekday in the period.')"
    :title="t('When customers come')"
  >
    <RptChart :height="230" :option="heatChart" />
    <RptGrid>
      <RptCard :title="t('By weekday (average per day)')">
        <RptChart :height="190" :option="weekdayChart" />
      </RptCard>
      <RptCard :title="t('By hour (total)')">
        <RptChart :height="190" :option="hourChart" />
      </RptCard>
    </RptGrid>
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { barOption, csvSection, donutOption, heatmapOption, hourLabel, int, money, num1, pct, ratio, weekdayCounts, weekdays } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // The period's customers: the best and most loyal, how many are new, how
  // often they come and spend, and when. Parts: best, frequent, who, when.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const topN = computed(() => props.ctx.options.topN)

  const fromAt = computed(() => new Date(`${props.ctx.from}T00:00:00`))
  const clients = computed(() => data.value.clients.map(c => ({
    ...c,
    is_new: new Date(c.first_purchase_at) >= fromAt.value,
    avg: ratio(c.net_total, c.visits),
  })))
  const frequent = computed(() => clients.value.toSorted((a, b) => b.visits - a.visits || b.net_total - a.net_total))

  const count = computed(() => clients.value.length)
  const newCount = computed(() => clients.value.filter(c => c.is_new).length)
  const repeat = computed(() => clients.value.filter(c => c.visits >= 2).length)
  const spent = computed(() => clients.value.reduce((sum, c) => sum + c.net_total, 0))
  const visits = computed(() => clients.value.reduce((sum, c) => sum + c.visits, 0))

  const kpis = computed(() => {
    const s = data.value.summary
    return [
      { label: t('Customers served'), value: int(count.value) },
      { label: t('New customers'), value: int(newCount.value), hint: pct(ratio(newCount.value, count.value)) },
      { label: t('Returning customers'), value: int(count.value - newCount.value) },
      { label: t('Came back 2+ times'), value: int(repeat.value), hint: pct(ratio(repeat.value, count.value)) },
      { label: t('Spend per customer'), value: money(ratio(spent.value, count.value)) },
      { label: t('Visits per customer'), value: num1(ratio(visits.value, count.value)) },
      { label: t('Sales with a customer'), value: pct(ratio(s.named_sales_count, s.sales_count)), hint: t('{n} of {total}', { n: int(s.named_sales_count), total: int(s.sales_count) }) },
      { label: t('Named customers\' sales'), value: money(spent.value), hint: t('{pct} of net sales', { pct: pct(ratio(spent.value, s.net_total)) }) },
    ]
  })

  const newTag = t('★ new')
  const topColumns = [
    { key: 'name', title: t('Customer'), value: r => (r.is_new ? `${r.name}  ${newTag}` : r.name) },
    { key: 'phone', title: t('Phone') },
    { key: 'home_branch', title: t('Home branch') },
    { key: 'visits', title: t('Visits'), type: 'int' },
    { key: 'items', title: t('Items'), type: 'int' },
    { key: 'net_total', title: t('Spent'), type: 'money' },
    { key: 'avg', title: t('Per visit'), type: 'money' },
    { key: 'last_at', title: t('Last visit'), type: 'datetime' },
    { key: 'lifetime_total', title: t('Lifetime'), type: 'money' },
  ]
  const frequentColumns = [
    { key: 'name', title: t('Customer') },
    { key: 'phone', title: t('Phone') },
    { key: 'visits', title: t('Visits'), type: 'int' },
    { key: 'net_total', title: t('Spent'), type: 'money' },
    { key: 'last_at', title: t('Last visit'), type: 'datetime' },
  ]

  function buckets (edges, value, label) {
    const counts = edges.map(() => 0)
    for (const c of clients.value) {
      const i = edges.findLastIndex(edge => value(c) >= edge)
      counts[Math.max(0, i)]++
    }
    return barOption({ categories: edges.map((edge, i) => label(edge, edges[i + 1])), series: [{ name: t('Customers'), data: counts }], labels: true })
  }
  const visitsChart = computed(() => buckets([1, 2, 3, 4, 6, 10], c => c.visits, (a, b) => (b === undefined ? `${a}+` : (b - a === 1 ? `${a}` : `${a}–${b - 1}`))))
  const spendChart = computed(() => buckets([0, 25, 50, 100, 200, 500, 1000], c => c.net_total / 100, (a, b) => (b === undefined ? `$${a}+` : `$${a}–${b}`)))
  const newChart = computed(() => donutOption({ items: [{ name: t('Returning'), value: count.value - newCount.value }, { name: t('New'), value: newCount.value }], kind: 'int' }))
  const homeChart = computed(() => {
    const byBranch = new Map()
    for (const c of clients.value) {
      const name = c.home_branch || t('Unknown')
      byBranch.set(name, (byBranch.get(name) ?? 0) + 1)
    }
    return donutOption({ items: [...byBranch].map(([name, value]) => ({ name, value })), kind: 'int' })
  })

  const named = computed(() => data.value.weekday_hour.map(c => ({ weekday: c.weekday, hour: c.hour, value: c.named_count })))
  const heatChart = computed(() => heatmapOption({ cells: named.value }))
  const weekdayChart = computed(() => {
    const counts = weekdayCounts(props.ctx.from, props.ctx.to)
    const totals = Array.from({ length: 8 }, () => 0)
    for (const c of named.value) {totals[c.weekday] += c.value}
    return barOption({ categories: weekdays(), series: [{ name: t('Customer sales per day'), data: weekdays().map((_, i) => Math.round(totals[i + 1] / (counts[i + 1] || 1) * 10) / 10) }], labels: true })
  })
  const hourChart = computed(() => {
    const totals = new Map()
    for (const c of named.value) {totals.set(c.hour, (totals.get(c.hour) ?? 0) + c.value)}
    const used = [...totals.keys()].filter(h => totals.get(h) > 0)
    const lo = used.length > 0 ? Math.min(...used) : 9
    const hi = used.length > 0 ? Math.max(...used) : 21
    const hrs = Array.from({ length: hi - lo + 1 }, (_, i) => lo + i)
    return barOption({ categories: hrs.map(h => hourLabel(h)), series: [{ name: t('Customer sales'), data: hrs.map(h => totals.get(h) ?? 0) }] })
  })

  defineExpose({
    csv: () => (has('best') || has('frequent') || has('who')
      ? [csvSection(t('Customers'), [
        { key: 'name', title: t('Customer') },
        ...topColumns.slice(1),
        { key: 'is_new', title: t('New'), value: r => (r.is_new ? '✓' : '') },
        { key: 'first_purchase_at', title: t('First purchase ever'), type: 'datetime' },
      ], clients.value)]
      : []),
  })
</script>
