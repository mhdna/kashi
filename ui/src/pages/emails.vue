<template>
  <div class="em d-flex flex-column fill-height">
    <div class="d-flex align-center flex-wrap flex-shrink-0 ga-3 pa-4 pb-2">
      <v-icon icon="mdi-email-outline" />
      <div class="text-h6 font-weight-medium">Emails</div>
      <v-chip
        v-if="configured !== null"
        :color="configured ? 'success' : 'warning'"
        size="small"
        :text="configured ? `From ${sender}` : 'Not set up: add the API key in Settings'"
        :to="auth.canViewPage('system', 'settings') ? '/settings?tab=email' : undefined"
        variant="tonal"
      />
      <v-spacer />
      <v-btn-toggle
        v-if="tab === 'history'"
        v-model="status"
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
      <v-btn
        v-if="tab === 'history'"
        icon="mdi-refresh"
        title="Refresh"
        variant="text"
        @click="load"
      />
      <v-btn
        v-if="auth.canEditPage('analytics', 'daily-income')"
        color="primary"
        prepend-icon="mdi-send-outline"
        text="Send report"
        variant="flat"
        @click="dialogOpen = true"
      />
    </div>

    <v-tabs v-model="tab" class="flex-shrink-0 px-4" color="primary" density="compact">
      <v-tab prepend-icon="mdi-history" text="History" value="history" />
      <v-tab prepend-icon="mdi-calendar-clock-outline" text="Schedules" value="schedules" />
    </v-tabs>
    <v-divider class="flex-shrink-0" />

    <div v-if="tab === 'schedules'" class="em__body">
      <EmailSchedules @sent="load" />
    </div>
    <div v-else class="em__body pa-4 pt-3">
      <v-data-table-server
        v-model:items-per-page="pageSize"
        v-model:page="page"
        density="compact"
        :headers="headers"
        item-value="id"
        :items="reports"
        :items-length="total"
        :items-per-page-options="[25, 50, 100, 250]"
        :loading="loading"
        no-data-text="No emails sent yet."
      >
        <template #item.created_at="{ item }">
          <span :title="new Date(item.created_at).toLocaleString()">{{ when(item.created_at) }}</span>
          <div v-if="item.sent_by" class="text-caption text-medium-emphasis">by {{ item.sent_by }}</div>
        </template>
        <template #item.label="{ item }">
          <div>{{ item.label }}</div>
          <div class="text-caption text-medium-emphasis">{{ item.filename }}</div>
          <div v-if="item.note" class="email-note text-caption" :title="item.note">“{{ item.note }}”</div>
        </template>
        <template #item.invoice_count="{ item }">
          {{ item.invoice_count.toLocaleString() }}
        </template>
        <template #item.total_cents="{ item }">
          {{ money(item.total_cents) }}
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
          <div v-if="item.error" class="text-caption text-error">{{ item.error }}</div>
        </template>
      </v-data-table-server>
    </div>

    <SendEmailReportDialog v-model="dialogOpen" @sent="load" />
  </div>
</template>

<script setup>
  import { onMounted, ref, watch } from 'vue'
  import EmailSchedules from '@/components/Email/EmailSchedules.vue'
  import SendEmailReportDialog from '@/components/Email/SendEmailReportDialog.vue'
  import { useEmailReports } from '@/composables/useEmailReports'
  import { useNotify } from '@/composables/useNotify'
  import { useAuthStore } from '@/stores/auth'

  // Every invoice report emailed from kashi (api/email_report.go), sent or
  // failed, newest first. Sender, reply-to and CSV columns are on
  // Settings > Email.

  const STATUS = {
    sent: { label: 'Sent', color: 'success', icon: 'mdi-check-circle' },
    failed: { label: 'Failed', color: 'error', icon: 'mdi-alert-circle' },
  }

  const headers = [
    { title: 'When', key: 'created_at', sortable: false },
    { title: 'Report', key: 'label', sortable: false },
    { title: 'To', key: 'recipients', sortable: false },
    { title: 'Invoices', key: 'invoice_count', sortable: false, align: 'end' },
    { title: 'Net total', key: 'total_cents', sortable: false, align: 'end' },
    { title: 'Status', key: 'status', sortable: false },
  ]

  const auth = useAuthStore()
  const notify = useNotify()
  const { getEmailSettings, listEmailReports } = useEmailReports()

  const reports = ref([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(1)
  const pageSize = ref(50)
  const status = ref('')
  const dialogOpen = ref(false)
  const tab = ref('history')
  const configured = ref(null)
  const sender = ref('')

  async function load () {
    loading.value = true
    try {
      const data = await listEmailReports({
        status: status.value,
        pageSize: pageSize.value,
        offset: (page.value - 1) * pageSize.value,
      })
      reports.value = data.reports
      total.value = data.total
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  onMounted(async () => {
    try {
      const data = await getEmailSettings()
      configured.value = Boolean(data?.configured)
      sender.value = data?.settings?.email_sender_address ?? ''
    } catch {
      // The chip just stays hidden.
    }
  })

  watch(status, () => {
    page.value = 1
    load()
  })
  watch([page, pageSize], load, { immediate: true })

  function when (value) {
    const d = new Date(value)
    const sameDay = d.toDateString() === new Date().toDateString()
    return sameDay
      ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
      : d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  function money (cents) {
    const n = Number(cents || 0) / 100
    const abs = Math.abs(n).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
    return `${n < 0 ? '−' : ''}$${abs}`
  }
</script>

<style scoped>
.em {
  min-height: 0;
}
.email-note {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.em__body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
</style>
