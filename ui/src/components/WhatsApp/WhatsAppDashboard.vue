<!--
  The WhatsApp tab's dashboard: the messages sent after sales and what clients
  answered in the rating poll, on one page and in one style. It reads the same
  two endpoints the Messages and Replies tabs build on -- /whatsapp/stats
  (api/whatsapp_rules.go) and /reviews (api/review_stats.go) -- and keeps to
  what's worth a glance: the totals, a day-by-day view, the answers, each
  branch side by side, and who needs a call back. The detail lives in the
  other tabs.
-->
<template>
  <div class="pa-4 pt-3">
    <div v-if="loading && !stats" class="d-flex justify-center pa-12">
      <v-progress-circular color="primary" indeterminate />
    </div>

    <template v-else-if="stats">
      <!-- Sale messages still not sent after the till's retries (the same
           watch as the WhatsApp menu dot). -->
      <v-alert
        v-if="failures?.failed_24h"
        class="mb-3"
        density="compact"
        :type="failures.status === 'critical' ? 'error' : 'warning'"
        variant="tonal"
      >
        <div class="font-weight-medium">
          {{ failures.failed_24h }} sale {{ failures.failed_24h === 1 ? 'message' : 'messages' }} failed in the last 24 hours<template v-if="failures.failed_1h">, {{ failures.failed_1h }} in the last hour</template>.
        </div>
        <div v-if="failures.last_reason" class="text-body-2">Latest: {{ failures.last_reason }}</div>
        <v-btn class="mt-1 px-0" size="small" variant="text" @click="emit('show-messages', { status: 'failed', search: '' })">
          See failed messages
        </v-btn>
      </v-alert>

      <!-- One strip of totals, messages then answers. -->
      <div class="kpis">
        <v-card
          v-for="kpi in kpis"
          :key="kpi.label"
          class="card kpi"
          :class="{ 'card--link': kpi.open }"
          variant="flat"
          @click="kpi.open?.()"
        >
          <div class="d-flex align-center ga-2">
            <span class="kpi__value" :class="kpi.tone && `text-${kpi.tone}`">{{ kpi.value }}</span>
            <v-chip
              v-if="kpi.delta != null"
              :color="kpi.delta >= 0 ? 'success' : 'error'"
              size="x-small"
              variant="tonal"
            >
              {{ kpi.delta >= 0 ? '▲' : '▼' }} {{ Math.abs(kpi.delta) }}%
            </v-chip>
          </div>
          <div class="kpi__label">{{ kpi.label }}</div>
          <div v-if="kpi.hint" class="kpi__hint">{{ kpi.hint }}</div>
        </v-card>
      </div>

      <div v-if="!hasAny" class="card empty mt-3">
        No WhatsApp messages in this range yet. Every sale to a client with a phone number shows up
        here once WhatsApp is on in Settings, whether it was sent, skipped by a rule or failed.
      </div>

      <template v-else>
        <v-row class="mt-1" density="compact">
          <v-col cols="12" md="8">
            <v-card class="card" variant="flat">
              <div class="card__title">Per day</div>
              <div class="card__subtitle">
                Messages by outcome<template v-if="ratings.totals.answers">; the line is the average answer (out of 5)</template>.
              </div>
              <v-chart autoresize class="chart" :option="dailyOption" />
            </v-card>
          </v-col>
          <v-col cols="12" md="4">
            <v-card class="card" variant="flat">
              <div class="card__title">Answers</div>
              <div class="card__subtitle">
                {{ ratings.totals.answers }} of {{ ratings.totals.polls }} clients answered the poll.
              </div>
              <div v-if="!ratings.totals.answers" class="empty">No answers in this range.</div>
              <div v-for="o in answerRows" v-else :key="o.label" class="answer">
                <div class="d-flex justify-space-between text-body-2 mb-1">
                  <span>{{ o.label }}</span>
                  <span class="text-medium-emphasis">{{ o.count }} · {{ o.pct }}%</span>
                </div>
                <v-progress-linear :color="answerColor(o.stars)" height="8" :model-value="o.pct" rounded />
              </div>
            </v-card>
          </v-col>

          <!-- Each branch's messages and answers side by side. -->
          <v-col v-if="branchRows.length > 1" cols="12">
            <v-card class="card" variant="flat">
              <div class="card__title mb-1">By branch</div>
              <v-table class="table" density="compact">
                <thead>
                  <tr>
                    <th>Branch</th>
                    <th class="text-end">Sent</th>
                    <th class="text-end">Delivered</th>
                    <th class="text-end">Read</th>
                    <th class="text-end">Answered</th>
                    <th class="text-end">Average</th>
                    <th class="text-end">Failed</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="b in branchRows" :key="b.id">
                    <td>{{ b.name }}</td>
                    <td class="text-end">{{ b.sent.toLocaleString() }}</td>
                    <td class="text-end">{{ percent(b.delivered, b.sent) }}</td>
                    <td class="text-end">{{ percent(b.read, b.sent) }}</td>
                    <td class="text-end">{{ percent(b.answers, b.polls) }}</td>
                    <td class="text-end">{{ b.answers ? `★ ${b.average.toFixed(1)}` : '—' }}</td>
                    <td class="text-end" :class="{ 'text-error': b.failed }">{{ b.failed }}</td>
                  </tr>
                </tbody>
              </v-table>
            </v-card>
          </v-col>

          <v-col cols="12" md="6">
            <v-card class="card" variant="flat">
              <div class="card__title">To call back</div>
              <div class="card__subtitle">Clients who gave the lowest answer.</div>
              <div v-if="ratings.low.length === 0" class="empty">Nobody gave the lowest answer in this range.</div>
              <div v-for="r in ratings.low.slice(0, LIST_SIZE)" :key="r.id" class="entry">
                <div class="d-flex align-center ga-3">
                  <div class="flex-grow-1">
                    <div class="text-body-2 font-weight-medium">{{ r.client_name || 'No name' }}</div>
                    <a class="text-caption" :href="`tel:+${r.client_phone}`">+{{ r.client_phone }}</a>
                  </div>
                  <div class="text-caption text-medium-emphasis text-end">
                    <div>{{ [r.branch_name, r.invoice_code].filter(Boolean).join(' · ') }}</div>
                    <div>{{ formatWhen(r.poll_voted_at) }}</div>
                  </div>
                </div>
                <div v-if="r.replies" class="reply text-body-2 mt-1">“{{ r.replies }}”</div>
              </div>
            </v-card>
          </v-col>
          <v-col cols="12" md="6">
            <v-card class="card" variant="flat">
              <div class="d-flex align-center">
                <div class="card__title flex-grow-1">What clients wrote</div>
                <v-btn
                  v-if="ratings.replies.length > 0"
                  append-icon="mdi-chevron-right"
                  size="small"
                  text="See all"
                  variant="text"
                  @click="emit('show-replies')"
                />
              </div>
              <div class="card__subtitle">The latest replies after the poll.</div>
              <div v-if="ratings.replies.length === 0" class="empty">No replies in this range.</div>
              <div v-for="r in ratings.replies.slice(0, LIST_SIZE)" :key="r.id" class="entry">
                <div class="d-flex align-start ga-2">
                  <div class="reply text-body-2 flex-grow-1">{{ r.body || KIND_LABELS[r.kind] || 'A message' }}</div>
                  <v-chip
                    v-if="r.poll_answer"
                    :color="answerColor(nullable(r.poll_rating))"
                    size="x-small"
                    variant="tonal"
                  >
                    {{ r.poll_answer }}
                  </v-chip>
                </div>
                <div class="text-caption text-medium-emphasis mt-1">
                  {{ [r.client_name || '+' + r.client_phone, r.branch_name, formatWhen(r.received_at)].filter(Boolean).join(' · ') }}
                </div>
              </div>
            </v-card>
          </v-col>
        </v-row>
      </template>
    </template>

    <!-- Meta only: the price each delivered message is costed at, from the
         Spent card. -->
    <v-dialog v-model="priceDialog" max-width="420">
      <v-card>
        <v-card-title>Price per message</v-card-title>
        <v-card-text>
          <p class="text-body-2 text-medium-emphasis mb-3">
            Meta charges each delivered template message at a rate set by the template's category and
            the client's country (WhatsApp Manager › Insights has the real bill). Spent uses this
            price, as it was when each message went out.
          </p>
          <v-text-field
            v-model="priceDraft"
            density="compact"
            :disabled="!canEdit"
            hide-details
            label="USD per message"
            min="0"
            prefix="$"
            step="0.0001"
            type="number"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Close" variant="text" @click="priceDialog = false" />
          <v-btn
            v-if="canEdit"
            color="primary"
            :loading="savingPrice"
            text="Save"
            variant="flat"
            @click="savePrice"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useReviews } from '@/composables/useReviews'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { useWhatsAppFailures } from '@/composables/useWhatsAppFailures'
  import { daysBetween, formatCost, STATUS } from '@/utils/whatsapp'

  const props = defineProps({
    // Sending through Meta's API: its costs show.
    meta: { type: Boolean, default: false },
    from: { type: String, required: true },
    to: { type: String, required: true },
    branchId: { type: Number, default: 0 },
    refreshKey: { type: Number, default: 0 },
  })
  const emit = defineEmits(['show-messages', 'show-replies'])

  // The two lists stay short; the Replies tab has the rest.
  const LIST_SIZE = 5

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getStats, setMessageCost } = useWhatsApp()
  const { getReviews } = useReviews()
  const { failures, refreshWhatsAppFailures } = useWhatsAppFailures()

  const stats = ref(null)
  const reviews = ref(null)
  const loading = ref(false)

  async function load () {
    loading.value = true
    refreshWhatsAppFailures().catch(() => {})
    const params = { from: props.from, to: props.to, branch_id: props.branchId }
    try {
      const [s, r] = await Promise.all([getStats(params), getReviews(params)])
      stats.value = s
      reviews.value = r
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  const EMPTY_RATINGS = {
    totals: { polls: 0, answers: 0, average: 0 },
    options: [],
    days: [],
    branches: [],
    low: [],
    replies: [],
  }
  const ratings = computed(() => {
    const r = reviews.value?.ratings
    if (!r) return EMPTY_RATINGS
    // Go marshals an empty slice it never filled as null.
    return {
      ...r,
      options: r.options ?? [],
      days: r.days ?? [],
      branches: r.branches ?? [],
      low: r.low ?? [],
      replies: r.replies ?? [],
    }
  })

  // ---- helpers -------------------------------------------------------------
  function percent (n, of) {
    return of ? `${Math.round((n / of) * 100)}%` : '—'
  }
  function delta (now, before) {
    if (!before) return null
    return Math.round(((now - before) / before) * 100)
  }
  // A Go sql.Null* arrives as { Time | Int16, Valid }.
  function nullable (v) {
    if (v && typeof v === 'object') return v.Valid ? (v.Time ?? v.Int16) : null
    return v
  }
  function formatWhen (at) {
    at = nullable(at)
    if (!at) return ''
    return new Date(at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }
  function answerColor (stars) {
    if (stars >= 4) return 'success'
    return stars === 3 ? 'warning' : 'error'
  }
  // A reply that isn't text (or a photo's, a video's caption under it).
  const KIND_LABELS = {
    image: '📷 Photo',
    video: '🎥 Video',
    audio: '🎤 Voice note: listen on the phone',
    sticker: 'A sticker',
    document: '📄 A file',
    other: 'A contact or location',
  }

  // ---- totals ----------------------------------------------------------------
  const hasAny = computed(() => {
    const t = stats.value?.totals
    return Boolean(t && (t.sent || t.skipped || t.failed))
  })
  const kpis = computed(() => {
    const t = stats.value.totals
    const p = stats.value.previous
    const r = ratings.value.totals
    const lowest = ratings.value.low.length
    const failed = t.failed + t.undelivered
    return [
      {
        label: 'Sent',
        value: t.sent.toLocaleString(),
        delta: delta(t.sent, p.sent),
        hint: `to ${t.recipients.toLocaleString()} client${t.recipients === 1 ? '' : 's'}`,
        open: () => emit('show-messages', { status: 'sent' }),
      },
      { label: 'Delivered', value: percent(t.delivered, t.sent), hint: `${t.delivered.toLocaleString()} messages` },
      { label: 'Read', value: percent(t.read, t.sent), hint: `${t.read.toLocaleString()} messages` },
      { label: 'Answered the poll', value: percent(r.answers, r.polls), hint: `${r.answers} of ${r.polls}` },
      { label: 'Average answer', value: r.answers ? `★ ${r.average.toFixed(1)}` : '—', hint: 'out of 5' },
      {
        label: 'To call back',
        value: lowest.toLocaleString(),
        tone: lowest ? 'error' : undefined,
        hint: 'lowest answer',
      },
      props.meta && {
        label: 'Spent',
        value: formatCost(t.cost_micros),
        delta: delta(t.cost_micros, p.cost_micros),
        hint: t.sent ? `${formatCost(t.cost_micros / t.sent)} per message` : 'set the price',
        open: openPrice,
      },
      {
        label: 'Failed',
        value: failed.toLocaleString(),
        tone: failed ? 'error' : undefined,
        hint: t.skipped ? `${t.skipped} skipped by rules` : '',
        open: () => emit('show-messages', { status: 'failed' }),
      },
    ].filter(Boolean)
  })

  // ---- answers ---------------------------------------------------------------
  // The poll's own options, best first; stars when the poll has none saved.
  const answerRows = computed(() => {
    const r = ratings.value
    const options = r.options.length > 0
      ? r.options
      : [5, 4, 3, 2, 1].map(n => ({ label: '★'.repeat(n), stars: n }))
    return options.map(o => {
      const count = r.totals[`stars${o.stars}`] ?? 0
      return { ...o, count, pct: r.totals.answers ? Math.round((count / r.totals.answers) * 100) : 0 }
    })
  })

  // ---- by branch -------------------------------------------------------------
  // Messages and answers joined per branch; a branch filter leaves one row,
  // which the table doesn't show.
  const branchRows = computed(() => {
    if (props.branchId) return []
    const answers = Object.fromEntries(ratings.value.branches.map(b => [b.branch_id, b]))
    return (stats.value?.branches ?? []).map(b => ({
      id: b.branch_id,
      name: b.branch_name || 'Deleted branch',
      sent: b.sent,
      delivered: b.delivered ?? 0,
      read: b.read ?? 0,
      failed: b.failed,
      polls: answers[b.branch_id]?.polls ?? 0,
      answers: answers[b.branch_id]?.answers ?? 0,
      average: answers[b.branch_id]?.average ?? 0,
    }))
  })

  // ---- per day ---------------------------------------------------------------
  const theme = useTheme()
  const isDark = computed(() => theme.current.value.dark)
  const colors = computed(() => ({
    sent: theme.current.value.colors.success,
    skipped: isDark.value ? '#757575' : '#bdbdbd',
    failed: theme.current.value.colors.error,
    rating: theme.current.value.colors.warning,
  }))
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))
  const STATUSES = ['sent', 'skipped', 'failed']

  const dailyOption = computed(() => {
    const messages = Object.fromEntries((stats.value?.days ?? []).map(d => [d.day, d]))
    const answers = Object.fromEntries(ratings.value.days.map(d => [d.day, d]))
    const days = daysBetween(props.from, props.to)
    const label = day => new Date(`${day}T00:00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    const axisLabel = { color: textColor.value, fontSize: 11 }
    const withRating = ratings.value.totals.answers > 0
    return {
      grid: { top: 28, right: withRating ? 36 : 12, bottom: 24, left: 36 },
      legend: { top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { color: textColor.value, fontSize: 10 } },
      tooltip: { trigger: 'axis', appendTo: 'body' },
      xAxis: {
        type: 'category',
        data: days.map(d => label(d)),
        axisLine: { lineStyle: { color: axisColor.value } },
        axisTick: { show: false },
        axisLabel,
      },
      yAxis: [
        { type: 'value', minInterval: 1, axisLabel, splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } } },
        withRating && { type: 'value', min: 1, max: 5, interval: 1, axisLabel, splitLine: { show: false } },
      ].filter(Boolean),
      series: [
        ...STATUSES.map(s => ({
          name: STATUS[s].label,
          type: 'bar',
          stack: 'messages',
          barMaxWidth: 28,
          itemStyle: { color: colors.value[s] },
          data: days.map(d => messages[d]?.[s] ?? 0),
        })),
        withRating && {
          name: 'Average answer',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          connectNulls: true,
          symbolSize: 6,
          itemStyle: { color: colors.value.rating },
          tooltip: { valueFormatter: v => (v == null ? '—' : `★ ${v.toFixed(1)}`) },
          data: days.map(d => answers[d]?.average ?? null),
        },
      ].filter(Boolean),
    }
  })

  // ---- price per message (Meta) ------------------------------------------------
  const priceDialog = ref(false)
  const priceDraft = ref('')
  const savingPrice = ref(false)
  function openPrice () {
    const micros = stats.value?.message_cost_micros ?? 0
    priceDraft.value = micros ? String(micros / 1e6) : ''
    priceDialog.value = true
  }
  async function savePrice () {
    const micros = Math.round((Number(priceDraft.value) || 0) * 1e6)
    if (micros < 0) return
    savingPrice.value = true
    try {
      await setMessageCost(micros)
      stats.value = { ...stats.value, message_cost_micros: micros }
      priceDialog.value = false
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
.card {
  padding: 12px 14px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.card--link {
  cursor: pointer;
}
.card__title {
  font-size: 0.9rem;
  font-weight: 600;
}
.card__subtitle {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 8px;
}
.kpis {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}
.kpi__value {
  font-size: 1.35rem;
  font-weight: 600;
  line-height: 1.3;
}
.kpi__label {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.7);
}
.kpi__hint {
  font-size: 0.7rem;
  color: rgba(var(--v-theme-on-surface), 0.45);
}
.chart {
  height: 260px;
  width: 100%;
}
.answer + .answer {
  margin-top: 14px;
}
.empty {
  padding: 20px 0;
  text-align: center;
  font-size: 0.875rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.card.empty {
  padding: 20px;
}
.entry {
  padding: 8px 0;
}
.entry + .entry {
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.reply {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.table :deep(th) {
  font-size: 0.75rem !important;
}
</style>
