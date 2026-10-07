<!--
  The bordered, uppercase-titled list panel from Sales Analytics'
  SalesAnalyticsTab: label on the left, value + muted share on the right,
  optional caption under the label.
    rows: [{ key, label, sub?, value, pct? }]
-->
<template>
  <v-card border class="d-flex flex-column fill-height" flat tile>
    <v-card-title class="text-caption text-uppercase text-grey py-1 px-3">{{ title }}</v-card-title>
    <v-divider />
    <div v-if="rows.length === 0" class="text-caption text-medium-emphasis px-3 py-4">{{ emptyText }}</div>
    <v-list v-else class="pa-0 flex-grow-1" density="compact">
      <v-list-item v-for="row in rows" :key="row.key" class="px-3" :subtitle="row.sub || undefined">
        <template #title>{{ row.label }}</template>
        <template #append>
          <span class="text-caption mr-3 text-no-wrap">{{ row.value }}</span>
          <span v-if="row.pct != null" class="list-panel__pct text-caption text-grey">{{ row.pct }}</span>
        </template>
      </v-list-item>
    </v-list>
  </v-card>
</template>

<script setup>
  defineProps({
    title: { type: String, required: true },
    rows: { type: Array, default: () => [] },
    emptyText: { type: String, default: 'Nothing recorded.' },
  })
</script>

<style scoped>
.list-panel__pct {
  min-width: 42px;
  text-align: end;
}
</style>
