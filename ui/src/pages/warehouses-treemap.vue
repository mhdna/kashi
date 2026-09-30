<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import WarehouseChart from '@/components/Stats/WarehouseChart.vue'
  import { useNotify } from '@/composables/useNotify'
  import { useWarehouseTreemap } from '@/composables/useWarehouseTreemap'

  const notify = useNotify()
  const { fetchRows, totals, locationOptions } = useWarehouseTreemap()

  const options = ref([{ title: 'All Locations', value: 'all' }])
  const selected = ref('all')

  const totalValue = computed(() => '$ ' + Math.round(totals(selected.value).valueCents / 100).toLocaleString('en-US'))
  const totalCount = computed(() => totals(selected.value).qty.toLocaleString('en-US') + ' units')

  onMounted(async () => {
    try {
      await fetchRows()
      options.value = [{ title: 'All Locations', value: 'all' }, ...locationOptions()]
    } catch (error) {
      notify.error(error.message)
    }
  })
</script>

<template>
  <div class="warehouses-treemap-page">
    <div class="filters d-flex align-center ga-3 pa-3">
      <v-select
        v-model="selected"
        density="compact"
        hide-details
        :items="options"
        label="Location"
        max-width="260"
        variant="outlined"
      />

      <v-spacer />
      <div class="text-center">
        <div class="text-h6 font-weight-bold">{{ totalValue }}</div>
        <div class="text-caption text-medium-emphasis">Total book value &middot; {{ totalCount }}</div>
      </div>
      <v-spacer />
    </div>

    <div class="chart-area">
      <WarehouseChart :location="selected" />
    </div>
  </div>
</template>

<style scoped>
.warehouses-treemap-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
}

.filters {
  flex: 0 0 auto;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.chart-area {
  flex: 1 1 auto;
  min-height: 0;
}
</style>
