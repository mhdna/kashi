<template>
  <div class="page-root">
    <v-card class="health-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-heart-pulse" />
        <span>Stock Health</span>
        <v-chip :color="statusColor" size="small" variant="flat">
          <v-icon icon="mdi-circle" size="10" start />
          {{ statusLabel }}
        </v-chip>
        <v-spacer />
        <v-btn icon="mdi-refresh" :loading="loading" variant="text" @click="load" />
      </v-card-title>
      <v-divider />

      <div class="health-content">
        <div class="pa-4 health-body">
          <PreparingState v-if="loading && !loaded" kind="chart" title="Checking stock health" />
          <template v-else>
            <v-row class="mb-2 flex-grow-0" density="compact">
              <v-col cols="12" sm="6">
                <v-card class="summary-card" variant="tonal">
                  <v-card-item>
                    <template #prepend>
                      <v-avatar color="primary" variant="tonal">
                        <v-icon icon="mdi-calendar-clock-outline" />
                      </v-avatar>
                    </template>
                    <v-card-title class="text-h4">{{ daysLabel }}</v-card-title>
                    <v-card-subtitle>Days of Inventory Left</v-card-subtitle>
                  </v-card-item>
                </v-card>
              </v-col>
              <v-col cols="12" sm="6">
                <v-card class="summary-card" variant="tonal">
                  <v-card-item>
                    <template #prepend>
                      <v-avatar color="secondary" variant="tonal">
                        <v-icon icon="mdi-cash-multiple" />
                      </v-avatar>
                    </template>
                    <v-card-title class="text-h4">{{ formatMoney(health.inventory_value) }}</v-card-title>
                    <v-card-subtitle>Inventory Value (cost)</v-card-subtitle>
                  </v-card-item>
                </v-card>
              </v-col>
            </v-row>

            <v-row class="health-columns mt-2" density="compact">
              <v-col cols="12" md="6">
                <v-card class="slow-mover-card d-flex flex-column" variant="flat">
                  <div class="d-flex flex-wrap align-center ga-2 px-4 pt-3 pb-2">
                    <v-icon color="error" icon="mdi-skull-outline" size="24" />
                    <span class="text-h6 font-weight-medium">Dead Stock</span>
                    <span class="text-subtitle-1 text-medium-emphasis">· {{ deadCount }}</span>
                    <v-spacer />
                    <v-text-field
                      v-model="deadSearch"
                      class="slow-mover-search"
                      clearable
                      density="compact"
                      hide-details
                      label="Search products"
                      prepend-inner-icon="mdi-magnify"
                      variant="outlined"
                    />
                  </div>
                  <v-divider />
                  <v-virtual-scroll
                    v-if="deadFiltered.length > 0"
                    class="slow-mover-scroll"
                    :item-height="64"
                    :items="deadFiltered"
                  >
                    <template #default="{ item }">
                      <v-list-item class="px-4 py-2">
                        <template #prepend>
                          <v-avatar
                            class="me-3"
                            color="error"
                            rounded="lg"
                            size="40"
                            variant="tonal"
                          >
                            <v-icon color="error" icon="mdi-skull-outline" size="22" />
                          </v-avatar>
                        </template>
                        <v-list-item-title class="font-weight-medium">{{ item.name }}</v-list-item-title>
                        <v-list-item-subtitle>{{ item.code }} · On hand: {{ item.on_hand }}</v-list-item-subtitle>
                        <template #append>
                          <span class="text-caption text-medium-emphasis">Never sold</span>
                        </template>
                      </v-list-item>
                      <v-divider />
                    </template>
                  </v-virtual-scroll>
                  <div v-else class="text-medium-emphasis text-center pa-6">
                    {{ deadSearch ? 'No matches.' : 'No dead stock -- everything active is selling.' }}
                  </div>
                </v-card>
              </v-col>
              <v-col cols="12" md="6">
                <v-card class="slow-mover-card d-flex flex-column" variant="flat">
                  <div class="d-flex flex-wrap align-center ga-2 px-4 pt-3 pb-2">
                    <v-icon color="warning" icon="mdi-tray-full" size="24" />
                    <span class="text-h6 font-weight-medium">Overstocked</span>
                    <span class="text-subtitle-1 text-medium-emphasis">· {{ overCount }}</span>
                    <v-spacer />
                    <v-text-field
                      v-model="overSearch"
                      class="slow-mover-search"
                      clearable
                      density="compact"
                      hide-details
                      label="Search products"
                      prepend-inner-icon="mdi-magnify"
                      variant="outlined"
                    />
                  </div>
                  <v-divider />
                  <v-virtual-scroll
                    v-if="overFiltered.length > 0"
                    class="slow-mover-scroll"
                    :item-height="64"
                    :items="overFiltered"
                  >
                    <template #default="{ item }">
                      <v-list-item class="px-4 py-2">
                        <template #prepend>
                          <v-avatar
                            class="me-3"
                            color="warning"
                            rounded="lg"
                            size="40"
                            variant="tonal"
                          >
                            <v-icon color="warning" icon="mdi-tray-full" size="22" />
                          </v-avatar>
                        </template>
                        <v-list-item-title class="font-weight-medium">{{ item.name }}</v-list-item-title>
                        <v-list-item-subtitle>{{ item.code }} · On hand: {{ item.on_hand }} · Sold 90d: {{ item.units_sold_90d }}</v-list-item-subtitle>
                        <template #append>
                          <span class="text-caption text-medium-emphasis">{{ item.days_of_inventory }}d supply</span>
                        </template>
                      </v-list-item>
                      <v-divider />
                    </template>
                  </v-virtual-scroll>
                  <div v-else class="text-medium-emphasis text-center pa-6">
                    {{ overSearch ? 'No matches.' : 'Nothing overstocked right now.' }}
                  </div>
                </v-card>
              </v-col>
            </v-row>
          </template>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useStockHealth } from '@/composables/useStockHealth'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  const { fetchStockHealth } = useStockHealth()

  const health = ref({})
  const loading = ref(false)
  const loaded = ref(false)
  const notify = useNotify()

  // The two slow-mover columns: fetched once as a capped list and rendered
  // with v-virtual-scroll, so a large catalog stays smooth without paging
  // controls -- this is a "everything at a glance" view, not a browsable table.
  const SLOW_MOVER_CAP = 300
  // The backend caps page_size at 100, so walk pages until we hit the
  // display cap or run out of rows.
  const SLOW_MOVER_PAGE_SIZE = 100
  const deadStock = ref([])
  const overstocked = ref([])

  async function fetchSlowMovers (category) {
    const out = []
    // page_id is a row offset, not a page index.
    for (let offset = 0; out.length < SLOW_MOVER_CAP; offset += SLOW_MOVER_PAGE_SIZE) {
      const params = new URLSearchParams({
        category,
        page_size: String(SLOW_MOVER_PAGE_SIZE),
        page_id: String(offset),
      })
      const res = await authFetch(`${API_BASE}/stock_health/slow_movers?${params}`)
      if (!res.ok) throw new Error(`Request failed with status ${res.status}`)
      const data = await res.json().catch(() => null)
      const products = data?.products ?? []
      out.push(...products)
      if (products.length < SLOW_MOVER_PAGE_SIZE) break
    }
    return out.slice(0, SLOW_MOVER_CAP)
  }

  async function load () {
    loading.value = true
    try {
      const [healthData, dead, over] = await Promise.all([
        fetchStockHealth(),
        fetchSlowMovers('dead'),
        fetchSlowMovers('overstocked'),
      ])
      health.value = healthData ?? {}
      deadStock.value = dead
      overstocked.value = over
      loaded.value = true
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  const statusColor = computed(() => ({ healthy: 'success', warning: 'warning', critical: 'error' }[health.value.status] || 'grey'))
  const statusLabel = computed(() => ({ healthy: 'Healthy', warning: 'Needs Attention', critical: 'Critical' }[health.value.status] || '—'))

  const daysLabel = computed(() => health.value.days_of_inventory ?? '—')

  // Client-side filter over the already-loaded (capped) lists -- keeps the
  // columns snappy without another round-trip per keystroke.
  const deadSearch = ref('')
  const overSearch = ref('')

  function matches (item, query) {
    const q = query.trim().toLowerCase()
    if (!q) return true
    return (item.name ?? '').toLowerCase().includes(q) || (item.code ?? '').toLowerCase().includes(q)
  }

  const deadFiltered = computed(() => deadStock.value.filter(i => matches(i, deadSearch.value)))
  const overFiltered = computed(() => overstocked.value.filter(i => matches(i, overSearch.value)))
  const deadCount = computed(() =>
    deadSearch.value.trim() ? deadFiltered.value.length : (health.value.dead_stock_count ?? deadStock.value.length),
  )
  const overCount = computed(() =>
    overSearch.value.trim() ? overFiltered.value.length : (health.value.overstocked_count ?? overstocked.value.length),
  )

  function formatMoney (cents) {
    return (Number(cents ?? 0) / 100).toLocaleString(undefined, { style: 'currency', currency: 'USD' })
  }

  onMounted(load)
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.health-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.health-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}
/* Let the two slow-mover columns grow into the space left below the KPI
   cards; each caps itself with a max-height. Mirrors the alerts page. */
.health-body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.health-columns {
  flex: 1 1 auto;
  min-height: 0;
}
.health-columns > :deep(.v-col-12) {
  display: flex;
  min-height: 0;
}
.slow-mover-card {
  flex: 1 1 auto;
  min-height: 0;
}
.slow-mover-search {
  flex: 0 1 220px;
}
.summary-card :deep(.v-card-item) {
  padding: 12px 16px;
}
.slow-mover-scroll {
  flex: 1 1 auto;
  min-height: 280px;
  max-height: 800px;
}
</style>
