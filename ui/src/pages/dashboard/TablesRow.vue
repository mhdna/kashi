<template>
  <div class="tables-row d-flex ga-3 h-100" style="flex: 1 1 0; min-height: 0">
    <DashboardList
      ref="salesList"
      :api-url="salesUrl"
      icon="mdi-cart-check"
      root-key="sales"
      style="flex: 1 1 0; min-width: 0"
      title="Sales"
    >
      <template #row="{ item }">
        <InitialsAvatar :name="item.client" :show-name="false" />
        <v-chip
          class="ms-2"
          :color="item.source === 'branch' ? 'primary' : 'secondary'"
          label
          size="x-small"
          variant="tonal"
        >
          {{ item.source }}
        </v-chip>
        <v-spacer />
        <span class="text-body-2 font-weight-medium me-2">{{ money(item.total) }}</span>
        <v-chip size="x-small" variant="tonal">{{ dateChip(item.occurred_at) }}</v-chip>
      </template>
      <template #details="{ item }">
        <DetailList
          :rows="[
            ['Source', item.source],
            ['Client', item.client],
            ['Amount', money(item.total)],
            ['Currency', item.currency],
            ['Date', dateFull(item.occurred_at)],
          ]"
        />
      </template>
    </DashboardList>

    <DashboardList
      ref="purchasesList"
      :api-url="purchasesUrl"
      icon="mdi-truck-delivery"
      root-key="purchases"
      style="flex: 1 1 0; min-width: 0"
      title="Purchases"
    >
      <template #row="{ item }">
        <span class="text-body-2 text-truncate">{{ item.supplier }}</span>
        <v-spacer />
        <span class="text-body-2 font-weight-medium me-2">{{ money(item.total) }}</span>
        <v-chip size="x-small" variant="tonal">{{ dateChip(item.purchased_at) }}</v-chip>
      </template>
      <template #details="{ item }">
        <DetailList
          :rows="[
            ['Supplier', item.supplier],
            ['Items', item.items],
            ['Amount', money(item.total)],
            ['Currency', item.currency],
            ['Date', dateFull(item.purchased_at)],
          ]"
        />
      </template>
    </DashboardList>

    <DashboardList
      ref="expensesList"
      :api-url="expensesUrl"
      icon="mdi-receipt-text-check"
      root-key="expenses"
      style="flex: 1 1 0; min-width: 0"
      title="Recent Expenses"
    >
      <template #row="{ item }">
        <span class="text-body-2 text-truncate">{{ item.category || 'Uncategorised' }}</span>
        <v-spacer />
        <span class="text-body-2 font-weight-medium me-2">{{ money(item.amount) }}</span>
        <v-chip class="me-1" size="x-small" variant="tonal">{{ dateChip(item.occurred_at) }}</v-chip>
        <ExpenseReceiptThumb :image-id="item.image_id" />
      </template>
      <template #details="{ item }">
        <DetailList
          :rows="[
            ['Source', item.source],
            ['Description', item.description],
            ['Category', item.category],
            ['Amount', money(item.amount)],
            ['Currency', item.currency],
            ['Date', dateFull(item.occurred_at)],
          ]"
        />
      </template>
    </DashboardList>
  </div>
</template>

<script setup>
  import { computed, ref } from 'vue'
  import { useDashboardLiveEvents } from '@/composables/useDashboardLiveEvents'
  import { API_BASE } from '@/config'
  import { useIncomeFilters } from './composables/useIncomeFilters'
  import DashboardList from './DashboardList.vue'
  import DetailList from './DetailList.vue'
  import ExpenseReceiptThumb from './ExpenseReceiptThumb.vue'
  import InitialsAvatar from './InitialsAvatar.vue'

  const { dashboardQuery } = useIncomeFilters()

  // Reactive: dashboardQuery() reads the shared filter refs, so each URL
  // recomputes on any filter change and DashboardList re-fetches from page 0.
  const salesUrl = computed(() => `${API_BASE}/dashboard/sales?${dashboardQuery()}`)
  const purchasesUrl = computed(() => `${API_BASE}/dashboard/purchases?${dashboardQuery()}`)
  const expensesUrl = computed(() => `${API_BASE}/dashboard/expenses?${dashboardQuery()}`)




  const salesList = ref(null)
  const purchasesList = ref(null)
  const expensesList = ref(null)

  // Filter changes reset the lists via their reactive api-url; a branch
  // sync push has to nudge them.
  useDashboardLiveEvents(() => {
    salesList.value?.reset()
    purchasesList.value?.reset()
    expensesList.value?.reset()
  })

  function money (cents) {
    const n = Number(cents || 0) / 100
    return `${Number.isInteger(n) ? n : n.toFixed(2)}$`
  }
  function dateChip (value) {
    return value ? new Date(value).toLocaleDateString([], { day: '2-digit', month: 'short' }) : '—'
  }
  function dateFull (value) {
    return value ? new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) : '—'
  }
</script>

<style scoped>
/* Three lists side by side only fit a desktop; below md they stack, each
   with a fixed height so its virtual scroller still has room to scroll. */
@media (max-width: 959.98px) {
  .tables-row {
    flex: 0 0 auto !important;
    flex-direction: column;
    height: auto !important;
  }
  .tables-row > * {
    flex: 0 0 360px !important;
  }
}
</style>
