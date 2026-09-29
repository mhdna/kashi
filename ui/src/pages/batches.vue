<template>
  <div class="page-root">
    <v-card class="batches-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-package-variant-closed" />
        <span>Batches</span>
      </v-card-title>
      <v-divider />

      <div class="batches-layout">
        <!-- ------------------------------------------------------ SIDEBAR -->
        <div class="batches-sidebar">
          <div class="pa-3">
            <v-text-field
              v-model="search"
              clearable
              density="compact"
              hide-details
              label="Search batches"
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
              @update:model-value="onSearch"
            />
          </div>
          <v-divider />

          <v-list v-if="batches.length > 0" class="batches-list" density="compact" nav>
            <v-list-item
              v-for="b in batches"
              :key="b.id"
              :active="b.id === selectedId"
              active-color="primary"
              @click="selectBatch(b.id)"
            >
              <v-list-item-title class="font-weight-medium">
                {{ b.code || `Batch #${b.id}` }}
              </v-list-item-title>
              <v-list-item-subtitle>
                {{ nullableString(b.supplier_name) || nullableString(b.purchase_code) || '—' }}
              </v-list-item-subtitle>
              <template #append>
                <v-chip size="x-small" variant="tonal">{{ b.item_count }} SKU{{ b.item_count === 1 ? '' : 's' }}</v-chip>
              </template>
            </v-list-item>
          </v-list>
          <div v-else class="pa-4 text-medium-emphasis text-body-2">
            {{ listLoading ? 'Loading…' : 'No batches yet.' }}
          </div>

          <v-spacer />
          <v-divider />
          <div class="pa-3">
            <v-btn
              v-if="canEdit"
              block
              color="primary"
              prepend-icon="mdi-plus"
              text="Add Batch"
              variant="flat"
              @click="openCreate"
            />
          </div>
        </div>

        <v-divider vertical />

        <!-- ------------------------------------------------------ CONTENT -->
        <div class="batches-content">
          <div v-if="!selectedId" class="empty-state">
            <v-icon color="medium-emphasis" icon="mdi-package-variant-closed" size="48" />
            <div class="text-medium-emphasis mt-2">Select a batch, or add one from a purchase invoice.</div>
          </div>

          <template v-else>
            <div class="d-flex align-center ga-3 px-4 py-3">
              <div>
                <div class="text-h6">{{ selectedBatch?.code || `Batch #${selectedId}` }}</div>
                <div class="text-caption text-medium-emphasis">
                  <span v-if="nullableString(selectedBatch?.purchase_code)">
                    From purchase {{ nullableString(selectedBatch?.purchase_code) }}
                  </span>
                  <span v-if="nullableString(selectedBatch?.supplier_name)"> — {{ nullableString(selectedBatch?.supplier_name) }}</span>
                  <span v-if="nullableString(selectedBatch?.inventory_name)"> — {{ nullableString(selectedBatch?.inventory_name) }}</span>
                </div>
              </div>
              <v-spacer />
              <v-btn
                v-if="canEdit"
                color="error"
                icon="mdi-delete"
                size="small"
                variant="text"
                @click="confirmDelete"
              />
            </div>
            <v-divider />

            <div v-if="detailLoading" class="pa-4 text-medium-emphasis">Loading…</div>
            <template v-else>
              <!-- summary strip -->
              <div class="stat-strip px-4 py-3">
                <div class="stat-tile">
                  <div class="text-caption text-medium-emphasis">Bought</div>
                  <div class="text-h6">{{ totals.bought }}</div>
                </div>
                <div class="stat-tile">
                  <div class="text-caption text-medium-emphasis">Sold</div>
                  <div class="text-h6">{{ totals.sold }}</div>
                </div>
                <div class="stat-tile">
                  <div class="text-caption text-medium-emphasis">Total Cost</div>
                  <div class="text-h6">${{ formatMoney(totals.cost) }}</div>
                </div>
                <div class="stat-tile">
                  <div class="text-caption text-medium-emphasis">Revenue</div>
                  <div class="text-h6 text-success">${{ formatMoney(totals.revenue) }}</div>
                </div>
                <div class="stat-tile">
                  <div class="text-caption text-medium-emphasis">Profit</div>
                  <div class="text-h6" :class="totals.profit >= 0 ? 'text-success' : 'text-error'">
                    ${{ formatMoney(totals.profit) }}
                  </div>
                </div>
              </div>
              <v-divider />

              <v-tabs v-model="tab" color="primary">
                <v-tab prepend-icon="mdi-format-list-bulleted" text="Products" value="products" />
                <v-tab prepend-icon="mdi-chart-bar" text="Insights" value="insights" />
              </v-tabs>
              <v-divider />

              <div class="tab-body">
                <!-- ------------------------------------------------ PRODUCTS -->
                <div v-show="tab === 'products'" class="pa-4">
                  <v-table class="products-table" density="comfortable">
                    <thead>
                      <tr>
                        <th>Product</th>
                        <th>Barcode</th>
                        <th class="text-end">Bought</th>
                        <th class="text-end">Sold</th>
                        <th class="text-end">Remaining</th>
                        <th class="text-end">Unit Cost</th>
                        <th class="text-end">Total Cost</th>
                        <th class="text-end">Revenue</th>
                        <th class="text-end">Profit</th>
                        <th>Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="it in sortedItems" :key="it.batch_item_id">
                        <td>{{ it.product_code }} — {{ it.product_name }}</td>
                        <td class="text-medium-emphasis">{{ it.variant_barcode || '—' }}</td>
                        <td class="text-end">{{ it.quantity_bought }}</td>
                        <td class="text-end">{{ it.quantity_sold }}</td>
                        <td class="text-end">{{ it.remaining_quantity }}</td>
                        <td class="text-end">${{ formatMoney(it.unit_cost) }}</td>
                        <td class="text-end">${{ formatMoney(it.quantity_bought * it.unit_cost) }}</td>
                        <td class="text-end text-success">${{ formatMoney(it.revenue) }}</td>
                        <td class="text-end" :class="itemProfit(it) >= 0 ? 'text-success' : 'text-error'">
                          ${{ formatMoney(itemProfit(it)) }}
                        </td>
                        <td>
                          <v-chip v-if="itemStatus(it)" :color="itemStatus(it).color" size="small" variant="tonal">
                            {{ itemStatus(it).label }}
                          </v-chip>
                        </td>
                      </tr>
                    </tbody>
                  </v-table>
                </div>

                <!-- ------------------------------------------------ INSIGHTS -->
                <div v-show="tab === 'insights'" class="pa-4">
                  <div class="insight-cards">
                    <v-sheet class="insight-card" color="surface-light" rounded="lg">
                      <div class="d-flex align-center ga-2 mb-1">
                        <v-icon color="success" icon="mdi-trending-up" size="18" />
                        <span class="text-caption text-medium-emphasis">Top Revenue</span>
                      </div>
                      <template v-if="topRevenue">
                        <div class="font-weight-medium">{{ topRevenue.product_name }}</div>
                        <div class="text-success">${{ formatMoney(topRevenue.revenue) }}</div>
                      </template>
                      <div v-else class="text-medium-emphasis">—</div>
                    </v-sheet>

                    <v-sheet class="insight-card" color="surface-light" rounded="lg">
                      <div class="d-flex align-center ga-2 mb-1">
                        <v-icon color="primary" icon="mdi-trophy" size="18" />
                        <span class="text-caption text-medium-emphasis">Best Seller</span>
                      </div>
                      <template v-if="bestSeller">
                        <div class="font-weight-medium">{{ bestSeller.product_name }}</div>
                        <div>{{ bestSeller.quantity_sold }} sold</div>
                      </template>
                      <div v-else class="text-medium-emphasis">—</div>
                    </v-sheet>
                  </div>

                  <v-row class="mt-2">
                    <v-col cols="12" md="4">
                      <div class="d-flex align-center ga-2 mb-2">
                        <v-icon color="grey" icon="mdi-skull-outline" size="18" />
                        <span class="text-subtitle-2">Dead Stock ({{ deadStock.length }})</span>
                      </div>
                      <v-list v-if="deadStock.length > 0" density="compact">
                        <v-list-item v-for="it in deadStock" :key="it.batch_item_id" :subtitle="`${it.quantity_bought} bought, never sold`" :title="it.product_name" />
                      </v-list>
                      <div v-else class="text-medium-emphasis text-body-2">Nothing dead in this batch.</div>
                    </v-col>

                    <v-col cols="12" md="4">
                      <div class="d-flex align-center ga-2 mb-2">
                        <v-icon color="warning" icon="mdi-alert-circle-outline" size="18" />
                        <span class="text-subtitle-2">Low Stock ({{ lowStock.length }})</span>
                      </div>
                      <v-list v-if="lowStock.length > 0" density="compact">
                        <v-list-item v-for="it in lowStock" :key="it.batch_item_id" :subtitle="`${it.remaining_quantity} left`" :title="it.product_name" />
                      </v-list>
                      <div v-else class="text-medium-emphasis text-body-2">Nothing low.</div>
                    </v-col>

                    <v-col cols="12" md="4">
                      <div class="d-flex align-center ga-2 mb-2">
                        <v-icon color="error" icon="mdi-close-circle-outline" size="18" />
                        <span class="text-subtitle-2">Out of Stock ({{ outOfStock.length }})</span>
                      </div>
                      <v-list v-if="outOfStock.length > 0" density="compact">
                        <v-list-item v-for="it in outOfStock" :key="it.batch_item_id" :subtitle="`${it.quantity_sold} sold total`" :title="it.product_name" />
                      </v-list>
                      <div v-else class="text-medium-emphasis text-body-2">Nothing out.</div>
                    </v-col>
                  </v-row>
                </div>
              </div>
            </template>
          </template>
        </div>
      </div>
    </v-card>

    <!-- add batch -->
    <v-dialog v-model="createDialog" max-width="480">
      <v-card>
        <v-card-title class="text-h6">Add Batch</v-card-title>
        <v-card-text>
          <v-autocomplete
            v-model="form.purchaseId"
            density="compact"
            hide-no-data
            :item-title="purchaseLabel"
            :item-value="p => p.id"
            :items="purchaseOptions"
            :loading="purchaseLoading"
            no-filter
            placeholder="Search purchase invoices"
            variant="outlined"
            @update:search="onPurchaseSearch"
          />
          <v-text-field
            v-model="form.code"
            class="mt-3"
            density="compact"
            hide-details="auto"
            label="Batch code (optional)"
            variant="outlined"
          />
          <v-textarea
            v-model="form.note"
            class="mt-3"
            density="compact"
            hide-details="auto"
            label="Note (optional)"
            rows="2"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="createDialog = false" />
          <v-btn
            color="primary"
            :loading="creating"
            text="Create"
            variant="flat"
            @click="submitCreate"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- delete -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card>
        <v-card-title class="text-h6">Delete Batch</v-card-title>
        <v-card-text>
          Delete <strong>{{ selectedBatch?.code || `Batch #${selectedId}` }}</strong>? This can't be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="deleteDialog = false" />
          <v-btn
            color="error"
            :loading="deleting"
            text="Delete"
            variant="flat"
            @click="doDelete"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { useBatches } from '@/composables/useBatches'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useNotify } from '@/composables/useNotify'
  import { formatMoney } from '@/utils/money'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const { searchBatches, searchPurchases, createBatch, fetchBatch, deleteBatch } = useBatches()

  function nullableString (value) {
    if (typeof value === 'string') return value
    if (!value || typeof value !== 'object') return ''
    const valid = value.Valid ?? value.valid
    if (valid === false) return ''
    return value.String ?? value.string ?? ''
  }

  // ---- left list ---------------------------------------------------
  const batches = ref([])
  const listLoading = ref(false)
  const notify = useNotify()
  const search = ref('')
  let searchTimer = null

  async function reloadBatches () {
    listLoading.value = true
    try {
      const data = await searchBatches(search.value)
      batches.value = data?.batches ?? []
    } catch (error) {
      notify.error(error.message)
    } finally {
      listLoading.value = false
    }
  }

  function onSearch () {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(reloadBatches, 250)
  }

  onMounted(reloadBatches)

  // ---- selection / detail -------------------------------------------
  const selectedId = ref(null)
  const selectedBatch = ref(null)
  const items = ref([])
  const detailLoading = ref(false)
  const tab = ref('products')

  async function selectBatch (id) {
    selectedId.value = id
    tab.value = 'products'
    detailLoading.value = true
    try {
      const data = await fetchBatch(id)
      selectedBatch.value = data?.batch ?? null
      items.value = data?.items ?? []
    } catch (error) {
      notify.error(error.message)
    } finally {
      detailLoading.value = false
    }
  }

  const sortedItems = computed(() => items.value.toSorted((a, b) => b.revenue - a.revenue))

  const totals = computed(() => {
    let bought = 0, sold = 0, cost = 0, revenue = 0
    for (const it of items.value) {
      bought += it.quantity_bought
      sold += it.quantity_sold
      cost += it.quantity_bought * it.unit_cost
      revenue += it.revenue
    }
    const cogs = items.value.reduce((sum, it) => sum + it.quantity_sold * it.unit_cost, 0)
    return { bought, sold, cost, revenue, profit: revenue - cogs }
  })

  function itemProfit (it) {
    return it.revenue - it.quantity_sold * it.unit_cost
  }

  function itemStatus (it) {
    if (it.remaining_quantity <= 0) return { label: 'Out of stock', color: 'error' }
    if (it.remaining_quantity <= it.low_stock_threshold) return { label: 'Low stock', color: 'warning' }
    if (it.quantity_sold === 0) return { label: 'Dead stock', color: 'grey' }
    return null
  }

  const topRevenue = computed(() => sortedItems.value[0] ?? null)
  const bestSeller = computed(() => {
    if (items.value.length === 0) return null
    return items.value.toSorted((a, b) => b.quantity_sold - a.quantity_sold)[0]
  })
  const deadStock = computed(() => items.value.filter(it => it.quantity_sold === 0))
  const lowStock = computed(() => items.value.filter(it => it.remaining_quantity > 0 && it.remaining_quantity <= it.low_stock_threshold))
  const outOfStock = computed(() => items.value.filter(it => it.remaining_quantity <= 0))

  // ---- add batch ------------------------------------------------------
  const createDialog = ref(false)
  const creating = ref(false)
  const form = reactive({ purchaseId: null, code: '', note: '' })
  const purchaseOptions = ref([])
  const purchaseLoading = ref(false)
  let purchaseSearchTimer = null

  function purchaseLabel (p) {
    const supplier = nullableString(p.supplier_name)
    return supplier ? `${p.code} — ${supplier}` : p.code
  }

  async function runPurchaseSearch (q) {
    purchaseLoading.value = true
    try {
      const data = await searchPurchases(q || '')
      purchaseOptions.value = data?.purchases ?? []
    } finally {
      purchaseLoading.value = false
    }
  }

  function onPurchaseSearch (q) {
    clearTimeout(purchaseSearchTimer)
    purchaseSearchTimer = setTimeout(() => runPurchaseSearch(q), 250)
  }

  function openCreate () {
    Object.assign(form, { purchaseId: null, code: '', note: '' })
    createDialog.value = true
    runPurchaseSearch('')
  }

  async function submitCreate () {
    if (!form.purchaseId) {
      notify.error('Select a purchase invoice.')
      return
    }
    creating.value = true
    try {
      const created = await createBatch({ purchaseId: form.purchaseId, code: form.code.trim(), note: form.note.trim() })
      createDialog.value = false
      await reloadBatches()
      const newId = created?.batch?.id
      if (newId) await selectBatch(newId)
    } catch (error) {
      notify.error(error.message)
    } finally {
      creating.value = false
    }
  }

  // ---- delete -----------------------------------------------------
  const deleteDialog = ref(false)
  const deleting = ref(false)

  function confirmDelete () {
    deleteDialog.value = true
  }

  async function doDelete () {
    deleting.value = true
    try {
      await deleteBatch(selectedId.value)
      deleteDialog.value = false
      selectedId.value = null
      selectedBatch.value = null
      items.value = []
      await reloadBatches()
    } catch (error) {
      notify.error(error.message)
    } finally {
      deleting.value = false
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
.batches-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.batches-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.batches-sidebar {
  flex: 0 0 300px;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.batches-list {
  flex: 1 1 auto;
  overflow-y: auto;
  min-height: 0;
}
.batches-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}
.empty-state {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.tab-body {
  flex: 1 1 auto;
  overflow-y: auto;
}
.stat-strip {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
}
.stat-tile {
  min-width: 110px;
}
.insight-cards {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}
.insight-card {
  flex: 1 1 220px;
  padding: 12px 16px;
}
</style>
