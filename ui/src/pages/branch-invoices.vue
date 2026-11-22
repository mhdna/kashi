<template>
  <v-dialog v-model="itemsDialog" max-width="760">
    <v-card class="invoice-details-card">
      <v-card-title class="d-flex align-center ga-3 px-6 py-4">
        <v-avatar color="primary" size="38" variant="tonal">
          <v-icon icon="mdi-invoice" size="22" />
        </v-avatar>
        <div>
          <div class="text-subtitle-1 font-weight-bold">Invoice details</div>
          <div class="text-caption text-medium-emphasis">{{ itemsTarget?.branch_invoice_code }}</div>
        </div>
      </v-card-title>
      <v-divider />
      <v-card-text class="invoice-details-content pa-6">
        <div v-if="itemsLoading" class="d-flex justify-center pa-4">
          <PreparingState kind="receipt" min-height="160px" title="Loading the items" />
        </div>
        <template v-else-if="itemsDetails">
          <v-sheet class="pa-0" rounded="lg">
            <v-row density="compact">
              <v-col cols="6" md="4">
                <span class="text-caption">Branch</span>
                <div>{{ branchName(itemsDetails.invoice?.branch_id) }}</div>
              </v-col>
              <v-col cols="6" md="4">
                <span class="text-caption">Kind</span>
                <div class="text-capitalize">{{ itemsDetails.invoice?.kind }}</div>
              </v-col>
              <v-col cols="6" md="4">
                <span class="text-caption">Salesperson</span>
                <div>{{ itemsDetails.invoice?.salesperson_name || '--' }}</div>
              </v-col>
              <v-col cols="6" md="4">
                <span class="text-caption">Loyalty Points</span>
                <div>{{ itemsDetails.invoice?.loyalty_points_delta }}</div>
              </v-col>
              <v-col cols="6" md="4">
                <span class="text-caption">Grand Total</span>
                <div class="font-weight-bold">{{ formatMoney(itemsDetails.invoice?.grand_total) }}</div>
              </v-col>
              <v-col v-if="nullableString(itemsDetails.invoice?.related_client_ref)" cols="6" md="4">
                <span class="text-caption">Original Sale Ref</span>
                <div class="text-truncate">{{ nullableString(itemsDetails.invoice.related_client_ref) }}</div>
              </v-col>
            </v-row>
          </v-sheet>

          <div class="text-subtitle-2 mt-6 mb-2">Items</div>
          <v-table class="details-table" density="comfortable">
            <thead>
              <tr>
                <th>Product</th>
                <th class="text-end">Qty</th>
                <th class="text-end">Unit Price</th>
                <th class="text-end">Discount</th>
                <th class="text-end">Line Total</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in itemsDetails.items" :key="item.id">
                <td>
                  <div>
                    {{ productName(item) }}
                    <!-- A line the till priced with a deal (000202). -->
                    <v-chip
                      v-if="item.deal && item.deal_discount"
                      class="ms-1"
                      :color="isFreeLine(item) ? 'success' : 'deep-purple'"
                      label
                      prepend-icon="mdi-gift-outline"
                      size="x-small"
                      :text="isFreeLine(item) ? `FREE · ${item.deal}` : item.deal"
                      variant="flat"
                    />
                  </div>
                  <div class="text-caption text-medium-emphasis">{{ productDetail(item) }}</div>
                </td>
                <td class="text-end">{{ item.quantity }}</td>
                <td class="text-end">{{ formatMoney(item.unit_price) }}</td>
                <td class="text-end">
                  <template v-if="item.deal_discount">-{{ formatMoney(Math.abs(item.deal_discount)) }}</template>
                  <template v-else>{{ item.discount }}%</template>
                </td>
                <td class="text-end">{{ formatMoney(item.line_total) }}</td>
              </tr>
            </tbody>
          </v-table>

          <div class="text-subtitle-2 mt-6 mb-2">Settlement</div>
          <v-table class="details-table" density="comfortable">
            <thead>
              <tr>
                <th>Account</th>
                <th class="text-end">Amount</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="payment in itemsDetails.payments" :key="payment.id">
                <td>{{ payment.account_name }}</td>
                <td class="text-end">{{ formatMoney(payment.amount) }}</td>
              </tr>
              <tr v-if="!itemsDetails.payments?.length">
                <td class="text-medium-emphasis" colspan="2">No settlement reported (older kashi-pos build).</td>
              </tr>
            </tbody>
          </v-table>
        </template>
      </v-card-text>
      <v-divider />
      <v-card-actions class="px-6 py-4">
        <InvoicePrintButton
          :branch-name="branchName(itemsTarget?.branch_id)"
          :invoice="itemsTarget"
          text
        />
        <v-btn
          v-if="auth.canEdit('branches') && canEditRefund(itemsTarget) && itemsDetails"
          data-test="edit-refund"
          prepend-icon="mdi-pencil"
          text="Edit refund"
          variant="text"
          @click="refundEditOpen = true"
        />
        <v-btn
          v-if="auth.canEdit('branches') && itemsTarget?.kind === 'sales' && !isCancelled(itemsTarget) && itemsDetails"
          data-test="change-settlement"
          prepend-icon="mdi-swap-horizontal"
          text="Change settlement"
          variant="text"
          @click="settlementOpen = true"
        />
        <v-btn
          v-if="auth.canEdit('branches') && canOpenForExchange(itemsDetails?.invoice) && itemsDetails"
          data-test="open-exchange"
          prepend-icon="mdi-calendar-clock-outline"
          text="Open for exchange"
          variant="text"
          @click="exchangeOpeningOpen = true"
        />
        <v-spacer />
        <v-btn
          v-if="itemsTarget?.kind === 'sales' && !isCancelled(itemsTarget)"
          color="error"
          text="Cancel invoice"
          variant="text"
          @click="cancelViewedInvoice"
        />
        <v-btn
          v-if="itemsTarget?.kind === 'sales' && !isCancelled(itemsTarget)"
          color="primary"
          text="Exchange"
          variant="text"
          @click="exchangeViewedInvoice"
        />
        <v-btn
          v-if="itemsTarget?.kind === 'sales' && !isCancelled(itemsTarget)"
          color="primary"
          text="Return"
          variant="tonal"
          @click="returnViewedInvoice"
        />
        <v-btn text="Close" @click="itemsDialog = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <div class="invoice-filters d-flex flex-wrap align-center ga-3 px-4 py-3">
    <v-btn
      data-test="open-filters"
      prepend-icon="mdi-filter-variant"
      :text="filterCount ? `Filters (${filterCount})` : 'Filters'"
      variant="outlined"
      @click="filterDialog = true"
    />
    <span
      v-if="filterSummary.length > 0"
      class="filter-summary text-body-2 text-medium-emphasis"
      data-test="filter-summary"
    >{{ filterSummary.join(', ') }}</span>
    <v-btn
      v-if="filterCount"
      data-test="clear-filters"
      prepend-icon="mdi-close"
      text="Clear"
      variant="text"
      @click="clearFilters"
    />
    <v-spacer />
    <v-btn
      color="success"
      :loading="exporting"
      prepend-icon="mdi-microsoft-excel"
      text="Export to Excel"
      variant="tonal"
      @click="exportToExcel"
    />
  </div>

  <InvoiceFilterDialog
    v-model="filterDialog"
    :branches="branches"
    :filters="filters"
    :registers="registerOptions"
    :suggest-fn="suggestInvoiceFilter"
    @apply="applyFilters"
  />

  <ServerSideTable
    ref="tableRef"
    :api-u-r-l="apiURL"
    density="comfortable"
    :external-search="search"
    flush
    :headers="headers"
    hover
    :query-params="queryParams"
    root-key="branch_invoices"
    :search-debounce="searchDebounce"
    :show-search-icon="false"
    @row-click="openItems"
  >
    <template #item.branch_id="{ item }">
      {{ branchName(item.branch_id) }}
    </template>
    <template #item.kind="{ item }">
      <v-chip :color="kindColor(item.kind)" size="small">{{ item.kind }}</v-chip>
      <v-chip
        v-if="isCancelled(item)"
        class="ml-1"
        color="error"
        size="small"
        variant="tonal"
      >Cancelled</v-chip>
    </template>
    <template #item.grand_total="{ item }">
      {{ formatMoney(item.grand_total) }}
    </template>
    <template #item.occurred_at="{ item }">
      {{ new Date(item.occurred_at).toLocaleString() }}
    </template>
    <template #item.received_at="{ item }">
      {{ new Date(item.received_at).toLocaleString() }}
    </template>
    <template #item.print="{ item }">
      <InvoicePrintButton :branch-name="branchName(item.branch_id)" :invoice="item" />
    </template>
  </ServerSideTable>

  <SettlementChangeDialog
    v-model="settlementOpen"
    :branch-name="branchName(itemsTarget?.branch_id)"
    :invoice="itemsDetails?.invoice ?? null"
    :payments="itemsDetails?.payments ?? []"
    @changed="refreshViewedInvoice"
  />

  <ExchangeOpeningDialog
    v-model="exchangeOpeningOpen"
    :branch-name="branchName(itemsTarget?.branch_id)"
    :invoice="itemsDetails?.invoice ?? null"
  />

  <ReturnEditDialog
    v-model="refundEditOpen"
    :branch-name="branchName(itemsTarget?.branch_id)"
    :invoice="itemsDetails?.invoice ?? null"
    :items="itemsDetails?.items ?? []"
  />

  <v-dialog v-model="remoteReturnDialog" :max-width="remoteReturnPhase === 'payments' ? 720 : 560">
    <v-card>
      <v-card-title>
        {{ remoteTitles[remoteMode] }} -- {{ remoteReturnTarget?.branch_invoice_code }}
      </v-card-title>
      <v-card-text>
        <template v-if="remoteReturnPhase === 'items' && remoteMode === 'cancel'">
          <v-alert class="mb-4" type="warning" variant="tonal">
            For a sale rung up by mistake. {{ branchName(remoteReturnTarget?.branch_id) }} returns the
            whole sale ({{ formatMoney(remoteReturnTarget?.grand_total) }}) to its payment account,
            and both the sale and the return are marked Cancelled: they stay listed here but no
            longer count in sales, targets or salesperson stats. It runs the next time the branch
            is online.
          </v-alert>
          <v-textarea v-model="remoteReturnReason" label="Reason (optional)" rows="2" />
          <RemoteReturnReceiptCheckbox v-model="remoteReturnPrint" />
        </template>
        <template v-else-if="remoteReturnPhase === 'items'">
          <v-alert class="mb-4" type="info" variant="tonal">
            This queues a command for {{ branchName(remoteReturnTarget?.branch_id) }} to process
            the {{ remoteMode }} itself. It isn't executed instantly -- the branch picks it up next
            time it's online.
            <template v-if="remoteMode === 'exchange'">
              The branch prices the new items from its own lists, and the difference goes on the
              sale's payment account.
            </template>
          </v-alert>

          <div v-if="remoteReturnItemsLoading" class="d-flex justify-center pa-4">
            <PreparingState kind="receipt" min-height="160px" title="Loading the invoice's items" />
          </div>
          <template v-else>
            <div class="text-body-2 mb-2">{{ remoteMode === 'exchange' ? 'Items to take back' : 'Items to return' }}</div>
            <v-table density="compact">
              <thead>
                <tr>
                  <th />
                  <th>Product</th>
                  <th class="text-end">Qty</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in remoteReturnRows" :key="row.item.id">
                  <td>
                    <v-checkbox-btn v-model="row.selected" density="compact" />
                  </td>
                  <td>
                    <div>{{ productName(row.item) }}</div>
                    <div class="text-caption text-medium-emphasis">{{ productDetail(row.item) }}</div>
                  </td>
                  <td class="text-end">
                    <v-text-field
                      v-model.number="row.quantity"
                      density="compact"
                      :disabled="!row.selected"
                      hide-details
                      :max="row.maxQuantity"
                      min="1"
                      style="max-width: 90px"
                      type="number"
                    />
                  </td>
                </tr>
              </tbody>
            </v-table>

            <template v-if="remoteMode === 'exchange'">
              <div class="text-body-2 mt-6 mb-2">Give in exchange</div>
              <div class="d-flex align-center ga-2">
                <v-autocomplete
                  v-model="replacementPick"
                  density="compact"
                  hide-details
                  hide-no-data
                  :item-title="v => v.label"
                  :item-value="v => v.id"
                  :items="skuOptions"
                  :loading="skuLoading"
                  no-filter
                  placeholder="Search SKU / barcode"
                  return-object
                  @update:search="onSkuSearch"
                />
                <v-text-field
                  v-model.number="replacementQuantity"
                  density="compact"
                  hide-details
                  label="Qty"
                  min="1"
                  style="max-width: 90px"
                  type="number"
                />
                <v-btn :disabled="!replacementPick" text="Add" @click="addReplacement" />
              </div>
              <v-table v-if="replacements.length > 0" class="mt-2" density="compact">
                <tbody>
                  <tr v-for="(row, index) in replacements" :key="row.variant.id">
                    <td>{{ row.variant.label }}</td>
                    <td class="text-end">{{ row.quantity }}</td>
                    <td class="text-end" style="width: 48px">
                      <v-btn
                        aria-label="Remove"
                        density="comfortable"
                        icon="mdi-close"
                        size="small"
                        variant="text"
                        @click="replacements.splice(index, 1)"
                      />
                    </td>
                  </tr>
                </tbody>
              </v-table>
            </template>
          </template>

          <v-textarea v-model="remoteReturnReason" class="mt-4" label="Reason (optional)" rows="2" />
          <RemoteReturnReceiptCheckbox v-model="remoteReturnPrint" />
        </template>

        <template v-else-if="remoteReturnPhase === 'payments'">
          <div v-if="remotePaymentsLoading" class="d-flex justify-center pa-6">
            <v-progress-circular indeterminate />
          </div>
          <template v-else>
            <p class="text-body-2 mb-2">
              {{ remoteMode === 'exchange' ? 'How the difference is settled' : 'How the refund is given back' }}, by payment
              method. This is kashi's estimate from what the items sold for{{ remoteMode === 'exchange' ? ' and the branch prices' : '' }};
              the till works out the exact amount and refuses payments that don't match it, saying its own figure.
            </p>
            <p v-if="unpricedNames.length > 0" class="text-body-2 text-error mb-2">
              No price at this branch for {{ unpricedNames.join(', ') }}; the estimate leaves it out.
            </p>
            <PaymentEntryPanel
              v-model="remoteEntries"
              :currencies="remoteCurrencies"
              :due-cents="remoteDueCents"
              :methods="remoteMethods"
            />
          </template>
        </template>

        <PreparingState
          v-else-if="remoteReturnPhase === 'polling'"
          kind="receipt"
          min-height="220px"
          :title="`Waiting for ${branchName(remoteReturnTarget?.branch_id)} to process the ${remoteMode === 'cancel' ? 'cancellation' : remoteMode}`"
        />

        <template v-else-if="remoteReturnPhase === 'result'">
          <v-alert
            :text="remoteReturnResult.status === 'success'
              ? remoteDoneText[remoteMode]
              : remoteReturnResult.status === 'failed'
                ? remoteReturnResult.error || 'The branch reported a failure with no error message.'
                : 'Still pending -- the branch may be offline right now. It will process the next time it\'s online, and this list will update live once it does.'"
            :type="remoteReturnResult.status === 'success' ? 'success' : remoteReturnResult.status === 'failed' ? 'error' : 'info'"
            variant="tonal"
          />
        </template>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn
          v-if="remoteReturnPhase === 'items'"
          text="Cancel"
          @click="closeRemoteReturn"
        />
        <v-btn
          v-if="remoteReturnPhase === 'items' && remoteMode === 'cancel'"
          color="error"
          :loading="remoteReturnLoading"
          text="Cancel invoice"
          @click="submitRemoteCancel"
        />
        <v-btn
          v-else-if="remoteReturnPhase === 'items'"
          color="primary"
          data-test="remote-next"
          :disabled="!hasSelectedReturnItems || (remoteMode === 'exchange' && replacements.length === 0)"
          text="Next: payments"
          @click="openRemotePayments"
        />
        <template v-else-if="remoteReturnPhase === 'payments'">
          <v-btn text="Back" @click="remoteReturnPhase = 'items'" />
          <v-btn
            data-test="remote-till-settles"
            :disabled="remotePaymentsLoading"
            :loading="remoteReturnLoading && !remoteSendingPayments"
            text="Let the till settle it"
            @click="submitRemoteReturn(false)"
          />
          <v-btn
            color="primary"
            data-test="remote-queue"
            :disabled="!remotePaymentStatus.done"
            :loading="remoteReturnLoading && remoteSendingPayments"
            :text="remoteMode === 'exchange' ? 'Queue exchange' : 'Queue return'"
            @click="submitRemoteReturn(true)"
          />
        </template>
        <template v-else-if="remoteReturnPhase === 'result' && remoteReturnResult?.status === 'failed' && remoteMode !== 'cancel'">
          <v-btn text="Close" @click="closeRemoteReturn" />
          <v-btn color="primary" text="Edit and send again" @click="remoteReturnPhase = 'payments'" />
        </template>
        <v-btn v-else text="Close" @click="closeRemoteReturn" />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import ExchangeOpeningDialog from '@/components/Invoice/ExchangeOpeningDialog.vue'
  import InvoiceFilterDialog from '@/components/Invoice/InvoiceFilterDialog.vue'
  import InvoicePrintButton from '@/components/Invoice/InvoicePrintButton.vue'
  import PaymentEntryPanel from '@/components/Invoice/PaymentEntryPanel.vue'
  import RemoteReturnReceiptCheckbox from '@/components/Invoice/RemoteReturnReceiptCheckbox.vue'
  import ReturnEditDialog from '@/components/Invoice/ReturnEditDialog.vue'
  import SettlementChangeDialog from '@/components/Invoice/SettlementChangeDialog.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchInvoices } from '@/composables/useBranchInvoices'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useInvoiceDetails } from '@/composables/useInvoiceDetails'
  import { useNotify } from '@/composables/useNotify'
  import { usePaymentEntry } from '@/composables/usePaymentEntry'
  import { useVariants } from '@/composables/useVariants'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'
  import { canOpenForExchange } from '@/utils/exchangeOpening'
  import { countInvoiceFilters, emptyFilters, invoiceFilterParams, invoiceFilterSummary } from '@/utils/invoiceFilters'
  import { estimateCents, paymentStatus, toPayload, unpricedReplacements } from '@/utils/paymentEntry'
  import { canEditRefund } from '@/utils/returnEdit'

  const props = defineProps({
    branches: { type: Array, default: () => [] },
    search: { type: String, default: '' },
    searchDebounce: { type: Number, default: 100 },
  })

  const { exportBranchInvoices, suggestInvoiceFilter, listBranchInvoiceItems, requestRemoteCancel, requestRemoteReturn } = useBranchInvoices()
  const { listBranchCommands } = useBranchSettings()
  const auth = useAuthStore()
  const { getBranchInvoiceDetails } = useInvoiceDetails()
  const notify = useNotify()

  // Live refresh -- api/branch_sync.go broadcasts "branch_invoice_created"
  // for every kind (sales/return/exchange), so a remotely-triggered return
  // (see requestRemoteReturn) shows up here as soon as the branch actually
  // processes and syncs it back, same as any branch-initiated invoice.
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  onMounted(() => {
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type !== 'branch_invoice_created') return
      if (filters.value.branchId && message.branch_id !== filters.value.branchId) return
      tableRef.value?.reload()
    })
  })
  onUnmounted(() => unsubscribe?.())

  function branchName (id) {
    return props.branches.find(b => b.id === id)?.name ?? `Branch #${id}`
  }

  function kindColor (kind) {
    if (kind === 'return') return 'warning'
    if (kind === 'exchange') return 'info'
    return 'primary'
  }

  // Lines carry the till's own product id, meaningless here; the name comes
  // from the variant kashi matched it to (blank when it never matched).
  function productName (item) {
    return item?.product_name || `Unknown product (#${item?.branch_product_id})`
  }

  // Free: the deal took each unit's whole price. deal_discount is signed like
  // the quantity on a returned line.
  function isFreeLine (item) {
    return Math.abs(item.deal_discount) === item.unit_price * Math.abs(item.quantity)
  }


  function productDetail (item) {
    return [item?.product_code, [item?.color_name, item?.size_name].filter(Boolean).join(' / '), item?.barcode]
      .filter(Boolean)
      .join(' · ')
  }

  function formatMoney (cents) {
    return (Number(cents) / 100).toFixed(2)
  }

  // cancelled_at is a Go sql.NullTime: { Time, Valid } on the wire.
  function isCancelled (invoice) {
    const value = invoice?.cancelled_at
    if (!value) return false
    if (typeof value === 'object') return Boolean(value.Valid ?? value.valid)
    return true
  }

  function nullableString (value) {
    if (typeof value === 'string') return value
    if (!value || typeof value !== 'object') return ''
    const valid = value.Valid ?? value.valid
    if (valid === false) return ''
    return value.String ?? value.string ?? ''
  }

  const headers = ref([
    { title: 'Branch', key: 'branch_id', align: 'start', sortable: false },
    { title: 'Kind', key: 'kind', align: 'start', sortable: false },
    { title: 'Code', key: 'branch_invoice_code', align: 'start', sortable: false },
    { title: 'Grand Total', key: 'grand_total', align: 'end', sortable: false },
    { title: 'Occurred At', key: 'occurred_at', align: 'start', sortable: false },
    { title: 'Received At', key: 'received_at', align: 'start', sortable: false },
    { title: '', key: 'print', align: 'end', sortable: false, width: 56 },
  ])

  const apiURL = `${API_BASE}/branch_invoices`
  const tableRef = ref(null)

  // Every filter lives in the Filters dialog; the list follows what was applied.
  const filterDialog = ref(false)
  const filters = ref(emptyFilters())
  function applyFilters (next) {
    filters.value = { ...emptyFilters(), ...next }
  }
  function clearFilters () {
    filters.value = emptyFilters()
  }
  const filterCount = computed(() => countInvoiceFilters(filters.value))

  // Registers to pick from (the dialog narrows them to the chosen branch).
  const { listAllRegisters } = useBranches()
  const allRegisters = ref([])
  onMounted(async () => {
    try {
      allRegisters.value = await listAllRegisters()
    } catch (error) {
      notify.error(error.message)
    }
  })
  const registerOptions = computed(() =>
    allRegisters.value.map(r => ({
      id: r.id,
      branch_id: r.branch_id,
      label: `${branchName(r.branch_id)} - ${r.name || r.code}`,
    })))
  const filterSummary = computed(() => invoiceFilterSummary(filters.value, {
    branch: branchName,
    register: id => registerOptions.value.find(r => r.id === id)?.label,
  }))

  const filterParams = computed(() => invoiceFilterParams(filters.value))
  const queryParams = filterParams

  const exporting = ref(false)
  async function exportToExcel () {
    exporting.value = true
    try {
      const params = { ...filterParams.value }
      if (props.search.trim()) params.search = props.search.trim()
      const blob = await exportBranchInvoices(params)
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `invoices-${new Date().toISOString().slice(0, 10)}.xlsx`
      link.click()
      URL.revokeObjectURL(link.href)
    } catch (error) {
      notify.error(error.message)
    } finally {
      exporting.value = false
    }
  }

  const itemsDialog = ref(false)
  const itemsTarget = ref(null)
  const itemsDetails = ref(null)
  const refundEditOpen = ref(false)
  const itemsLoading = ref(false)

  // ?open=<id> (a sale clicked in the app bar's Branch sync list): its
  // details, fetched by id since it may not be on the page that's loaded.
  const route = useRoute()
  watch(() => route.query.open, async open => {
    const id = Number(open)
    if (!id) return
    itemsTarget.value = null
    itemsDialog.value = true
    itemsLoading.value = true
    itemsDetails.value = null
    try {
      itemsDetails.value = await getBranchInvoiceDetails(id)
      itemsTarget.value = itemsDetails.value?.invoice ?? null
    } catch (error) {
      notify.error(error.message)
      itemsDialog.value = false
    } finally {
      itemsLoading.value = false
    }
  }, { immediate: true })

  async function openItems (invoice) {
    itemsTarget.value = invoice
    itemsDialog.value = true
    itemsLoading.value = true
    itemsDetails.value = null
    try {
      itemsDetails.value = await getBranchInvoiceDetails(invoice.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      itemsLoading.value = false
    }
  }

  // Change settlement: the till re-splits the sale, then syncs it back here.
  const settlementOpen = ref(false)
  // Open for exchange: the till lets an old sale be exchanged until a chosen day.
  const exchangeOpeningOpen = ref(false)
  async function refreshViewedInvoice () {
    tableRef.value?.reload()
    if (!itemsTarget.value) return
    try {
      itemsDetails.value = await getBranchInvoiceDetails(itemsTarget.value.id)
    } catch {
      // the list still shows it; the details reload on next open
    }
  }

  function returnViewedInvoice () {
    const invoice = itemsTarget.value
    itemsDialog.value = false
    openRemoteReturn(invoice)
  }

  function exchangeViewedInvoice () {
    const invoice = itemsTarget.value
    itemsDialog.value = false
    openRemoteReturn(invoice, 'exchange')
  }

  function cancelViewedInvoice () {
    const invoice = itemsTarget.value
    itemsDialog.value = false
    remoteMode.value = 'cancel'
    remoteReturnTarget.value = invoice
    remoteReturnReason.value = ''
    remoteReturnPrint.value = true
    remoteReturnPhase.value = 'items'
    remoteReturnDialog.value = true
  }

  async function submitRemoteCancel () {
    remoteReturnLoading.value = true
    try {
      const command = await requestRemoteCancel(remoteReturnTarget.value.branch_id, remoteReturnTarget.value, remoteReturnReason.value, remoteReturnPrint.value)
      remoteReturnPhase.value = 'polling'
      pollRemoteReturnCommand(remoteReturnTarget.value.branch_id, command?.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      remoteReturnLoading.value = false
    }
  }

  const remoteReturnDialog = ref(false)
  // 'return' (pick lines to take back), 'exchange' (the same, plus new items
  // to hand out) or 'cancel' (the whole sale, flagged as a cancellation) --
  // the same remote command and wait every way.
  const remoteMode = ref('return')
  const remoteTitles = { return: 'Return remotely', exchange: 'Exchange remotely', cancel: 'Cancel invoice' }
  const remoteDoneText = {
    return: 'Return processed successfully.',
    exchange: 'Exchange processed successfully.',
    cancel: 'Invoice cancelled.',
  }

  // Replacement lines for an exchange, picked from kashi's catalogue; kashi
  // turns each into the till's own product keys (see withTillReplacements).
  const { variants: skuOptions, loading: skuLoading, searchVariants } = useVariants()
  const replacements = ref([])
  const replacementPick = ref(null)
  const replacementQuantity = ref(1)
  let skuSearchTimer = null
  function onSkuSearch (q) {
    clearTimeout(skuSearchTimer)
    skuSearchTimer = setTimeout(() => searchVariants(q || ''), 250)
  }

  function addReplacement () {
    const variant = replacementPick.value
    const quantity = Math.max(1, Math.trunc(Number(replacementQuantity.value) || 1))
    const existing = replacements.value.find(row => row.variant.id === variant.id)
    if (existing) {
      existing.quantity += quantity
    } else {
      replacements.value.push({ variant, quantity })
    }
    replacementPick.value = null
    replacementQuantity.value = 1
  }
  const remoteReturnTarget = ref(null)
  const remoteReturnReason = ref('')
  // "Print the receipt on the till"; ticked again each time the dialog opens.
  const remoteReturnPrint = ref(true)
  const remoteReturnLoading = ref(false)
  const remoteReturnRows = ref([])
  const remoteReturnItemsLoading = ref(false)
  // 'items' (picking what to return) -> 'polling' (command queued, waiting
  // on the branch) -> 'result' (final success/failed/still-pending state).
  const remoteReturnPhase = ref('items')
  const remoteReturnResult = ref(null)
  let remoteReturnPollToken = 0

  const hasSelectedReturnItems = computed(() => remoteReturnRows.value.some(row => row.selected))

  // 'payments': how the refund (or an exchange's difference) is split
  // between the till's payment methods, entered on the calculator.
  const { loadPaymentSetup, variantPrices } = usePaymentEntry()
  const remoteMethods = ref([])
  const remoteCurrencies = ref([])
  const remoteEntries = ref([])
  const remotePrices = ref({})
  const remotePaymentsLoading = ref(false)
  const remoteSendingPayments = ref(false)
  let remoteEntriesDue = null
  const selectedReturnRows = computed(() => remoteReturnRows.value.filter(row => row.selected).map(row => ({
    ...row,
    quantity: Math.min(Math.max(1, Number(row.quantity) || 1), row.maxQuantity),
  })))
  const remoteDueCents = computed(() => estimateCents(
    selectedReturnRows.value,
    remoteMode.value === 'exchange' ? replacements.value : [],
    remotePrices.value,
  ))
  const remotePaymentStatus = computed(() => paymentStatus(remoteEntries.value, remoteDueCents.value, remoteCurrencies.value))
  const unpricedNames = computed(() => remoteMode.value === 'exchange'
    ? unpricedReplacements(replacements.value, remotePrices.value).map(row => row.variant.label)
    : [])

  async function openRemotePayments () {
    const target = remoteReturnTarget.value
    remoteReturnPhase.value = 'payments'
    remotePaymentsLoading.value = true
    try {
      const [setup, prices] = await Promise.all([
        remoteMethods.value.length > 0 ? { methods: remoteMethods.value, currencies: remoteCurrencies.value } : loadPaymentSetup(target.branch_id),
        remoteMode.value === 'exchange' ? variantPrices(target.branch_id, replacements.value.map(row => row.variant.id)) : {},
      ])
      remoteMethods.value = setup.methods
      remoteCurrencies.value = setup.currencies
      remotePrices.value = prices
      // Entries made against another total (the items changed) start over.
      if (remoteDueCents.value !== remoteEntriesDue) {
        remoteEntries.value = []
        remoteEntriesDue = remoteDueCents.value
      }
    } catch (error) {
      notify.error(error.message)
      remoteReturnPhase.value = 'items'
    } finally {
      remotePaymentsLoading.value = false
    }
  }

  async function openRemoteReturn (invoice, mode = 'return') {
    remoteMode.value = mode
    remoteReturnTarget.value = invoice
    remoteReturnReason.value = ''
    remoteReturnPrint.value = true
    remoteReturnRows.value = []
    replacements.value = []
    replacementPick.value = null
    replacementQuantity.value = 1
    remoteMethods.value = []
    remoteEntries.value = []
    remoteEntriesDue = null
    remotePrices.value = {}
    if (mode === 'exchange') {
      searchVariants('')
    }
    remoteReturnPhase.value = 'items'
    remoteReturnDialog.value = true
    remoteReturnItemsLoading.value = true
    try {
      const invoiceItems = await listBranchInvoiceItems(invoice.id)
      // Every line starts selected at its full quantity, so leaving
      // everything checked reproduces today's whole-invoice return.
      remoteReturnRows.value = invoiceItems.map(item => ({
        item,
        branch_product_id: item.branch_product_id,
        unit_price: item.unit_price,
        maxQuantity: item.quantity,
        quantity: item.quantity,
        selected: true,
      }))
    } catch (error) {
      notify.error(error.message)
    } finally {
      remoteReturnItemsLoading.value = false
    }
  }

  async function submitRemoteReturn (withPayments = false) {
    remoteReturnLoading.value = true
    remoteSendingPayments.value = withPayments
    try {
      const items = remoteReturnRows.value
        .filter(row => row.selected)
        .map(row => ({
          branch_product_id: row.branch_product_id,
          // A product can sit on several lines of one sale (a deal's paid
          // and free units, or two prices); the price tells the till which.
          unit_price: row.unit_price,
          quantity: Math.min(Math.max(1, Number(row.quantity) || 1), row.maxQuantity),
        }))
      const replacementLines = remoteMode.value === 'exchange'
        ? replacements.value.map(row => ({ variant_id: row.variant.id, quantity: row.quantity }))
        : []
      const payments = withPayments ? toPayload(remoteEntries.value) : []
      const command = await requestRemoteReturn(remoteReturnTarget.value.branch_id, remoteReturnTarget.value, remoteReturnReason.value, items, replacementLines, remoteReturnPrint.value, payments)
      remoteReturnPhase.value = 'polling'
      pollRemoteReturnCommand(remoteReturnTarget.value.branch_id, command?.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      remoteReturnLoading.value = false
    }
  }

  // The branch usually executes within a second or two if it's online (see
  // requestRemoteReturn), so a handful of quick polls covers the common
  // case; if it's still pending after that, it just hasn't come back
  // online yet -- the branch-invoices table's own WS listener (above) will
  // reflect it live whenever that happens, no need to keep polling forever.
  async function pollRemoteReturnCommand (branchId, commandId, attempt = 0) {
    const token = ++remoteReturnPollToken
    if (!commandId) {
      remoteReturnResult.value = { status: 'pending' }
      remoteReturnPhase.value = 'result'
      return
    }
    let commands
    try {
      commands = await listBranchCommands(branchId)
    } catch {
      commands = []
    }
    if (token !== remoteReturnPollToken) return // dialog closed/reopened meanwhile
    const command = commands.find(cmd => cmd.id === commandId)
    const status = command?.status ?? 'pending'
    if (status !== 'pending' || attempt >= 9) {
      remoteReturnResult.value = { status, error: command?.error?.Valid ? command.error.String : '' }
      remoteReturnPhase.value = 'result'
      return
    }
    setTimeout(() => pollRemoteReturnCommand(branchId, commandId, attempt + 1), 1500)
  }

  function closeRemoteReturn () {
    remoteReturnPollToken++ // abort any poll chain still in flight
    remoteReturnDialog.value = false
  }
</script>

<style scoped>
.filter-summary {
  flex: 0 1 auto;
  min-width: 0;
}

.invoice-details-card {
  display: flex;
  height: 720px !important;
  max-height: calc(100vh - 48px);
  flex-direction: column;
}

.invoice-details-content {
  flex: 1 1 auto;
  overflow-y: auto;
}

.details-table {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
}
</style>
