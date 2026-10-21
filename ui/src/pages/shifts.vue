<template>
  <div class="page-root">
    <v-card class="shifts-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-clock-outline" />
        <span>Shifts</span>
        <v-spacer />
        <v-text-field
          v-if="tab === 'central'"
          v-model="search"
          class="shifts-search"
          clearable
          density="compact"
          hide-details
          label="Search cashbox or #id"
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
        />
        <v-select
          v-if="tab === 'central'"
          v-model="statusFilter"
          clearable
          density="compact"
          hide-details
          :items="STATUS_OPTIONS"
          placeholder="Any status"
          style="max-width: 150px"
          variant="outlined"
        />
        <v-select
          v-if="tab === 'central'"
          v-model="cashboxFilter"
          clearable
          density="compact"
          hide-details
          :item-title="c => `${c.name} (${c.code})`"
          :item-value="c => c.id"
          :items="cashboxes"
          placeholder="Any cashbox"
          style="max-width: 220px"
          variant="outlined"
        />
        <v-btn-toggle
          v-model="tab"
          color="primary"
          density="comfortable"
          mandatory
          variant="outlined"
        >
          <v-btn value="branch">Branch</v-btn>
          <v-btn value="central">Central</v-btn>
          <v-btn value="visitors">Visitors</v-btn>
        </v-btn-toggle>
        <v-btn
          v-if="canEdit && tab === 'central'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Open Shift"
          variant="flat"
          @click="openCreate"
        />
      </v-card-title>
      <v-divider />

      <!-- ------------------------------------------------------ CENTRAL -->
      <div v-if="tab === 'central'" class="shifts-content">
        <ServerSideTable
          ref="tableRef"
          :api-u-r-l="`${API_BASE}/shifts`"
          class="shifts-table"
          density="comfortable"
          :external-search="search ?? ''"
          flush
          :headers="centralHeaders"
          :query-params="centralQueryParams"
          root-key="shifts"
          :show-search-icon="false"
          :sort-keys="['id', 'cashbox', 'created_at', 'closed_at']"
        >
          <template #item.cashbox="{ item }">
            <div class="d-flex flex-column">
              <span>{{ item.cashbox_name }}</span>
              <span class="text-medium-emphasis text-caption">{{ item.cashbox_code }}</span>
            </div>
          </template>
          <template #item.is_closed="{ item }">
            <v-chip :color="item.is_closed ? 'default' : 'success'" size="small" variant="tonal">
              {{ item.is_closed ? 'Closed' : 'Open' }}
            </v-chip>
          </template>
          <template #item.created_at="{ item }">{{ formatDate(item.created_at) }}</template>
          <template #item.closed_at="{ item }">{{ formatDate(item.closed_at) }}</template>
          <template #item.actions="{ item }">
            <v-btn
              v-if="canEdit"
              :disabled="item.is_closed"
              :loading="closingId === item.id"
              size="small"
              text="Close"
              variant="text"
              @click="close(item)"
            />
          </template>
        </ServerSideTable>
      </div>

      <!-- ------------------------------------------------------- BRANCH -->
      <div v-else-if="tab === 'branch'" class="shifts-layout">
        <PageSidebar
          v-model="selectedBranch"
          all-title="All branches"
          empty-text="No branches yet."
          :items="branchItems"
          :loading="branchesLoading"
          show-all
        />

        <v-divider vertical />

        <div class="shifts-content">
          <ServerSideTable
            ref="branchTableRef"
            :api-u-r-l="`${API_BASE}/branch_shifts`"
            class="shifts-table"
            density="comfortable"
            flush
            :headers="branchHeaders"
            :query-params="branchQueryParams"
            root-key="branch_shifts"
            :show-search-icon="false"
          >
            <template #item.branch_id="{ item }">{{ branchName(item.branch_id) }}</template>
            <template #item.opened_at="{ item }">{{ formatDate(item.opened_at) }}</template>
            <template #item.closed_at="{ item }">{{ formatDate(item.closed_at) }}</template>
            <template #item.expected_usd="{ item }">
              <div class="font-weight-medium">{{ shiftTotal(item, 'expected') }} USD</div>
              <div v-for="line in methodLines(item, 'expected')" :key="line.label" class="method-line">
                <span class="method-label">{{ line.label }}</span>{{ line.text }}
              </div>
            </template>
            <template #item.counted_usd="{ item }">
              <div class="font-weight-medium">{{ shiftTotal(item, 'counted') }} USD</div>
              <div v-for="line in methodLines(item, 'counted')" :key="line.label" class="method-line">
                <span class="method-label">{{ line.label }}</span>{{ line.text }}
              </div>
            </template>
            <template #item.variance_usd="{ item }">
              <div class="font-weight-medium">
                <v-chip :color="varianceColor(shiftTotalCents(item, 'variance'))" size="small" variant="tonal">
                  {{ signed(shiftTotalCents(item, 'variance')) }} USD
                </v-chip>
              </div>
              <div v-for="line in methodLines(item, 'variance')" :key="line.label" class="method-line">
                <span class="method-label">{{ line.label }}</span>
                <v-chip :color="varianceColor(line.value)" size="small" variant="tonal">{{ line.text }}</v-chip>
              </div>
            </template>
            <template #item.receipt="{ item }">
              <v-btn
                v-if="item.has_receipt_image"
                color="primary"
                icon="mdi-camera"
                size="small"
                variant="text"
                @click="openReceipt(item)"
              />
              <v-icon v-else class="text-disabled" icon="mdi-camera-off" size="small" />
            </template>
            <template #item.received_at="{ item }">{{ formatDate(item.received_at) }}</template>
            <template #item.print="{ item }">
              <v-btn
                v-if="canPrintBranchShift"
                icon="mdi-printer"
                :loading="printingShiftId === item.id"
                size="small"
                title="Print this close shift on the till that closed it"
                variant="text"
                @click="printShift(item)"
              />
            </template>
          </ServerSideTable>
        </div>
      </div>

      <!-- ----------------------------------------------------- VISITORS -->
      <!-- Branch-reported nav-drawer people-counter presses, rolled up to
           one row per branch per local day (gross in / gross out / net).
           Same branch-sidebar + ServerSideTable shape as the Branch tab. -->
      <div v-else class="shifts-layout">
        <PageSidebar
          v-model="selectedBranch"
          all-title="All branches"
          empty-text="No branches yet."
          :items="branchItems"
          :loading="branchesLoading"
          show-all
        />

        <v-divider vertical />

        <div class="shifts-content">
          <ServerSideTable
            ref="visitorsTableRef"
            :api-u-r-l="`${API_BASE}/branch_visitor_events`"
            class="shifts-table"
            density="comfortable"
            flush
            :headers="visitorHeaders"
            :query-params="branchQueryParams"
            root-key="visitor_days"
            :show-search-icon="false"
          >
            <template #item.branch_id="{ item }">{{ branchName(item.branch_id) }}</template>
            <template #item.net="{ item }">
              <v-chip :color="(item.in_count - item.out_count) < 0 ? 'error' : 'default'" size="small" variant="tonal">
                {{ item.in_count - item.out_count }}
              </v-chip>
            </template>
          </ServerSideTable>
        </div>
      </div>
    </v-card>

    <v-dialog v-model="dialog" max-width="420">
      <v-card class="px-4">
        <v-card-title>Open a New Shift</v-card-title>
        <v-card-text>
          <form @submit.prevent="submit">
            <v-select
              v-model="cashboxId.value.value"
              density="compact"
              :error-messages="cashboxId.errorMessage.value"
              :item-title="c => `${c.name} (${c.code})`"
              :item-value="c => c.id"
              :items="cashboxes"
              label="Cashbox"
            />

            <v-card-actions class="px-0">
              <v-spacer />
              <v-btn text="Cancel" @click="closeDialog" />
              <v-btn color="primary" :loading="submitting" text="Open Shift" type="submit" />
            </v-card-actions>
          </form>
        </v-card-text>
      </v-card>
    </v-dialog>

    <v-dialog v-model="receiptDialog" max-width="720">
      <v-card>
        <v-card-title class="d-flex align-center ga-2">
          <v-icon icon="mdi-camera" />
          <span>Close-shift receipt</span>
          <v-spacer />
          <span v-if="receiptShift" class="text-caption text-medium-emphasis">
            {{ branchName(receiptShift.branch_id) }} · {{ receiptShift.closing_person_name || '—' }}
          </span>
        </v-card-title>
        <v-divider />
        <!-- Fixed-height stage so the dialog doesn't resize when the photo lands. -->
        <div class="image-stage d-flex align-center justify-center pa-2">
          <v-progress-circular v-if="receiptLoading" indeterminate />
          <ZoomableImage
            v-else-if="receiptUrl"
            alt="Close-shift receipt"
            :busy="receiptChanging"
            height="100%"
            :progress="uploadProgress"
            :replaceable="canReplace"
            :rotatable="canRotate"
            :src="receiptUrl"
            @replace="file => changeReceipt(id => replacePhoto('shift_receipt', id, file))"
            @rotate="delta => changeReceipt(id => rotatePhoto('shift_receipt', id, delta))"
          />
        </div>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="receiptDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { useField, useForm } from 'vee-validate'
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import PageSidebar from '@/components/PageSidebar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useBranches } from '@/composables/useBranches'
  import { useCashboxes } from '@/composables/useCashboxes'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { usePhotoActions } from '@/composables/usePhotoActions'
  import { useShiftReceiptImages } from '@/composables/useShiftReceiptImages'
  import { useShifts } from '@/composables/useShifts'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const STATUS_OPTIONS = [
    { title: 'Open', value: 'open' },
    { title: 'Closed', value: 'closed' },
  ]

  const { createShift, closeShift } = useShifts()
  const { cashboxes, fetchCashboxes } = useCashboxes()
  const { branches, fetchBranches } = useBranches()
  fetchCashboxes()

  const branchesLoading = ref(true)
  fetchBranches().finally(() => { branchesLoading.value = false })

  const tab = ref('branch')

  const centralHeaders = [
    { title: 'ID', key: 'id', align: 'start', width: 80 },
    { title: 'Cashbox', key: 'cashbox', align: 'start' },
    { title: 'Status', key: 'is_closed', align: 'start' },
    { title: 'Opened At', key: 'created_at', align: 'end' },
    { title: 'Closed At', key: 'closed_at', align: 'end' },
    { title: '', key: 'actions', align: 'end', sortable: false, width: 90 },
  ]

  const branchHeaders = [
    { title: 'Branch', key: 'branch_id', align: 'start', sortable: false },
    { title: 'Closed By', key: 'closing_person_name', align: 'start', sortable: false },
    { title: 'Opened', key: 'opened_at', align: 'start', sortable: false },
    { title: 'Closed', key: 'closed_at', align: 'start', sortable: false },
    { title: 'Expected', key: 'expected_usd', align: 'end', sortable: false },
    { title: 'Counted', key: 'counted_usd', align: 'end', sortable: false },
    { title: 'Variance', key: 'variance_usd', align: 'end', sortable: false },
    { title: 'Receipt', key: 'receipt', align: 'center', sortable: false, width: 90 },
    { title: 'Received', key: 'received_at', align: 'end', sortable: false },
    { title: '', key: 'print', align: 'end', sortable: false, width: 56 },
  ]

  const visitorHeaders = [
    { title: 'Branch', key: 'branch_id', align: 'start', sortable: false },
    { title: 'Day', key: 'day', align: 'start', sortable: false },
    { title: 'Customers In', key: 'in_count', align: 'end', sortable: false },
    { title: 'Customers Out', key: 'out_count', align: 'end', sortable: false },
    { title: 'Net', key: 'net', align: 'end', sortable: false },
  ]

  const search = ref('')
  const statusFilter = ref(null)
  const cashboxFilter = ref(null)
  const selectedBranch = ref(null)

  const branchItems = computed(() => branches.value.map(b => ({
    value: b.id,
    title: b.name,
    subtitle: b.code,
    raw: b,
  })))

  function branchName (id) {
    return branches.value.find(b => b.id === id)?.name ?? (id ? `Branch #${id}` : '—')
  }

  // ServerSideTable merges these into every request and refetches from page 1
  // whenever they change (see the API's listShifts / listBranchShifts params).
  const centralQueryParams = computed(() => ({
    status: statusFilter.value || '',
    cashbox_id: cashboxFilter.value || '',
  }))
  const branchQueryParams = computed(() => ({
    branch_id: selectedBranch.value || '',
  }))

  const tableRef = ref(null)
  const visitorsTableRef = ref(null)
  const branchTableRef = ref(null)
  const dialog = ref(false)

  // --- Close-shift receipt photo (see api/branch_shift_receipt_image.go) ---
  const { fetchShiftReceiptImageForShift, fetchShiftReceiptImageObjectUrl } = useShiftReceiptImages()
  const receiptDialog = ref(false)
  const receiptShift = ref(null)
  const receiptUrl = ref('')
  const receiptLoading = ref(false)
  const receiptImageId = ref(null)
  const notify = useNotify()

  // Reprinting a branch's close shift is a command to the till that closed
  // it (print_shift_report), and queuing commands takes Branches edit.
  const auth = useAuthStore()
  const canPrintBranchShift = computed(() => auth.canEdit('branches'))
  const { printBranchShift } = useBranchCommands()
  const printingShiftId = ref(null)

  async function printShift (item) {
    printingShiftId.value = item.id
    try {
      await printBranchShift(item)
      notify.success(`Sent to ${branchName(item.branch_id)} — it prints within a few seconds if the till is online.`)
    } catch (error) {
      notify.error(error.message)
    } finally {
      printingShiftId.value = null
    }
  }

  function revokeReceiptUrl () {
    if (receiptUrl.value) {
      URL.revokeObjectURL(receiptUrl.value)
      receiptUrl.value = ''
    }
  }

  async function openReceipt (item) {
    receiptShift.value = item
    receiptImageId.value = null
    revokeReceiptUrl()
    receiptDialog.value = true
    receiptLoading.value = true
    try {
      const image = await fetchShiftReceiptImageForShift(item.id)
      if (!image) {
        notify.warning('No receipt photo for this shift yet.')
        return
      }
      receiptImageId.value = image.id
      receiptUrl.value = await fetchShiftReceiptImageObjectUrl(image.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      receiptLoading.value = false
    }
  }

  // Saves a rotation or replacement (save gets the image id), then shows the
  // photo as the server now serves it.
  const { canRotate, canReplace, rotatePhoto, replacePhoto, uploadProgress } = usePhotoActions()
  const receiptChanging = ref(false)
  async function changeReceipt (save) {
    const imageId = receiptImageId.value
    receiptChanging.value = true
    try {
      await save(imageId)
      const url = await fetchShiftReceiptImageObjectUrl(imageId)
      if (receiptImageId.value !== imageId) {
        URL.revokeObjectURL(url)
        return
      }
      revokeReceiptUrl()
      receiptUrl.value = url
    } catch (error) {
      notify.error(error.message)
    } finally {
      receiptChanging.value = false
    }
  }

  // Live-refresh the Visitors tab when a branch reports a counter press, so
  // the per-day rollup doesn't need a manual reload. The visitors table is
  // only mounted while that tab is open, so this is a no-op otherwise and
  // switching to the tab refetches on mount anyway. Debounced so a burst of
  // presses is one reload, not one network round trip per press.
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribeAdmin = null
  let visitorReloadTimer = null
  function scheduleVisitorsReload () {
    if (visitorReloadTimer) return
    visitorReloadTimer = setTimeout(() => {
      visitorReloadTimer = null
      visitorsTableRef.value?.reload()
    }, 1000)
  }
  onMounted(() => {
    ensureConnected()
    unsubscribeAdmin = onMessage(message => {
      if (message.type === 'branch_visitor_event') {
        scheduleVisitorsReload()
      } else if (message.type === 'branch_shift_receipt_uploaded') {
        branchTableRef.value?.reload()
      }
    })
  })
  onUnmounted(() => {
    clearTimeout(visitorReloadTimer)
    unsubscribeAdmin?.()
    revokeReceiptUrl()
  })
  const submitting = ref(false)
  const closingId = ref(null)

  function money (cents) {
    return (Number(cents) / 100).toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
  }

  // The tills report the dollar drawers in cents but LBP in whole pounds.
  const SHIFT_METHODS = [
    { key: 'usd', label: 'USD', format: money },
    { key: 'lbp', label: 'LBP', format: units => Number(units).toLocaleString(undefined, { maximumFractionDigits: 0 }) },
    { key: 'visa', label: 'Visa', format: money },
    { key: 'whish', label: 'Whish', format: money },
  ]

  // A shift's amounts as one USD figure in cents: the LBP drawer is priced at
  // the rate the till closed at (kashi fills 90000 for shifts from before the
  // rate was reported).
  function shiftTotalCents (shift, kind) {
    const rate = Number(shift.lbp_rate) || 90_000
    const lbpCents = Math.round((Number(shift[`${kind}_lbp`]) || 0) / rate * 100)
    return ['usd', 'visa', 'whish'].reduce((sum, key) => sum + (Number(shift[`${kind}_${key}`]) || 0), lbpCents)
  }

  function shiftTotal (shift, kind) {
    return money(shiftTotalCents(shift, kind))
  }

  function signed (cents) {
    return cents > 0 ? `+${money(cents)}` : money(cents)
  }

  // One line per payment method for an expected / counted / variance column:
  // USD always, the others only when the shift moved money through them.
  function methodLines (shift, kind) {
    return SHIFT_METHODS
      .filter(m => m.key === 'usd' || shift[`expected_${m.key}`] || shift[`counted_${m.key}`])
      .map(m => {
        const value = Number(shift[`${kind}_${m.key}`]) || 0
        const text = m.format(value)
        return { label: m.label, value, text: kind === 'variance' && value > 0 ? `+${text}` : text }
      })
  }

  function varianceColor (cents) {
    if (Number(cents) < 0) return 'error'
    if (Number(cents) > 0) return 'warning'
    return 'default'
  }

  function formatDate (value) {
    // closed_at / opened_at come back as Go's sql.NullTime -> { Time, Valid }.
    if (value && typeof value === 'object') {
      if (!value.Valid) return '—'
      value = value.Time
    }
    if (!value) return '—'
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
  }

  const { handleSubmit, handleReset } = useForm({
    validationSchema: {
      cashboxId (value) {
        return !!value || 'Select a cashbox.'
      },
    },
  })

  const cashboxId = useField('cashboxId')

  function openCreate () {
    handleReset()
    dialog.value = true
  }

  function closeDialog () {
    dialog.value = false
    handleReset()
  }

  const submit = handleSubmit(async values => {
    submitting.value = true
    try {
      await createShift({ cashbox_id: values.cashboxId })
      closeDialog()
      tableRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  })

  async function close (item) {
    closingId.value = item.id
    try {
      await closeShift(item.id)
      tableRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      closingId.value = null
    }
  }
</script>

<style scoped>
.method-line {
  white-space: nowrap;
  font-size: 0.8125rem;
  opacity: 0.75;
}
.method-label {
  margin-inline-end: 8px;
  font-size: 0.75rem;
  opacity: 0.6;
}

.image-stage {
  height: min(70vh, 640px);
  width: 100%;
}

.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.shifts-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.shifts-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.shifts-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}
.shifts-search {
  flex: 0 1 260px;
}
</style>
