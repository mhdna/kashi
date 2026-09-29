<template>
  <v-card class="alert-list-card d-flex flex-column" variant="flat">
    <div class="d-flex flex-wrap align-center ga-2 px-4 pt-3 pb-2">
      <v-icon :color="color" :icon="icon" size="24" />
      <span class="text-h6 font-weight-medium">{{ title }}</span>
      <span class="text-subtitle-1 text-medium-emphasis">· {{ total }}</span>
      <v-spacer />
      <v-text-field
        v-model="search"
        class="alert-list-search"
        clearable
        density="compact"
        hide-details
        label="Search products"
        prepend-inner-icon="mdi-magnify"
        variant="outlined"
      />
    </div>
    <v-divider />

    <div v-if="error" class="text-error text-center text-body-2 pa-6">{{ error }}</div>

    <v-virtual-scroll
      v-else-if="items.length > 0"
      class="alert-scroll"
      :item-height="72"
      :items="items"
      @scroll.passive="onScroll"
    >
      <template #default="{ item }">
        <v-list-item class="px-4 py-2" @click="$emit('select', item)">
          <template #prepend>
            <v-avatar
              class="me-3"
              :color="color"
              rounded="lg"
              size="40"
              variant="tonal"
            >
              <v-icon :color="color" :icon="icon" size="22" />
            </v-avatar>
          </template>
          <v-list-item-title class="font-weight-medium">{{ item.name }}</v-list-item-title>
          <v-list-item-subtitle>
            {{ item.code }} · On hand: {{ item.on_hand }} · Threshold: {{ thresholdLabel(item) }}
          </v-list-item-subtitle>
          <template #append>
            <v-icon class="text-disabled" icon="mdi-chevron-right" size="20" />
          </template>
        </v-list-item>
        <v-divider />
      </template>
    </v-virtual-scroll>

    <div v-else-if="loading" class="text-center pa-6">
      <v-progress-circular color="primary" indeterminate size="24" width="2" />
    </div>
    <div v-else class="text-medium-emphasis text-center pa-6">{{ emptyText }}</div>

    <div v-if="loading && items.length > 0" class="d-flex justify-center py-2">
      <v-progress-circular color="primary" indeterminate size="16" width="2" />
    </div>
  </v-card>
</template>

<script setup lang="ts">
  import { onMounted } from 'vue'
  import { useAlertList } from '@/composables/useAlertList'

  const props = defineProps({
    endpoint: { type: String, required: true },
    status: { type: String, default: '' },
    title: { type: String, required: true },
    icon: { type: String, required: true },
    color: { type: String, required: true },
    emptyText: { type: String, default: 'Nothing here.' },
  })
  defineEmits(['select'])

  const { items, total, loading, error, search, reload, onScroll } = useAlertList(props.endpoint, props.status)

  // Go's sql.NullInt64 marshals as {Int64, Valid} -- same shape handled in
  // the alerts page for the per-product override.
  function nullableInt (value) {
    if (value == null) return null
    if (typeof value !== 'object') return value
    const valid = value.Valid ?? value.valid
    if (valid === false) return null
    return value.Int64 ?? value.int64 ?? null
  }

  function thresholdLabel (item) {
    return nullableInt(item.low_stock_threshold) ?? item.effective_threshold ?? '—'
  }

  onMounted(reload)
  defineExpose({ reload })
</script>

<style scoped>
.alert-list-search {
  flex: 0 1 220px;
}
.alert-list-card {
  flex: 1 1 auto;
  min-height: 0;
}
/* Grow to fill the column, but never run away on a tall monitor. */
.alert-scroll {
  flex: 1 1 auto;
  min-height: 280px;
  max-height: 800px;
}
</style>
