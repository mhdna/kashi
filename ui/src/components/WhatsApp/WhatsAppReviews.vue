<template>
  <div class="pa-4 pt-3">
    <div v-if="loading && !stats" class="d-flex justify-center pa-12">
      <v-progress-circular color="primary" indeterminate />
    </div>

    <template v-else-if="stats">
      <v-alert
        v-if="!stats.configured || !stats.has_base_url"
        class="mb-3"
        density="compact"
        type="info"
        variant="tonal"
      >
        Set the Google place and the link address under <strong>Review link</strong> below. Then add
        <code>review</code> (the link) or <code>review_button</code> (a "visit website" button ending
        in <code>{{ '{' }}{{ '{' }}1}}</code>) to the template's placeholders, and every message gets its own
        link.
      </v-alert>

      <!-- KPIs -->
      <v-row density="compact">
        <v-col
          v-for="kpi in kpis"
          :key="kpi.label"
          cols="6"
          lg="2"
          md="4"
        >
          <v-card class="pa-3 d-flex flex-column justify-center fill-height" :color="kpi.color" variant="tonal">
            <div class="text-h6 font-weight-medium">{{ kpi.value }}</div>
            <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
            <div v-if="kpi.hint" class="text-caption text-disabled">{{ kpi.hint }}</div>
          </v-card>
        </v-col>
      </v-row>

      <v-row class="mt-1" density="compact">
        <v-col cols="12" md="8">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Review links per day</div>
            <div class="chart-card__subtitle">Messages sent with a link, and how often a link was opened.</div>
            <v-chart autoresize class="chart" :option="dailyOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">How far they got</div>
            <div class="chart-card__subtitle">Of the messages sent with a review link.</div>
            <v-chart autoresize class="chart" :option="funnelOption" />
          </v-card>
        </v-col>

        <!-- Google, per place -->
        <v-col
          v-for="place in stats.places"
          :key="place.place_id"
          cols="12"
          :md="stats.places.length > 1 ? 6 : 12"
        >
          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-start ga-2">
              <div class="flex-grow-1">
                <div class="chart-card__title">{{ place.name || 'Google reviews' }}</div>
                <div class="chart-card__subtitle">
                  {{ place.branches.length > 0 ? place.branches.join(', ') : 'Every branch without its own place' }}
                </div>
              </div>
              <v-btn
                :href="place.review_url"
                icon="mdi-open-in-new"
                rel="noopener"
                size="small"
                target="_blank"
                title="Open the review page"
                variant="text"
              />
            </div>
            <div v-if="!place.checked" class="text-body-2 text-medium-emphasis py-6 text-center">
              {{ stats.has_api_key ? 'Not read from Google yet.' : 'Add a Places API key below to see the rating and reviews.' }}
            </div>
            <template v-else>
              <div class="d-flex align-center flex-wrap ga-4 mb-2">
                <div class="d-flex align-center ga-2">
                  <span class="text-h5 font-weight-medium">{{ place.rating.toFixed(1) }}</span>
                  <v-rating
                    color="amber"
                    density="compact"
                    half-increments
                    :model-value="place.rating"
                    readonly
                    size="small"
                  />
                </div>
                <div class="text-body-2">
                  <strong>{{ place.review_count.toLocaleString() }}</strong> reviews
                </div>
                <v-chip
                  v-if="place.new_reviews != null"
                  :color="place.new_reviews > 0 ? 'success' : undefined"
                  size="small"
                  variant="tonal"
                >
                  {{ place.new_reviews >= 0 ? '+' : '' }}{{ place.new_reviews }} in this range
                </v-chip>
                <span class="text-caption text-disabled">checked {{ formatWhen(place.checked_at) }}</span>
              </div>
              <v-chart
                v-if="place.snapshots.length > 1"
                autoresize
                class="chart chart--small"
                :option="placeOption(place)"
              />
            </template>
          </v-card>
        </v-col>

        <v-col v-if="!branchId && stats.branches.length > 0" cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title mb-2">By branch</div>
            <v-table density="compact">
              <thead>
                <tr>
                  <th>Branch</th>
                  <th class="text-end">Sent</th>
                  <th class="text-end">Read</th>
                  <th class="text-end">Opened</th>
                  <th class="text-end">Rate</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="b in stats.branches" :key="b.branch_id">
                  <td>{{ b.branch_name || 'No branch' }}</td>
                  <td class="text-end">{{ b.sent }}</td>
                  <td class="text-end">{{ b.read }}</td>
                  <td class="text-end">{{ b.clicked }}</td>
                  <td class="text-end">{{ percent(b.clicked, b.sent) }}</td>
                </tr>
              </tbody>
            </v-table>
          </v-card>
        </v-col>

        <v-col cols="12" :md="!branchId && stats.branches.length > 0 ? 6 : 12">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title mb-2">Latest Google reviews</div>
            <div v-if="stats.reviews.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
              {{ stats.has_api_key ? 'No reviews read yet.' : 'Add a Places API key below to see reviews here.' }}
            </div>
            <div v-for="r in stats.reviews" :key="r.review_name" class="review py-2">
              <div class="d-flex align-center ga-2">
                <v-avatar v-if="r.author_photo" :image="r.author_photo" size="24" />
                <span class="text-body-2 font-weight-medium">{{ r.author || 'Google user' }}</span>
                <v-rating
                  color="amber"
                  density="compact"
                  :model-value="r.rating"
                  readonly
                  size="x-small"
                />
                <v-spacer />
                <span class="text-caption text-disabled">{{ formatWhen(r.published_at) }}</span>
              </div>
              <div v-if="r.body" class="text-body-2 mt-1 review__body">{{ r.body }}</div>
            </div>
            <div v-if="stats.reviews.length > 0" class="text-caption text-disabled mt-1">
              Google shares up to five reviews per place at a time; kashi keeps every one it has seen.
            </div>
          </v-card>
        </v-col>

        <!-- Settings -->
        <v-col cols="12">
          <v-card class="chart-card pb-3" variant="flat">
            <div class="d-flex align-center">
              <div class="chart-card__title">Review link</div>
              <v-spacer />
              <v-btn
                v-if="canEdit && stats.has_api_key"
                :loading="checking"
                prepend-icon="mdi-refresh"
                size="small"
                text="Check Google now"
                variant="text"
                @click="checkNow"
              />
            </div>
            <p class="text-caption text-medium-emphasis mb-3">
              Each message's link opens the Google "write a review" box of the branch that made the sale:
              its own place when set, else the main one. Paste a place id, or a whole review link.
              WhatsApp shows the title and line below on the link's card.
            </p>
            <v-form v-if="draft" :disabled="!canEdit" @submit.prevent="save">
              <v-row density="compact">
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.review_place_id"
                    density="compact"
                    hint="Google place id, e.g. ChIJ…"
                    label="Main Google place"
                    persistent-hint
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.review_base_url"
                    density="compact"
                    hint="Where the links live; each message adds its own code"
                    label="Link address"
                    persistent-hint
                    placeholder="https://grandbz.com/review"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.review_title"
                    density="compact"
                    hide-details
                    label="Card title"
                    placeholder="Grand Bazaar Feedback"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.review_description"
                    density="compact"
                    hide-details
                    label="Card line (optional)"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.google_places_api_key"
                    :append-inner-icon="showKey ? 'mdi-eye-off' : 'mdi-eye'"
                    autocomplete="off"
                    density="compact"
                    hint="Google Cloud > Places API (New). Read every 3 hours."
                    label="Places API key"
                    persistent-hint
                    :type="showKey ? 'text' : 'password'"
                    variant="outlined"
                    @click:append-inner="showKey = !showKey"
                  />
                </v-col>
              </v-row>

              <div class="text-body-2 font-weight-medium mt-4 mb-1">Branches with their own Google place</div>
              <v-row density="compact">
                <v-col
                  v-for="b in draft.branches"
                  :key="b.branch_id"
                  cols="12"
                  md="6"
                >
                  <v-text-field
                    v-model="b.place_id"
                    density="compact"
                    hide-details
                    :label="b.branch_name"
                    placeholder="Uses the main place"
                    variant="outlined"
                  />
                </v-col>
              </v-row>

              <div v-if="canEdit" class="d-flex justify-end mt-3">
                <v-btn
                  color="primary"
                  :disabled="!dirty"
                  :loading="saving"
                  text="Save"
                  type="submit"
                  variant="flat"
                />
              </div>
            </v-form>
          </v-card>
        </v-col>
      </v-row>
    </template>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { daysBetween } from '@/utils/whatsapp'

  // Review links in the post-sale message (api/review_link.go) and what Google
  // shows for each place they go to (api/google_places.go).

  const props = defineProps({
    from: { type: String, required: true },
    to: { type: String, required: true },
    branchId: { type: Number, default: 0 },
    refreshKey: { type: Number, default: 0 },
  })

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getReviews, refreshReviews, getReviewSettings, saveReviewSettings } = useWhatsApp()

  const stats = ref(null)
  const loading = ref(false)

  async function load () {
    loading.value = true
    try {
      stats.value = await getReviews({ from: props.from, to: props.to, branch_id: props.branchId })
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  function percent (n, of) {
    return of ? `${Math.round((n / of) * 100)}%` : '—'
  }
  function formatWhen (at) {
    if (!at) return ''
    return new Date(at).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
  }

  // ---- KPIs ----------------------------------------------------------------
  const kpis = computed(() => {
    const f = stats.value.funnel
    const places = stats.value.places.filter(p => p.checked)
    const newReviews = places.reduce((sum, p) => sum + (p.new_reviews ?? 0), 0)
    const known = places.some(p => p.new_reviews != null)
    const rated = places.length === 1 ? places[0] : null
    return [
      { label: 'Sent with a review link', value: f.with_link.toLocaleString(), hint: `of ${f.sent.toLocaleString()} messages` },
      { label: 'Read', value: f.read.toLocaleString(), hint: f.with_link ? `${percent(f.read, f.with_link)} of sent` : '' },
      {
        label: 'Opened the link',
        value: f.clicked.toLocaleString(),
        color: f.clicked ? 'success' : undefined,
        hint: f.with_link ? `${percent(f.clicked, f.with_link)} of sent` : '',
      },
      { label: 'Link opens', value: stats.value.clicks.toLocaleString(), hint: 'every open, repeats included' },
      {
        label: 'Google rating',
        value: rated ? rated.rating.toFixed(1) : (places.length > 0 ? `${places.length} places` : '—'),
        hint: rated ? `${rated.review_count.toLocaleString()} reviews` : '',
      },
      { label: 'New Google reviews', value: known ? `+${newReviews}` : '—', hint: known ? 'in this range' : 'counted from the first check' },
    ]
  })

  // ---- charts --------------------------------------------------------------
  const theme = useTheme()
  const isDark = computed(() => theme.current.value.dark)
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
  const dayLabel = day => new Date(`${day}T00:00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })

  const dailyOption = computed(() => {
    const byDay = Object.fromEntries((stats.value?.days ?? []).map(d => [d.day, d]))
    const days = daysBetween(props.from, props.to)
    return {
      grid: { top: 28, right: 12, bottom: 24, left: 36 },
      legend: legend.value,
      tooltip: { trigger: 'axis', appendTo: 'body' },
      xAxis: { type: 'category', data: days.map(d => dayLabel(d)), ...catAxis.value },
      yAxis: { type: 'value', minInterval: 1, ...valAxis.value },
      series: [
        {
          name: 'Sent with a link',
          type: 'bar',
          itemStyle: { color: isDark.value ? '#555' : '#cfcfcf', borderRadius: [3, 3, 0, 0] },
          data: days.map(d => byDay[d]?.sent ?? 0),
        },
        {
          name: 'Link opens',
          type: 'line',
          smooth: true,
          symbolSize: 5,
          itemStyle: { color: theme.current.value.colors.success },
          data: days.map(d => byDay[d]?.clicks ?? 0),
        },
      ],
    }
  })

  const funnelOption = computed(() => {
    const f = stats.value.funnel
    const steps = [
      ['Sent', f.with_link],
      ['Delivered', f.delivered],
      ['Read', f.read],
      ['Opened', f.clicked],
    ]
    return {
      grid: { top: 8, right: 48, bottom: 8, left: 72 },
      tooltip: { trigger: 'item', appendTo: 'body' },
      xAxis: { type: 'value', show: false },
      yAxis: { type: 'category', inverse: true, data: steps.map(s => s[0]), ...catAxis.value },
      series: [{
        type: 'bar',
        barWidth: '55%',
        itemStyle: { color: theme.current.value.colors.success, borderRadius: [0, 3, 3, 0] },
        label: {
          show: true,
          position: 'right',
          color: textColor.value,
          fontSize: 11,
          formatter: p => `${p.value} · ${percent(p.value, f.with_link)}`,
        },
        data: steps.map(s => s[1]),
      }],
    }
  })

  function placeOption (place) {
    const points = place.snapshots.map(s => [new Date(s.at).getTime(), s.review_count])
    return {
      grid: { top: 12, right: 12, bottom: 24, left: 44 },
      tooltip: { trigger: 'axis', appendTo: 'body' },
      xAxis: { type: 'time', ...catAxis.value },
      yAxis: { type: 'value', minInterval: 1, scale: true, ...valAxis.value },
      series: [{
        name: 'Reviews',
        type: 'line',
        step: 'end',
        symbol: 'none',
        itemStyle: { color: '#f5b301' },
        areaStyle: { opacity: 0.08 },
        data: points,
      }],
    }
  }

  // ---- settings ------------------------------------------------------------
  const saved = ref(null)
  const draft = ref(null)
  const saving = ref(false)
  const checking = ref(false)
  const showKey = ref(false)
  const dirty = computed(() => JSON.stringify(draft.value) !== JSON.stringify(saved.value))

  function seed (settings) {
    saved.value = settings
    draft.value = structuredClone(settings)
  }
  async function loadSettings () {
    try {
      seed(await getReviewSettings())
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function save () {
    saving.value = true
    try {
      seed(await saveReviewSettings(draft.value))
      notify.success('Review link saved.')
      load()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
  async function checkNow () {
    checking.value = true
    try {
      await refreshReviews()
      await load()
    } catch (error) {
      notify.error(`Google: ${error.message}`)
    } finally {
      checking.value = false
    }
  }

  onMounted(loadSettings)
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
.chart--small {
  height: 160px;
}
.review + .review {
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.review__body {
  white-space: pre-line;
}
</style>
