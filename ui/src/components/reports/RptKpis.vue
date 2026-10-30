<template>
  <div class="rpt-kpis">
    <RptHide class="rpt-kpis__hide" :part="part" />
    <div v-for="kpi in items" :key="kpi.label" class="rpt-kpi">
      <!-- What it is first, then the figure: the label names the answer
           ("Quietest weekday"), so it must not hide under the value. -->
      <div class="rpt-kpi__label">{{ kpi.label }}</div>
      <div class="rpt-kpi__value">{{ kpi.value }}</div>
      <div v-if="kpi.hint" class="rpt-kpi__hint">{{ kpi.hint }}</div>
    </div>
  </div>
</template>

<script setup>
  // Headline numbers: [{ label, value, hint? }], value already formatted.
  import RptHide from './RptHide.vue'

  defineProps({
    items: { type: Array, required: true },
    // The part key, for its Hide button.
    part: { type: String, default: '' },
  })
</script>
