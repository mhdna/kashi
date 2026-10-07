<template>
  <div class="page-root">
    <v-card class="coupons-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-ticket-percent" />
        <span>Coupons</span>
        <v-spacer />
        <v-text-field
          v-if="tab === 'coupons'"
          v-model="search"
          class="table-search"
          clearable
          density="compact"
          hide-details
          label="Search code"
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
        />
        <v-btn-toggle
          v-if="tab === 'coupons'"
          v-model="statusFilter"
          color="primary"
          density="comfortable"
          mandatory
          variant="outlined"
        >
          <v-btn value="all">All</v-btn>
          <v-btn value="active">Active</v-btn>
          <v-btn value="inactive">Inactive</v-btn>
          <v-btn value="redeemed">Redeemed</v-btn>
        </v-btn-toggle>
        <v-btn
          v-if="canEdit && tab === 'coupons'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Coupon"
          variant="flat"
          @click="openCreate"
        />
        <v-btn
          v-if="canEdit && tab === 'types'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Type"
          variant="flat"
          @click="openCreateType"
        />
        <v-btn
          v-if="canEdit && tab === 'lists'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Generate List"
          variant="flat"
          @click="openCreateList"
        />
      </v-card-title>
      <v-divider />

      <div class="coupons-layout">
        <v-tabs
          v-model="tab"
          class="coupons-tabs rail-tabs"
          color="primary"
          :direction="$vuetify.display.smAndDown ? 'horizontal' : 'vertical'"
        >
          <v-tab prepend-icon="mdi-ticket-percent" text="Coupons" value="coupons" />
          <v-tab prepend-icon="mdi-tag-multiple" text="Types" value="types" />
          <v-tab prepend-icon="mdi-format-list-numbered" text="Lists" value="lists" />
        </v-tabs>

        <v-divider vertical />

        <div class="coupons-content">
          <!-- ------------------------------------------------------ COUPONS -->
          <div v-show="tab === 'coupons'" class="coupons-tab-layout">
            <CategorySidebar
              v-model="selectedCategory"
              :categories="couponCategories"
              @add="openCreateCategory"
              @delete="removeCategory"
              @edit="openEditCategory"
            />

            <v-divider vertical />

            <div class="coupons-tab-content">
              <div class="pa-4">
                <BulkDeleteBar
                  :count="selectedCoupons.length"
                  :error="couponSelError"
                  :loading="couponDeleting"
                  @clear="resetCouponSel"
                  @confirm="bulkDeleteCoupons"
                />

                <ServerSideTable
                  ref="tableRef"
                  :api-u-r-l="apiURL"
                  density="comfortable"
                  :external-search="search ?? ''"
                  flush
                  :headers="headers"
                  hover
                  item-value="code"
                  :max-page-size="100"
                  :query-params="queryParams"
                  root-key="coupons"
                  :selectable="canEdit"
                  :show-search-icon="false"
                  @row-click="openEdit"
                  @update:selected="v => (selectedCoupons = v)"
                >
                  <template #item.category_id="{ item }">
                    <CategoryChip :category="categoryFor(item.category_id)" />
                  </template>
                  <template #item.type_name="{ item }">
                    {{ item.type_name || '—' }}
                  </template>
                  <template #item.status="{ item }">
                    <v-chip :color="statusColor(item.status)" size="small">
                      {{ item.status }}
                    </v-chip>
                  </template>
                  <template #item.value_cents="{ item }">
                    {{ formatCents(item.value_cents) }}
                  </template>
                  <template #item.valid_until="{ item }">
                    {{ formatDate(item.valid_until) }}
                  </template>
                  <template #item.actions="{ item }">
                    <v-btn
                      v-if="canEdit"
                      :disabled="item.status !== 'active'"
                      size="small"
                      text="Deactivate"
                      variant="text"
                      @click.stop="deactivate(item)"
                    />
                  </template>
                </ServerSideTable>
              </div>
            </div>
          </div>

          <!-- -------------------------------------------------------- TYPES -->
          <div v-show="tab === 'types'" class="pa-4">
            <v-data-table
              density="comfortable"
              :headers="typeHeaders"
              :items="couponTypes"
              :items-per-page="25"
              @click:row="(e, { item }) => openEditType(item)"
            >
              <template #item.value_cents="{ item }">
                {{ formatCents(item.value_cents) }} {{ item.currency_code }}
              </template>
              <template #item.is_active="{ item }">
                <v-chip :color="item.is_active ? 'success' : 'default'" size="small">
                  {{ item.is_active ? 'active' : 'inactive' }}
                </v-chip>
              </template>
              <template #item.flags="{ item }">
                <v-chip v-if="item.reference_only" class="mr-1" size="small" variant="tonal">reference-only</v-chip>
                <v-chip v-if="item.calculate_on_checkout" class="mr-1" size="small" variant="tonal">auto-apply</v-chip>
                <v-chip v-if="!item.code_optional" size="small" variant="tonal">code required</v-chip>
              </template>
              <template #item.actions="{ item }">
                <v-btn
                  v-if="canEdit"
                  icon="mdi-delete-outline"
                  size="small"
                  variant="text"
                  @click.stop="removeType(item)"
                />
              </template>
            </v-data-table>
          </div>

          <!-- -------------------------------------------------------- LISTS -->
          <div v-show="tab === 'lists'" class="pa-4">
            <v-data-table
              density="comfortable"
              :headers="listHeaders"
              :items="couponLists"
              :items-per-page="25"
            >
              <template #item.coupon_type_name="{ item }">{{ item.coupon_type_name }}</template>
              <template #item.coupon_count="{ item }">
                {{ item.redeemed_count }} / {{ item.coupon_count }} redeemed
              </template>
              <template #item.is_active="{ item }">
                <v-chip :color="item.is_active ? 'success' : 'default'" size="small">
                  {{ item.is_active ? 'active' : 'closed' }}
                </v-chip>
              </template>
              <template #item.actions="{ item }">
                <v-btn
                  v-if="canEdit"
                  :disabled="!item.is_active"
                  size="small"
                  text="Close"
                  variant="text"
                  @click.stop="closeList(item)"
                />
              </template>
            </v-data-table>
          </div>
        </div>
      </div>
    </v-card>

    <!-- add / edit coupon -->
    <v-dialog v-model="dialog" max-width="560">
      <v-card class="px-4">
        <v-card-title>{{ editingCode ? 'Edit Coupon' : 'Add a New Coupon' }}</v-card-title>
        <v-card-text>
          <form @submit.prevent="submit">
            <v-text-field
              v-model="form.code"
              density="compact"
              :disabled="!!editingCode"
              :error-messages="formErrors.code"
              label="Code"
            />
            <v-select
              v-model="form.typeId"
              density="compact"
              :error-messages="formErrors.typeId"
              item-title="name"
              item-value="id"
              :items="couponTypes"
              label="Coupon Type"
            />
            <v-select
              v-model="form.categoryId"
              clearable
              density="compact"
              item-title="name"
              item-value="id"
              :items="activeCategories"
              label="Category"
            />
            <v-select
              v-model="form.clientId"
              clearable
              density="compact"
              :item-title="c => `${c.name} (${c.phone})`"
              :item-value="c => c.id"
              :items="selectableClients"
              label="Client (optional)"
            />
            <v-select
              v-model="form.status"
              density="compact"
              :items="['active', 'inactive']"
              label="Status"
            />
            <v-text-field
              v-model="form.validUntil"
              density="compact"
              label="Valid Until (optional)"
              type="date"
            />

            <v-card-actions class="px-0">
              <v-spacer />
              <v-btn text="Cancel" @click="closeDialog" />
              <v-btn
                color="primary"
                :loading="submitting"
                :text="editingCode ? 'Save' : 'Add Coupon'"
                type="submit"
              />
            </v-card-actions>
          </form>
        </v-card-text>
      </v-card>
    </v-dialog>

    <CategoryDialog
      v-model="categoryDialog"
      :category="editingCategory"
      name-hint="Coupon group, e.g. Seasonal, VIP"
      :on-submit="submitCategory"
    />

    <!-- add / edit coupon type -->
    <v-dialog v-model="typeDialog" max-width="520">
      <v-card class="px-4">
        <v-card-title>{{ editingTypeId ? 'Edit Coupon Type' : 'Add a New Coupon Type' }}</v-card-title>
        <v-card-text>
          <form @submit.prevent="submitType">
            <v-text-field
              v-model="typeForm.name"
              density="compact"
              :error-messages="typeFormErrors.name"
              label="Name"
            />
            <v-row>
              <v-col cols="8">
                <v-text-field
                  v-model.number="typeForm.valueAmount"
                  density="compact"
                  :error-messages="typeFormErrors.value"
                  label="Value"
                  min="0"
                  step="0.01"
                  type="number"
                />
              </v-col>
              <v-col cols="4">
                <v-text-field
                  v-model="typeForm.currencyCode"
                  density="compact"
                  label="Currency"
                />
              </v-col>
            </v-row>
            <v-select
              v-model="typeForm.categoryId"
              clearable
              density="compact"
              item-title="name"
              item-value="id"
              :items="activeCategories"
              label="Category"
            />
            <v-checkbox
              v-model="typeForm.isActive"
              density="compact"
              hide-details
              label="Active"
            />
            <v-checkbox
              v-model="typeForm.referenceOnly"
              density="compact"
              hide-details
              label="Reference-only (no monetary value calculated at checkout)"
            />
            <v-checkbox
              v-model="typeForm.calculateOnCheckout"
              density="compact"
              hide-details
              label="Auto-apply the value at checkout"
            />
            <v-checkbox
              v-model="typeForm.codeRequired"
              class="mb-2"
              density="compact"
              hide-details
              label="A code must be typed to redeem"
            />

            <v-card-actions class="px-0">
              <v-spacer />
              <v-btn text="Cancel" @click="closeTypeDialog" />
              <v-btn
                color="primary"
                :loading="typeSubmitting"
                :text="editingTypeId ? 'Save' : 'Add Type'"
                type="submit"
              />
            </v-card-actions>
          </form>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- generate coupon list -->
    <v-dialog v-model="listDialog" max-width="520">
      <v-card class="px-4">
        <template v-if="!generatedCodes">
          <v-card-title>Generate a Coupon List</v-card-title>
          <v-card-text>
            <form @submit.prevent="submitList">
              <v-text-field
                v-model="listForm.name"
                density="compact"
                :error-messages="listFormErrors.name"
                label="Campaign name"
              />
              <v-select
                v-model="listForm.typeId"
                density="compact"
                :error-messages="listFormErrors.typeId"
                item-title="name"
                item-value="id"
                :items="couponTypes"
                label="Coupon Type"
              />
              <v-text-field
                v-model.number="listForm.count"
                density="compact"
                :error-messages="listFormErrors.count"
                label="How many codes?"
                min="1"
                type="number"
              />
              <v-text-field
                v-model="listForm.validUntil"
                density="compact"
                label="Valid Until (optional)"
                type="date"
              />

              <v-card-actions class="px-0">
                <v-spacer />
                <v-btn text="Cancel" @click="closeListDialog" />
                <v-btn color="primary" :loading="listSubmitting" text="Generate" type="submit" />
              </v-card-actions>
            </form>
          </v-card-text>
        </template>
        <template v-else>
          <v-card-title class="d-flex align-center ga-2">
            <v-icon color="success" icon="mdi-check-circle" />
            <span>{{ generatedCodes.length }} codes generated</span>
          </v-card-title>
          <v-card-text>
            <v-textarea
              auto-grow
              density="compact"
              label="Codes"
              :model-value="generatedCodes.join('\n')"
              readonly
              rows="6"
            />
            <v-btn
              block
              prepend-icon="mdi-content-copy"
              variant="tonal"
              @click="copyGeneratedCodes"
            >
              Copy all codes
            </v-btn>
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn color="primary" text="Done" @click="closeListDialog" />
          </v-card-actions>
        </template>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import CategoryChip from '@/components/CategoryChip.vue'
  import CategorySidebar from '@/components/CategorySidebar.vue'
  import CategoryDialog from '@/components/Forms/CategoryDialog.vue'
  import BulkDeleteBar from '@/components/Tables/BulkDeleteBar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useBulkDelete } from '@/composables/useBulkDelete'
  import { useClients } from '@/composables/useClients'
  import { useCouponCategories } from '@/composables/useCouponCategories'
  import { useCouponLists } from '@/composables/useCouponLists'
  import { useCoupons } from '@/composables/useCoupons'
  import { useCouponTypes } from '@/composables/useCouponTypes'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()
  const notify = useNotify()
  const tab = ref('coupons')

  const { createCoupon, updateCoupon, deactivateCoupon } = useCoupons()
  const { clients, fetchClients } = useClients()
  // Inactive (deactivated) clients can't be picked for something new; the
  // one already selected stays listed so an existing value still renders.
  const selectableClients = computed(() =>
    clients.value.filter(c => c.is_active !== false || c.id === form.clientId))
  const {
    couponCategories,
    fetchCouponCategories,
    createCouponCategory,
    updateCouponCategory,
    deleteCouponCategory,
  } = useCouponCategories()
  const { couponTypes, fetchCouponTypes, createCouponType, updateCouponType, deleteCouponType } = useCouponTypes()
  const { couponLists, fetchCouponLists, createCouponList, deactivateCouponList } = useCouponLists()

  const apiURL = `${API_BASE}/coupons`
  const headers = ref([
    { title: 'Code', key: 'code', align: 'start' },
    { title: 'Type', key: 'type_name', align: 'start', sortable: false },
    { title: 'Category', key: 'category_id', align: 'start', sortable: false },
    { title: 'Status', key: 'status', align: 'start' },
    { title: 'Value', key: 'value_cents', align: 'end' },
    { title: 'Valid Until', key: 'valid_until', align: 'end' },
    { title: 'Actions', key: 'actions', align: 'end', sortable: false },
  ])
  const typeHeaders = ref([
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'Value', key: 'value_cents', align: 'end' },
    { title: 'Status', key: 'is_active', align: 'start' },
    { title: 'Flags', key: 'flags', align: 'start', sortable: false },
    { title: '', key: 'actions', align: 'end', sortable: false },
  ])
  const listHeaders = ref([
    { title: 'Campaign', key: 'name', align: 'start' },
    { title: 'Type', key: 'coupon_type_name', align: 'start' },
    { title: 'Codes', key: 'coupon_count', align: 'start', sortable: false },
    { title: 'Status', key: 'is_active', align: 'start' },
    { title: '', key: 'actions', align: 'end', sortable: false },
  ])

  const tableRef = ref(null)
  const search = ref('')
  const statusFilter = ref('all')
  const selectedCategory = ref(null)

  const queryParams = computed(() => ({
    status: statusFilter.value === 'all' ? '' : statusFilter.value,
    category_id: selectedCategory.value || '',
  }))

  const activeCategories = computed(() => couponCategories.value.filter(c => c.is_active))

  function categoryFor (id) {
    return couponCategories.value.find(c => c.id === id) ?? null
  }

  function statusColor (status) {
    if (status === 'active') return 'success'
    if (status === 'redeemed') return 'info'
    return 'default'
  }

  function formatDate (value) {
    if (!value) return '—'
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString()
  }

  function formatCents (cents) {
    return `$${((cents ?? 0) / 100).toFixed(2)}`
  }

  function reload () {
    tableRef.value?.reload()
  }

  onMounted(() => {
    fetchClients()
    fetchCouponCategories(true).catch(error => notify.error(error.message))
    fetchCouponTypes(true).catch(error => notify.error(error.message))
    fetchCouponLists(true).catch(error => notify.error(error.message))
  })

  // ---- bulk delete ---------------------------------------------------
  const {
    selected: selectedCoupons,
    deleting: couponDeleting,
    error: couponSelError,
    reset: resetCouponSel,
    run: runCouponBulk,
  } = useBulkDelete()

  async function bulkDeleteCoupons () {
    try {
      await runCouponBulk('/coupons/bulk_delete', { codes: selectedCoupons.value })
      resetCouponSel()
      reload()
    } catch { /* error shown in the bar */ }
  }

  watch([statusFilter, selectedCategory], () => resetCouponSel())

  // ---- add / edit coupon -------------------------------------------------
  const dialog = ref(false)
  const editingCode = ref('')
  const submitting = ref(false)
  const form = reactive({
    code: '',
    typeId: null,
    categoryId: null,
    clientId: null,
    status: 'active',
    validUntil: '',
  })
  const formErrors = reactive({ code: '', typeId: '' })

  function resetForm () {
    Object.assign(form, {
      code: '', typeId: null, categoryId: null, clientId: null, status: 'active', validUntil: '',
    })
    Object.assign(formErrors, { code: '', typeId: '' })
  }

  function openCreate () {
    editingCode.value = ''
    resetForm()
    dialog.value = true
  }

  function openEdit (item) {
    if (!canEdit.value) return
    editingCode.value = item.code
    resetForm()
    Object.assign(form, {
      code: item.code,
      typeId: item.type_id ?? null,
      categoryId: item.category_id ?? null,
      clientId: item.client_id ?? null,
      status: item.status,
      validUntil: item.valid_until?.slice(0, 10) ?? '',
    })
    dialog.value = true
  }

  function closeDialog () {
    dialog.value = false
    editingCode.value = ''
    resetForm()
  }

  function validate () {
    formErrors.code = editingCode.value || form.code?.trim().length >= 1 ? '' : 'Code is required.'
    formErrors.typeId = form.typeId ? '' : 'Coupon type is required.'
    return !formErrors.code && !formErrors.typeId
  }

  async function submit () {
    if (!validate()) return
    submitting.value = true
    const payload = {
      status: form.status,
      type_id: form.typeId,
      client_id: form.clientId ?? 0,
      valid_until: form.validUntil ? new Date(form.validUntil).toISOString() : '',
      category_id: form.categoryId ?? 0,
    }
    try {
      await (editingCode.value ? updateCoupon(editingCode.value, payload) : createCoupon({ code: form.code.trim(), ...payload }))
      closeDialog()
      reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  }

  async function deactivate (item) {
    try {
      await deactivateCoupon(item.code)
      reload()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- categories --------------------------------------------------------
  const categoryDialog = ref(false)
  const editingCategory = ref(null)

  function openCreateCategory () {
    editingCategory.value = null
    categoryDialog.value = true
  }

  function openEditCategory (category) {
    editingCategory.value = category
    categoryDialog.value = true
  }

  async function submitCategory (payload) {
    await (editingCategory.value
      ? updateCouponCategory({ id: editingCategory.value.id, ...payload })
      : createCouponCategory(payload))
    await fetchCouponCategories(true)
    reload()
  }

  async function removeCategory (category) {
    if (!confirm(`Delete category "${category.name}"? Categories still used by a coupon can't be deleted.`)) return
    try {
      await deleteCouponCategory(category.id)
      if (selectedCategory.value === category.id) selectedCategory.value = null
      await fetchCouponCategories(true)
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- coupon types --------------------------------------------------------
  const typeDialog = ref(false)
  const editingTypeId = ref(null)
  const typeSubmitting = ref(false)
  const typeForm = reactive({
    name: '', valueAmount: 0, currencyCode: 'USD', categoryId: null,
    isActive: true, referenceOnly: false, calculateOnCheckout: false, codeRequired: true,
  })
  const typeFormErrors = reactive({ name: '', value: '' })

  function resetTypeForm () {
    Object.assign(typeForm, {
      name: '', valueAmount: 0, currencyCode: 'USD', categoryId: null,
      isActive: true, referenceOnly: false, calculateOnCheckout: false, codeRequired: true,
    })
    Object.assign(typeFormErrors, { name: '', value: '' })
  }

  function openCreateType () {
    editingTypeId.value = null
    resetTypeForm()
    typeDialog.value = true
  }

  function openEditType (item) {
    if (!canEdit.value) return
    editingTypeId.value = item.id
    Object.assign(typeForm, {
      name: item.name,
      valueAmount: (item.value_cents ?? 0) / 100,
      currencyCode: item.currency_code,
      categoryId: item.category_id ?? null,
      isActive: item.is_active,
      referenceOnly: item.reference_only,
      calculateOnCheckout: item.calculate_on_checkout,
      codeRequired: !item.code_optional,
    })
    typeDialog.value = true
  }

  function closeTypeDialog () {
    typeDialog.value = false
    editingTypeId.value = null
    resetTypeForm()
  }

  function validateType () {
    typeFormErrors.name = typeForm.name?.trim().length >= 1 ? '' : 'Name is required.'
    typeFormErrors.value = typeForm.valueAmount > 0 ? '' : 'Value must be greater than zero.'
    return !typeFormErrors.name && !typeFormErrors.value
  }

  async function submitType () {
    if (!validateType()) return
    typeSubmitting.value = true
    const payload = {
      name: typeForm.name.trim(),
      value_cents: Math.round(typeForm.valueAmount * 100),
      currency_code: typeForm.currencyCode.trim().toUpperCase(),
      is_active: typeForm.isActive,
      reference_only: typeForm.referenceOnly,
      calculate_on_checkout: typeForm.calculateOnCheckout,
      code_optional: !typeForm.codeRequired,
      category_id: typeForm.categoryId ?? 0,
    }
    try {
      await (editingTypeId.value ? updateCouponType(editingTypeId.value, payload) : createCouponType(payload))
      closeTypeDialog()
    } catch (error) {
      notify.error(error.message)
    } finally {
      typeSubmitting.value = false
    }
  }

  async function removeType (item) {
    if (!confirm(`Delete coupon type "${item.name}"? Types still used by a coupon can't be deleted.`)) return
    try {
      await deleteCouponType(item.id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- coupon lists --------------------------------------------------------
  const listDialog = ref(false)
  const listSubmitting = ref(false)
  const generatedCodes = ref(null)
  const listForm = reactive({ name: '', typeId: null, count: 100, validUntil: '' })
  const listFormErrors = reactive({ name: '', typeId: '', count: '' })

  function resetListForm () {
    Object.assign(listForm, { name: '', typeId: null, count: 100, validUntil: '' })
    Object.assign(listFormErrors, { name: '', typeId: '', count: '' })
  }

  function openCreateList () {
    resetListForm()
    generatedCodes.value = null
    listDialog.value = true
  }

  function closeListDialog () {
    listDialog.value = false
    generatedCodes.value = null
    resetListForm()
  }

  function validateList () {
    listFormErrors.name = listForm.name?.trim().length >= 1 ? '' : 'Campaign name is required.'
    listFormErrors.typeId = listForm.typeId ? '' : 'Coupon type is required.'
    listFormErrors.count = listForm.count > 0 ? '' : 'Enter how many codes to generate.'
    return !listFormErrors.name && !listFormErrors.typeId && !listFormErrors.count
  }

  async function submitList () {
    if (!validateList()) return
    listSubmitting.value = true
    try {
      const result = await createCouponList({
        name: listForm.name.trim(),
        coupon_type_id: listForm.typeId,
        count: listForm.count,
        valid_until: listForm.validUntil ? new Date(listForm.validUntil).toISOString() : '',
      })
      generatedCodes.value = result.codes
      if (result.generated < result.requested) {
        notify.error(`Only generated ${result.generated} of ${result.requested} codes -- retry to top up the rest.`)
      }
    } catch (error) {
      notify.error(error.message)
    } finally {
      listSubmitting.value = false
    }
  }

  async function closeList (item) {
    if (!confirm(`Close campaign "${item.name}"? Every still-active code it minted will be deactivated too.`)) return
    try {
      await deactivateCouponList(item.id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function copyGeneratedCodes () {
    if (!generatedCodes.value) return
    try {
      await navigator.clipboard.writeText(generatedCodes.value.join('\n'))
      notify.success('Codes copied to clipboard.')
    } catch {
      // Clipboard access can be denied; the textarea above still lets
      // someone select-and-copy manually.
    }
  }
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.coupons-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.coupons-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.coupons-tabs {
  flex: 0 0 auto;
}

.coupons-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}

.coupons-tab-layout {
  display: flex;
  height: 100%;
}

.coupons-tab-content {
  flex: 1 1 auto;
  min-width: 0;
}

.table-search {
  flex: 0 1 320px;
}
</style>
