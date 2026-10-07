<template>
  <div class="product-revenues d-flex flex-column fill-height" style="overflow-y: auto">
    <!-- Header / controls -->
    <div class="d-flex align-center flex-wrap ga-3 pa-4 pb-2">
      <v-icon icon="mdi-package-variant" />
      <div class="text-h6 font-weight-medium">Product Revenues</div>
      <v-spacer />
      <v-select
        v-model="rangeDays"
        density="compact"
        hide-details
        :items="RANGE_OPTIONS"
        style="max-width: 170px"
        variant="outlined"
      />
      <v-select
        v-model="selectedCategory"
        clearable
        density="compact"
        hide-details
        :items="categoryOptions"
        label="Category"
        style="min-width: 160px; max-width: 200px"
        variant="outlined"
      />
      <v-select
        v-model="selectedSubcategory"
        clearable
        density="compact"
        hide-details
        :items="subcategoryOptions"
        label="Sub-category"
        style="min-width: 160px; max-width: 200px"
        variant="outlined"
      />
      <v-select
        v-model="selectedBrand"
        clearable
        density="compact"
        hide-details
        :items="brandOptions"
        label="Brand"
        style="min-width: 160px; max-width: 200px"
        variant="outlined"
      />
      <v-select
        v-if="tab === 'pos'"
        v-model="selectedBranch"
        clearable
        density="compact"
        hide-details
        :items="branchOptions"
        label="Branch"
        style="min-width: 160px; max-width: 200px"
        variant="outlined"
      />
    </div>

    <v-tabs v-model="tab" class="px-4" color="primary">
      <v-tab value="pos">Branches</v-tab>
      <v-tab value="kashi">Wholesale</v-tab>
    </v-tabs>
    <v-divider />

    <v-window v-model="tab" class="flex-grow-1" style="overflow-y: auto">
      <v-window-item value="kashi">
        <v-container class="pa-4 pt-2" fluid>
          <ProductRevenuePanel
            :daily="dailyRevenue"
            :error="error"
            :loading="loading"
            :products="products"
          />
        </v-container>
      </v-window-item>

      <v-window-item value="pos">
        <v-container class="pa-4 pt-2" fluid>
          <ProductRevenuePanel
            :daily="posDailyRevenue"
            :error="posError"
            :loading="posLoading"
            :products="posProducts"
            revenue-note="The branch tills don't yet report the exact price charged per line item centrally, so revenue here is an ESTIMATE: quantity sold × each SKU's current catalog price, not the actual transaction total. Units sold and branch/category/brand breakdowns are real."
          >
            <template #extra-charts>
              <v-row dense>
                <v-col cols="12">
                  <v-card class="chart-card" variant="flat">
                    <div class="chart-card__title">Revenue by branch</div>
                    <v-chart autoresize class="chart" :option="branchOption" />
                  </v-card>
                </v-col>
              </v-row>
            </template>
          </ProductRevenuePanel>
        </v-container>
      </v-window-item>
    </v-window>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import VChart from 'vue-echarts'
  import { useTheme } from 'vuetify'
  import ProductRevenuePanel from '@/components/Products/ProductRevenuePanel.vue'
  import { useAttributes } from '@/composables/useAttributes'
  import { useBranches } from '@/composables/useBranches'
  import { useProductRevenues } from '@/composables/useProductRevenues'

  const theme = useTheme()
  const c = computed(() => theme.current.value.colors)
  const isDark = computed(() => theme.current.value.dark)
  const primary = computed(() => c.value.primary)
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

  const tab = ref('pos')

  const RANGE_OPTIONS = [
    { title: 'Last 30 days', value: 30 },
    { title: 'Last 90 days', value: 90 },
    { title: 'Last 6 months', value: 182 },
    { title: 'Last 12 months', value: 365 },
  ]
  const rangeDays = ref(90)
  const selectedCategory = ref(null)
  const selectedSubcategory = ref(null)
  const selectedBrand = ref(null)
  const selectedBranch = ref(null)

  const { valuesByType, fetchAll: fetchAttributes } = useAttributes()
  const { branches, fetchBranches } = useBranches()
  const {
    listProductRevenue, listProductRevenueDaily,
    listBranchProductRevenue, listBranchProductRevenueDaily, listBranchRevenueByBranch,
  } = useProductRevenues()

  function toOptions (attributeName) {
    return computed(() =>
      (valuesByType.value[attributeName] ?? []).map(v => ({ title: v.value, value: v.id })),
    )
  }
  const categoryOptions = toOptions('category')
  const subcategoryOptions = toOptions('sub-category')
  const brandOptions = toOptions('brand')
  const branchOptions = computed(() =>
    [...branches.value].toSorted((a, b) => a.name.localeCompare(b.name)).map(b => ({ title: b.name, value: b.id })),
  )

  function ymd (d) {
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }
  const windowBounds = computed(() => {
    const today = new Date()
    const from = new Date(today)
    from.setDate(from.getDate() - (rangeDays.value - 1))
    return { from: ymd(from), to: ymd(today) }
  })

  const baseFilters = computed(() => ({
    from: windowBounds.value.from,
    to: windowBounds.value.to,
    categoryValueId: selectedCategory.value,
    subcategoryValueId: selectedSubcategory.value,
    brandValueId: selectedBrand.value,
  }))
  const posFilters = computed(() => ({ ...baseFilters.value, branchId: selectedBranch.value }))

  // --- Kashi (admin) sales --------------------------------------------------
  const loading = ref(false)
  const error = ref('')
  const products = ref([])
  const dailyRevenue = ref([])

  async function loadKashi () {
    loading.value = true
    error.value = ''
    try {
      const [productRows, dayRows] = await Promise.all([
        listProductRevenue(baseFilters.value),
        listProductRevenueDaily(baseFilters.value),
      ])
      products.value = productRows
      dailyRevenue.value = dayRows
    } catch (error_) {
      error.value = error_.message
    } finally {
      loading.value = false
    }
  }

  // --- Kashi-POS (branch) sales ----------------------------------------------
  const posLoading = ref(false)
  const posError = ref('')
  const posProducts = ref([])
  const posDailyRevenue = ref([])
  const revenueByBranch = ref([])

  async function loadPos () {
    posLoading.value = true
    posError.value = ''
    try {
      const [productRows, dayRows, branchRows] = await Promise.all([
        listBranchProductRevenue(posFilters.value),
        listBranchProductRevenueDaily(posFilters.value),
        listBranchRevenueByBranch(baseFilters.value),
      ])
      posProducts.value = productRows
      posDailyRevenue.value = dayRows
      revenueByBranch.value = branchRows
    } catch (error_) {
      posError.value = error_.message
    } finally {
      posLoading.value = false
    }
  }

  onMounted(async () => {
    await Promise.all([fetchAttributes(), fetchBranches()])
    loadKashi()
    loadPos()
  })
  watch([rangeDays, selectedCategory, selectedSubcategory, selectedBrand], () => {
    loadKashi()
    loadPos()
  })
  watch(selectedBranch, loadPos)

  const branchOption = computed(() => {
    const totals = revenueByBranch.value
      .toSorted((a, b) => a.revenue - b.revenue)
    return {
      grid: { top: 8, right: 56, bottom: 24, left: 8, containLabel: true },
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        axisPointer: { type: 'shadow' },
        valueFormatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
      },
      xAxis: {
        type: 'value',
        ...valAxis.value,
        axisLabel: { ...valAxis.value.axisLabel, formatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}` },
      },
      yAxis: { type: 'category', data: totals.map(t => t.branch_name), ...catAxis.value },
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
        data: totals.map(t => t.revenue),
      }],
    }
  })
</script>

<style scoped>
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
.chart {
  height: 280px;
  width: 100%;
}
</style>
