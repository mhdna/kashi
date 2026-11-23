<template>
  <div class="page-root">
    <v-card class="attendance-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-clock-check-outline" />
        <span>{{ t('Attendance') }}</span>
        <v-spacer />
        <v-select
          v-model="branchFilter"
          class="attendance-branch"
          density="compact"
          hide-details
          item-title="name"
          item-value="id"
          :items="branchOptions"
          :label="t('Branch')"
          variant="outlined"
        />
        <v-btn
          prepend-icon="mdi-clipboard-check-outline"
          :text="pendingRequests.length > 0 ? t('Requests ({n})', { n: pendingRequests.length }) : t('Requests')"
          variant="tonal"
          @click="openRequests"
        />
        <v-btn
          prepend-icon="mdi-file-chart-outline"
          :text="t('Hours report')"
          to="/reports?report=staff&parts=hours,grid"
          variant="tonal"
        />
        <v-btn
          prepend-icon="mdi-refresh"
          :text="t('Sync from POS')"
          variant="text"
          @click="reload"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-file-delimited-outline"
          :text="t('Import CSV')"
          variant="flat"
          @click="fileInput?.click()"
        />
        <input
          ref="fileInput"
          accept=".csv,text/csv"
          hidden
          type="file"
          @change="onFilePicked"
        >
      </v-card-title>
      <v-tabs v-model="tab" class="px-4" color="primary" density="compact">
        <v-tab prepend-icon="mdi-chart-timeline" :text="t('Timeline')" value="timeline" />
        <v-tab prepend-icon="mdi-table-clock" :text="t('In / out pairs')" value="pairs" />
        <v-tab prepend-icon="mdi-calendar" :text="t('Monthly grid')" value="grid" />
        <v-tab prepend-icon="mdi-format-list-bulleted" :text="t('Punches')" value="punches" />
      </v-tabs>
      <v-divider />

      <!-- Timeline, pairs and grid are three views of the same timesheet,
           which pairs each person's IN/OUT punches per day so breaks show as
           gaps between spans. One component keeps its date range and loaded
           punches while switching between them. -->
      <AttendanceTimesheet
        v-if="tab !== 'punches'"
        ref="timesheetRef"
        :branch-id="branchId"
        :branch-names="branchNames"
        :view="tab"
      />

      <ServerSideTable
        v-else
        ref="tableRef"
        :api-u-r-l="apiURL"
        density="comfortable"
        flush
        :headers="headers"
        hover
        item-value="id"
        :query-params="{ branch_id: branchId }"
        root-key="branch_attendance_events"
        :show-search-icon="false"
      >
        <template #item.branch_id="{ item }">
          {{ branchNames[item.branch_id] ?? item.branch_id }}
        </template>
        <template #item.type="{ item }">
          <v-chip
            :color="item.type?.toLowerCase().includes('out') ? 'deep-orange' : 'green'"
            size="x-small"
            :text="item.type || '—'"
          />
        </template>
        <!-- A punch an approved change created (a missed punch added, or a
             corrected time) says who approved it instead of passing for a
             device punch. -->
        <template #item.status="{ item }">
          <v-chip
            v-if="item.changed_by"
            color="info"
            size="x-small"
            :title="item.change_note"
            variant="tonal"
          >
            {{ t('{note} · by {by}', { note: item.change_note || t('Changed'), by: item.changed_by }) }}
          </v-chip>
          <span v-else>{{ item.status }}</span>
        </template>
        <template #item.actions="{ item }">
          <v-btn
            v-if="canEdit"
            icon="mdi-pencil"
            size="x-small"
            variant="text"
            @click.stop="openEdit(item)"
          />
          <v-btn
            v-if="canEdit"
            color="error"
            icon="mdi-delete"
            size="x-small"
            variant="text"
            @click.stop="remove(item)"
          />
        </template>
      </ServerSideTable>
    </v-card>

    <v-dialog v-model="dialog" max-width="460">
      <v-card class="px-4 py-2">
        <v-card-title>{{ t('Edit attendance punch') }}</v-card-title>
        <v-card-text>
          <v-text-field v-model="form.salesperson_name" density="compact" :label="t('Name')" />
          <v-row dense>
            <v-col cols="6">
              <DateField v-model="form.event_date" density="compact" :label="t('Date')" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.event_time" density="compact" :label="t('Time')" type="time" />
            </v-col>
          </v-row>
          <v-row dense>
            <v-col cols="6">
              <v-select v-model="form.type" density="compact" :items="['IN', 'OUT']" :label="t('Type')" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.status" density="compact" :label="t('Status')" />
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn :text="t('Cancel')" @click="dialog = false" />
          <v-btn color="primary" :loading="submitting" :text="t('Save')" @click="submit" />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Change requests cashiers filed on a till (fix, delete or add a punch)
         that nobody has decided yet. A decision is sent to the till that
         filed it and applies when that till is next online; the till's
         decision then syncs back and the request leaves this list. -->
    <v-dialog v-model="requestsDialog" max-width="760">
      <v-card>
        <v-card-title>{{ t('Attendance requests') }}</v-card-title>
        <v-card-text>
          <div v-if="requestsLoading" class="d-flex justify-center pa-4">
            <PreparingState kind="people" min-height="160px" :title="t('Loading requests')" />
          </div>
          <v-alert v-else-if="pendingRequests.length === 0" type="info" variant="tonal">
            {{ t('No pending attendance requests.') }}
          </v-alert>
          <v-table v-else density="compact">
            <thead>
              <tr><th>{{ t('Branch') }}</th><th>{{ t('Date') }}</th><th>{{ t('Name') }}</th><th>{{ t('Request') }}</th><th /></tr>
            </thead>
            <tbody>
              <tr v-for="request in pendingRequests" :key="request.id">
                <td>{{ branchNames[request.branch_id] ?? t('Branch #{id}', { id: request.branch_id }) }}</td>
                <td>{{ request.attendance_date }}</td>
                <td>{{ request.salesperson_name }}</td>
                <td>{{ requestSummary(request) }}</td>
                <td class="text-end text-no-wrap">
                  <span v-if="sentRequests[request.id]" class="text-caption text-medium-emphasis">
                    {{ t('{sent} -- waiting for the till', { sent: t(sentRequests[request.id]) }) }}
                  </span>
                  <template v-else-if="canEdit">
                    <v-btn
                      color="error"
                      :loading="reviewingId === request.id"
                      size="small"
                      :text="t('Reject')"
                      variant="text"
                      @click="review(request, false)"
                    />
                    <v-btn
                      color="primary"
                      :loading="reviewingId === request.id"
                      size="small"
                      :text="t('Approve')"
                      variant="tonal"
                      @click="review(request, true)"
                    />
                  </template>
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn :text="t('Close')" @click="requestsDialog = false" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import AttendanceTimesheet from '@/components/AttendanceTimesheet.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAttendance } from '@/composables/useAttendance'
  import { useBranches } from '@/composables/useBranches'
  import { confirmDialog } from '@/composables/useDialog'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { t } from '@/i18n'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const { importCsv, updateEvent, deleteEvent, listPendingRequests, reviewRequest } = useAttendance()
  const { branches, fetchBranches } = useBranches()

  const apiURL = `${API_BASE}/branch_attendance_events`
  const headers = computed(() => [
    { title: t('Branch'), key: 'branch_id', align: 'start', sortable: false },
    { title: t('Date'), key: 'event_date', align: 'start' },
    { title: t('Time'), key: 'event_time' },
    { title: t('Name'), key: 'salesperson_name', align: 'start' },
    { title: t('Type'), key: 'type' },
    { title: t('Status'), key: 'status' },
    { title: t('User ID'), key: 'attendance_user_id' },
    { title: '', key: 'actions', align: 'end', sortable: false },
  ])

  // 'all' lists every branch's punches; importing still needs one branch.
  const ALL_BRANCHES = 'all'
  const branchFilter = ref(ALL_BRANCHES)
  const branchId = computed(() => (branchFilter.value === ALL_BRANCHES ? null : branchFilter.value))
  const branchOptions = computed(() => [{ id: ALL_BRANCHES, name: t('All branches') }, ...branches.value])
  const branchNames = computed(() => Object.fromEntries(branches.value.map(b => [b.id, b.name])))
  const tab = ref('timeline')
  const tableRef = ref(null)
  const timesheetRef = ref(null)
  const fileInput = ref(null)
  const notify = useNotify()

  function reload () {
    tableRef.value?.reload()
    timesheetRef.value?.reload()
  }

  onMounted(() => {
    fetchBranches().catch(() => {})
    loadRequests()
  })

  // ---- change requests filed on the tills ------------------------------
  const requestsDialog = ref(false)
  const requestsLoading = ref(false)
  const pendingRequests = ref([])
  const reviewingId = ref(null)
  // request id -> "Approved" / "Rejected", once the decision is queued; the
  // row stays until the till's own decision syncs back.
  const sentRequests = reactive({})

  async function loadRequests () {
    requestsLoading.value = true
    try {
      pendingRequests.value = await listPendingRequests(branchId.value)
    } catch (error) {
      notify.error(error.message)
    } finally {
      requestsLoading.value = false
    }
  }

  function openRequests () {
    requestsDialog.value = true
    loadRequests()
  }

  function requestSummary (request) {
    if (request.action === 'add') return t('Add {type} {time}', { type: request.requested_type, time: request.requested_time })
    if (request.action === 'delete') return t('Delete {time}', { time: request.original_time })
    return t('Replace {original} with {type} {time}', { original: request.original_time, type: request.requested_type, time: request.requested_time })
  }

  async function review (request, approved) {
    reviewingId.value = request.id
    try {
      await reviewRequest(request.branch_id, request.id, approved)
      sentRequests[request.id] = approved ? 'Approved' : 'Rejected'
    } catch (error) {
      notify.error(error.message)
    } finally {
      reviewingId.value = null
    }
  }

  async function onFilePicked (event) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (!branchId.value) {
      notify.warning(t('Pick a branch (not "All branches") before importing.'))
      return
    }
    try {
      const res = await importCsv(file, branchId.value)
      notify.success(t('Imported {inserted} punches ({skipped} already present).', { inserted: res?.inserted ?? 0, skipped: res?.skipped ?? 0 }))
      reload()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- edit / delete ---------------------------------------------------
  const dialog = ref(false)
  const submitting = ref(false)
  const editingId = ref(null)
  const form = reactive({ salesperson_name: '', event_date: '', event_time: '', type: '', status: '', attendance_user_id: '', event_at: '' })

  function openEdit (item) {
    editingId.value = item.id
    Object.assign(form, {
      salesperson_name: item.salesperson_name ?? '',
      event_date: item.event_date ?? '',
      event_time: item.event_time ?? '',
      type: item.type ?? '',
      status: item.status ?? '',
      attendance_user_id: item.attendance_user_id ?? '',
      event_at: item.event_at ?? '',
    })
    dialog.value = true
  }

  async function submit () {
    submitting.value = true
    try {
      await updateEvent(editingId.value, {
        salesperson_name: form.salesperson_name,
        attendance_user_id: form.attendance_user_id,
        event_date: form.event_date,
        event_time: form.event_time,
        event_at: `${form.event_date} ${form.event_time}`.trim(),
        type: form.type,
        status: form.status,
      })
      dialog.value = false
      reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  }

  async function remove (item) {
    const name = item.salesperson_name || t('this employee')
    if (!(await confirmDialog({ text: t('Delete the {date} {time} punch for {name}?', { date: item.event_date, time: item.event_time, name }), okText: t('Delete'), danger: true }))) return
    try {
      await deleteEvent(item.id)
      reload()
    } catch (error) {
      notify.error(error.message)
    }
  }
</script>

<style scoped>
.page-root {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.attendance-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.attendance-branch {
  flex: 0 1 220px;
}
/* v-alert defaults to flex: 1 1; keep notices at their natural height
   instead of stretching to fill the card. */
.notice {
  flex: 0 0 auto;
}
</style>
