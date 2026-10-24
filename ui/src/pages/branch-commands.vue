<template>
  <div class="page-root">
    <v-card class="bc-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-console-line" />
        <span>Commands</span>
        <v-spacer />
        <v-btn
          v-if="canEdit && activeBranch"
          prepend-icon="mdi-send-outline"
          text="Send command"
          variant="tonal"
          @click="openSendDialog"
        />
        <v-btn prepend-icon="mdi-refresh" text="Reload" variant="text" @click="reloadAll" />
      </v-card-title>
      <v-divider />

      <div class="bc-layout">
        <PageSidebar
          v-model="selectedId"
          all-title="All branches"
          empty-text="No branches yet."
          :items="branchItems"
          :loading="branchesLoading"
          show-all
        >
          <template #prepend="{ item }">
            <span class="bc-dot me-2" :class="isBranchOnline(item) ? 'is-online' : 'is-offline'" />
          </template>
          <template #append="{ item }">
            <v-badge
              v-if="summaryFor(item.id).failed_recent > 0"
              color="error"
              :content="summaryFor(item.id).failed_recent"
              inline
            />
            <v-badge
              v-if="summaryFor(item.id).pending > 0"
              color="warning"
              :content="summaryFor(item.id).pending"
              inline
            />
          </template>
        </PageSidebar>

        <v-divider vertical />

        <div class="bc-content pa-5">
          <p class="text-body-2 text-medium-emphasis mb-4">
            A command waits here until the till runs it: instantly over its live connection, or on
            its next sync if it's offline. The till runs it with its own local code and reports back
            whether it worked.
          </p>

          <!-- ------------------------------------------------ overview -->
          <div class="bc-stats mb-4">
            <div class="bc-stat">
              <div class="bc-stat-label">Queued</div>
              <div class="bc-stat-value" :class="{ 'text-warning': totals.pending > 0 }">{{ totals.pending }}</div>
              <div class="bc-stat-sub">
                {{ totals.oldestPendingAt ? `oldest ${ago(totals.oldestPendingAt)}` : 'all commands have run' }}
              </div>
            </div>
            <div class="bc-stat">
              <div class="bc-stat-label">Failed (24h)</div>
              <div class="bc-stat-value" :class="{ 'text-error': totals.failed > 0 }">{{ totals.failed }}</div>
              <div class="bc-stat-sub">retry from the list below</div>
            </div>
            <div class="bc-stat">
              <div class="bc-stat-label">Registers online</div>
              <div class="bc-stat-value">{{ onlineRegisterCount }} / {{ visibleRegisters.length }}</div>
              <div class="bc-stat-sub">connected right now</div>
            </div>
            <div class="bc-stat">
              <div class="bc-stat-label">Last command</div>
              <div class="bc-stat-value bc-stat-value--small">
                {{ totals.lastIssuedAt ? ago(totals.lastIssuedAt) : '—' }}
              </div>
              <div class="bc-stat-sub">{{ totals.lastIssuedAt ? formatDate(totals.lastIssuedAt) : 'none issued yet' }}</div>
            </div>
          </div>

          <div v-if="activeBranch && visibleRegisters.length > 0" class="d-flex flex-wrap ga-2 mb-4">
            <v-chip
              v-for="r in visibleRegisters"
              :key="r.id"
              :color="registerOnline(r) ? 'success' : undefined"
              size="small"
              variant="tonal"
            >
              <span class="bc-dot me-2" :class="registerOnline(r) ? 'is-online' : 'is-offline'" />
              {{ r.name ? `${r.code} — ${r.name}` : r.code }}
              <span class="text-medium-emphasis ms-2">
                {{ registerOnline(r) ? 'online' : `seen ${r.last_seen_at?.Valid ? ago(r.last_seen_at.Time) : 'never'}` }}
              </span>
            </v-chip>
          </div>

          <!-- ------------------------------------------------- filters -->
          <div class="d-flex flex-wrap align-center ga-3 mb-2">
            <v-chip-group v-model="statusFilter" color="primary" mandatory>
              <v-chip
                v-for="s in STATUS_FILTERS"
                :key="s.value"
                size="small"
                :value="s.value"
                variant="outlined"
              >
                {{ s.title }}
              </v-chip>
            </v-chip-group>
            <v-spacer />
            <v-select
              v-model="typeFilter"
              clearable
              density="compact"
              hide-details
              :items="typeOptions"
              placeholder="Any type"
              style="max-width: 200px"
              variant="outlined"
            />
            <v-select
              v-if="activeBranch"
              v-model="registerFilter"
              clearable
              density="compact"
              hide-details
              item-title="label"
              item-value="id"
              :items="registerOptions"
              placeholder="Any register"
              style="max-width: 200px"
              variant="outlined"
            />
          </div>

          <ServerSideTable
            ref="tableRef"
            :api-u-r-l="`${API_BASE}/branch_commands`"
            density="compact"
            expandable
            :headers="headers"
            :query-params="queryParams"
            root-key="commands"
            :show-search-icon="false"
          >
            <template #item.type="{ item }">{{ typeLabel(item.type) }}</template>
            <template #item.branch_name="{ item }">{{ item.branch_name }}</template>
            <template #item.status="{ item }">
              <v-chip :color="statusColor(item.status)" size="small" variant="tonal">{{ item.status }}</v-chip>
            </template>
            <template #item.issued_by_name="{ item }">{{ item.issued_by_name?.Valid ? item.issued_by_name.String : '—' }}</template>
            <template #item.created_at="{ item }">
              <span :title="formatDate(item.created_at)">{{ ago(item.created_at) }}</span>
            </template>
            <template #item.completed_at="{ item }">
              <span v-if="item.completed_at?.Valid" :title="formatDate(item.completed_at.Time)">
                {{ tookLabel(item) }}
              </span>
              <span v-else class="text-medium-emphasis">waiting</span>
            </template>
            <template #item.actions="{ item }">
              <v-btn
                v-if="canEdit && item.status === 'pending'"
                prepend-icon="mdi-cancel"
                size="small"
                text="Cancel"
                variant="text"
                @click.stop="cancel(item)"
              />
              <v-btn
                v-if="canEdit && (item.status === 'failed' || item.status === 'cancelled')"
                prepend-icon="mdi-replay"
                size="small"
                text="Retry"
                variant="text"
                @click.stop="retry(item)"
              />
            </template>
            <template #expanded-row="{ columns, item }">
              <tr>
                <td :colspan="columns.length">
                  <div class="bc-detail py-3">
                    <div v-if="item.error?.Valid" class="mb-2">
                      <div class="bc-detail-label">Error</div>
                      <div class="text-error">{{ item.error.String }}</div>
                    </div>
                    <div class="bc-detail-label">Payload</div>
                    <pre class="bc-json">{{ prettyPayload(item.payload) }}</pre>
                    <div v-if="item.result?.Valid" class="mt-2">
                      <div class="bc-detail-label">Result</div>
                      <pre class="bc-json">{{ prettyPayload(item.result.RawMessage) }}</pre>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </ServerSideTable>
        </div>
      </div>
    </v-card>

    <!-- send a one-off command -->
    <v-dialog v-model="sendDialog" max-width="520">
      <v-card v-if="activeBranch">
        <v-card-title class="text-subtitle-1">Send a command to {{ activeBranch.name }}</v-card-title>
        <v-card-text>
          <v-select
            v-model="sendForm.type"
            class="mb-3"
            density="compact"
            hide-details
            :items="SENDABLE_TYPES"
            label="Command"
            variant="outlined"
          />

          <template v-if="sendForm.type === 'sync_now'">
            <p class="text-body-2 mb-3">
              The till uploads what it has queued and pulls the latest from kashi right away instead
              of waiting for its next sync.
            </p>
            <v-select
              v-model="sendForm.registerId"
              density="compact"
              hide-details
              item-title="label"
              item-value="id"
              :items="[{ id: 0, label: 'Every register' }, ...registerOptions]"
              label="Register"
              variant="outlined"
            />
          </template>

          <template v-else-if="sendForm.type === 'print_shift_report'">
            <p class="text-body-2 mb-3">
              Prints the end-of-shift receipt again on the printer of the till that closed it.
            </p>
            <v-autocomplete
              v-model="sendForm.shiftId"
              density="compact"
              hide-details
              item-title="label"
              item-value="id"
              :items="shiftOptions"
              label="Closed shift"
              :loading="pickerLoading"
              no-data-text="No closed shifts reported yet"
              variant="outlined"
            />
          </template>

          <template v-else-if="sendForm.type === 'replace_shift_balance'">
            <p class="text-body-2 mb-3">
              Moves value between two payment methods in the till's open shift, the same as its
              Swap balance dialog. The sales themselves don't change.
            </p>
            <v-select
              v-model="sendForm.registerId"
              class="mb-3"
              density="compact"
              hide-details
              item-title="label"
              item-value="id"
              :items="registerOptions"
              label="Register"
              variant="outlined"
            />
            <div class="d-flex ga-3 mb-3">
              <v-select
                v-model="sendForm.fromAccount"
                density="compact"
                hide-details
                :items="balanceAccountOptions"
                label="From"
                :loading="pickerLoading"
                variant="outlined"
              />
              <v-select
                v-model="sendForm.toAccount"
                density="compact"
                hide-details
                :items="balanceAccountOptions.filter(name => name !== sendForm.fromAccount)"
                label="To"
                :loading="pickerLoading"
                variant="outlined"
              />
            </div>
            <v-text-field
              v-model.number="sendForm.amountUsd"
              density="compact"
              hide-details
              label="Amount (USD)"
              min="0"
              prefix="$"
              step="0.01"
              type="number"
              variant="outlined"
            />
            <v-checkbox
              v-if="canTransferNegative"
              v-model="sendForm.allowNegative"
              density="compact"
              hide-details
              label="Transfer: take the amount out of the source even if it goes below zero"
            />
            <p v-if="sendForm.fromAccount && sendForm.toAccount && sendForm.amountUsd > 0" class="text-caption text-medium-emphasis mt-2">
              <template v-if="sendForm.allowNegative">
                {{ sendForm.fromAccount }} goes down by {{ formatMoney(amountCents) }} and
                {{ sendForm.toAccount }} goes up by {{ formatMoney(amountCents) }}.
              </template>
              <template v-else>
                Swaps {{ formatMoney(amountCents) }} of {{ sendForm.fromAccount }}'s balance into
                {{ sendForm.toAccount }}, keeping its sign. The till refuses it if that's more than
                {{ sendForm.fromAccount }} holds.
              </template>
            </p>
          </template>

          <template v-else-if="sendForm.type === 'reprint_invoice'">
            <p class="text-body-2 mb-3">
              Prints the receipt again on the printer of the till that made the sale.
            </p>
            <v-autocomplete
              v-model="sendForm.clientRef"
              v-model:search="invoiceSearch"
              density="compact"
              hide-details
              item-title="label"
              item-value="client_ref"
              :items="invoiceOptions"
              label="Invoice"
              :loading="pickerLoading"
              no-data-text="No matching invoices"
              no-filter
              variant="outlined"
            />
          </template>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Close" variant="text" @click="sendDialog = false" />
          <v-btn
            color="primary"
            :disabled="!sendReady"
            :loading="sending"
            text="Send"
            variant="flat"
            @click="submitSend"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import PageSidebar from '@/components/PageSidebar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { authFetch } from '@/composables/useApi'
  import { COMMAND_TYPE_LABELS, useBranchCommands } from '@/composables/useBranchCommands'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchPaymentMethods } from '@/composables/useBranchPaymentMethods'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'
  import { formatMoney } from '@/utils/money'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const route = useRoute()
  const router = useRouter()

  const { branches, fetchBranches, listRegisters, fetchConnectedRegisters } = useBranches()
  const { summary, refreshCommandSummary, sendCommand, cancelCommand, retryCommand } = useBranchCommands()
  const { listBranchPaymentMethods } = useBranchPaymentMethods()
  const auth = useAuthStore()
  // "Negative Balance Transfers" in Settings > Roles; admins always can.
  // The API checks it again (api/branch_command.go).
  const canTransferNegative = computed(() => auth.canEdit('shift_negative_balance'))

  const STATUS_FILTERS = [
    { title: 'All', value: '' },
    { title: 'Queued', value: 'pending' },
    { title: 'Failed', value: 'failed' },
    { title: 'Done', value: 'success' },
    { title: 'Cancelled', value: 'cancelled' },
  ]
  const SENDABLE_TYPES = [
    { title: 'Sync now', value: 'sync_now' },
    { title: 'Print a close shift', value: 'print_shift_report' },
    { title: 'Reprint an invoice', value: 'reprint_invoice' },
    { title: 'Swap a shift balance', value: 'replace_shift_balance' },
  ]
  const typeOptions = Object.entries(COMMAND_TYPE_LABELS).map(([value, title]) => ({ value, title }))

  const branchesLoading = ref(false)
  const selectedId = ref(null)
  const statusFilter = ref('')
  const typeFilter = ref(null)
  const registerFilter = ref(null)
  const tableRef = ref(null)

  const activeBranch = computed(() => branches.value.find(b => b.id === selectedId.value) ?? null)
  const branchItems = computed(() => branches.value.map(b => ({ value: b.id, title: b.name, subtitle: b.code, raw: b })))

  const headers = computed(() => [
    { title: 'Command', key: 'type', sortable: false },
    ...(activeBranch.value ? [] : [{ title: 'Branch', key: 'branch_name', sortable: false }]),
    { title: 'Register', key: 'register_code', sortable: false },
    { title: 'Status', key: 'status', sortable: false },
    { title: 'Issued by', key: 'issued_by_name', sortable: false },
    { title: 'Issued', key: 'created_at', sortable: false },
    { title: 'Ran', key: 'completed_at', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ])

  const queryParams = computed(() => ({
    branch_id: selectedId.value ?? 0,
    register_id: registerFilter.value ?? 0,
    status: statusFilter.value ?? '',
    type: typeFilter.value ?? '',
  }))

  function summaryFor (branchId) {
    return summary.value.find(s => s.branch_id === branchId) ?? { pending: 0, failed_recent: 0 }
  }

  // Sidebar "All" sums every branch; a branch shows its own row.
  const totals = computed(() => {
    const rows = selectedId.value == null ? summary.value : summary.value.filter(s => s.branch_id === selectedId.value)
    const times = key => rows.map(r => r[key]).filter(Boolean).map(t => new Date(t).getTime())
    const oldest = times('oldest_pending_at')
    const last = times('last_issued_at')
    return {
      pending: rows.reduce((n, r) => n + r.pending, 0),
      failed: rows.reduce((n, r) => n + r.failed_recent, 0),
      oldestPendingAt: oldest.length > 0 ? new Date(Math.min(...oldest)) : null,
      lastIssuedAt: last.length > 0 ? new Date(Math.max(...last)) : null,
    }
  })

  // ---- registers & who's online ------------------------------------------
  const registersByBranch = ref({})
  const connectedRegisterIds = ref(new Set())

  async function loadRegisters () {
    const lists = await Promise.all(branches.value.map(b => listRegisters(b.id).catch(() => [])))
    const byBranch = {}
    for (const [i, b] of branches.value.entries()) byBranch[b.id] = lists[i]
    registersByBranch.value = byBranch
  }
  async function loadConnected () {
    try {
      connectedRegisterIds.value = new Set(await fetchConnectedRegisters())
    } catch { /* informational */ }
  }
  const visibleRegisters = computed(() => selectedId.value == null
    ? Object.values(registersByBranch.value).flat()
    : registersByBranch.value[selectedId.value] ?? [])
  const registerOptions = computed(() => (registersByBranch.value[selectedId.value] ?? []).map(r => ({
    id: r.id,
    label: r.name ? `${r.code} — ${r.name}` : r.code,
  })))
  const registerOnline = r => connectedRegisterIds.value.has(r.id)
  const onlineRegisterCount = computed(() => visibleRegisters.value.filter(r => registerOnline(r)).length)
  function isBranchOnline (branch) {
    return (registersByBranch.value[branch.id] ?? []).some(r => registerOnline(r))
  }

  // ---- list actions ------------------------------------------------------
  async function cancel (item) {
    try {
      await cancelCommand(item.id)
      notify.success('Cancelled — the till won\'t run it.')
      reloadTable()
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function retry (item) {
    try {
      await retryCommand(item.id)
      notify.success(`Queued again for ${item.register_code}.`)
      reloadTable()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- send dialog -------------------------------------------------------
  const sendDialog = ref(false)
  const sending = ref(false)
  const pickerLoading = ref(false)
  const sendForm = reactive({ type: 'sync_now', registerId: 0, shiftId: null, clientRef: null, fromAccount: null, toAccount: null, amountUsd: null, allowNegative: false })
  const shifts = ref([])
  const invoices = ref([])
  const invoiceSearch = ref('')
  const paymentMethods = ref([])
  // On Account / On Coupon aren't money in a drawer; the till refuses them.
  const balanceAccountOptions = computed(() => paymentMethods.value
    .filter(m => m.is_enabled && !/^(on account|on coupon)$/i.test(m.name.trim()))
    .map(m => m.name))
  const amountCents = computed(() => Math.round(Number(sendForm.amountUsd || 0) * 100))

  const shiftOptions = computed(() => shifts.value.map(s => ({
    id: s.id,
    label: `${s.closed_at?.Valid ? formatDate(s.closed_at.Time) : formatDate(s.occurred_at)} · ${s.closing_person_name || 'Unknown'} · counted ${formatMoney(s.counted_usd)}`,
  })))
  const invoiceOptions = computed(() => invoices.value.map(i => ({
    client_ref: i.client_ref,
    label: `${i.branch_invoice_code} · ${formatDate(i.occurred_at)}`,
  })))
  const sendReady = computed(() => {
    if (sendForm.type === 'print_shift_report') return sendForm.shiftId != null
    if (sendForm.type === 'reprint_invoice') return !!sendForm.clientRef
    if (sendForm.type === 'replace_shift_balance') {
      return !!sendForm.registerId && !!sendForm.fromAccount && !!sendForm.toAccount
        && sendForm.fromAccount !== sendForm.toAccount && amountCents.value > 0
    }
    return true
  })

  async function getJSON (url) {
    const res = await authFetch(url)
    if (!res.ok) throw new Error(`Request failed with status ${res.status}`)
    return res.json()
  }

  function openSendDialog () {
    Object.assign(sendForm, { type: 'sync_now', registerId: 0, shiftId: null, clientRef: null, fromAccount: null, toAccount: null, amountUsd: null, allowNegative: false })
    sendDialog.value = true
  }

  watch(() => sendForm.type, async type => {
    if (!sendDialog.value || !activeBranch.value) return
    pickerLoading.value = true
    try {
      switch (type) {
        case 'print_shift_report': {
          const data = await getJSON(`${API_BASE}/branch_shifts?branch_id=${activeBranch.value.id}&page_size=50&page_id=0`)
          shifts.value = data?.branch_shifts ?? []
          break
        }
        case 'reprint_invoice': {
          await loadInvoices('')
          break
        }
        case 'replace_shift_balance': {
          // One register: the swap changes that till's shift only.
          sendForm.registerId = registerOptions.value.length === 1 ? registerOptions.value[0].id : 0
          paymentMethods.value = await listBranchPaymentMethods(activeBranch.value.id)
          break
        }
      }
    } catch (error) {
      notify.error(error.message)
    } finally {
      pickerLoading.value = false
    }
  })

  async function loadInvoices (search) {
    const q = new URLSearchParams({ branch_id: activeBranch.value.id, page_size: 30, page_id: 0, search: search ?? '' })
    const data = await getJSON(`${API_BASE}/branch_invoices?${q}`)
    invoices.value = data?.branch_invoices ?? []
  }
  let invoiceSearchTimer = null
  watch(invoiceSearch, search => {
    if (sendForm.type !== 'reprint_invoice') return
    clearTimeout(invoiceSearchTimer)
    invoiceSearchTimer = setTimeout(() => loadInvoices(search).catch(() => {}), 300)
  })

  async function submitSend () {
    sending.value = true
    try {
      const branchId = activeBranch.value.id
      let sent
      switch (sendForm.type) {
        case 'print_shift_report': {
          sent = await sendCommand(branchId, 'print_shift_report', { shift_id: sendForm.shiftId })
          break
        }
        case 'reprint_invoice': {
          sent = await sendCommand(branchId, 'reprint_invoice', { client_ref: sendForm.clientRef })
          break
        }
        case 'replace_shift_balance': {
          sent = await sendCommand(branchId, 'replace_shift_balance', {
            from_account: sendForm.fromAccount,
            to_account: sendForm.toAccount,
            amount_cents: amountCents.value,
            allow_negative: canTransferNegative.value && sendForm.allowNegative,
          }, sendForm.registerId)
          break
        }
        default: {
          sent = await sendCommand(branchId, sendForm.type, {}, sendForm.registerId)
        }
      }
      notify.success(`Sent to ${sent.length} register${sent.length === 1 ? '' : 's'}.`)
      sendDialog.value = false
      reloadTable()
    } catch (error) {
      notify.error(error.message)
    } finally {
      sending.value = false
    }
  }

  // ---- formatting --------------------------------------------------------
  function typeLabel (type) {
    return COMMAND_TYPE_LABELS[type] ?? type
  }
  function statusColor (status) {
    return { success: 'success', failed: 'error', pending: 'warning' }[status] ?? 'default'
  }
  function formatDate (value) {
    return value ? new Date(value).toLocaleString() : '—'
  }
  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000))
    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`
    if (seconds < 86_400) return `${Math.floor(seconds / 3600)} h ago`
    return `${Math.floor(seconds / 86_400)} d ago`
  }
  function tookLabel (item) {
    const seconds = Math.round((new Date(item.completed_at.Time) - new Date(item.created_at)) / 1000)
    if (seconds < 60) return `after ${seconds}s`
    if (seconds < 3600) return `after ${Math.round(seconds / 60)} min`
    return `after ${Math.round(seconds / 3600)} h`
  }
  function prettyPayload (raw) {
    try {
      return JSON.stringify(typeof raw === 'string' ? JSON.parse(raw) : raw, null, 2)
    } catch {
      return String(raw)
    }
  }

  // ---- loading & live refresh ---------------------------------------------
  function reloadTable () {
    tableRef.value?.reload?.()
    refreshCommandSummary().catch(() => {})
  }
  async function reloadAll () {
    tableRef.value?.reloadFromFirstPage?.()
    await Promise.all([refreshCommandSummary().catch(() => {}), loadConnected(), loadRegisters()])
  }

  watch(selectedId, id => {
    registerFilter.value = null
    router.replace({ query: { ...route.query, branch: id == null ? undefined : String(id) } })
  })

  // A queued command usually runs within seconds; refresh while any wait.
  let poll = null
  onMounted(async () => {
    branchesLoading.value = true
    try {
      await fetchBranches(true)
      const wanted = Number(route.query.branch)
      selectedId.value = branches.value.find(b => b.id === wanted)?.id ?? null
      if (route.query.status) statusFilter.value = String(route.query.status)
      await Promise.all([refreshCommandSummary(), loadConnected(), loadRegisters()])
    } catch (error) {
      notify.error(error.message)
    } finally {
      branchesLoading.value = false
    }
    poll = setInterval(() => {
      loadConnected()
      if (totals.value.pending > 0) reloadTable()
      else refreshCommandSummary().catch(() => {})
    }, 10_000)
  })
  onUnmounted(() => {
    clearInterval(poll)
    clearTimeout(invoiceSearchTimer)
  })
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bc-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bc-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.bc-content {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
}
.bc-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
  flex-shrink: 0;
}
.bc-dot.is-online {
  background: rgb(var(--v-theme-success));
}
.bc-dot.is-offline {
  background: rgba(128, 128, 128, 0.5);
}
.bc-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}
.bc-stat {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  padding: 12px 14px;
}
.bc-stat-label,
.bc-detail-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.bc-stat-value {
  font-size: 1.6rem;
  font-weight: 600;
  line-height: 1.3;
}
.bc-stat-value--small {
  font-size: 1.1rem;
  padding: 6px 0;
}
.bc-stat-sub {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.bc-json {
  font-size: 0.75rem;
  white-space: pre-wrap;
  word-break: break-word;
  background: rgba(var(--v-theme-on-surface), 0.04);
  border-radius: 6px;
  padding: 8px 10px;
  max-height: 260px;
  overflow: auto;
}
</style>
