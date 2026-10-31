<template>
  <RptKpis v-if="has('kpis.customers')" :items="kpis" part="kpis.customers" />

  <RptSection v-if="has('best')" part="best" :title="t('Best customers (top {n})', { n: topN })">
    <RptTable :columns="topColumns" :limit="topN" numbered :rows="clients" />
  </RptSection>

  <RptSection v-if="has('frequent')" part="frequent" :title="t('Most frequent visitors')">
    <RptTable
      :columns="frequentColumns"
      compact
      :limit="10"
      numbered
      :rows="frequent"
    />
  </RptSection>

  <RptSection
    v-if="has('who.visits') || has('who.spend') || has('who.home') || has('who.top')"
    :break-before="has('best')"
    :part="['who.visits', 'who.spend', 'who.home', 'who.top']"
    :title="t('Who they are')"
  >
    <RptCard v-if="has('who.visits')" :note="t('Customers by how many times they bought in the period.')" part="who.visits" :title="t('By number of visits')">
      <RptTable :columns="visitColumns" compact :rows="visitRows" />
    </RptCard>
    <RptCard v-if="has('who.spend')" :note="t('Customers by what they spent in the period, net of returns.')" part="who.spend" :title="t('By amount spent')">
      <RptTable :columns="spendColumns" compact :rows="spendRows" />
    </RptCard>
    <RptCard v-if="has('who.top') && count >= 10" :note="t('How much of the period\'s sales the best customers carry.')" part="who.top" :title="t('How much the top customers bring')">
      <RptTable :columns="topShareColumns" compact :rows="topShareRows" />
    </RptCard>
    <RptCard v-if="has('who.home') && homeRows.length > 1" part="who.home" :title="t('Customers by home branch')">
      <RptTable :columns="homeColumns" compact :rows="homeRows" />
    </RptCard>
  </RptSection>

  <RptSection
    v-if="has('when.heatmap') || has('when.busiest')"
    :note="t('Sales with a named customer, by weekday and hour on the till.')"
    :part="['when.heatmap', 'when.busiest']"
    :title="t('When customers come')"
  >
    <RptGrid template="3fr 2fr">
      <RptChart v-if="has('when.heatmap')" :height="250" :option="heatChart" />
      <RptCard v-if="has('when.busiest')" part="when.busiest" :title="t('Busiest times')">
        <RptTable :columns="slotColumns" compact :rows="topSlots" />
      </RptCard>
    </RptGrid>
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { csvSection, heatmapOption, hourLabel, int, money, num1, pct, ratio, weekdays } from './reportKit'
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
  const newSpent = computed(() => clients.value.filter(c => c.is_new).reduce((sum, c) => sum + c.net_total, 0))

  const kpis = computed(() => {
    const s = data.value.summary
    return [
      { label: t('Customers served'), value: int(count.value) },
      { label: t('New customers'), value: int(newCount.value), hint: t('{pct} · spent {amount}', { pct: pct(ratio(newCount.value, count.value)), amount: money(newSpent.value) }) },
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

  // Customers grouped by a measure, with what each group spent: the share
  // of customers next to the share of money says who carries the sales.
  function bucketRows (edges, value, label) {
    const rows = edges.map((edge, i) => ({ group: label(edge, edges[i + 1]), customers: 0, spent: 0 }))
    for (const c of clients.value) {
      const row = rows[Math.max(0, edges.findLastIndex(edge => value(c) >= edge))]
      row.customers++
      row.spent += c.net_total
    }
    return rows.filter(r => r.customers > 0).map(r => ({
      ...r,
      customerShare: ratio(r.customers, count.value),
      spentShare: ratio(r.spent, spent.value),
      perCustomer: ratio(r.spent, r.customers),
    }))
  }
  const visitRows = computed(() => bucketRows([1, 2, 3, 4, 6, 10], c => c.visits, (a, b) => (b === undefined ? `${a}+` : (b - a === 1 ? `${a}` : `${a}–${b - 1}`))))
  const spendRows = computed(() => bucketRows([0, 25, 50, 100, 200, 500, 1000], c => c.net_total / 100, (a, b) => (b === undefined ? `$${a}+` : `$${a}–${b}`)))
  function groupColumns (title) {
    return [
      { key: 'group', title },
      { key: 'customers', title: t('Customers'), type: 'int' },
      { key: 'customerShare', title: t('% of customers'), type: 'pct' },
      { key: 'spent', title: t('Spent'), type: 'money' },
      { key: 'spentShare', title: t('% of spend'), type: 'pct', bar: true },
      { key: 'perCustomer', title: t('Per customer'), type: 'money' },
    ]
  }
  const visitColumns = groupColumns(t('Visits'))
  const spendColumns = groupColumns(t('Spent'))

  // The best customers' weight: the top 10, then the top 10%, 20% and half.
  const topShareRows = computed(() => {
    const n = count.value
    const groups = [
      { group: t('Top 10 customers'), size: 10 },
      { group: t('Top 10%'), size: Math.ceil(n * 0.1) },
      { group: t('Top 20%'), size: Math.ceil(n * 0.2) },
      { group: t('Top half'), size: Math.ceil(n * 0.5) },
    ]
    return groups.map(g => {
      const total = clients.value.slice(0, g.size).reduce((sum, c) => sum + c.net_total, 0)
      return { ...g, spent: total, spentShare: ratio(total, spent.value), salesShare: ratio(total, data.value.summary.net_total) }
    })
  })
  const topShareColumns = [
    { key: 'group', title: t('Customers') },
    { key: 'size', title: t('How many'), type: 'int' },
    { key: 'spent', title: t('Spent'), type: 'money' },
    { key: 'spentShare', title: t('% of customers\' spend'), type: 'pct', bar: true },
    { key: 'salesShare', title: t('% of net sales'), type: 'pct' },
  ]

  const homeRows = computed(() => {
    const byBranch = new Map()
    for (const c of clients.value) {
      const name = c.home_branch || t('Unknown')
      const row = byBranch.get(name) ?? { branch: name, customers: 0, spent: 0, isNew: 0 }
      row.customers++
      row.spent += c.net_total
      row.isNew += c.is_new ? 1 : 0
      byBranch.set(name, row)
    }
    return [...byBranch.values()].toSorted((a, b) => b.spent - a.spent)
  })
  const homeColumns = [
    { key: 'branch', title: t('Home branch') },
    { key: 'customers', title: t('Customers'), type: 'int' },
    { key: 'isNew', title: t('New customers'), type: 'int' },
    { key: 'spent', title: t('Spent'), type: 'money' },
  ]

  const named = computed(() => data.value.weekday_hour.map(c => ({ weekday: c.weekday, hour: c.hour, value: c.named_count })))
  const heatChart = computed(() => heatmapOption({ cells: named.value }))
  const names = weekdays()
  const topSlots = computed(() => named.value.filter(c => c.value > 0).toSorted((a, b) => b.value - a.value).slice(0, 8).map(c => ({ when: `${names[c.weekday - 1]} ${hourLabel(c.hour)}`, sales: c.value })))
  const slotColumns = [
    { key: 'when', title: t('When') },
    { key: 'sales', title: t('Customer sales'), type: 'int' },
  ]

  defineExpose({
    csv: () => (has('best') || has('frequent') || has('who.visits') || has('who.spend')
      ? [csvSection(t('Customers'), [
        { key: 'name', title: t('Customer') },
        ...topColumns.slice(1),
        { key: 'is_new', title: t('New'), value: r => (r.is_new ? '✓' : '') },
        { key: 'first_purchase_at', title: t('First purchase ever'), type: 'datetime' },
      ], clients.value)]
      : []),
  })
</script>
