<!--
  The Emails page's Schedules tab: every scheduled report, when it runs
  next, how its last run went; switch one off, edit it, send it now or
  delete it.
-->
<template>
  <div class="pa-4 pt-3">
    <div class="d-flex align-center mb-2">
      <span class="text-caption text-medium-emphasis">
        Sent automatically at their time (Beirut). Each send shows in History.
      </span>
      <v-spacer />
      <v-btn
        v-if="canEdit"
        color="primary"
        prepend-icon="mdi-plus"
        size="small"
        text="New schedule"
        variant="flat"
        @click="edit(null)"
      />
    </div>

    <v-data-table
      density="compact"
      :headers="headers"
      item-value="id"
      :items="schedules"
      :items-per-page="-1"
      :loading="loading"
      no-data-text="No scheduled reports yet."
    >
      <template #item.enabled="{ item }">
        <v-switch
          color="primary"
          density="compact"
          :disabled="!canEdit"
          hide-details
          :model-value="item.enabled"
          @update:model-value="value => toggle(item, value)"
        />
      </template>
      <template #item.name="{ item }">
        <div class="font-weight-medium">{{ item.name || 'Untitled' }}</div>
        <div class="text-caption text-medium-emphasis">
          {{ item.period_label }} · {{ registersLabel(item) }} · {{ (item.format || 'csv').toUpperCase() }}
        </div>
      </template>
      <template #item.when="{ item }">
        <div>{{ when(item) }}</div>
        <div v-if="item.enabled && item.next_run_at" class="text-caption text-medium-emphasis">
          next {{ dateTime(item.next_run_at) }}
        </div>
      </template>
      <template #item.recipients="{ item }">
        <span class="text-caption">{{ item.recipients }}</span>
        <div v-if="item.bcc" class="text-caption text-medium-emphasis">Bcc: {{ item.bcc }}</div>
      </template>
      <template #item.last="{ item }">
        <template v-if="item.last_run_at">
          <v-chip
            :color="LAST[item.last_status]?.color"
            size="x-small"
            :text="LAST[item.last_status]?.label ?? item.last_status"
            variant="tonal"
          />
          <div class="text-caption text-medium-emphasis">{{ dateTime(item.last_run_at) }}</div>
          <div v-if="item.last_status === 'failed'" class="text-caption text-error">{{ item.last_error }}</div>
        </template>
        <span v-else class="text-caption text-medium-emphasis">Not yet</span>
      </template>
      <template #item.actions="{ item }">
        <div v-if="canEdit" class="d-flex justify-end">
          <v-btn
            icon="mdi-send-outline"
            :loading="running === item.id"
            size="small"
            title="Send now"
            variant="text"
            @click="runNow(item)"
          />
          <v-btn
            icon="mdi-pencil"
            size="small"
            title="Edit"
            variant="text"
            @click="edit(item)"
          />
          <v-btn
            icon="mdi-delete"
            size="small"
            title="Delete"
            variant="text"
            @click="remove(item)"
          />
        </div>
      </template>
    </v-data-table>

    <EmailScheduleDialog v-model="dialogOpen" :schedule="editing" @saved="load" />
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import EmailScheduleDialog from '@/components/Email/EmailScheduleDialog.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useEmailReports } from '@/composables/useEmailReports'
  import { useNotify } from '@/composables/useNotify'
  import { useAuthStore } from '@/stores/auth'

  const emit = defineEmits(['sent'])

  const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const LAST = {
    sent: { label: 'Sent', color: 'success' },
    skipped: { label: 'Nothing to send', color: undefined },
    failed: { label: 'Failed', color: 'error' },
  }
  const headers = [
    { title: 'On', key: 'enabled', sortable: false, width: 70 },
    { title: 'Report', key: 'name', sortable: false },
    { title: 'When', key: 'when', sortable: false },
    { title: 'To', key: 'recipients', sortable: false },
    { title: 'Last run', key: 'last', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ]

  const auth = useAuthStore()
  const notify = useNotify()
  const canEdit = computed(() => auth.canEditPage('analytics', 'daily-income'))
  const {
    listEmailSchedules,
    saveEmailSchedule,
    deleteEmailSchedule,
    runEmailSchedule,
    listEmailReportRegisters,
  } = useEmailReports()

  const schedules = ref([])
  const registers = ref([])
  const loading = ref(false)

  async function load () {
    loading.value = true
    try {
      const [list, regs] = await Promise.all([
        listEmailSchedules(),
        registers.value.length > 0 ? registers.value : listEmailReportRegisters(),
      ])
      schedules.value = list
      registers.value = regs
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  // A run the scheduler just made shows up without a refresh.
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  onMounted(() => {
    load()
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message?.type === 'email_schedule_ran') load()
    })
  })
  onUnmounted(() => unsubscribe?.())

  function registersLabel (item) {
    const ids = item.register_ids ?? []
    if (ids.length === 0) return 'All registers'
    const names = ids.map(id => {
      const r = registers.value.find(reg => reg.id === id)
      return r ? `${r.branch_name} ${r.code}` : `#${id}`
    })
    return names.length > 2 ? `${names.slice(0, 2).join(', ')} +${names.length - 2}` : names.join(', ')
  }

  function when (item) {
    const at = `${String(Math.floor(item.at_minute / 60)).padStart(2, '0')}:${String(item.at_minute % 60).padStart(2, '0')}`
    if (item.frequency === 'weekly') {
      return `${(item.weekdays ?? []).map(d => WEEKDAYS[d]).join(', ')} at ${at}`
    }
    if (item.frequency === 'monthly') return `Day ${item.month_day} of each month at ${at}`
    return `Every day at ${at}`
  }

  function dateTime (value) {
    return new Date(value).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  // The API takes the whole schedule back on save.
  function payloadOf (item, changes = {}) {
    return {
      name: item.name,
      enabled: item.enabled,
      period: item.period,
      register_ids: item.register_ids ?? [],
      recipients: item.recipients,
      bcc: item.bcc,
      format: item.format,
      note: item.note,
      skip_empty: item.skip_empty,
      frequency: item.frequency,
      time: `${String(Math.floor(item.at_minute / 60)).padStart(2, '0')}:${String(item.at_minute % 60).padStart(2, '0')}`,
      weekdays: item.weekdays ?? [],
      month_day: item.month_day,
      ...changes,
    }
  }

  async function toggle (item, enabled) {
    try {
      await saveEmailSchedule(item.id, payloadOf(item, { enabled }))
      await load()
    } catch (error) {
      notify.error(error.message)
    }
  }

  const dialogOpen = ref(false)
  const editing = ref(null)
  function edit (item) {
    editing.value = item
    dialogOpen.value = true
  }

  const running = ref(null)
  async function runNow (item) {
    running.value = item.id
    try {
      const report = await runEmailSchedule(item.id)
      notify.success(`Sent to ${report?.recipients ?? item.recipients}.`)
      emit('sent')
    } catch (error) {
      notify.error(error.message)
    } finally {
      running.value = null
      load()
    }
  }

  async function remove (item) {
    if (!window.confirm(`Delete “${item.name || 'Untitled'}”? Reports it already sent stay in History.`)) return
    try {
      await deleteEmailSchedule(item.id)
      await load()
    } catch (error) {
      notify.error(error.message)
    }
  }
</script>
