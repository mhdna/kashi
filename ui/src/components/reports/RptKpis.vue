<template>
  <div class="rpt-kpis">
    <div v-for="kpi in items" :key="kpi.label" class="rpt-kpi">
      <div class="rpt-kpi__value">{{ kpi.value }}</div>
      <div class="rpt-kpi__label">{{ kpi.label }}</div>
      <div v-if="kpi.hint" class="rpt-kpi__hint">{{ kpi.hint }}</div>
    </div>
  </div>
</template>

<script setup>
  // Headline numbers: [{ label, value, hint? }], value already formatted.
  defineProps({ items: { type: Array, required: true } })
</script>
