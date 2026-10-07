<template>
  <div class="page-root">
    <v-card class="attendance-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-clock-check-outline" />
        <span>Attendance</span>
        <v-spacer />
        <v-select
          v-model="branchFilter"
          class="attendance-branch"
          density="compact"
          hide-details
          item-title="name"
          item-value="id"
          :items="branchOptions"
          label="Branch"
          variant="outlined"
        />
        <v-btn
          prepend-icon="mdi-refresh"
          text="Sync from POS"
          variant="text"
          @click="reload"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-file-delimited-outline"
          text="Import CSV"
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
      <v-divider />

      <ServerSideTable
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
        <v-card-title>Edit attendance punch</v-card-title>
        <v-card-text>
          <v-text-field v-model="form.salesperson_name" density="compact" label="Name" />
          <v-row dense>
            <v-col cols="6">
              <v-text-field v-model="form.event_date" density="compact" label="Date" type="date" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.event_time" density="compact" label="Time" type="time" />
            </v-col>
          </v-row>
          <v-row dense>
            <v-col cols="6">
              <v-select v-model="form.type" density="compact" :items="['IN', 'OUT']" label="Type" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.status" density="compact" label="Status" />
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" @click="dialog = false" />
          <v-btn color="primary" :loading="submitting" text="Save" @click="submit" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAttendance } from '@/composables/useAttendance'
  import { useBranches } from '@/composables/useBranches'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const { importCsv, updateEvent, deleteEvent } = useAttendance()
  const { branches, fetchBranches } = useBranches()

  const apiURL = `${API_BASE}/branch_attendance_events`
  const headers = [
    { title: 'Branch', key: 'branch_id', align: 'start', sortable: false },
    { title: 'Date', key: 'event_date', align: 'start' },
    { title: 'Time', key: 'event_time' },
    { title: 'Name', key: 'salesperson_name', align: 'start' },
    { title: 'Type', key: 'type' },
    { title: 'Status', key: 'status' },
    { title: 'User ID', key: 'attendance_user_id' },
    { title: '', key: 'actions', align: 'end', sortable: false },
  ]

  // 'all' lists every branch's punches; importing still needs one branch.
  const ALL_BRANCHES = 'all'
  const branchFilter = ref(ALL_BRANCHES)
  const branchId = computed(() => (branchFilter.value === ALL_BRANCHES ? null : branchFilter.value))
  const branchOptions = computed(() => [{ id: ALL_BRANCHES, name: 'All branches' }, ...branches.value])
  const branchNames = computed(() => Object.fromEntries(branches.value.map(b => [b.id, b.name])))
  const tableRef = ref(null)
  const fileInput = ref(null)
  const notify = useNotify()

  function reload () {
    tableRef.value?.reload()
  }

  onMounted(() => {
    fetchBranches().catch(() => {})
  })

  async function onFilePicked (event) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (!branchId.value) {
      notify.warning('Pick a branch (not "All branches") before importing.')
      return
    }
    try {
      const res = await importCsv(file, branchId.value)
      notify.success(`Imported ${res?.inserted ?? 0} punches (${res?.skipped ?? 0} already present).`)
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
    if (!confirm(`Delete the ${item.event_date} ${item.event_time} punch for ${item.salesperson_name || 'this employee'}?`)) return
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
