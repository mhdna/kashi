<!--
  Master/detail manager shared by pages/price-lists.vue and
  pages/discount-lists.vue. They are the same entity: a list of lists, each
  with per-product line items and a branch assignment. The only real
  difference is one value column — `price` (cents) vs `discount` (percent) —
  captured in `cfg` below and keyed off the `kind` prop.

  Line items are paged server-side through ServerSideTable; nothing here
  holds the whole item set in memory.
-->
<template>
  <div class="page-root">
    <v-card class="vlm-card" flat>
      <v-card-title class="page-heading d-flex align-center ga-3 px-4 py-3">
        <v-icon :icon="cfg.icon" />
        <span>{{ cfg.title }}</span>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-plus"
          :text="`Add ${cfg.singularTitle}`"
          variant="flat"
          @click="openCreateList"
        />
      </v-card-title>
      <v-divider />

      <div class="vlm-layout">
        <PageSidebar v-model="selectedListId" :items="listItems" :width="240">
          <template #append="{ item }">
            <v-icon v-if="item.is_special" color="amber-darken-2" icon="mdi-star" size="16" />
            <v-chip v-if="item.is_default" color="primary" size="x-small" text="Default" />
            <v-icon
              v-else-if="!item.is_active"
              color="grey"
              icon="mdi-close-circle"
              size="16"
            />
          </template>
        </PageSidebar>

        <v-divider vertical />

        <div class="vlm-content">
          <div v-if="!selectedList" class="pa-6 text-medium-emphasis">
            Select a {{ cfg.singular }} on the left, or add one.
          </div>

          <template v-else>
            <div class="d-flex align-center flex-wrap ga-2 pa-4 pb-2">
              <span class="text-h6">{{ selectedList.name }}</span>
              <v-chip v-if="selectedList.is_default" color="primary" size="small" text="Default" />
              <v-chip
                :color="selectedList.is_active ? 'success' : 'grey'"
                size="small"
                :text="selectedList.is_active ? 'Active' : 'Inactive'"
              />
              <v-chip
                v-if="selectedList.is_special"
                color="amber-darken-2"
                prepend-icon="mdi-star"
                size="small"
                text="Special"
              />
              <span class="text-medium-emphasis text-body-2">
                {{ formatDate(selectedList.valid_from) }} – {{ formatDate(selectedList.valid_to) }}
              </span>
              <v-spacer />
              <v-btn
                v-if="canEdit"
                prepend-icon="mdi-store-outline"
                size="small"
                text="Branches"
                variant="tonal"
                @click="branchesDialog = true"
              />
              <v-btn
                v-if="canEdit"
                icon="mdi-pencil"
                size="small"
                variant="text"
                @click="openEditList"
              />
              <v-btn
                v-if="canEdit"
                color="error"
                icon="mdi-delete"
                size="small"
                variant="text"
                @click="deleteListDialog = true"
              />
            </div>

            <v-divider />

            <div v-if="canEdit" class="d-flex align-end ga-3 pa-4">
              <v-autocomplete
                v-model="addVariantId"
                class="flex-grow-1"
                clearable
                density="compact"
                hide-details
                item-title="label"
                item-value="id"
                :items="variantOptions"
                label="Product variant"
                :loading="variantSearchLoading"
                no-filter
                variant="outlined"
                @update:search="onVariantSearch"
              />
              <v-text-field
                v-model="addValue"
                density="compact"
                hide-details
                :label="cfg.valueLabel"
                :prefix="cfg.prefix"
                style="max-width: 140px;"
                :suffix="cfg.suffix"
                type="number"
                variant="outlined"
                @keyup.enter="addItem"
              />
              <v-btn
                color="primary"
                :disabled="!addVariantId || addValue === ''"
                :loading="addingItem"
                text="Add"
                variant="flat"
                @click="addItem"
              />
              <v-btn
                v-if="kind === 'discount'"
                prepend-icon="mdi-playlist-plus"
                text="Paste codes"
                variant="tonal"
                @click="bulkDialog = true"
              />
            </div>

            <v-text-field
              v-model="itemSearch"
              class="mx-4"
              clearable
              density="compact"
              hide-details
              label="Search products in this list"
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
            />

            <BulkDeleteBar
              :count="selectedItems.length"
              :error="itemSelError"
              :loading="itemDeleting"
              @clear="resetItemSel"
              @confirm="bulkDeleteItems"
            />

            <ServerSideTable
              ref="tableRef"
              :api-u-r-l="itemsURL(selectedList.id)"
              class="mt-2"
              density="comfortable"
              :external-search="itemSearch ?? ''"
              flush
              :headers="itemHeaders"
              item-value="product_variant_id"
              root-key="items"
              :selectable="canEdit"
              :show-search-icon="false"
              @row-click="openEditItem"
              @update:selected="v => (selectedItems = v)"
            >
              <template #item.variant="{ item }">
                {{ variantText(item) }}
              </template>
              <template #[`item.${cfg.valueKey}`]="{ item }">
                {{ cfg.display(item[cfg.valueKey]) }}
              </template>
            </ServerSideTable>
          </template>
        </div>
      </div>
    </v-card>

    <!-- create / edit list -->
    <v-dialog v-model="listDialog" max-width="520">
      <v-card>
        <v-card-title>{{ editingList ? `Edit ${cfg.singularTitle}` : `Add ${cfg.singularTitle}` }}</v-card-title>
        <v-card-text>
          <v-text-field v-model="listForm.name" density="compact" label="Name" variant="outlined" />
          <v-row>
            <v-col cols="6">
              <DateField
                v-model="listForm.validFrom"
                density="compact"
                label="Valid From"
                variant="outlined"
              />
            </v-col>
            <v-col cols="6">
              <DateField
                v-model="listForm.validTo"
                density="compact"
                label="Valid To"
                variant="outlined"
              />
            </v-col>
          </v-row>
          <div class="d-flex ga-6">
            <v-switch
              v-model="listForm.isActive"
              color="primary"
              density="compact"
              hide-details
              label="Active"
            />
            <v-switch
              v-model="listForm.isDefault"
              color="primary"
              density="compact"
              hide-details
              label="Default"
            />
            <v-switch
              v-model="listForm.isSpecial"
              color="amber-darken-2"
              density="compact"
              hide-details
              label="Special"
            />
          </div>
          <p class="text-caption text-medium-emphasis mt-1">
            Special lists get a star badge on the till's invoice screen, so cashiers notice
            the price came from a promo/loyalty list rather than the usual catalog price.
          </p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="listDialog = false" />
          <v-btn
            color="primary"
            :loading="savingList"
            text="Save"
            variant="flat"
            @click="saveList"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- edit item value -->
    <v-dialog v-model="bulkDialog" max-width="520">
      <v-card>
        <v-card-title>Add many products to this list</v-card-title>
        <v-card-text>
          <p class="text-body-2 text-medium-emphasis mb-3">
            Paste product codes or barcodes, one per line or separated by commas. A product code
            covers all of its colours and sizes. Tills pick the change up within seconds.
          </p>
          <v-textarea
            v-model="bulkEntries"
            auto-grow
            autofocus
            density="compact"
            hide-details
            label="Codes or barcodes"
            rows="5"
            variant="outlined"
          />
          <v-text-field
            v-model="bulkValue"
            class="mt-3"
            density="compact"
            hide-details
            label="Discount"
            style="max-width: 160px;"
            suffix="%"
            type="number"
            variant="outlined"
          />
          <v-alert
            v-if="bulkUnmatched.length > 0"
            class="mt-3"
            density="compact"
            type="warning"
            variant="tonal"
          >
            Not found: {{ bulkUnmatched.join(', ') }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Close" variant="text" @click="bulkDialog = false" />
          <v-btn
            color="primary"
            :disabled="!bulkEntries.trim() || bulkValue === ''"
            :loading="bulkSaving"
            text="Apply"
            variant="flat"
            @click="applyBulk"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="editItemDialog" max-width="420">
      <v-card>
        <v-card-title>
          {{ editItemTarget?.product_code }} — {{ editItemTarget?.product_name }}
          <div class="text-caption text-medium-emphasis">
            {{ variantText(editItemTarget) }} · {{ editItemTarget?.barcode }}
          </div>
        </v-card-title>
        <v-card-text>
          <v-text-field
            v-model="editItemValue"
            autofocus
            density="compact"
            hide-details
            :label="cfg.valueLabel"
            :prefix="cfg.prefix"
            :suffix="cfg.suffix"
            type="number"
            variant="outlined"
            @keyup.enter="saveEditItem"
          />
        </v-card-text>
        <v-card-actions>
          <v-btn
            color="error"
            prepend-icon="mdi-delete"
            text="Remove"
            variant="text"
            @click="deleteCurrentItem"
          />
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="editItemDialog = false" />
          <v-btn
            color="primary"
            :loading="savingItem"
            text="Save"
            variant="flat"
            @click="saveEditItem"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- delete list -->
    <v-dialog v-model="deleteListDialog" max-width="420">
      <v-card>
        <v-card-title>Delete {{ cfg.singular }}</v-card-title>
        <v-card-text>
          Delete <strong>{{ selectedList?.name }}</strong> and all its entries? This can't be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="deleteListDialog = false" />
          <v-btn
            color="error"
            :loading="deletingList"
            text="Delete"
            variant="flat"
            @click="doDeleteList"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <BranchAssignDialog
      v-model="branchesDialog"
      :kind="cfg.singular"
      :load-fn="() => fetchBranchIds(selectedList.id)"
      :save-fn="ids => setBranches(selectedList.id, ids)"
      :title="`Branches using “${selectedList?.name}”`"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import BranchAssignDialog from '@/components/BranchAssignDialog.vue'
  import PageSidebar from '@/components/PageSidebar.vue'
  import BulkDeleteBar from '@/components/Tables/BulkDeleteBar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { authFetch } from '@/composables/useApi'
  import { useBulkDelete } from '@/composables/useBulkDelete'
  import { useDiscountLists } from '@/composables/useDiscountLists'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { usePriceLists } from '@/composables/usePriceLists'
  import { useVariants } from '@/composables/useVariants'
  import { API_BASE } from '@/config'
  import { formatMoney } from '@/utils/money'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()
  const notify = useNotify()

  const props = defineProps({
    kind: { type: String, required: true, validator: v => v === 'price' || v === 'discount' },
  })

  const CONFIGS = {
    price: {
      icon: 'mdi-format-list-bulleted',
      title: 'Price Lists',
      singular: 'price list',
      singularTitle: 'Price List',
      valueKey: 'price',
      valueLabel: 'Price',
      prefix: '$',
      suffix: undefined,
      parse: v => Math.round(Number.parseFloat(v) * 100),
      toInput: v => (v / 100).toString(),
      display: v => `$${formatMoney(v)}`,
    },
    discount: {
      icon: 'mdi-sale',
      title: 'Discount Lists',
      singular: 'discount list',
      singularTitle: 'Discount List',
      valueKey: 'discount',
      valueLabel: 'Discount',
      prefix: undefined,
      suffix: '%',
      parse: v => Math.round(Number.parseFloat(v)),
      toInput: String,
      display: v => `${v}%`,
    },
  }
  const cfg = computed(() => CONFIGS[props.kind])

  const store = props.kind === 'price' ? usePriceLists() : useDiscountLists()
  const {
    list: lists, fetchLists, createList, updateList, deleteList,
    itemsURL, saveItem, deleteItem, fetchBranchIds, setBranches,
  } = store

  // Code + name no longer identify a row on their own — every variant of a
  // product shares them — so the variant's colour/size and barcode are shown
  // alongside.
  const itemHeaders = computed(() => [
    { title: 'Code', key: 'product_code', align: 'start', width: 140 },
    { title: 'Product', key: 'product_name', align: 'start' },
    { title: 'Variant', key: 'variant', align: 'start', width: 160, sortable: false },
    { title: 'Barcode', key: 'barcode', align: 'start', width: 150 },
    { title: cfg.value.valueLabel, key: cfg.value.valueKey, align: 'end', width: 140 },
  ])

  // color_name/size_name arrive as nullable strings from the join.
  function variantText (item) {
    return [item?.color_name, item?.size_name].filter(Boolean).join(' / ') || '—'
  }

  const selectedListId = ref(null)
  const selectedList = computed(() => lists.value.find(l => l.id === selectedListId.value) ?? null)
  const listItems = computed(() => lists.value.map(l => ({ value: l.id, title: l.name, raw: l })))
  const tableRef = ref(null)
  const itemSearch = ref('')

  function reloadItems () {
    tableRef.value?.reload()
  }

  // ---- item bulk delete ------------------------------------------------
  const {
    selected: selectedItems,
    deleting: itemDeleting,
    error: itemSelError,
    reset: resetItemSel,
    run: runItemBulk,
  } = useBulkDelete()

  async function bulkDeleteItems () {
    if (!selectedList.value) return
    try {
      await runItemBulk(`/${props.kind === 'price' ? 'price_lists' : 'discount_lists'}/${selectedList.value.id}/items/bulk_delete`, {
        product_variant_ids: selectedItems.value,
      })
      resetItemSel()
      reloadItems()
    } catch { /* error shown in the bar */ }
  }

  watch(selectedListId, () => {
    resetItemSel()
    itemSearch.value = ''
  })

  onMounted(async () => {
    try {
      await fetchLists(true)
      if (selectedListId.value == null && lists.value.length > 0) {
        selectedListId.value = lists.value[0].id
      }
    } catch (error) {
      notify.error(error.message)
    }
  })

  // ---- variant search ---------------------------------------------------
  // List items key on the variant, so the picker searches SKUs (GET /barcodes)
  // rather than products -- reusing useVariants, the same source the sales
  // invoice picker uses, so a row reads identically in both places.
  const { variants: variantOptions, loading: variantSearchLoading, searchVariants } = useVariants()
  const addVariantId = ref(null)
  const addValue = ref('')
  const addingItem = ref(false)
  let variantSearchTimer

  function onVariantSearch (query) {
    clearTimeout(variantSearchTimer)
    variantSearchTimer = setTimeout(() => searchVariants(query?.trim() ?? ''), 300)
  }

  onMounted(() => searchVariants(''))

  async function addItem () {
    if (!addVariantId.value || addValue.value === '') return
    const parsed = cfg.value.parse(addValue.value)
    if (!Number.isFinite(parsed) || parsed < 0) {
      notify.error(`Enter a valid ${cfg.value.valueLabel.toLowerCase()}.`)
      return
    }
    addingItem.value = true
    try {
      await saveItem(selectedList.value.id, addVariantId.value, parsed)
      addVariantId.value = null
      addValue.value = ''
      reloadItems()
    } catch (error) {
      notify.error(error.message)
    } finally {
      addingItem.value = false
    }
  }

  // ---- paste many codes onto a discount list ---------------------------------
  const bulkDialog = ref(false)
  const bulkEntries = ref('')
  const bulkValue = ref('')
  const bulkSaving = ref(false)
  const bulkUnmatched = ref([])

  async function applyBulk () {
    const discount = Number(bulkValue.value)
    if (!Number.isInteger(discount) || discount < 0 || discount > 100) {
      notify.error('Enter a discount from 0 to 100.')
      return
    }
    bulkSaving.value = true
    try {
      const res = await authFetch(`${API_BASE}/discount_lists/items/bulk`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          discount_list_id: selectedList.value.id,
          entries: bulkEntries.value.split(/[\s,;]+/).filter(Boolean),
          discount,
        }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
      bulkUnmatched.value = data.unmatched ?? []
      notify.success(`${data.set} product(s) set to ${discount}%.`)
      if (bulkUnmatched.value.length === 0) {
        bulkDialog.value = false
        bulkEntries.value = ''
      }
      reloadItems()
    } catch (error) {
      notify.error(error.message)
    } finally {
      bulkSaving.value = false
    }
  }

  // ---- edit / delete item -------------------------------------------------
  const editItemDialog = ref(false)
  const editItemTarget = ref(null)
  const editItemValue = ref('')
  const savingItem = ref(false)

  function openEditItem (item) {
    if (!canEdit.value) return
    editItemTarget.value = item
    editItemValue.value = cfg.value.toInput(item[cfg.value.valueKey])
    editItemDialog.value = true
  }

  function deleteCurrentItem () {
    const item = editItemTarget.value
    editItemDialog.value = false
    if (item) removeItem(item)
  }

  async function saveEditItem () {
    const parsed = cfg.value.parse(editItemValue.value)
    if (!Number.isFinite(parsed) || parsed < 0) return
    savingItem.value = true
    try {
      await saveItem(selectedList.value.id, editItemTarget.value.product_variant_id, parsed)
      editItemDialog.value = false
      reloadItems()
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingItem.value = false
    }
  }

  async function removeItem (item) {
    try {
      await deleteItem(selectedList.value.id, item.product_variant_id)
      reloadItems()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- create / edit / delete list ------------------------------------
  const listDialog = ref(false)
  const editingList = ref(false)
  const savingList = ref(false)
  const listForm = reactive({ name: '', validFrom: '', validTo: '', isActive: true, isDefault: false, isSpecial: false })

  function openCreateList () {
    editingList.value = false
    Object.assign(listForm, { name: '', validFrom: '', validTo: '', isActive: true, isDefault: false, isSpecial: false })
    listDialog.value = true
  }

  function openEditList () {
    editingList.value = true
    Object.assign(listForm, {
      name: selectedList.value.name,
      validFrom: selectedList.value.valid_from?.slice(0, 10) ?? '',
      validTo: selectedList.value.valid_to?.slice(0, 10) ?? '',
      isActive: selectedList.value.is_active,
      isDefault: selectedList.value.is_default,
      isSpecial: selectedList.value.is_special,
    })
    listDialog.value = true
  }

  async function saveList () {
    if (!listForm.name.trim() || !listForm.validFrom || !listForm.validTo) {
      notify.error('Name and both dates are required.')
      return
    }
    savingList.value = true
    const payload = {
      name: listForm.name.trim(),
      is_active: listForm.isActive,
      is_default: listForm.isDefault,
      is_special: listForm.isSpecial,
      valid_from: new Date(listForm.validFrom).toISOString(),
      valid_to: new Date(listForm.validTo).toISOString(),
    }
    try {
      if (editingList.value) {
        await updateList({ id: selectedList.value.id, ...payload })
      } else {
        const created = await createList(payload)
        if (created?.id) selectedListId.value = created.id
      }
      listDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingList.value = false
    }
  }

  const deleteListDialog = ref(false)
  const deletingList = ref(false)

  async function doDeleteList () {
    deletingList.value = true
    try {
      await deleteList(selectedList.value.id)
      deleteListDialog.value = false
      selectedListId.value = lists.value.length > 0 ? lists.value[0].id : null
    } catch (error) {
      notify.error(error.message)
    } finally {
      deletingList.value = false
    }
  }

  const branchesDialog = ref(false)

  function formatDate (value) {
    return value ? new Date(value).toLocaleDateString() : '—'
  }
</script>

<style scoped>
.page-root, .vlm-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.vlm-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.vlm-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}
</style>
