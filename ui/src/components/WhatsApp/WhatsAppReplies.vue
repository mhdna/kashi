<template>
  <div class="pa-4 pt-3">
    <div class="d-flex align-center flex-wrap ga-3 mb-2">
      <span class="text-body-2 text-medium-emphasis">
        What clients wrote back to the business number. Needs the webhook set up in
        <router-link to="/settings?tab=whatsapp">Settings</router-link>.
      </span>
      <v-spacer />
      <span class="text-caption text-medium-emphasis">{{ total.toLocaleString() }} replies</span>
      <v-text-field
        v-model="search"
        clearable
        density="compact"
        hide-details
        placeholder="Client, phone, text, invoice…"
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
      :items="replies"
      :items-length="total"
      :items-per-page-options="tablePageSizeOptions(pageSize)"
      :loading="loading"
      no-data-text="No replies in this range."
    >
      <template #item.received_at="{ item }">
        <span :title="new Date(item.received_at).toLocaleString()">{{ when(item.received_at) }}</span>
      </template>
      <template #item.client="{ item }">
        <div>{{ item.client_name || '—' }}</div>
        <div class="text-caption text-medium-emphasis">{{ formatPhone(item.client_phone) }}</div>
      </template>
      <template #item.body="{ item }">
        <span v-if="item.body" class="wr-body">{{ item.body }}</span>
        <span v-else class="text-caption text-medium-emphasis">({{ item.kind || 'message' }})</span>
      </template>
      <template #item.answers="{ item }">
        <template v-if="item.invoice_code">
          <div>{{ item.invoice_code }}<span v-if="item.branch_name" class="text-medium-emphasis"> · {{ item.branch_name }}</span></div>
          <div class="text-caption text-medium-emphasis">{{ item.template_name }}</div>
        </template>
        <span v-else class="text-caption text-medium-emphasis">—</span>
      </template>
    </v-data-table-server>
  </div>
</template>

<script setup>
  import { onUnmounted, ref, watch } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { formatPhone } from '@/utils/whatsapp'
  import { tablePageSizeOptions, useSettingsStore } from '@/stores/settings'

  const props = defineProps({
    from: { type: String, required: true },
    to: { type: String, required: true },
    branchId: { type: Number, default: 0 },
    refreshKey: { type: Number, default: 0 },
  })

  const notify = useNotify()
  const { listReplies } = useWhatsApp()

  const headers = [
    { title: 'When', key: 'received_at', sortable: false },
    { title: 'Client', key: 'client', sortable: false },
    { title: 'Message', key: 'body', sortable: false },
    { title: 'Answering', key: 'answers', sortable: false },
  ]

  const replies = ref([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(1)
  const pageSize = ref(useSettingsStore().tablePageSize)
  const search = ref('')

  async function load () {
    loading.value = true
    try {
      const data = await listReplies({
        from: props.from,
        to: props.to,
        branch_id: props.branchId,
        search: (search.value ?? '').trim(),
        page_size: pageSize.value,
        page_id: (page.value - 1) * pageSize.value,
      })
      replies.value = data.replies ?? []
      total.value = data.total ?? 0
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  function when (value) {
    const d = new Date(value)
    return d.toDateString() === new Date().toDateString()
      ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
      : d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  let searchTimer = 0
  watch(search, () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      page.value = 1
      load()
    }, 300)
  })
  watch(() => [props.from, props.to, props.branchId], () => {
    page.value = 1
    load()
  })
  watch([page, pageSize, () => props.refreshKey], load, { immediate: true })
  onUnmounted(() => clearTimeout(searchTimer))
</script>

<style scoped>
.wr-body {
  white-space: pre-wrap;
}
</style>
