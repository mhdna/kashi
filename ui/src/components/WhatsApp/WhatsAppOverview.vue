<template>
  <div class="pa-4 pt-3">
    <div v-if="loading && !stats" class="d-flex justify-center pa-12">
      <v-progress-circular color="primary" indeterminate />
    </div>

    <template v-else-if="stats">
      <!-- KPIs -->
      <v-row density="compact">
        <v-col
          v-for="kpi in kpis"
          :key="kpi.label"
          cols="6"
          lg="2"
          md="4"
        >
          <v-card
            class="pa-3 d-flex flex-column justify-center fill-height"
            :class="{ 'kpi--link': kpi.status }"
            :color="kpi.color"
            variant="tonal"
            @click="kpi.status ? emit('show-messages', { status: kpi.status }) : undefined"
          >
            <div class="d-flex align-center justify-space-between">
              <div class="text-h6 font-weight-medium">{{ kpi.value }}</div>
              <v-chip
                v-if="kpi.delta != null"
                :color="kpi.delta >= 0 ? 'primary' : 'error'"
                size="small"
                variant="tonal"
              >
                <v-icon :icon="kpi.delta >= 0 ? 'mdi-arrow-up' : 'mdi-arrow-down'" start />
                {{ Math.abs(kpi.delta) }}%
              </v-chip>
            </div>
            <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
            <div v-if="kpi.hint" class="text-caption text-disabled">{{ kpi.hint }}</div>
          </v-card>
        </v-col>
      </v-row>

      <v-alert
        v-if="!hasAny"
        class="my-3"
        density="compact"
        type="info"
        variant="tonal"
      >
        No WhatsApp messages in this range yet. Every sale to a client with a phone number shows up
        here once WhatsApp is on in Settings, whether it was sent, skipped by a rule or failed.
      </v-alert>

      <v-row class="mt-1" density="compact">
        <v-col cols="12" md="8">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Messages per day</div>
            <div class="chart-card__subtitle">Bars count messages by outcome; the line is what was spent.</div>
            <v-chart autoresize class="chart" :option="dailyOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Outcome</div>
            <div class="chart-card__subtitle">Of every sale WhatsApp considered.</div>
            <v-chart autoresize class="chart" :option="outcomeOption" @click="onOutcomeClick" />
          </v-card>
        </v-col>

        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">By rule</div>
            <div class="chart-card__subtitle">Which rule decided each message; "Default" is no rule matching.</div>
            <v-chart autoresize class="chart" :option="ruleOption" :style="{ height: ruleChartHeight }" />
          </v-card>
        </v-col>
        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Time of day</div>
            <div class="chart-card__subtitle">Messages sent per hour (Beirut time).</div>
            <v-chart autoresize class="chart" :option="hourOption" />
          </v-card>
        </v-col>

        <v-col v-if="!branchId" cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">By branch</div>
            <v-chart autoresize class="chart" :option="branchOption" />
          </v-card>
        </v-col>
        <v-col cols="12" :md="branchId ? 12 : 6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title mb-2">Clients messaged most</div>
            <div v-if="stats.recipients.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
              Nobody was messaged in this range.
            </div>
            <v-table v-else density="compact">
              <thead>
                <tr>
                  <th>Client</th>
                  <th class="text-end">Messages</th>
                  <th class="text-end">Their sales</th>
                  <th class="text-end">Cost</th>
                  <th>Last</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="r in stats.recipients"
                  :key="r.client_phone"
                  class="row--link"
                  @click="emit('show-messages', { search: r.client_phone })"
                >
                  <td>
                    <div>{{ r.client_name || '—' }}</div>
                    <div class="text-caption text-medium-emphasis">{{ formatPhone(r.client_phone) }}</div>
                  </td>
                  <td class="text-end">{{ r.messages }}</td>
                  <td class="text-end">{{ formatDollars(r.sales_cents) }}</td>
                  <td class="text-end">{{ formatCost(r.cost_micros) }}</td>
                  <td class="text-caption">{{ new Date(r.last_sent_at).toLocaleDateString() }}</td>
                </tr>
              </tbody>
            </v-table>
          </v-card>
        </v-col>

        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title mb-2">Why messages failed</div>
            <div v-if="stats.failures.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
              Nothing failed. 👍
            </div>
            <v-list v-else class="pa-0" density="compact">
              <v-list-item
                v-for="f in stats.failures"
                :key="f.reason"
                class="px-1"
                @click="emit('show-messages', { status: 'failed', search: '' })"
              >
                <template #prepend>
                  <v-icon color="error" icon="mdi-alert-circle-outline" size="18" />
                </template>
                <v-list-item-title class="text-body-2 text-wrap">{{ f.reason || 'Unknown' }}</v-list-item-title>
                <template #append>
                  <span class="text-caption">×{{ f.messages }}</span>
                </template>
              </v-list-item>
            </v-list>
          </v-card>
        </v-col>

        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title mb-1">Price per message</div>
            <p class="text-caption text-medium-emphasis mb-3">
              Meta charges for each delivered template message, at a rate set by the template's
              category (marketing, utility…) and the client's country — see WhatsApp Manager &gt;
              Insights for your actual bill. Costs here use this price, as it was when each
              message went out.
            </p>
            <div class="d-flex align-center ga-3">
              <v-text-field
                v-model="priceDraft"
                density="compact"
                :disabled="!canEdit"
                hide-details
                label="USD per message"
                min="0"
                prefix="$"
                step="0.0001"
                style="max-width: 200px"
                type="number"
                variant="outlined"
              />
              <v-btn
                v-if="canEdit"
                color="primary"
                :disabled="!priceDirty"
                :loading="savingPrice"
                text="Save"
                variant="flat"
                @click="savePrice"
              />
            </div>
          </v-card>
        </v-col>
      </v-row>
    </template>
  </div>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { daysBetween, formatCost, formatDollars, formatPhone, STATUS } from '@/utils/whatsapp'

  const props = defineProps({
    from: { type: String, required: true },
    to: { type: String, required: true },
    branchId: { type: Number, default: 0 },
    refreshKey: { type: Number, default: 0 },
  })
  const emit = defineEmits(['show-messages'])

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getStats, setMessageCost } = useWhatsApp()

  const stats = ref(null)
  const loading = ref(false)

  async function load () {
    loading.value = true
    try {
      const keepPrice = priceDirty.value
      stats.value = await getStats({ from: props.from, to: props.to, branch_id: props.branchId })
      if (!keepPrice) seedPrice()
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  // ---- KPIs ----------------------------------------------------------------
  function delta (now, before) {
    if (!before) return null
    return Math.round(((now - before) / before) * 100)
  }
  const hasAny = computed(() => {
    const t = stats.value?.totals
    return Boolean(t && (t.sent || t.skipped || t.failed))
  })
  const kpis = computed(() => {
    const t = stats.value.totals
    const p = stats.value.previous
    return [
      { label: 'Messages sent', value: t.sent.toLocaleString(), delta: delta(t.sent, p.sent), status: 'sent' },
      { label: 'Clients reached', value: t.recipients.toLocaleString(), delta: delta(t.recipients, p.recipients) },
      {
        label: 'Spent',
        value: formatCost(t.cost_micros),
        delta: delta(t.cost_micros, p.cost_micros),
        hint: t.sent ? `${formatCost(t.cost_micros / t.sent)} per message` : '',
      },
      {
        label: 'Sales that got a message',
        value: formatDollars(t.sent_sales_cents),
      },
      { label: 'Skipped by rules', value: t.skipped.toLocaleString(), status: 'skipped' },
      {
        label: 'Failed',
        value: t.failed.toLocaleString(),
        status: 'failed',
        color: t.failed ? 'error' : undefined,
      },
    ]
  })

  // ---- charts --------------------------------------------------------------
  const theme = useTheme()
  const isDark = computed(() => theme.current.value.dark)
  const colors = computed(() => ({
    sent: theme.current.value.colors.success,
    skipped: isDark.value ? '#757575' : '#bdbdbd',
    failed: theme.current.value.colors.error,
    cost: theme.current.value.colors.primary,
  }))
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))
  const catAxis = computed(() => ({
    axisLine: { lineStyle: { color: axisColor.value } },
    axisTick: { show: false },
    axisLabel: { color: textColor.value, fontSize: 11 },
  }))
  const valAxis = computed(() => ({
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
  }))
  const legend = computed(() => ({
    top: 0,
    right: 0,
    itemWidth: 10,
    itemHeight: 10,
    textStyle: { color: textColor.value, fontSize: 10 },
  }))
  const STATUSES = ['sent', 'skipped', 'failed']

  const dailyOption = computed(() => {
    const byDay = Object.fromEntries((stats.value?.days ?? []).map(d => [d.day, d]))
    const days = daysBetween(props.from, props.to)
    const label = day => new Date(`${day}T00:00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    return {
      grid: { top: 28, right: 48, bottom: 24, left: 36 },
      legend: legend.value,
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        valueFormatter: v => (typeof v === 'number' && !Number.isInteger(v) ? `$${v.toFixed(4)}` : v),
      },
      xAxis: { type: 'category', data: days.map(d => label(d)), ...catAxis.value },
      yAxis: [
        { type: 'value', minInterval: 1, ...valAxis.value },
        {
          type: 'value',
          ...valAxis.value,
          splitLine: { show: false },
          axisLabel: { ...valAxis.value.axisLabel, formatter: v => `$${v}` },
        },
      ],
      series: [
        ...STATUSES.map(s => ({
          name: STATUS[s].label,
          type: 'bar',
          stack: 'messages',
          barMaxWidth: 28,
          itemStyle: { color: colors.value[s] },
          data: days.map(d => byDay[d]?.[s] ?? 0),
        })),
        {
          name: 'Spent',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          showSymbol: false,
          itemStyle: { color: colors.value.cost },
          data: days.map(d => (byDay[d]?.cost_micros ?? 0) / 1e6),
        },
      ],
    }
  })

  const outcomeOption = computed(() => {
    const t = stats.value?.totals ?? {}
    return {
      tooltip: { trigger: 'item', appendTo: 'body' },
      legend: { ...legend.value, top: 'bottom', right: 'center' },
      series: [{
        type: 'pie',
        radius: ['50%', '75%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: true,
        label: { show: false },
        data: STATUSES.map(s => ({
          name: STATUS[s].label,
          value: t[s] ?? 0,
          status: s,
          itemStyle: { color: colors.value[s] },
        })),
      }],
    }
  })
  function onOutcomeClick (params) {
    if (params?.data?.status) emit('show-messages', { status: params.data.status })
  }

  // One bar per rule, split by outcome.
  const ruleRows = computed(() => {
    const byRule = {}
    for (const r of stats.value?.rules ?? []) {
      const name = r.rule_name || 'Default'
      byRule[name] ??= { name, sent: 0, skipped: 0, failed: 0 }
      byRule[name][r.status] += r.messages
    }
    return Object.values(byRule)
      .toSorted((a, b) => (a.sent + a.skipped + a.failed) - (b.sent + b.skipped + b.failed))
  })
  const ruleChartHeight = computed(() => `${Math.max(280, ruleRows.value.length * 34 + 60)}px`)
  const ruleOption = computed(() => ({
    grid: { top: 28, right: 16, bottom: 24, left: 8, containLabel: true },
    legend: legend.value,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, appendTo: 'body' },
    xAxis: { type: 'value', minInterval: 1, ...valAxis.value },
    yAxis: { type: 'category', data: ruleRows.value.map(r => r.name), ...catAxis.value },
    series: STATUSES.map(s => ({
      name: STATUS[s].label,
      type: 'bar',
      stack: 'rule',
      barMaxWidth: 22,
      itemStyle: { color: colors.value[s] },
      data: ruleRows.value.map(r => r[s]),
    })),
  }))

  const hourOption = computed(() => {
    const byHour = Object.fromEntries((stats.value?.hours ?? []).map(h => [h.hour, h.sent]))
    const hours = Array.from({ length: 24 }, (_, h) => h)
    return {
      grid: { top: 16, right: 12, bottom: 24, left: 36 },
      tooltip: { trigger: 'axis', appendTo: 'body' },
      xAxis: { type: 'category', data: hours.map(h => `${String(h).padStart(2, '0')}:00`), ...catAxis.value },
      yAxis: { type: 'value', minInterval: 1, ...valAxis.value },
      series: [{
        name: 'Sent',
        type: 'bar',
        barWidth: '60%',
        itemStyle: { color: colors.value.sent, borderRadius: [3, 3, 0, 0] },
        data: hours.map(h => byHour[h] ?? 0),
      }],
    }
  })

  const branchOption = computed(() => {
    const rows = stats.value?.branches ?? []
    return {
      grid: { top: 28, right: 12, bottom: 24, left: 36 },
      legend: legend.value,
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, appendTo: 'body' },
      xAxis: { type: 'category', data: rows.map(r => r.branch_name || 'Deleted branch'), ...catAxis.value },
      yAxis: { type: 'value', minInterval: 1, ...valAxis.value },
      series: STATUSES.map(s => ({
        name: STATUS[s].label,
        type: 'bar',
        stack: 'branch',
        barMaxWidth: 36,
        itemStyle: { color: colors.value[s] },
        data: rows.map(r => r[s]),
      })),
    }
  })

  // ---- price per message ---------------------------------------------------
  const priceDraft = ref('')
  const savedPriceMicros = computed(() => stats.value?.message_cost_micros ?? 0)
  function seedPrice () {
    priceDraft.value = savedPriceMicros.value ? String(savedPriceMicros.value / 1e6) : ''
  }
  const draftMicros = computed(() => Math.round((Number(priceDraft.value) || 0) * 1e6))
  const priceDirty = computed(() => stats.value != null && draftMicros.value !== savedPriceMicros.value)
  const savingPrice = ref(false)
  async function savePrice () {
    if (draftMicros.value < 0) return
    savingPrice.value = true
    try {
      await setMessageCost(draftMicros.value)
      stats.value = { ...stats.value, message_cost_micros: draftMicros.value }
      seedPrice()
      notify.success('Price per message saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingPrice.value = false
    }
  }

  // Last, so everything load() touches exists before its first run.
  watch(() => [props.from, props.to, props.branchId, props.refreshKey], load, { immediate: true })
</script>

<style scoped>
.chart-card {
  padding: 12px 14px 4px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.chart-card__title {
  font-size: 0.9rem;
  font-weight: 600;
  margin-bottom: 2px;
}
.chart-card__subtitle {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 4px;
}
.chart {
  height: 280px;
  width: 100%;
}
.kpi--link,
.row--link {
  cursor: pointer;
}
.row--link:hover {
  background: rgba(var(--v-theme-on-surface), 0.04);
}
</style>
