<template>
  <div class="inventory-analysis d-flex flex-column fill-height" style="overflow-y: auto">
    <!-- Header / controls -->
    <div class="d-flex align-center flex-wrap ga-3 pa-4 pb-2">
      <v-icon icon="mdi-warehouse" />
      <div class="text-h6 font-weight-medium">Inventory Analysis</div>
      <v-chip
        v-if="overview"
        class="ml-1"
        :color="statusColor"
        size="small"
        variant="flat"
      >
        <v-icon icon="mdi-circle" size="10" start />
        {{ statusLabel }}
      </v-chip>
      <v-spacer />
      <v-select
        v-model="selectedLocationId"
        clearable
        density="compact"
        hide-details
        :items="locationOptions"
        label="Branch / Location"
        style="min-width: 220px; max-width: 300px"
        variant="outlined"
      />
      <v-btn icon="mdi-refresh" :loading="loading" variant="text" @click="load" />
    </div>

    <v-tabs v-model="tab" class="px-4" color="primary">
      <v-tab value="overview">Overview</v-tab>
      <v-tab value="locations">By Branch / Location</v-tab>
      <v-tab value="low-stock">Low Stock</v-tab>
      <v-tab value="slow-movers">Slow Movers</v-tab>
      <v-tab value="movements">Movements</v-tab>
    </v-tabs>
    <v-divider />

    <div v-if="loading" class="d-flex justify-center pa-12">
      <v-progress-circular color="primary" indeterminate />
    </div>

    <v-window v-else v-model="tab" class="flex-grow-1" style="overflow-y: auto">
      <!-- Overview ------------------------------------------------------ -->
      <v-window-item value="overview">
        <v-container class="pa-4 pt-2" fluid>
          <v-alert
            v-if="selectedLocation"
            class="mb-3"
            density="compact"
            type="info"
            variant="tonal"
          >
            Showing figures for <strong>{{ selectedLocation.name }}</strong> only. Clear the location filter for company-wide totals.
          </v-alert>

          <v-row dense>
            <v-col v-for="kpi in kpis" :key="kpi.label" cols="6" md="2">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium text-truncate">{{ kpi.value }}</div>
                <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
              </v-card>
            </v-col>
          </v-row>

          <v-row dense>
            <v-col cols="12" md="6">
              <v-card class="chart-card" variant="flat">
                <div class="chart-card__title">Inventory value by location</div>
                <v-chart autoresize class="chart" :option="valueByLocationOption" />
              </v-card>
            </v-col>
            <v-col cols="12" md="6">
              <v-card class="chart-card" variant="flat">
                <div class="chart-card__title">Days of inventory by location</div>
                <div class="chart-card__subtitle">Lower is faster-moving; locations with no sales in 30 days aren't shown</div>
                <v-chart autoresize class="chart" :option="daysByLocationOption" />
              </v-card>
            </v-col>
          </v-row>
        </v-container>
      </v-window-item>

      <!-- By Branch / Location ------------------------------------------ -->
      <v-window-item value="locations">
        <v-container class="pa-4 pt-2" fluid>
          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-center flex-wrap ga-3 mb-2">
              <div class="chart-card__title mb-0">Stock by branch / location</div>
              <v-btn-toggle v-model="locationTypeFilter" density="compact" mandatory variant="outlined">
                <v-btn size="small" value="all">All</v-btn>
                <v-btn size="small" value="store">Branches</v-btn>
                <v-btn size="small" value="warehouse">Warehouses</v-btn>
              </v-btn-toggle>
              <v-spacer />
              <v-text-field
                v-model="locationSearch"
                clearable
                density="compact"
                hide-details
                prepend-inner-icon="mdi-magnify"
                style="max-width: 240px"
                variant="outlined"
              />
            </div>
            <v-data-table
              class="ia-table"
              :headers="locationHeaders"
              :items="filteredLocations"
              :items-per-page="15"
              :search="locationSearch"
              :sort-by="[{ key: 'inventory_value', order: 'desc' }]"
            >
              <template #item.type="{ item }">
                <v-chip size="small" variant="tonal">{{ item.type }}</v-chip>
              </template>
              <template #item.total_units="{ item }">{{ item.total_units.toLocaleString() }}</template>
              <template #item.inventory_value="{ item }">{{ formatMoney(item.inventory_value) }}</template>
              <template #item.units_sold_30d="{ item }">{{ item.units_sold_30d.toLocaleString() }}</template>
              <template #item.days_of_inventory="{ item }">
                <span v-if="item.days_of_inventory != null">{{ item.days_of_inventory }}</span>
                <span v-else class="text-disabled">—</span>
              </template>
            </v-data-table>
          </v-card>
        </v-container>
      </v-window-item>

      <!-- Low Stock ------------------------------------------------------ -->
      <v-window-item value="low-stock">
        <v-container class="pa-4 pt-2" fluid>
          <div class="text-caption text-medium-emphasis mb-2">
            Total on-hand across every location, company-wide — not filterable by branch.
          </div>
          <v-row dense>
            <v-col cols="6" md="3">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium">{{ (lowStockSummary?.low_stock_count ?? 0).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis mt-1">Low stock</div>
              </v-card>
            </v-col>
            <v-col cols="6" md="3">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium">{{ (lowStockSummary?.out_of_stock_count ?? 0).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis mt-1">Out of stock</div>
              </v-card>
            </v-col>
            <v-col cols="6" md="3">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium">{{ (lowStockSummary?.total_products ?? 0).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis mt-1">Active products</div>
              </v-card>
            </v-col>
          </v-row>

          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-center flex-wrap ga-3 mb-2">
              <div class="chart-card__title mb-0">Products at or below threshold</div>
              <v-btn-toggle v-model="lowStockStatusFilter" density="compact" mandatory variant="outlined">
                <v-btn size="small" value="">All</v-btn>
                <v-btn size="small" value="low_stock">Low</v-btn>
                <v-btn size="small" value="out_of_stock">Out</v-btn>
              </v-btn-toggle>
              <v-spacer />
              <v-text-field
                v-model="lowStockSearch"
                clearable
                density="compact"
                hide-details
                prepend-inner-icon="mdi-magnify"
                style="max-width: 240px"
                variant="outlined"
              />
            </div>
            <ServerSideTable
              :api-u-r-l="lowStockApiURL"
              density="compact"
              :external-search="lowStockSearch"
              flush
              :headers="lowStockHeaders"
              hover
              :query-params="{ status: lowStockStatusFilter }"
              root-key="products"
              :show-search-icon="false"
            >
              <template #item.on_hand="{ item }">
                <span :class="item.on_hand <= 0 ? 'text-error font-weight-medium' : ''">{{ item.on_hand.toLocaleString() }}</span>
              </template>
              <template #item.effective_threshold="{ item }">{{ item.effective_threshold.toLocaleString() }}</template>
            </ServerSideTable>
          </v-card>
        </v-container>
      </v-window-item>

      <!-- Slow Movers ------------------------------------------------------ -->
      <v-window-item value="slow-movers">
        <v-container class="pa-4 pt-2" fluid>
          <div class="text-caption text-medium-emphasis mb-2">
            Carrying stock company-wide — not filterable by branch.
          </div>
          <v-row dense>
            <v-col cols="6" md="3">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium">{{ (overview?.dead_stock_count ?? 0).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis mt-1">Dead stock (no sales in 90d)</div>
              </v-card>
            </v-col>
            <v-col cols="6" md="3">
              <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
                <div class="text-h6 font-weight-medium">{{ (overview?.overstocked_count ?? 0).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis mt-1">Overstocked (&gt;90 days supply)</div>
              </v-card>
            </v-col>
          </v-row>

          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-center flex-wrap ga-3 mb-2">
              <div class="chart-card__title mb-0">Slow-moving products</div>
              <v-btn-toggle v-model="slowMoverCategoryFilter" density="compact" mandatory variant="outlined">
                <v-btn size="small" value="">All</v-btn>
                <v-btn size="small" value="dead">Dead</v-btn>
                <v-btn size="small" value="overstocked">Overstocked</v-btn>
              </v-btn-toggle>
              <v-spacer />
              <v-text-field
                v-model="slowMoverSearch"
                clearable
                density="compact"
                hide-details
                prepend-inner-icon="mdi-magnify"
                style="max-width: 240px"
                variant="outlined"
              />
            </div>
            <ServerSideTable
              :api-u-r-l="slowMoversApiURL"
              density="compact"
              :external-search="slowMoverSearch"
              flush
              :headers="slowMoverHeaders"
              hover
              :query-params="{ category: slowMoverCategoryFilter }"
              root-key="products"
              :show-search-icon="false"
            >
              <template #item.on_hand="{ item }">{{ item.on_hand.toLocaleString() }}</template>
              <template #item.units_sold_90d="{ item }">{{ item.units_sold_90d.toLocaleString() }}</template>
              <template #item.days_of_inventory="{ item }">
                <span v-if="item.days_of_inventory != null">{{ item.days_of_inventory }}</span>
                <span v-else class="text-disabled">—</span>
              </template>
              <template #item.category="{ item }">
                <v-chip :color="item.category === 'dead' ? 'error' : 'warning'" size="small" variant="tonal">{{ item.category }}</v-chip>
              </template>
            </ServerSideTable>
          </v-card>
        </v-container>
      </v-window-item>

      <!-- Movements -------------------------------------------------------- -->
      <v-window-item value="movements">
        <v-container class="pa-4 pt-2" fluid>
          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-center flex-wrap ga-3 mb-2">
              <div class="chart-card__title mb-0">Stock movement ledger</div>
              <v-select
                v-model="movementReasonFilter"
                clearable
                density="compact"
                hide-details
                :items="reasonOptions"
                label="Reason"
                style="max-width: 170px"
                variant="outlined"
              />
              <v-text-field
                v-model="movementFromDate"
                density="compact"
                hide-details
                label="From"
                style="max-width: 160px"
                type="date"
                variant="outlined"
              />
              <v-text-field
                v-model="movementToDate"
                density="compact"
                hide-details
                label="To"
                style="max-width: 160px"
                type="date"
                variant="outlined"
              />
            </div>
            <ServerSideTable
              :api-u-r-l="movementsApiURL"
              density="compact"
              flush
              :headers="movementHeaders"
              hover
              :query-params="movementQueryParams"
              root-key="movements"
            >
              <template #item.created_at="{ item }">{{ new Date(item.created_at).toLocaleString() }}</template>
              <template #item.reason="{ item }">
                <v-chip size="small" variant="tonal">{{ item.reason.replaceAll('_', ' ') }}</v-chip>
              </template>
              <template #item.quantity="{ item }">
                <span :class="item.quantity < 0 ? 'text-error' : 'text-success'">{{ item.quantity > 0 ? '+' : '' }}{{ item.quantity.toLocaleString() }}</span>
              </template>
              <template #item.change="{ item }">{{ item.quantity_before.toLocaleString() }} → {{ item.quantity_after.toLocaleString() }}</template>
              <template #item.variant="{ item }">
                {{ item.product_name }}
                <span v-if="item.color_name || item.size_name" class="text-caption text-medium-emphasis">
                  ({{ [item.color_name, item.size_name].filter(Boolean).join(' / ') }})
                </span>
              </template>
            </ServerSideTable>
          </v-card>
        </v-container>
      </v-window-item>
    </v-window>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import VChart from 'vue-echarts'
  import { useTheme } from 'vuetify'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useLowStockAlerts } from '@/composables/useLowStockAlerts'
  import { useStockHealth } from '@/composables/useStockHealth'
  import { STOCK_MOVEMENT_REASONS } from '@/composables/useStockMovements'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { formatMoney } from '@/utils/money'

  const theme = useTheme()
  const c = computed(() => theme.current.value.colors)
  const isDark = computed(() => theme.current.value.dark)
  const primary = computed(() => c.value.primary)
  const warning = computed(() => c.value.warning)
  const negative = computed(() => c.value.error)
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))

  const catAxis = computed(() => ({
    axisLine: { lineStyle: { color: axisColor.value } },
    axisTick: { show: false },
    axisLabel: { color: textColor.value, fontSize: 11 },
  }))
  const valAxis = computed(() => ({
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
  }))

  const tab = ref('overview')
  const { fetchStockHealth, fetchStockHealthByInventory } = useStockHealth()

  const loading = ref(false)
  const notify = useNotify()
  const overview = ref(null)
  const locations = ref([])
  const selectedLocationId = ref(null)

  async function load () {
    loading.value = true
    try {
      const [health, byInventory] = await Promise.all([
        fetchStockHealth(),
        fetchStockHealthByInventory(),
      ])
      overview.value = health
      locations.value = byInventory.inventories
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      loading.value = false
    }
  }
  onMounted(() => {
    load()
    loadLowStockSummary()
  })

  const statusColor = computed(() => ({ healthy: 'success', warning: 'warning', critical: 'error' }[overview.value?.status] || 'grey'))
  const statusLabel = computed(() => ({ healthy: 'Healthy', warning: 'Needs Attention', critical: 'Critical' }[overview.value?.status] || '—'))

  const locationOptions = computed(() =>
    locations.value
      .toSorted((a, b) => a.name.localeCompare(b.name))
      .map(l => ({ title: `${l.name} (${l.type})`, value: l.id })),
  )
  const selectedLocation = computed(() => locations.value.find(l => l.id === selectedLocationId.value) ?? null)

  // --- Overview KPIs -------------------------------------------------------
  const kpis = computed(() => {
    const src = selectedLocation.value ?? overview.value
    if (!src) return []
    const daysOfInventory = selectedLocation.value ? selectedLocation.value.days_of_inventory : overview.value.days_of_inventory
    const base = [
      { label: 'Days of inventory', value: daysOfInventory == null ? '—' : String(daysOfInventory) },
      { label: 'Inventory value', value: formatMoney(src.inventory_value) },
      { label: 'Units on hand', value: (src.total_units ?? 0).toLocaleString() },
      { label: 'Units sold (30d)', value: (src.units_sold_30d ?? 0).toLocaleString() },
    ]
    if (!selectedLocation.value && overview.value) {
      base.push(
        { label: 'Dead / overstocked', value: `${overview.value.dead_stock_count} / ${overview.value.overstocked_count}` },
        { label: 'Active products', value: (overview.value.total_active_products ?? 0).toLocaleString() },
      )
    }
    return base
  })

  // --- Overview charts -------------------------------------------------------
  const valueByLocationOption = computed(() => {
    const top = locations.value
      .toSorted((a, b) => a.inventory_value - b.inventory_value)
      .slice(-12)
    return {
      grid: { top: 8, right: 56, bottom: 24, left: 8, containLabel: true },
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        axisPointer: { type: 'shadow' },
        valueFormatter: v => formatMoney(v),
      },
      xAxis: {
        type: 'value',
        ...valAxis.value,
        axisLabel: { ...valAxis.value.axisLabel, formatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}` },
      },
      yAxis: { type: 'category', data: top.map(l => l.name), ...catAxis.value },
      series: [{
        type: 'bar',
        barWidth: '55%',
        label: {
          show: true,
          position: 'right',
          color: textColor.value,
          fontSize: 11,
          formatter: p => `$${(p.value / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
        },
        itemStyle: { color: primary.value, borderRadius: [0, 3, 3, 0] },
        data: top.map(l => l.inventory_value),
      }],
    }
  })

  function daysBarColor (days) {
    if (days < 3) return negative.value
    if (days < 14) return warning.value
    return primary.value
  }
  const daysByLocationOption = computed(() => {
    const withDays = locations.value
      .filter(l => l.days_of_inventory != null)
      .toSorted((a, b) => a.days_of_inventory - b.days_of_inventory)
    return {
      grid: { top: 8, right: 40, bottom: 24, left: 8, containLabel: true },
      tooltip: { trigger: 'axis', appendTo: 'body', axisPointer: { type: 'shadow' } },
      xAxis: { type: 'value', ...valAxis.value },
      yAxis: { type: 'category', data: withDays.map(l => l.name), inverse: true, ...catAxis.value },
      series: [{
        type: 'bar',
        barWidth: '55%',
        label: { show: true, position: 'right', color: textColor.value, fontSize: 11 },
        data: withDays.map(l => ({ value: l.days_of_inventory, itemStyle: { color: daysBarColor(l.days_of_inventory) } })),
      }],
    }
  })

  // --- By Location table ------------------------------------------------
  const locationTypeFilter = ref('all')
  const locationSearch = ref('')
  const filteredLocations = computed(() =>
    locationTypeFilter.value === 'all'
      ? locations.value
      : locations.value.filter(l => l.type === locationTypeFilter.value),
  )
  const locationHeaders = [
    { title: 'Location', key: 'name' },
    { title: 'Type', key: 'type' },
    { title: 'Units on hand', key: 'total_units', align: 'end' },
    { title: 'Inventory value', key: 'inventory_value', align: 'end' },
    { title: 'Sold (30d)', key: 'units_sold_30d', align: 'end' },
    { title: 'Days of inventory', key: 'days_of_inventory', align: 'end' },
  ]

  // --- Low stock -----------------------------------------------------------
  const lowStockApiURL = `${API_BASE}/low_stock`
  const lowStockStatusFilter = ref('')
  const lowStockSearch = ref('')
  const lowStockSummary = ref(null)
  const { fetchSummary: fetchLowStockSummary } = useLowStockAlerts()
  async function loadLowStockSummary () {
    lowStockSummary.value = await fetchLowStockSummary().catch(() => null)
  }
  const lowStockHeaders = [
    { title: 'Product', key: 'name' },
    { title: 'Code', key: 'code' },
    { title: 'On hand', key: 'on_hand', align: 'end' },
    { title: 'Threshold', key: 'effective_threshold', align: 'end' },
  ]

  // --- Slow movers -----------------------------------------------------------
  const slowMoversApiURL = `${API_BASE}/stock_health/slow_movers`
  const slowMoverCategoryFilter = ref('')
  const slowMoverSearch = ref('')
  const slowMoverHeaders = [
    { title: 'Product', key: 'name' },
    { title: 'Code', key: 'code' },
    { title: 'On hand', key: 'on_hand', align: 'end' },
    { title: 'Sold (90d)', key: 'units_sold_90d', align: 'end' },
    { title: 'Days of inventory', key: 'days_of_inventory', align: 'end' },
    { title: 'Category', key: 'category' },
  ]

  // --- Movements -----------------------------------------------------------
  const movementsApiURL = `${API_BASE}/stock_movements`
  const movementReasonFilter = ref(null)
  const movementFromDate = ref('')
  const movementToDate = ref('')
  const reasonOptions = STOCK_MOVEMENT_REASONS.map(r => ({ title: r.replaceAll('_', ' '), value: r }))
  const movementQueryParams = computed(() => ({
    inventory_id: selectedLocationId.value,
    reason: movementReasonFilter.value,
    from_date: movementFromDate.value ? `${movementFromDate.value}T00:00:00Z` : null,
    to_date: movementToDate.value ? `${movementToDate.value}T23:59:59Z` : null,
  }))
  const movementHeaders = [
    { title: 'Date', key: 'created_at' },
    { title: 'Location', key: 'inventory_name' },
    { title: 'Product', key: 'variant' },
    { title: 'Reason', key: 'reason' },
    { title: 'Qty', key: 'quantity', align: 'end' },
    { title: 'Before → After', key: 'change' },
    { title: 'Note', key: 'note' },
  ]
</script>

<style scoped>
.ia-notice {
  flex: 0 0 auto;
}
.chart-card {
  padding: 12px 14px 4px;
  margin-bottom: 8px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.chart-card__title {
  font-size: 0.9rem;
  font-weight: 600;
  margin-bottom: 2px;
}
.chart-card__subtitle {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 4px;
}
.chart {
  height: 280px;
  width: 100%;
}
</style>
