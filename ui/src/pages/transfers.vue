<template>
  <v-dialog v-model="dialog" max-width="560">
    <v-card class="px-4">
      <v-card-title>{{ editingId ? 'Edit Transfer' : 'Add a New Transfer' }}</v-card-title>
      <v-card-text>
        <form @submit.prevent="submit">
          <v-row>
            <v-col cols="6">
              <v-select
                v-model="fromInventoryId.value.value"
                density="compact"
                :error-messages="fromInventoryId.errorMessage.value"
                :item-title="i => i.name"
                :item-value="i => i.id"
                :items="inventories"
                label="From Inventory"
              />
            </v-col>
            <v-col cols="6">
              <v-select
                v-model="toInventoryId.value.value"
                density="compact"
                :error-messages="toInventoryId.errorMessage.value"
                :item-title="i => i.name"
                :item-value="i => i.id"
                :items="inventories"
                label="To Inventory"
              />
            </v-col>
          </v-row>
          <v-text-field v-model="note.value.value" density="compact" label="Note (optional)" />

          <v-card-actions class="px-0">
            <v-spacer />
            <v-btn text="Cancel" @click="closeDialog" />
            <v-btn color="primary" :loading="submitting" :text="editingId ? 'Save' : 'Create Transfer'" type="submit" />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <v-dialog v-model="itemsDialog" max-width="720">
    <v-card class="px-4">
      <v-card-title>
        Transfer #{{ itemsTarget?.id }}
        <v-chip class="ml-2" :color="statusColor(itemsTarget?.status)" size="small">{{ itemsTarget?.status }}</v-chip>
      </v-card-title>
      <v-card-text>
        <div v-if="itemsTarget?.branch_name" class="text-body-2 text-medium-emphasis mb-2">
          Requested by <strong>{{ itemsTarget.branch_name }}</strong>
          <span v-if="itemsTarget.note"> — {{ itemsTarget.note }}</span>
        </div>
        <div class="text-body-2 mb-2">
          <v-chip v-if="itemsTarget?.confirmed_at" color="success" size="small" variant="tonal">
            <v-icon icon="mdi-check-circle-outline" start />
            Confirmed by destination {{ formatDateTime(itemsTarget.confirmed_at) }}
          </v-chip>
          <v-chip v-else-if="itemsTarget?.status === 'requested'" color="warning" size="small" variant="tonal">
            Awaiting destination confirmation
          </v-chip>
          <div v-if="itemsTarget?.confirmed_note" class="text-caption text-medium-emphasis mt-1">
            “{{ itemsTarget.confirmed_note }}”
          </div>
        </div>
        <v-img
          v-if="attachmentObjectUrl"
          class="mb-4 transfer-attachment"
          max-height="260"
          rounded
          :src="attachmentObjectUrl"
        />
        <div v-if="canEdit && itemsTarget?.status === 'draft'" class="d-flex align-center mb-4" style="gap: 12px;">
          <v-autocomplete
            v-model="itemVariantId"
            density="compact"
            hide-details
            hide-no-data
            :item-title="v => v.label"
            :item-value="v => v.id"
            :items="skuOptions"
            :loading="skuLoading"
            no-filter
            placeholder="Search SKU / barcode"
            @update:search="onSkuSearch"
          />
          <v-text-field
            v-model.number="itemQuantity"
            density="compact"
            hide-details
            label="Quantity"
            style="max-width: 140px;"
            type="number"
          />
          <v-btn :loading="itemSubmitting" text="Add" @click="addItem" />
        </div>
        <v-table density="compact">
          <thead>
            <tr>
              <th>SKU</th>
              <th>Barcode</th>
              <th class="text-end">Sent</th>
              <th class="text-end">Received</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.id">
              <td>{{ item.product_code }} — {{ item.product_name }}</td>
              <td>{{ item.variant_barcode }}</td>
              <td class="text-end">{{ item.quantity }}</td>
              <td class="text-end">
                <v-text-field
                  v-if="canEdit"
                  v-model.number="item.received_quantity"
                  density="compact"
                  hide-details
                  :placeholder="String(item.quantity)"
                  style="max-width: 100px; display: inline-block;"
                  type="number"
                  @update:model-value="v => onReceivedQuantityChange(item, v)"
                />
                <span
                  v-else
                  :class="{ 'text-warning': item.received_quantity != null && item.received_quantity !== item.quantity }"
                >
                  {{ item.received_quantity ?? '—' }}
                </span>
              </td>
            </tr>
          </tbody>
        </v-table>
      </v-card-text>
      <v-card-actions>
        <v-btn
          v-if="canEdit && itemsTarget?.status === 'requested'"
          color="success"
          :loading="staging"
          prepend-icon="mdi-check"
          text="Approve"
          @click="stage('approve')"
        />
        <v-btn
          v-if="canEdit && itemsTarget?.status === 'draft'"
          color="primary"
          :loading="staging"
          text="Dispatch"
          @click="stage('dispatch')"
        />
        <v-btn
          v-if="canEdit && itemsTarget?.status === 'dispatched'"
          color="success"
          :loading="staging"
          text="Receive"
          @click="stage('receive')"
        />
        <v-btn
          v-if="canEdit && ['requested', 'draft', 'dispatched', 'received'].includes(itemsTarget?.status)"
          color="error"
          :loading="staging"
          :text="itemsTarget?.status === 'requested' ? 'Reject' : 'Cancel Transfer'"
          variant="text"
          @click="stage('cancel')"
        />
        <v-spacer />
        <v-btn text="Close" @click="closeItemsDialog" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <div class="page-root">
    <v-card class="transfers-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-transfer" />
        <span>Transfers</span>
        <v-btn-toggle
          v-model="statusFilter"
          color="primary"
          density="comfortable"
          mandatory
          variant="outlined"
        >
          <v-btn value="">All</v-btn>
          <v-btn value="requested">Requested</v-btn>
          <v-btn value="draft">Draft</v-btn>
          <v-btn value="dispatched">Dispatched</v-btn>
          <v-btn value="received">Received</v-btn>
        </v-btn-toggle>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Transfer"
          variant="flat"
          @click="openCreate"
        />
      </v-card-title>
      <v-divider />

      <div class="transfers-content">
        <div class="pa-4">
          <ServerSideTable
            ref="tableRef"
            :api-u-r-l="apiURL"
            density="comfortable"
            flush
            :headers="headers"
            :max-page-size="10"
            :query-params="queryParams"
            root-key="transfers"
            :show-search-icon="false"
          >
            <template #item.branch_name="{ item }">
              {{ item.branch_name || '—' }}
            </template>
            <template #item.status="{ item }">
              <v-chip :color="statusColor(item.status)" size="small">{{ item.status }}</v-chip>
            </template>
            <template #item.actions="{ item }">
              <v-icon-btn icon="mdi-format-list-bulleted" size="small" variant="text" @click="openItems(item)" />
              <v-icon-btn
                v-if="canEdit && item.status === 'draft'"
                icon="mdi-pencil"
                size="small"
                variant="text"
                @click="openEdit(item)"
              />
            </template>
          </ServerSideTable>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { useField, useForm } from 'vee-validate'
  import { computed, onBeforeUnmount, ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { authFetch } from '@/composables/useApi'
  import { useInventories } from '@/composables/useInventories'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useTransfers } from '@/composables/useTransfers'
  import { useVariants } from '@/composables/useVariants'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const {
    createTransfer, updateTransfer, fetchTransferItems, createTransferItem,
    setTransferItemReceivedQuantity,
    dispatchTransfer, receiveTransfer, approveTransfer, cancelTransfer,
  } = useTransfers()
  const { inventories, fetchInventories } = useInventories()
  const { variants: skuOptions, loading: skuLoading, searchVariants } = useVariants()
  fetchInventories()
  searchVariants('')

  let skuSearchTimer = null
  function onSkuSearch (q) {
    clearTimeout(skuSearchTimer)
    skuSearchTimer = setTimeout(() => searchVariants(q || ''), 250)
  }

  const apiURL = `${API_BASE}/transfers`
  const headers = ref([
    { title: 'ID', key: 'id', align: 'start' },
    { title: 'From', key: 'from_inventory_name', align: 'start' },
    { title: 'To', key: 'to_inventory_name', align: 'start' },
    { title: 'Branch', key: 'branch_name', align: 'start' },
    { title: 'Items', key: 'item_count', align: 'end' },
    { title: 'Status', key: 'status', align: 'start' },
    { title: 'Created At', key: 'created_at', align: 'end' },
    { title: '', key: 'actions', align: 'end', sortable: false, width: 96 },
  ])

  function statusColor (s) {
    return {
      requested: 'info', draft: 'grey', dispatched: 'warning', received: 'success', cancelled: 'error',
    }[s] || 'grey'
  }

  function formatDateTime (value) {
    if (!value) return ''
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
  }

  const statusFilter = ref('')
  const queryParams = computed(() => ({ status: statusFilter.value }))
  const tableRef = ref(null)
  const dialog = ref(false)
  const submitting = ref(false)
  const notify = useNotify()
  const editingId = ref(null)

  const { handleSubmit, handleReset, setValues } = useForm({
    validationSchema: {
      fromInventoryId (value) {
        return !!value || 'Select the source inventory.'
      },
      toInventoryId (value, { form }) {
        if (!value) return 'Select the destination inventory.'
        if (value === form.fromInventoryId) return 'From and To inventories must be different.'
        return true
      },
    },
  })

  const fromInventoryId = useField('fromInventoryId')
  const toInventoryId = useField('toInventoryId')
  const note = useField('note')

  function openCreate () {
    editingId.value = null
    handleReset()
    dialog.value = true
  }

  function openEdit (item) {
    editingId.value = item.id
    setValues({ fromInventoryId: item.from_inventory_id, toInventoryId: item.to_inventory_id, note: item.note })
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
        ? updateTransfer({
          id: editingId.value,
          from_inventory_id: values.fromInventoryId,
          to_inventory_id: values.toInventoryId,
          type: 'products',
          note: values.note || '',
        })
        : createTransfer({
          fromInventoryId: values.fromInventoryId,
          toInventoryId: values.toInventoryId,
          note: values.note || '',
          items: [],
        }))
      closeDialog()
      tableRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  })

  const itemsDialog = ref(false)
  const itemsTarget = ref(null)
  const items = ref([])
  const itemVariantId = ref(null)
  const itemQuantity = ref(null)
  const itemSubmitting = ref(false)
  const staging = ref(false)
  const attachmentObjectUrl = ref(null)

  // <v-img src> can't carry an Authorization header, so the attachment is
  // fetched as a blob and handed back as an object URL -- same pattern as
  // useShiftReceiptImages.js's fetchShiftReceiptImageObjectUrl.
  function releaseAttachmentObjectUrl () {
    if (attachmentObjectUrl.value) {
      URL.revokeObjectURL(attachmentObjectUrl.value)
      attachmentObjectUrl.value = null
    }
  }

  async function loadAttachment (transferId) {
    releaseAttachmentObjectUrl()
    try {
      const res = await authFetch(`${API_BASE}/transfers/${transferId}/attachment`)
      if (!res.ok) return
      const blob = await res.blob()
      attachmentObjectUrl.value = URL.createObjectURL(blob)
    } catch {
      // No attachment, or it failed to load -- the dialog just shows none.
    }
  }

  onBeforeUnmount(releaseAttachmentObjectUrl)

  function closeItemsDialog () {
    itemsDialog.value = false
    releaseAttachmentObjectUrl()
  }

  async function openItems (item) {
    itemsTarget.value = item
    itemsDialog.value = true
    if (item.attachment_path) loadAttachment(item.id)
    try {
      items.value = await fetchTransferItems(item.id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function addItem () {
    if (!itemVariantId.value || !itemQuantity.value) return
    itemSubmitting.value = true
    try {
      await createTransferItem({
        transferId: itemsTarget.value.id,
        variantId: itemVariantId.value,
        quantity: itemQuantity.value,
      })
      items.value = await fetchTransferItems(itemsTarget.value.id)
      itemVariantId.value = null
      itemQuantity.value = null
    } catch (error) {
      notify.error(error.message)
    } finally {
      itemSubmitting.value = false
    }
  }

  // Debounced so admin isn't firing a PUT on every keystroke while typing
  // a corrected count.
  let receivedQuantityTimers = {}
  function onReceivedQuantityChange (item, value) {
    clearTimeout(receivedQuantityTimers[item.id])
    receivedQuantityTimers[item.id] = setTimeout(async () => {
      if (value === '' || value == null || Number.isNaN(Number(value))) return
      try {
        await setTransferItemReceivedQuantity(item.id, Number(value))
      } catch (error) {
        notify.error(error.message)
      }
    }, 500)
  }

  async function stage (action) {
    staging.value = true
    try {
      const fn = {
        dispatch: dispatchTransfer, receive: receiveTransfer, approve: approveTransfer, cancel: cancelTransfer,
      }[action]
      const res = await fn(itemsTarget.value.id)
      itemsTarget.value = res?.transfer ?? itemsTarget.value
      tableRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      staging.value = false
    }
  }
</script>

<style scoped>
/* Fill the layout's flex-column scroll wrapper so the scroll lives inside the
   table, not the whole page. */
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.transfers-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.transfers-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}
</style>
