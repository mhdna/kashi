<template>
  <v-card class="metric-card" rounded="lg" variant="outlined" width="100%">
    <div class="metric-card-head">
      <v-icon color="indigo-lighten-2" :icon="props.icon" size="18" />
      <span class="metric-card-title text-label-small text-uppercase text-medium-emphasis">
        {{ props.title }}
      </span>
    </div>

    <!-- Only the very first load (no value yet) shows a spinner. Once a
      number is on screen it stays put through every background refresh. -->
    <v-progress-circular
      v-if="props.loading && props.value == null"
      class="mt-3"
      color="indigo-lighten-2"
      indeterminate
      size="22"
      width="2"
    />
    <template v-else>
      <div class="metric-value text-headline-small font-weight-bold">
        <span v-if="props.unit === '$' && display !== '—'">{{ props.unit }}</span>{{ display }}
      </div>

      <div class="metric-card-foot text-body-small">
        <span v-if="deltaPct != null" class="metric-delta" :class="deltaClass">
          <v-icon :icon="deltaUp ? 'mdi-arrow-up' : 'mdi-arrow-down'" size="12" />{{ deltaText }}
        </span>
        <span class="metric-prev text-medium-emphasis">
          {{ deltaPct != null ? props.compareLabel : 'Prev' }}: {{ prevDisplay }}
        </span>
      </div>
    </template>
  </v-card>
</template>

<script setup>
  import { computed } from 'vue'
  import { formatMoney } from '@/utils/money'

  const props = defineProps({
    title: {
      type: String,
      required: true,
    },
    icon: {
      type: String,
      required: true,
    },
    unit: {
      type: String,
      required: true,
    },
    // Raw metric value from /dashboard/summary. Money metrics ($ unit)
    // arrive in cents. null renders as an em dash.
    value: {
      type: Number,
      default: null,
    },
    // Same metric for the immediately-preceding period.
    prevValue: {
      type: Number,
      default: null,
    },
    loading: {
      type: Boolean,
      default: false,
    },
    // What prevValue is, e.g. "vs last week" on the per-day page.
    compareLabel: {
      type: String,
      default: 'vs last period',
    },
  })

  function fmt (v) {
    if (v == null || Number.isNaN(v)) return '—'
    if (props.unit === '$') return formatMoney(v)
    if (props.unit === '%') return `${v.toLocaleString()}%`
    return v.toLocaleString()
  }

  const display = computed(() => fmt(props.value))
  const prevDisplay = computed(() => {
    const v = fmt(props.prevValue)
    return props.unit === '$' && v !== '—' ? `${props.unit}${v}` : v
  })

  const deltaPct = computed(() => {
    if (props.value == null || props.prevValue == null || props.prevValue === 0) return null
    return ((props.value - props.prevValue) / Math.abs(props.prevValue)) * 100
  })
  const deltaUp = computed(() => (deltaPct.value ?? 0) >= 0)
  const deltaClass = computed(() => {
    if (deltaPct.value == null) return ''
    return deltaUp.value ? 'text-success' : 'text-error'
  })
  const deltaText = computed(() => {
    if (deltaPct.value == null) return ''
    const sign = deltaPct.value >= 0 ? '+' : '−'
    const abs = Math.abs(deltaPct.value)
    // Off a tiny base (2 → 2,392) the percentage is noise; cap it.
    if (abs >= 1000) return `${sign}999%+`
    return `${sign}${abs.toFixed(abs >= 100 ? 0 : 1)}%`
  })
</script>

<style scoped>
  .metric-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px;
    min-width: 0;
  }

  .metric-card-head {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .metric-card-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    letter-spacing: 0.04em;
  }

  .metric-value {
    line-height: 1.2;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .metric-card-foot {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    column-gap: 8px;
    row-gap: 2px;
    line-height: 1.3;
  }

  .metric-delta {
    display: inline-flex;
    align-items: center;
    font-weight: 600;
  }

  /* Fixed-width digits so the number doesn't jitter sideways between
     filter/period selections. */
  .metric-value,
  .metric-delta,
  .metric-prev {
    font-variant-numeric: tabular-nums;
  }
</style>
