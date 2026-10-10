<template>
  <div class="pa-4 pt-3">
    <div class="d-flex align-center flex-wrap ga-3 mb-2">
      <v-btn-toggle
        v-model="statusModel"
        color="primary"
        density="compact"
        mandatory
        variant="outlined"
      >
        <v-btn text="All" value="" />
        <v-btn
          v-for="(meta, key) in STATUS"
          :key="key"
          :prepend-icon="meta.icon"
          :text="meta.label"
          :value="key"
        />
      </v-btn-toggle>
      <v-spacer />
      <span class="text-caption text-medium-emphasis">{{ total.toLocaleString() }} messages</span>
      <v-text-field
        v-model="searchModel"
        clearable
        density="compact"
        hide-details
        placeholder="Client, phone, invoice, rule…"
        prepend-inner-icon="mdi-magnify"
        style="max-width: 300px"
        variant="outlined"
      />
    </div>

    <v-data-table-server
      v-model:items-per-page="pageSize"
      v-model:page="page"
      density="compact"
      :headers="headers"
      item-value="id"
      :items="messages"
      :items-length="total"
      :items-per-page-options="[25, 50, 100, 250]"
      :loading="loading"
      no-data-text="No messages match."
    >
      <template #item.created_at="{ item }">
        <span :title="new Date(item.created_at).toLocaleString()">{{ when(item.created_at) }}</span>
      </template>
      <template #item.client="{ item }">
        <div>{{ item.client_name || '—' }}</div>
        <div class="text-caption text-medium-emphasis">{{ formatPhone(item.client_phone) }}</div>
      </template>
      <template #item.grand_total_cents="{ item }">
        {{ formatDollars(item.grand_total_cents) }}
        <div v-if="item.item_count" class="text-caption text-medium-emphasis">
          {{ item.item_count }} item{{ item.item_count === 1 ? '' : 's' }}
        </div>
      </template>
      <template #item.status="{ item }">
        <v-chip
          :color="STATUS[item.status]?.color"
          :prepend-icon="STATUS[item.status]?.icon"
          size="x-small"
          variant="tonal"
        >
          {{ STATUS[item.status]?.label ?? item.status }}
        </v-chip>
      </template>
      <template #item.rule="{ item }">
        <div>{{ item.rule_name || 'Default' }}</div>
        <div v-if="item.template_name" class="text-caption text-medium-emphasis">{{ item.template_name }}</div>
      </template>
      <template #item.reason="{ item }">
        <!-- A skip's reason is just its rule, already in the column before. -->
        <span v-if="item.status !== 'skipped'" class="text-caption" :class="{ 'text-error': item.status === 'failed' }">
          {{ item.reason }}
        </span>
      </template>
      <template #item.cost_micros="{ item }">
        {{ item.cost_micros ? formatCost(item.cost_micros) : '' }}
      </template>
    </v-data-table-server>
  </div>
</template>

<script setup>
  import { computed, onUnmounted, ref, watch } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { formatCost, formatDollars, formatPhone, STATUS } from '@/utils/whatsapp'

  const props = defineProps({
    from: { type: String, required: true },
    to: { type: String, required: true },
    branchId: { type: Number, default: 0 },
    refreshKey: { type: Number, default: 0 },
    status: { type: String, default: '' },
    search: { type: String, default: '' },
  })
  const emit = defineEmits(['update:status', 'update:search'])

  const notify = useNotify()
  const { listMessages } = useWhatsApp()

  // Status and search live on the page, so Overview can open this tab
  // already filtered.
  const statusModel = computed({ get: () => props.status, set: v => emit('update:status', v ?? '') })
  const searchModel = computed({ get: () => props.search, set: v => emit('update:search', v ?? '') })

  const headers = [
    { title: 'When', key: 'created_at', sortable: false },
    { title: 'Client', key: 'client', sortable: false },
    { title: 'Branch', key: 'branch_name', sortable: false },
    { title: 'Invoice', key: 'invoice_code', sortable: false },
    { title: 'Total', key: 'grand_total_cents', sortable: false, align: 'end' },
    { title: 'Outcome', key: 'status', sortable: false },
    { title: 'Rule / template', key: 'rule', sortable: false },
    { title: 'Why', key: 'reason', sortable: false },
    { title: 'Cost', key: 'cost_micros', sortable: false, align: 'end' },
  ]

  const messages = ref([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(1)
  const pageSize = ref(50)

  async function load () {
    loading.value = true
    try {
      const data = await listMessages({
        from: props.from,
        to: props.to,
        branch_id: props.branchId,
        status: props.status,
        search: (props.search ?? '').trim(),
        page_size: pageSize.value,
        page_id: (page.value - 1) * pageSize.value,
      })
      messages.value = data.messages ?? []
      total.value = data.total ?? 0
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  function when (value) {
    const d = new Date(value)
    const sameDay = d.toDateString() === new Date().toDateString()
    return sameDay
      ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
      : d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  let searchTimer = 0
  watch(() => props.search, () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      page.value = 1
      load()
    }, 300)
  })
  watch(() => [props.from, props.to, props.branchId, props.status], () => {
    page.value = 1
    load()
  })
  watch([page, pageSize, () => props.refreshKey], load, { immediate: true })
  onUnmounted(() => clearTimeout(searchTimer))
</script>
