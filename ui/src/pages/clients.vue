<template>
  <v-dialog v-model="dialog" max-width="480">
    <v-card class="px-4">
      <v-card-title>{{ editingId ? 'Edit Client' : 'Add a New Client' }}</v-card-title>
      <v-card-text>
        <form @submit.prevent="submit">
          <v-text-field
            v-model="name.value.value"
            density="compact"
            :error-messages="name.errorMessage.value"
            label="Name"
          />
          <v-text-field
            v-model="phone.value.value"
            density="compact"
            :error-messages="phone.errorMessage.value"
            label="Phone"
          />
          <v-select
            v-model="clientType"
            density="compact"
            :items="[{ title: 'Retail', value: 'retail' }, { title: 'Wholesale', value: 'wholesale' }]"
            label="Type"
          />

          <v-card-actions class="px-0">
            <v-spacer />
            <v-btn text="Cancel" @click="closeDialog" />
            <v-btn color="primary" :loading="submitting" :text="editingId ? 'Save' : 'Add Client'" type="submit" />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <v-dialog v-model="exportDialog" max-width="520">
    <v-card>
      <v-card-title>Export Clients to CSV</v-card-title>
      <v-card-text>
        <div class="text-caption text-medium-emphasis mb-1">Fields</div>
        <div class="d-flex flex-wrap mb-2">
          <v-checkbox
            v-for="field in exportFieldOptions"
            :key="field.value"
            v-model="exportFields"
            density="compact"
            hide-details
            :label="field.title"
            style="flex: 0 0 50%"
            :value="field.value"
          />
        </div>

        <v-select
          v-model="exportBranchIds"
          chips
          clearable
          closable-chips
          density="compact"
          hide-details
          item-title="name"
          item-value="id"
          :items="branches"
          label="Branches (empty = all)"
          multiple
          variant="outlined"
        />

        <div class="mt-3">
          <DateRangePicker v-model="exportRange" clearable empty-label="Came any time" />
        </div>
        <v-checkbox
          v-model="exportNoDuplicates"
          density="compact"
          hide-details
          label="No duplicates"
        />
        <v-checkbox
          v-model="exportIncludeExcluded"
          density="compact"
          hide-details
          label="Include cancelled invoices and hidden branches"
        />
        <div class="text-caption text-medium-emphasis mt-2">
          A client belongs to a branch if they bought something there (cancelled invoices and
          branches hidden from reports are skipped unless you include them). Each selected branch gets
          its own file (a zip when there are several); with none selected you get one file of
          every {{ tab }} client. Dates keep only those who bought in that period, and Total
          spent / Invoices count just that scope.
          <b>No duplicates</b> removes anyone with the same phone in more than one file from all
          of them, and keeps one row per phone inside a file. <b>Phone check</b> adds a column
          marking numbers that aren't valid Lebanese numbers as Invalid.
        </div>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Cancel" @click="exportDialog = false" />
        <v-btn
          color="primary"
          :disabled="exportFields.length === 0"
          :loading="exporting"
          text="Export"
          @click="exportCsv"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="deleteDialog" max-width="420">
    <v-card>
      <v-card-title>Delete Client</v-card-title>
      <v-card-text>
        Delete "{{ deleteTarget?.name }}"? A client with invoices, loyalty history or a
        branch link is deactivated instead, so their history stays intact and every till
        stops offering them. You can reactivate them later.
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Cancel" @click="deleteDialog = false" />
        <v-btn color="error" :loading="deleting" text="Delete" @click="confirmDelete" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <div class="clients-toolbar d-flex flex-wrap align-center justify-space-between ga-3 mb-2">
    <v-tabs v-model="tab" color="primary">
      <v-tab value="retail">Retail</v-tab>
      <v-tab value="wholesale">Wholesale</v-tab>
    </v-tabs>

    <div class="d-flex align-center ga-2">
      <v-menu>
        <template #activator="{ props: menuProps }">
          <v-btn
            prepend-icon="mdi-filter"
            rounded="lg"
            style="border-color: rgb(var(--v-theme-surface-light));"
            v-bind="menuProps"
            variant="outlined"
          >
            {{ selectedBranchLabel }}
          </v-btn>
        </template>
        <v-list>
          <v-list-item :active="selectedBranchIds.length === 0" @click="selectedBranchIds = []">
            <v-list-item-title>All Branches</v-list-item-title>
          </v-list-item>
          <v-list-item
            v-for="branch in branches"
            :key="branch.id"
            :active="selectedBranchIds.includes(branch.id)"
            @click="toggleBranch(branch.id)"
          >
            <template #prepend>
              <v-icon v-if="selectedBranchIds.includes(branch.id)" icon="mdi-check" size="small" />
            </template>
            <v-list-item-title>{{ branch.name }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
      <DateRangePicker v-model="dateRange" clearable />
      <v-btn
        prepend-icon="mdi-file-delimited-outline"
        rounded="lg"
        style="border-color: rgb(var(--v-theme-surface-light));"
        text="Export CSV"
        variant="outlined"
        @click="openExport"
      />
      <v-btn
        v-if="canEdit"
        color="primary"
        prepend-icon="mdi-plus"
        text="Add Client"
        @click="openCreate"
      />
    </div>
  </div>

  <ClientLoyaltyBoard
    :key="tab"
    ref="boardRef"
    :branch-ids="selectedBranchIds"
    :client-type="tab"
    :date-range="dateRange"
    @delete="openDelete"
    @edit="openEdit"
    @reactivate="reactivate"
  />

</template>

<script setup>
  import { useField, useForm } from 'vee-validate'
  import { computed, onMounted, ref } from 'vue'
  import ClientLoyaltyBoard from '@/components/Clients/ClientLoyaltyBoard.vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useClients } from '@/composables/useClients'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'
  import { rangeBounds } from '@/utils/dateRange'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const { createClient, updateClient, deleteClient, setClientActive } = useClients()
  const notify = useNotify()
  const { branches, fetchBranches } = useBranches()

  onMounted(() => fetchBranches())

  const selectedBranchIds = ref([])
  const selectedBranchLabel = computed(() => {
    if (selectedBranchIds.value.length === 0) return 'All Branches'
    if (selectedBranchIds.value.length === 1) {
      return branches.value.find(b => b.id === selectedBranchIds.value[0])?.name ?? 'All Branches'
    }
    return `${selectedBranchIds.value.length} Branches`
  })

  function toggleBranch (id) {
    selectedBranchIds.value = selectedBranchIds.value.includes(id)
      ? selectedBranchIds.value.filter(b => b !== id)
      : [...selectedBranchIds.value, id]
  }

  // Clients who bought in this window (and their spend in it); [] = any time.
  const dateRange = ref([])

  // ---- CSV export ----------------------------------------------------------
  const exportFieldOptions = [
    { title: 'Name', value: 'name' },
    { title: 'Phone', value: 'phone' },
    { title: 'Phone check (Lebanese)', value: 'phone_check' },
    { title: 'Type', value: 'client_type' },
    { title: 'Status', value: 'status' },
    { title: 'Member since', value: 'created_at' },
    { title: 'Loyalty points', value: 'loyalty' },
    { title: 'Total spent', value: 'total_spent' },
    { title: 'Invoices', value: 'invoices' },
    { title: 'Last visit', value: 'last_visit' },
  ]
  const exportDialog = ref(false)
  const exporting = ref(false)
  const exportFields = ref(['name', 'phone'])
  const exportBranchIds = ref([])
  const exportNoDuplicates = ref(false)
  const exportIncludeExcluded = ref(false)
  const exportRange = ref([])

  function openExport () {
    // Start from the branches already filtered on the page.
    exportBranchIds.value = [...selectedBranchIds.value]
    exportRange.value = [...dateRange.value]
    exportDialog.value = true
  }

  async function exportCsv () {
    exporting.value = true
    try {
      const params = new URLSearchParams()
      params.set('client_type', tab.value)
      // Keep the checkbox order stable rather than click order.
      params.set('fields', exportFieldOptions.map(f => f.value).filter(v => exportFields.value.includes(v)).join(','))
      for (const id of exportBranchIds.value) params.append('branch_ids', id)
      if (exportNoDuplicates.value) params.set('no_duplicates', true)
      if (exportIncludeExcluded.value) params.set('include_excluded', true)
      const { from, to } = rangeBounds(exportRange.value)
      if (from) params.set('from', from)
      if (to) params.set('to', to)

      const res = await authFetch(`${API_BASE}/clients/export?${params}`)
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || `Export failed with status ${res.status}`)
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      // One file comes back as a .csv, several branches as a .zip.
      link.download = res.headers.get('Content-Disposition')?.match(/filename="([^"]+)"/)?.[1] ?? (blob.type === 'application/zip' ? 'clients.zip' : 'clients.csv')
      link.click()
      URL.revokeObjectURL(url)
      exportDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      exporting.value = false
    }
  }

  const tab = ref('retail')
  const boardRef = ref(null)
  const dialog = ref(false)
  const submitting = ref(false)
  const editingId = ref(null)

  const { handleSubmit, handleReset, setValues } = useForm({
    validationSchema: {
      name (value) {
        if (value?.length >= 2) return true
        return 'Name needs to be at least 2 characters.'
      },
      phone (value) {
        if (/^[0-9-]{7,}$/.test(value ?? '')) return true
        return 'Phone number needs to be at least 7 digits.'
      },
    },
  })

  const name = useField('name')
  const phone = useField('phone')
  const clientType = ref('retail')

  function openCreate () {
    editingId.value = null
    handleReset()
    // Default to whichever tab is open, so adding from the Wholesale tab
    // doesn't silently create a retail client.
    clientType.value = tab.value
    dialog.value = true
  }

  function openEdit (item) {
    editingId.value = item.id
    clientType.value = item.client_type ?? 'retail'
    setValues({ name: item.name, phone: item.phone })
    dialog.value = true
  }

  function closeDialog () {
    dialog.value = false
    handleReset()
    editingId.value = null
  }

  const submit = handleSubmit(async values => {
    submitting.value = true
    try {
      await (editingId.value
        ? updateClient({ id: editingId.value, name: values.name, phone: values.phone, client_type: clientType.value })
        : createClient({ name: values.name, phone: values.phone, client_type: clientType.value }))
      closeDialog()
      boardRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  })

  const deleteDialog = ref(false)
  const deleteTarget = ref(null)
  const deleting = ref(false)

  function openDelete (item) {
    deleteTarget.value = item
    deleteDialog.value = true
  }

  async function confirmDelete () {
    deleting.value = true
    try {
      const outcome = await deleteClient(deleteTarget.value.id)
      deleteDialog.value = false
      notify.success(outcome === 'deactivated'
        ? `${deleteTarget.value.name} has history, so they were deactivated instead`
        : `${deleteTarget.value.name} deleted`)
      boardRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      deleting.value = false
    }
  }

  async function reactivate (item) {
    try {
      await setClientActive(item.id, true)
      notify.success(`${item.name} reactivated`)
      boardRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    }
  }
</script>
