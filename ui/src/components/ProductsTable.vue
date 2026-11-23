<template>
  <ServerSideTable
    ref="tableRef"
    :api-u-r-l="apiURL"
    density="comfortable"
    :external-search="props.search"
    flush
    :headers="headers"
    hover
    :query-params="queryParams"
    root-key="products"
    :show-search-icon="false"
    @row-click="$emit('edit', $event)"
  >
    <template #item.is_active="{ item }">
      <v-icon
        :color="item.is_active ? 'success' : 'disabled'"
        :icon="item.is_active ? 'mdi-check-circle' : 'mdi-close-circle'"
        size="18"
      />
    </template>
    <template #item.name="{ item }">
      {{ item.name }}
      <v-chip v-if="isFromPhoto(item)" class="ms-1" data-test="from-photo" label size="x-small">{{ t('From photo') }}</v-chip>
    </template>
    <template #item.created_at="{ item }">
      {{ formatDate(item.created_at) }}
    </template>
  </ServerSideTable>
</template>

<script setup>
  import { computed, ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'
  import { isFromPhoto, productQueryParams } from '@/utils/productQuery'

  const props = defineProps({
    search: {
      type: String,
      default: '',
    },
    filters: {
      type: Object,
      default: () => ({}),
    },
  })

  defineEmits(['edit'])

  const apiURL = `${API_BASE}/products`
  const headers = computed(() => [
    { title: t('ID'), key: 'id', align: 'start' },
    { title: t('Code'), key: 'code', align: 'start' },
    { title: t('Name'), key: 'name', align: 'start' },
    { title: t('Active'), key: 'is_active', align: 'center' },
    { title: t('Created At'), key: 'created_at', align: 'end' },
  ])

  const queryParams = computed(() => productQueryParams(props.filters))

  function formatDate (value) {
    if (!value) return '—'
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString()
  }

  const tableRef = ref(null)
  function reload () {
    tableRef.value?.reload()
  }

  defineExpose({ reload })
</script>
